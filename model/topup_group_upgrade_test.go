package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func configureTopUpGroupUpgradeTest(t *testing.T, groups map[string]float64, rules []operation_setting.TopUpGroupUpgradeRule) {
	t.Helper()

	previousGroups := ratio_setting.GetGroupRatioCopy()
	previousRules := operation_setting.GetTopUpGroupUpgradeRules()
	t.Cleanup(func() {
		previousGroupsJSON, err := common.Marshal(previousGroups)
		require.NoError(t, err)
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(string(previousGroupsJSON)))
		operation_setting.GetPaymentSetting().TopUpGroupUpgradeRules = previousRules
	})

	groupsJSON, err := common.Marshal(groups)
	require.NoError(t, err)
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(string(groupsJSON)))
	operation_setting.GetPaymentSetting().TopUpGroupUpgradeRules = rules
}

func createTopUpGroupUpgradeEpayOrder(t *testing.T, userID int, tradeNo string, amount int64, money float64) {
	t.Helper()
	require.NoError(t, DB.Create(&TopUp{
		UserId:          userID,
		Amount:          amount,
		Money:           money,
		TradeNo:         tradeNo,
		PaymentMethod:   "alipay",
		PaymentProvider: PaymentProviderEpay,
		CreateTime:      common.GetTimestamp(),
		Status:          common.TopUpStatusPending,
	}).Error)
}

func getUserGroupForTopUpGroupUpgradeTest(t *testing.T, userID int) string {
	t.Helper()
	var user User
	require.NoError(t, DB.Select("id", commonGroupCol).Where("id = ?", userID).First(&user).Error)
	return user.Group
}

func TestRechargeEpayAppliesFirstMatchingTopUpGroupUpgradeAndRefreshesCache(t *testing.T) {
	truncateTables(t)
	useUserCacheMiniRedis(t)

	oldQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1
	t.Cleanup(func() { common.QuotaPerUnit = oldQuotaPerUnit })
	configureTopUpGroupUpgradeTest(t, map[string]float64{
		"default": 1,
		"vip":     2,
		"svip":    3,
	}, []operation_setting.TopUpGroupUpgradeRule{
		{Type: operation_setting.TopUpGroupUpgradeRuleTypeSingle, Amount: "50.00", Group: "vip"},
		{Type: operation_setting.TopUpGroupUpgradeRuleTypeCumulative, Amount: "1.00", Group: "svip"},
	})

	user := insertUserForPaymentGuardTest(t, 801, 0)
	user.Group = "default"
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Update("group", user.Group).Error)
	require.NoError(t, populateUserCache(*user))
	createTopUpGroupUpgradeEpayOrder(t, user.Id, "EPAY-GROUP-PRIORITY", 1, 100)

	alreadyDone, err := RechargeEpay("EPAY-GROUP-PRIORITY", "alipay", "127.0.0.1")
	require.NoError(t, err)
	assert.False(t, alreadyDone)
	assert.Equal(t, "vip", getUserGroupForTopUpGroupUpgradeTest(t, user.Id))
	assert.Equal(t, 1, getUserQuotaForPaymentGuardTest(t, user.Id))

	cached, err := cacheGetUserBase(user.Id)
	require.NoError(t, err)
	assert.Equal(t, "vip", cached.Group)

	alreadyDone, err = RechargeEpay("EPAY-GROUP-PRIORITY", "alipay", "127.0.0.1")
	require.NoError(t, err)
	assert.True(t, alreadyDone)
	assert.Equal(t, "vip", getUserGroupForTopUpGroupUpgradeTest(t, user.Id))
	assert.Equal(t, 1, getUserQuotaForPaymentGuardTest(t, user.Id))
}

