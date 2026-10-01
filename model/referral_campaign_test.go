package model

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func createReferralCampaignTestUser(t *testing.T, quota int) User {
	t.Helper()
	user := User{
		Username: "referral-" + common.GetRandomString(12),
		Password: "unused-password-hash",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
		Quota:    quota,
		AffCode:  "referral-" + common.GetRandomString(8),
	}
	require.NoError(t, DB.Create(&user).Error)
	return user
}

func createReferralCampaignTestCampaign(t *testing.T, now int64, direction, spendBasis string) ReferralCampaign {
	t.Helper()
	campaign := ReferralCampaign{
		Name:               "Campaign " + common.GetRandomString(6),
		Priority:           10,
		StartAt:            now - 60,
		EndAt:              now + 3600,
		Direction:          direction,
		SpendBasis:         spendBasis,
		RequiredFriends:    2,
		PaidThresholdQuota: 10,
		BaseBps:            200,
		InviterBps:         500,
		InviteePoolBps:     400,
	}
	require.NoError(t, SaveReferralCampaign(&campaign))
	assert.False(t, campaign.Enabled, "new campaigns are inactive")
	require.NoError(t, DB.Model(&ReferralCampaign{}).Where("id = ?", campaign.Id).Update("enabled", true).Error)
	campaign.Enabled = true
	return campaign
}

func TestReferralUnlockUsesStrictPaidThresholdAndSharesInviteePool(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix()
	campaign := createReferralCampaignTestCampaign(t, now, ReferralDirectionBoth, ReferralSpendAll)
	inviter := createReferralCampaignTestUser(t, 0)
	friendOne := createReferralCampaignTestUser(t, 20_000)
	friendTwo := createReferralCampaignTestUser(t, 20_000)
	require.NoError(t, RegisterReferralFriend(friendOne.Id, inviter.Id, now))
	require.NoError(t, RegisterReferralFriend(friendTwo.Id, inviter.Id, now))

	require.NoError(t, RecordReferralPaidQuota("paid-one-exact", friendOne.Id, 10, now))
	var exact ReferralFriend
	require.NoError(t, DB.First(&exact, "user_id = ?", friendOne.Id).Error)
	assert.EqualValues(t, 10, exact.PaidQuota)

	require.NoError(t, CreditReferralSpend("wallet", "friend-one-base", friendOne.Id, 1_000, 0, now))
	require.NoError(t, DB.First(&exact, "user_id = ?", friendOne.Id).Error)
	unlocked, err := referralUnlocked(DB, &campaign, inviter.Id, now)
	require.NoError(t, err)
	assert.False(t, unlocked)
	var inviterAccount ReferralAccount
	require.NoError(t, DB.First(&inviterAccount, "user_id = ?", inviter.Id).Error)
	assert.EqualValues(t, 20, inviterAccount.Balance, "base rate applies before unlock")

	require.NoError(t, RecordReferralPaidQuota("paid-one-over", friendOne.Id, 1, now))
	require.NoError(t, RecordReferralPaidQuota("paid-two-over", friendTwo.Id, 11, now))
	require.NoError(t, CreditReferralSpend("wallet", "friend-one-unlocked", friendOne.Id, 1_000, 0, now))
	require.NoError(t, CreditReferralSpend("wallet", "inviter-pool", inviter.Id, 10_000, 0, now))

	require.NoError(t, DB.First(&inviterAccount, "user_id = ?", inviter.Id).Error)
	assert.EqualValues(t, 70, inviterAccount.Balance, "only future qualifying friend spend uses the inviter rate")
	var friendOneAccount, friendTwoAccount ReferralAccount
	require.NoError(t, DB.First(&friendOneAccount, "user_id = ?", friendOne.Id).Error)
	require.NoError(t, DB.First(&friendTwoAccount, "user_id = ?", friendTwo.Id).Error)
	assert.EqualValues(t, 200, friendOneAccount.Balance)
	assert.EqualValues(t, 200, friendTwoAccount.Balance, "the 4% pool is split between qualified friends")

	var snapshot ReferralLedger
	require.NoError(t, DB.Where("source_id = ? AND user_id = ?", "inviter-pool", friendOne.Id).
		First(&snapshot).Error)
	assert.Equal(t, campaign.Name, mustCampaignSnapshotName(t, snapshot.CampaignSnapshot))
}