func TestRechargeEpayAppliesCumulativeTopUpGroupUpgradeIncludingCurrentPayment(t *testing.T) {
	truncateTables(t)

	oldQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1
	t.Cleanup(func() { common.QuotaPerUnit = oldQuotaPerUnit })
	configureTopUpGroupUpgradeTest(t, map[string]float64{
		"default": 1,
		"vip":     2,
	}, []operation_setting.TopUpGroupUpgradeRule{
		{Type: operation_setting.TopUpGroupUpgradeRuleTypeCumulative, Amount: "100.00", Group: "vip"},
	})

	user := insertUserForPaymentGuardTest(t, 802, 0)
	user.Group = "default"
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Update("group", user.Group).Error)
	require.NoError(t, DB.Create(&TopUp{
		UserId:          user.Id,
		Amount:          1,
		Money:           60,
		TradeNo:         "EPAY-GROUP-PREVIOUS",
		PaymentMethod:   "alipay",
		PaymentProvider: PaymentProviderEpay,
		CreateTime:      common.GetTimestamp(),
		CompleteTime:    common.GetTimestamp(),
		Status:          common.TopUpStatusSuccess,
	}).Error)
	// Subscription records and zero-payment rows must not contribute to the
	// cumulative top-up amount.
	require.NoError(t, DB.Create(&TopUp{
		UserId:       user.Id,
		Money:        1_000,
		TradeNo:      "SUBSCRIPTION-NOT-TOPUP",
		CreateTime:   common.GetTimestamp(),
		CompleteTime: common.GetTimestamp(),
		Status:       common.TopUpStatusSuccess,
	}).Error)
	require.NoError(t, DB.Create(&TopUp{
		UserId:       user.Id,
		Amount:       1,
		Money:        0,
		TradeNo:      "ZERO-PAYMENT-NOT-TOPUP",
		CreateTime:   common.GetTimestamp(),
		CompleteTime: common.GetTimestamp(),
		Status:       common.TopUpStatusSuccess,
	}).Error)
	createTopUpGroupUpgradeEpayOrder(t, user.Id, "EPAY-GROUP-CUMULATIVE", 1, 40)

	_, err := RechargeEpay("EPAY-GROUP-CUMULATIVE", "alipay", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, "vip", getUserGroupForTopUpGroupUpgradeTest(t, user.Id))
}

func TestRechargeEpayDoesNotUpgradeBelowSingleTopUpThreshold(t *testing.T) {
	truncateTables(t)

	oldQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1
	t.Cleanup(func() { common.QuotaPerUnit = oldQuotaPerUnit })
	configureTopUpGroupUpgradeTest(t, map[string]float64{
		"default": 1,
		"vip":     2,
	}, []operation_setting.TopUpGroupUpgradeRule{
		{Type: operation_setting.TopUpGroupUpgradeRuleTypeSingle, Amount: "100.00", Group: "vip"},
	})

	user := insertUserForPaymentGuardTest(t, 804, 0)
	user.Group = "default"
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Update("group", user.Group).Error)
	createTopUpGroupUpgradeEpayOrder(t, user.Id, "EPAY-GROUP-BELOW-THRESHOLD", 1, 99.99)

	_, err := RechargeEpay("EPAY-GROUP-BELOW-THRESHOLD", "alipay", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, "default", getUserGroupForTopUpGroupUpgradeTest(t, user.Id))
}

func TestRechargeEpayTopUpGroupUpgradeNeverDowngradesGroup(t *testing.T) {
	truncateTables(t)

	oldQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1
	t.Cleanup(func() { common.QuotaPerUnit = oldQuotaPerUnit })
	configureTopUpGroupUpgradeTest(t, map[string]float64{
		"default": 1,
		"vip":     2,
		"svip":    3,
	}, []operation_setting.TopUpGroupUpgradeRule{
		{Type: operation_setting.TopUpGroupUpgradeRuleTypeSingle, Amount: "1.00", Group: "vip"},
	})

	user := insertUserForPaymentGuardTest(t, 803, 0)
	user.Group = "svip"
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Update("group", user.Group).Error)
	createTopUpGroupUpgradeEpayOrder(t, user.Id, "EPAY-GROUP-NO-DOWNGRADE", 1, 1)

	_, err := RechargeEpay("EPAY-GROUP-NO-DOWNGRADE", "alipay", "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, "svip", getUserGroupForTopUpGroupUpgradeTest(t, user.Id))
	assert.Equal(t, 1, getUserQuotaForPaymentGuardTest(t, user.Id))
}