func mustCampaignSnapshotName(t *testing.T, raw string) string {
	t.Helper()
	var campaign ReferralCampaign
	require.NoError(t, common.Unmarshal([]byte(raw), &campaign))
	return campaign.Name
}

func TestReferralSpendBasisSelectsWalletSubscriptionAndPaidWalletConsumption(t *testing.T) {
	testCases := []struct {
		name       string
		basis      string
		sourceKind string
		quota      int64
		paidQuota  int64
		wantReward int64
	}{
		{name: "wallet excludes subscription", basis: ReferralSpendWallet, sourceKind: "subscription", quota: 1_000},
		{name: "all includes subscription", basis: ReferralSpendAll, sourceKind: "subscription", quota: 1_000, wantReward: 20},
		{name: "paid wallet counts only tracked paid quota", basis: ReferralSpendPaidWallet, sourceKind: "wallet", quota: 1_000, paidQuota: 500, wantReward: 10},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			truncateTables(t)
			now := time.Now().Unix()
			campaign := createReferralCampaignTestCampaign(t, now, ReferralDirectionInviter, testCase.basis)
			inviter := createReferralCampaignTestUser(t, 0)
			friend := createReferralCampaignTestUser(t, 2_000)
			require.NoError(t, RegisterReferralFriend(friend.Id, inviter.Id, now))
			sourceId := common.GetRandomString(8) + testCase.name
			require.NoError(t, CreditReferralSpend(testCase.sourceKind, sourceId, friend.Id,
				testCase.quota, testCase.paidQuota, now))

			var ledger []ReferralLedger
			require.NoError(t, DB.Where("campaign_id = ?", campaign.Id).Find(&ledger).Error)
			if testCase.wantReward == 0 {
				assert.Empty(t, ledger)
				return
			}
			require.Len(t, ledger, 1)
			assert.Equal(t, testCase.wantReward, ledger[0].Amount)
		})
	}
}

func TestReferralRefundReversesOriginalRewardAndRecordsWalletDebt(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix()
	campaign := createReferralCampaignTestCampaign(t, now, ReferralDirectionInviter, ReferralSpendWallet)
	require.NoError(t, SaveReferralPolicy(&ReferralPolicy{MinTransferQuota: 1, BucketMinutes: 60}))
	inviter := createReferralCampaignTestUser(t, 0)
	friend := createReferralCampaignTestUser(t, 20_000)
	require.NoError(t, RegisterReferralFriend(friend.Id, inviter.Id, now))
	require.NoError(t, CreditReferralSpend("wallet", "refundable-spend", friend.Id, 1_000, 0, now))

	_, err := TransferReferralBalance(inviter.Id, 20)
	require.NoError(t, err)
	require.NoError(t, RefundReferralSpend("wallet", "refundable-spend", 0))
	require.NoError(t, RefundReferralSpend("wallet", "refundable-spend", 0))

	var account ReferralAccount
	require.NoError(t, DB.First(&account, "user_id = ?", inviter.Id).Error)
	assert.EqualValues(t, -20, account.Balance, "already transferred reward becomes future-rebate debt")
	assert.EqualValues(t, 20, account.LifetimeEarned, "lifetime earnings remain gross")

	var entries []ReferralLedger
	require.NoError(t, DB.Where("source_id = ?", "refundable-spend").Order("id asc").Find(&entries).Error)
	require.Len(t, entries, 2)
	assert.EqualValues(t, 20, entries[0].Amount)
	assert.EqualValues(t, -20, entries[1].Amount)
	assert.Equal(t, campaign.Name, mustCampaignSnapshotName(t, entries[1].CampaignSnapshot))
}

func TestReferralWalletSpendTracksPaidSourceAndRefundsItBeforeLegacyQuota(t *testing.T) {
	truncateTables(t)
	user := createReferralCampaignTestUser(t, 200)
	require.NoError(t, DB.Create(&ReferralPaidWallet{UserId: user.Id, PaidRemaining: 100}).Error)

	paid, err := SetReferralWalletSpend(user.Id, "wallet-spend", 150, true)
	require.NoError(t, err)
	assert.EqualValues(t, 50, paid)
	require.NoError(t, DB.First(&user, user.Id).Error)
	assert.Equal(t, 50, user.Quota)

	paid, err = SetReferralWalletSpend(user.Id, "wallet-spend", 80, false)
	require.NoError(t, err)
	assert.Zero(t, paid)
	require.NoError(t, DB.First(&user, user.Id).Error)
	assert.Equal(t, 120, user.Quota)

	var wallet ReferralPaidWallet
	require.NoError(t, DB.First(&wallet, "user_id = ?", user.Id).Error)
	assert.EqualValues(t, 100, wallet.PaidRemaining)
}

func TestReferralOverlapAttributionAndPriceEditsApplyOnlyToNewSettlements(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix()
	first := createReferralCampaignTestCampaign(t, now, ReferralDirectionInviter, ReferralSpendWallet)
	second := createReferralCampaignTestCampaign(t, now, ReferralDirectionInviter, ReferralSpendWallet)
	second.StartAt = now - 30
	require.NoError(t, SaveReferralCampaign(&second))
	third := createReferralCampaignTestCampaign(t, now, ReferralDirectionInviter, ReferralSpendWallet)
	third.StartAt = second.StartAt
	require.NoError(t, SaveReferralCampaign(&third))
	inviter := createReferralCampaignTestUser(t, 0)
	friend := createReferralCampaignTestUser(t, 5_000)
	require.NoError(t, RegisterReferralFriend(friend.Id, inviter.Id, now))
	var attribution ReferralFriend
	require.NoError(t, DB.First(&attribution, "user_id = ?", friend.Id).Error)
	assert.Equal(t, third.Id, attribution.CampaignId, "latest start and then highest ID wins")

	require.NoError(t, CreditReferralSpend("wallet", "before-price-edit", friend.Id, 1_000, 0, now))
	third.BaseBps = 600
	require.NoError(t, SaveReferralCampaign(&third))
	require.NoError(t, CreditReferralSpend("wallet", "before-price-edit", friend.Id, 1_000, 0, now+1))
	require.NoError(t, CreditReferralSpend("wallet", "after-price-edit", friend.Id, 1_000, 0, now+1))
	entries, err := ListReferralLedger(inviter.Id, 0, 100)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.EqualValues(t, 60, entries[0].Amount)
	assert.EqualValues(t, 20, entries[1].Amount)
	assert.Equal(t, 200, entries[1].RateBps, "history retains the original rate")

	first.Priority = 100
	require.NoError(t, SaveReferralCampaign(&first))
	require.NoError(t, CreditReferralSpend("wallet", "exclusive-winner", friend.Id, 1_000, 0, now+2))
	entries, err = ListReferralLedger(inviter.Id, 0, 100)
	require.NoError(t, err)
	assert.Len(t, entries, 2, "a losing campaign cannot award attributed friends")
	require.NoError(t, RecordReferralPaidQuota("losing-campaign-payment", friend.Id, 100, now+2))
	require.NoError(t, DB.First(&attribution, "user_id = ?", friend.Id).Error)
	assert.EqualValues(t, 100, attribution.PaidQuota, "online credits count in the friend's attributed period")
}

func TestReferralDirectionsAwardOnlySelectedRecipients(t *testing.T) {
	for _, testCase := range []struct {
		direction string
		inviter   int64
		invitee   int64
	}{
		{ReferralDirectionInviter, 20, 0},
		{ReferralDirectionInvitee, 0, 20},
		{ReferralDirectionBoth, 20, 20},
	} {
		t.Run(testCase.direction, func(t *testing.T) {
			truncateTables(t)
			now := time.Now().Unix()
			createReferralCampaignTestCampaign(t, now, testCase.direction, ReferralSpendWallet)
			inviter := createReferralCampaignTestUser(t, 1_000)
			friend := createReferralCampaignTestUser(t, 1_000)
			require.NoError(t, RegisterReferralFriend(friend.Id, inviter.Id, now))
			require.NoError(t, CreditReferralSpend("wallet", "friend-direction", friend.Id, 1_000, 0, now))
			require.NoError(t, CreditReferralSpend("wallet", "inviter-direction", inviter.Id, 1_000, 0, now))
			for _, recipient := range []struct {
				userId int
				amount int64
			}{
				{inviter.Id, testCase.inviter}, {friend.Id, testCase.invitee},
			} {
				var amount int64
				require.NoError(t, DB.Model(&ReferralLedger{}).Where("user_id = ?", recipient.userId).
					Select("COALESCE(SUM(amount), 0)").Scan(&amount).Error)
				assert.Equal(t, recipient.amount, amount)
			}
		})
	}
}

func TestReferralPaidCallbacksExcludeManualAndGiftCredits(t *testing.T) {
	truncateTables(t)
	oldUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1
	t.Cleanup(func() { common.QuotaPerUnit = oldUnit })
	now := time.Now().Unix()
	createReferralCampaignTestCampaign(t, now, ReferralDirectionInviter, ReferralSpendWallet)
	inviter := createReferralCampaignTestUser(t, 0)
	friend := createReferralCampaignTestUser(t, 0)
	require.NoError(t, RegisterReferralFriend(friend.Id, inviter.Id, now))
	for _, order := range []TopUp{
		{UserId: friend.Id, TradeNo: "online-credit", Amount: 11, Money: 11, Status: common.TopUpStatusPending, PaymentProvider: PaymentProviderEpay, PaymentMethod: "alipay"},
		{UserId: friend.Id, TradeNo: "manual-credit", Amount: 50, Money: 50, Status: common.TopUpStatusPending, PaymentProvider: PaymentProviderEpay, PaymentMethod: "alipay"},
		{UserId: friend.Id, TradeNo: "online-gift-credit", Amount: 50, Money: 0, Status: common.TopUpStatusPending, PaymentProvider: PaymentProviderEpay, PaymentMethod: "alipay"},
	} {
		require.NoError(t, DB.Create(&order).Error)
	}
	done, err := RechargeEpay("online-credit", "alipay", "127.0.0.1")
	require.NoError(t, err)
	assert.False(t, done)
	done, err = RechargeEpay("online-credit", "alipay", "127.0.0.1")
	require.NoError(t, err)
	assert.True(t, done)
	require.NoError(t, ManualCompleteTopUp("manual-credit", "127.0.0.1"))
	_, err = RechargeEpay("online-gift-credit", "alipay", "127.0.0.1")
	require.NoError(t, err)
	require.NoError(t, IncreaseUserQuota(friend.Id, 100, true))
	redemption := Redemption{Name: "Referral gift credit", Key: common.GetRandomString(32), Quota: 25,
		Status: common.RedemptionCodeStatusEnabled, MaxUses: 1, MaxUsesPerUser: 1}
	require.NoError(t, DB.Create(&redemption).Error)
	_, err = Redeem(redemption.Key, friend.Id)
	require.NoError(t, err)
	var attribution ReferralFriend
	require.NoError(t, DB.First(&attribution, "user_id = ?", friend.Id).Error)
	assert.EqualValues(t, 11, attribution.PaidQuota)
	var wallet ReferralPaidWallet
	require.NoError(t, DB.First(&wallet, "user_id = ?", friend.Id).Error)
	assert.EqualValues(t, 11, wallet.PaidRemaining)
	var count int64
	require.NoError(t, DB.Model(&ReferralLedger{}).Count(&count).Error)
	assert.Zero(t, count, "top-ups never generate rebates")
}

func TestReferralConcurrentTransfersPreserveBalanceAndCampaignProvenance(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix()
	campaign := createReferralCampaignTestCampaign(t, now, ReferralDirectionInviter, ReferralSpendWallet)
	require.NoError(t, SaveReferralPolicy(&ReferralPolicy{MinTransferQuota: 1, BucketMinutes: 7}))
	inviter := createReferralCampaignTestUser(t, 0)
	friend := createReferralCampaignTestUser(t, 5_000)
	require.NoError(t, RegisterReferralFriend(friend.Id, inviter.Id, now))
	require.NoError(t, CreditReferralSpend("wallet", "concurrent-withdraw-source", friend.Id, 5_000, 0, now))
	start := make(chan struct{})
	results := make(chan error, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Go(func() {
			<-start
			_, err := TransferReferralBalance(inviter.Id, 75)
			results <- err
		})
	}
	close(start)
	workers.Wait()
	close(results)
	succeeded := 0
	for err := range results {
		if err == nil {
			succeeded++
		}
	}
	assert.Equal(t, 1, succeeded)
	var account ReferralAccount
	require.NoError(t, DB.First(&account, "user_id = ?", inviter.Id).Error)
	assert.EqualValues(t, 25, account.Balance)
	transfers, err := ListReferralTransfers(inviter.Id, 0, 20)
	require.NoError(t, err)
	require.Len(t, transfers, 1)
	var sources []ReferralTransferSource
	require.NoError(t, common.UnmarshalJsonStr(transfers[0].SourceSnapshot, &sources))
	require.Len(t, sources, 1)
	assert.Equal(t, campaign.Id, sources[0].CampaignId)
	assert.EqualValues(t, 75, sources[0].Amount, "statistics buckets do not constrain transfer amounts")
}

func TestReferralPaidWalletPartialRefundReversesThePaidPortionFirst(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix()
	createReferralCampaignTestCampaign(t, now, ReferralDirectionInviter, ReferralSpendPaidWallet)
	inviter := createReferralCampaignTestUser(t, 0)
	friend := createReferralCampaignTestUser(t, 1_000)
	require.NoError(t, RegisterReferralFriend(friend.Id, inviter.Id, now))
	require.NoError(t, CreditReferralSpend("wallet", "paid-partial-refund", friend.Id, 1_000, 500, now))
	require.NoError(t, RefundReferralSpend("wallet", "paid-partial-refund", 500))
	var account ReferralAccount
	require.NoError(t, DB.First(&account, "user_id = ?", inviter.Id).Error)
	assert.Zero(t, account.Balance)
}

func TestReferralFreshSQLiteStartupMigrationIsIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "referral-fresh.db")), &gorm.Config{})
	require.NoError(t, err)
	previousDB, previousLogDB := DB, LOG_DB
	DB, LOG_DB = db, db
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		connection, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, connection.Close())
	})
	for range 2 {
		require.NoError(t, migrateDB())
	}
	var version string
	require.NoError(t, db.Raw("select sqlite_version()").Scan(&version).Error)
	t.Logf("SQLite version: %s; fresh startup migration ran twice", version)
	campaigns, err := ListReferralCampaigns()
	require.NoError(t, err)
	assert.Empty(t, campaigns, "startup never enables or backfills campaigns")
}

func TestReferralPartialRefundRetainsOriginalRateRounding(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix()
	createReferralCampaignTestCampaign(t, now, ReferralDirectionInviter, ReferralSpendWallet)
	inviter := createReferralCampaignTestUser(t, 0)
	friend := createReferralCampaignTestUser(t, 51)
	require.NoError(t, RegisterReferralFriend(friend.Id, inviter.Id, now))
	require.NoError(t, CreditReferralSpend("wallet", "rounding-refund", friend.Id, 51, 0, now))
	require.NoError(t, RefundReferralSpend("wallet", "rounding-refund", 50))
	var account ReferralAccount
	require.NoError(t, DB.First(&account, "user_id = ?", inviter.Id).Error)
	assert.EqualValues(t, 1, account.Balance, "50 quota at the original 2% rate still earns one unit")
	require.NoError(t, RefundReferralSpend("wallet", "rounding-refund", 49))
	require.NoError(t, DB.First(&account, "user_id = ?", inviter.Id).Error)
	assert.Zero(t, account.Balance)
}

func TestReferralLegacyTransferUsesConfiguredMinimumAndNetDebt(t *testing.T) {
	truncateTables(t)
	require.NoError(t, SaveReferralPolicy(&ReferralPolicy{MinTransferQuota: 300, BucketMinutes: 7}))
	user := createReferralCampaignTestUser(t, 0)
	require.NoError(t, DB.Model(&user).Update("aff_quota", 500).Error)
	require.NoError(t, DB.Create(&ReferralAccount{UserId: user.Id, Balance: -100}).Error)
	require.Error(t, user.TransferAffQuotaToQuota(200))
	require.Error(t, user.TransferAffQuotaToQuota(450), "legacy credits cannot bypass rebate debt")
	require.NoError(t, user.TransferAffQuotaToQuota(300))
	assert.Equal(t, 300, user.Quota)
	summary, err := GetReferralSummary(user.Id)
	require.NoError(t, err)
	assert.EqualValues(t, 100, summary.Balance)
	transfers, err := ListReferralTransfers(user.Id, 0, 20)
	require.NoError(t, err)
	require.Len(t, transfers, 1)
	assert.EqualValues(t, 300, transfers[0].LegacyPart)
}

func TestReferralCampaignUsesInclusiveStartAndExclusiveEnd(t *testing.T) {
	truncateTables(t)
	campaign := createReferralCampaignTestCampaign(t, 1_000, ReferralDirectionInviter, ReferralSpendWallet)
	campaign.StartAt, campaign.EndAt = 1_000, 1_100
	require.NoError(t, SaveReferralCampaign(&campaign))
	inviter := createReferralCampaignTestUser(t, 0)
	friend := createReferralCampaignTestUser(t, 1_000)
	require.NoError(t, RegisterReferralFriend(friend.Id, inviter.Id, 999))
	var count int64
	require.NoError(t, DB.Model(&ReferralFriend{}).Count(&count).Error)
	assert.Zero(t, count)
	require.NoError(t, RegisterReferralFriend(friend.Id, inviter.Id, 1_000))
	require.NoError(t, CreditReferralSpend("wallet", "start-boundary", friend.Id, 1_000, 0, 1_000))
	require.NoError(t, CreditReferralSpend("wallet", "end-boundary", friend.Id, 1_000, 0, 1_100))
	require.NoError(t, RecordReferralPaidQuota("end-boundary-payment", friend.Id, 100, 1_100))
	entries, err := ListReferralLedger(inviter.Id, 0, 20)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "start-boundary", entries[0].SourceId)
	var attribution ReferralFriend
	require.NoError(t, DB.First(&attribution, "user_id = ?", friend.Id).Error)
	assert.Zero(t, attribution.PaidQuota)
}

func TestReferralThresholdEditUnlocksImmediatelyWithoutHistoricalBackfill(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix()
	campaign := createReferralCampaignTestCampaign(t, now, ReferralDirectionBoth, ReferralSpendWallet)
	inviter := createReferralCampaignTestUser(t, 1_000)
	for range 2 {
		friend := createReferralCampaignTestUser(t, 1_000)
		require.NoError(t, RegisterReferralFriend(friend.Id, inviter.Id, now))
		require.NoError(t, RecordReferralPaidQuota(common.GetRandomString(12), friend.Id, 5, now))
		require.NoError(t, CreditReferralSpend("wallet", common.GetRandomString(12), friend.Id, 1_000, 0, now))
	}
	campaign.PaidThresholdQuota = 4
	require.NoError(t, SaveReferralCampaign(&campaign))
	var unlock ReferralUnlock
	require.NoError(t, DB.First(&unlock, "inviter_id = ? AND campaign_id = ?", inviter.Id, campaign.Id).Error)
	assert.Positive(t, unlock.UnlockedAt)
	var account ReferralAccount
	require.NoError(t, DB.First(&account, "user_id = ?", inviter.Id).Error)
	assert.EqualValues(t, 40, account.Balance, "only the two original base rebates remain")
	campaign.RequiredFriends = 10
	require.NoError(t, SaveReferralCampaign(&campaign))
	summary, err := GetReferralSummary(inviter.Id)
	require.NoError(t, err)
	assert.True(t, summary.Unlocked, "an already achieved unlock is retained for the period")
}

func TestReferralRefundBeforeCreditBlocksLateRewardCallback(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix()
	createReferralCampaignTestCampaign(t, now, ReferralDirectionInviter, ReferralSpendWallet)
	inviter := createReferralCampaignTestUser(t, 0)
	friend := createReferralCampaignTestUser(t, 1_000)
	require.NoError(t, RegisterReferralFriend(friend.Id, inviter.Id, now))
	require.NoError(t, RefundReferralSpend("wallet", "late-refund-credit", 0))
	require.NoError(t, CreditReferralSpend("wallet", "late-refund-credit", friend.Id, 1_000, 0, now))
	entries, err := ListReferralLedger(inviter.Id, 0, 20)
	require.NoError(t, err)
	assert.Empty(t, entries, "refunded consumption cannot earn a delayed rebate")
}

func TestReferralGenericWalletDebitConsumesUnpaidThenPaidFunds(t *testing.T) {
	for _, batched := range []bool{false, true} {
		t.Run(map[bool]string{false: "direct", true: "batch-enabled"}[batched], func(t *testing.T) {
			truncateTables(t)
			previous := common.BatchUpdateEnabled
			common.BatchUpdateEnabled = batched
			t.Cleanup(func() { common.BatchUpdateEnabled = previous })
			user := createReferralCampaignTestUser(t, 150)
			require.NoError(t, DB.Create(&ReferralPaidWallet{UserId: user.Id, PaidRemaining: 100}).Error)
			require.NoError(t, DecreaseUserQuota(user.Id, 75, false))
			require.NoError(t, DB.First(&user, user.Id).Error)
			assert.Equal(t, 75, user.Quota, "source-tracked debits commit with the wallet")
			var funds ReferralPaidWallet
			require.NoError(t, DB.First(&funds, "user_id = ?", user.Id).Error)
			assert.EqualValues(t, 75, funds.PaidRemaining)
		})
	}
}

// Set this to a disposable database produced by the released gateway, not a
// production database. The test retains its representative wallet/legacy data.
func TestReferralReleasedSQLiteUpgrade(t *testing.T) {
	path := os.Getenv("TEST_REFERRAL_UPGRADE_SQLITE")
	if path == "" {
		t.Skip("TEST_REFERRAL_UPGRADE_SQLITE is not configured")
	}
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	require.NoError(t, err)
	var user User
	require.NoError(t, db.First(&user, "username = ?", "refqa").Error)
	originalQuota := user.Quota
	require.NoError(t, db.Model(&User{}).Where("id = ?", user.Id).Updates(map[string]any{
		"aff_quota": 750_000, "aff_history": 1_000_000,
	}).Error)
	require.NoError(t, db.Create(&TopUp{
		UserId: user.Id, Amount: 20, Money: 20, TradeNo: "released-online-topup",
		PaymentProvider: PaymentProviderEpay, PaymentMethod: "alipay",
		Status: common.TopUpStatusSuccess, CreateTime: time.Now().Unix(), CompleteTime: time.Now().Unix(),
	}).Error)
	previousDB, previousLogDB := DB, LOG_DB
	DB, LOG_DB = db, db
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		connection, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, connection.Close())
	})
	for range 2 {
		require.NoError(t, migrateDB())
	}
	require.NoError(t, DB.First(&user, user.Id).Error)
	assert.Equal(t, originalQuota, user.Quota)
	assert.Equal(t, 750_000, user.AffQuota)
	assert.Equal(t, 1_000_000, user.AffHistoryQuota)
	summary, err := GetReferralSummary(user.Id)
	require.NoError(t, err)
	assert.EqualValues(t, 750_000, summary.Balance)
	assert.Nil(t, summary.Campaign)
	paid, err := SetReferralWalletSpend(user.Id, "released-mixed-wallet", 50, true)
	require.NoError(t, err)
	assert.Zero(t, paid, "old online top-ups and mixed balances are not backfilled as paid")
	_, err = SetReferralWalletSpend(user.Id, "released-mixed-wallet", 0, false)
	require.NoError(t, err)
	var version string
	require.NoError(t, DB.Raw("select sqlite_version()").Scan(&version).Error)
	t.Logf("SQLite version: %s; released database upgrade ran twice", version)
}
