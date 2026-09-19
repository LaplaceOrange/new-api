package model

import (
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSearchRedemptionsFiltersAndPaginates(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Redemption{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	})

	now := common.GetTimestamp()
	redemptions := []Redemption{
		{Id: 1, Name: "alpha-active", Key: "00000000000000000000000000000001", Status: common.RedemptionCodeStatusEnabled, ExpiredTime: 0},
		{Id: 2, Name: "alpha-future", Key: "00000000000000000000000000000002", Status: common.RedemptionCodeStatusEnabled, ExpiredTime: now + 3600},
		{Id: 3, Name: "alpha-expired", Key: "00000000000000000000000000000003", Status: common.RedemptionCodeStatusEnabled, ExpiredTime: now - 10},
		{Id: 4, Name: "beta-disabled", Key: "00000000000000000000000000000004", Status: common.RedemptionCodeStatusDisabled, ExpiredTime: 0},
		{Id: 5, Name: "beta-used", Key: "00000000000000000000000000000005", Status: common.RedemptionCodeStatusUsed, ExpiredTime: 0},
	}
	require.NoError(t, DB.Create(&redemptions).Error)

	tests := []struct {
		name      string
		keyword   string
		status    string
		startIdx  int
		num       int
		wantTotal int64
		wantIds   []int
	}{
		{
			name:      "no filters returns all rows",
			num:       10,
			wantTotal: 5,
			wantIds:   []int{5, 4, 3, 2, 1},
		},
		{
			name:      "keyword filters by name prefix",
			keyword:   "alpha",
			num:       10,
			wantTotal: 3,
			wantIds:   []int{3, 2, 1},
		},
		{
			name:      "enabled status excludes expired rows",
			status:    "1",
			num:       10,
			wantTotal: 2,
			wantIds:   []int{2, 1},
		},
		{
			name:      "expired status returns enabled expired rows",
			status:    "expired",
			num:       10,
			wantTotal: 1,
			wantIds:   []int{3},
		},
		{
			name:      "disabled status",
			status:    "2",
			num:       10,
			wantTotal: 1,
			wantIds:   []int{4},
		},
		{
			name:      "used status",
			status:    "3",
			num:       10,
			wantTotal: 1,
			wantIds:   []int{5},
		},
		{
			name:      "pagination keeps unpaged total",
			startIdx:  1,
			num:       2,
			wantTotal: 5,
			wantIds:   []int{4, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, total, err := SearchRedemptions(tt.keyword, tt.status, tt.startIdx, tt.num)
			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, total)
			gotIds := make([]int, 0, len(rows))
			for _, row := range rows {
				gotIds = append(gotIds, row.Id)
			}
			assert.Equal(t, tt.wantIds, gotIds)
		})
	}
}

func setupRedeemTables(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&Redemption{}, &RedemptionUsage{}, &User{}, &SubscriptionPlan{}, &UserSubscription{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&RedemptionUsage{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&RedemptionUsage{}).Error)
		DB.Exec("DELETE FROM user_subscriptions")
		DB.Exec("DELETE FROM subscription_plans")
		DB.Exec("DELETE FROM users")
		DB.Exec("DELETE FROM logs")
	})
}

func setupRedeemFixture(t *testing.T, quota int) (userId int, key string) {
	t.Helper()
	setupRedeemTables(t)

	user := &User{Username: "redeem-user", Password: "password", Status: common.UserStatusEnabled, Quota: 0, AffCode: "rdm1"}
	require.NoError(t, DB.Create(user).Error)

	key = "10000000000000000000000000000001"
	redemption := &Redemption{
		Name:           "redeem-test",
		Key:            key,
		Status:         common.RedemptionCodeStatusEnabled,
		Quota:          quota,
		CreatedTime:    common.GetTimestamp(),
		MaxUses:        1,
		MaxUsesPerUser: 1,
	}
	require.NoError(t, DB.Create(redemption).Error)
	return user.Id, key
}

func TestRedeemCreditsQuotaExactlyOnce(t *testing.T) {
	userId, key := setupRedeemFixture(t, 500)

	result, err := Redeem(key, userId)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 500, result.Quota)

	var user User
	require.NoError(t, DB.First(&user, "id = ?", userId).Error)
	assert.Equal(t, 500, user.Quota)

	var redemption Redemption
	require.NoError(t, DB.First(&redemption, "name = ?", "redeem-test").Error)
	assert.Equal(t, common.RedemptionCodeStatusUsed, redemption.Status)
	assert.Equal(t, userId, redemption.UsedUserId)
	assert.Equal(t, 1, redemption.UsedCount)

	// Redeeming the same code again must fail and must not credit quota.
	_, err = Redeem(key, userId)
	require.Error(t, err)
	require.NoError(t, DB.First(&user, "id = ?", userId).Error)
	assert.Equal(t, 500, user.Quota)
}

func TestRedeemRejectsWalletOverflow(t *testing.T) {
	userId, key := setupRedeemFixture(t, 11)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", userId).Update("quota", common.MaxWalletQuota-10).Error)

	_, err := Redeem(key, userId)
	require.ErrorIs(t, err, ErrRedeemFailed)

	var user User
	require.NoError(t, DB.First(&user, "id = ?", userId).Error)
	assert.Equal(t, common.MaxWalletQuota-10, user.Quota)

	var redemption Redemption
	require.NoError(t, DB.First(&redemption, "key = ?", key).Error)
	assert.Equal(t, common.RedemptionCodeStatusEnabled, redemption.Status)
	assert.Equal(t, 0, redemption.UsedCount)
}

func TestRedemptionQuotaRejectsWalletOverflow(t *testing.T) {
	setupRedeemFixture(t, 500)

	redemption := &Redemption{
		Name:        "overflow-redemption",
		Key:         "10000000000000000000000000000002",
		Status:      common.RedemptionCodeStatusEnabled,
		Quota:       common.MaxWalletQuota + 1,
		CreatedTime: common.GetTimestamp(),
	}
	require.Error(t, redemption.Insert())
}

// Exactly one of several concurrent redeems of the same code may win, and
// quota must be credited exactly once.
func TestRedeemConcurrentSingleSuccess(t *testing.T) {
	userId, key := setupRedeemFixture(t, 300)

	const goroutines = 5
	successes := make([]bool, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(idx int) {
			defer wg.Done()
			if _, err := Redeem(key, userId); err == nil {
				successes[idx] = true
			}
		}(i)
	}
	wg.Wait()

	successCount := 0
	for _, ok := range successes {
		if ok {
			successCount++
		}
	}
	assert.Equal(t, 1, successCount, "exactly one concurrent redeem should succeed")

	var user User
	require.NoError(t, DB.First(&user, "id = ?", userId).Error)
	assert.Equal(t, 300, user.Quota, "quota must be credited exactly once")
}

func createRedeemUser(t *testing.T, username string) *User {
	t.Helper()
	user := &User{Username: username, Password: "password", Status: common.UserStatusEnabled, Quota: 0, Group: "default", AffCode: username}
	require.NoError(t, DB.Create(user).Error)
	return user
}

func TestRedeemAllowsMultipleUsesUntilExhausted(t *testing.T) {
	setupRedeemTables(t)
	user := createRedeemUser(t, "redeem-multi")
	key := "10000000000000000000000000000011"
	require.NoError(t, (&Redemption{
		Name:           "multi-use",
		Key:            key,
		Status:         common.RedemptionCodeStatusEnabled,
		Quota:          100,
		CreatedTime:    common.GetTimestamp(),
		MaxUses:        2,
		MaxUsesPerUser: 0,
	}).Insert())

	first, err := Redeem(key, user.Id)
	require.NoError(t, err)
	assert.Equal(t, 100, first.Quota)

	var redemption Redemption
	require.NoError(t, DB.First(&redemption, "key = ?", key).Error)
	assert.Equal(t, common.RedemptionCodeStatusEnabled, redemption.Status)
	assert.Equal(t, 1, redemption.UsedCount)

	second, err := Redeem(key, user.Id)
	require.NoError(t, err)
	assert.Equal(t, 100, second.Quota)

	require.NoError(t, DB.First(&redemption, "key = ?", key).Error)
	assert.Equal(t, common.RedemptionCodeStatusUsed, redemption.Status)
	assert.Equal(t, 2, redemption.UsedCount)

	_, err = Redeem(key, user.Id)
	require.ErrorIs(t, err, ErrRedeemFailed)

	require.NoError(t, DB.First(user, "id = ?", user.Id).Error)
	assert.Equal(t, 200, user.Quota)
}

func TestRedeemEnforcesPerUserLimit(t *testing.T) {
	setupRedeemTables(t)
	firstUser := createRedeemUser(t, "redeem-user-a")
	secondUser := createRedeemUser(t, "redeem-user-b")
	key := "10000000000000000000000000000012"
	require.NoError(t, (&Redemption{
		Name:           "per-user",
		Key:            key,
		Status:         common.RedemptionCodeStatusEnabled,
		Quota:          50,
		CreatedTime:    common.GetTimestamp(),
		MaxUses:        5,
		MaxUsesPerUser: 1,
	}).Insert())

	_, err := Redeem(key, firstUser.Id)
	require.NoError(t, err)
	_, err = Redeem(key, firstUser.Id)
	require.ErrorIs(t, err, ErrRedeemFailed)

	_, err = Redeem(key, secondUser.Id)
	require.NoError(t, err)

	require.NoError(t, DB.First(firstUser, "id = ?", firstUser.Id).Error)
	require.NoError(t, DB.First(secondUser, "id = ?", secondUser.Id).Error)
	assert.Equal(t, 50, firstUser.Quota)
	assert.Equal(t, 50, secondUser.Quota)

	var redemption Redemption
	require.NoError(t, DB.First(&redemption, "key = ?", key).Error)
	assert.Equal(t, 2, redemption.UsedCount)
	assert.Equal(t, common.RedemptionCodeStatusEnabled, redemption.Status)
}

func TestRedeemUnlimitedUsesStayEnabled(t *testing.T) {
	setupRedeemTables(t)
	user := createRedeemUser(t, "redeem-unlimited")
	key := "10000000000000000000000000000013"
	require.NoError(t, (&Redemption{
		Name:           "unlimited",
		Key:            key,
		Status:         common.RedemptionCodeStatusEnabled,
		Quota:          10,
		CreatedTime:    common.GetTimestamp(),
		MaxUses:        0,
		MaxUsesPerUser: 0,
	}).Insert())

	for range 3 {
		_, err := Redeem(key, user.Id)
		require.NoError(t, err)
	}

	var redemption Redemption
	require.NoError(t, DB.First(&redemption, "key = ?", key).Error)
	assert.Equal(t, common.RedemptionCodeStatusEnabled, redemption.Status)
	assert.Equal(t, 3, redemption.UsedCount)
	assert.Equal(t, 0, redemption.MaxUses)
}

func TestRedeemConcurrentHonorsMaxUses(t *testing.T) {
	setupRedeemTables(t)
	user := createRedeemUser(t, "redeem-concurrent-multi")
	key := "10000000000000000000000000000014"
	require.NoError(t, (&Redemption{
		Name:           "concurrent-multi",
		Key:            key,
		Status:         common.RedemptionCodeStatusEnabled,
		Quota:          20,
		CreatedTime:    common.GetTimestamp(),
		MaxUses:        3,
		MaxUsesPerUser: 0,
	}).Insert())

	const goroutines = 8
	successes := make([]bool, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(idx int) {
			defer wg.Done()
			if _, err := Redeem(key, user.Id); err == nil {
				successes[idx] = true
			}
		}(i)
	}
	wg.Wait()

	successCount := 0
	for _, ok := range successes {
		if ok {
			successCount++
		}
	}
	assert.Equal(t, 3, successCount)

	require.NoError(t, DB.First(user, "id = ?", user.Id).Error)
	assert.Equal(t, 60, user.Quota)

	var redemption Redemption
	require.NoError(t, DB.First(&redemption, "key = ?", key).Error)
	assert.Equal(t, 3, redemption.UsedCount)
	assert.Equal(t, common.RedemptionCodeStatusUsed, redemption.Status)
}

func TestRedeemGrantsSubscription(t *testing.T) {
	setupRedeemTables(t)
	user := createRedeemUser(t, "redeem-sub")
	plan := &SubscriptionPlan{
		Title:         "Gift Plan",
		PriceAmount:   9,
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
		TotalAmount:   1000,
	}
	require.NoError(t, DB.Create(plan).Error)

	key := "10000000000000000000000000000015"
	require.NoError(t, (&Redemption{
		Name:           "sub-code",
		Key:            key,
		Status:         common.RedemptionCodeStatusEnabled,
		Quota:          0,
		PlanId:         plan.Id,
		CreatedTime:    common.GetTimestamp(),
		MaxUses:        1,
		MaxUsesPerUser: 1,
	}).Insert())

	result, err := Redeem(key, user.Id)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 0, result.Quota)
	assert.Equal(t, plan.Id, result.PlanId)
	assert.Equal(t, "Gift Plan", result.PlanTitle)

	var subs []UserSubscription
	require.NoError(t, DB.Where("user_id = ? AND plan_id = ?", user.Id, plan.Id).Find(&subs).Error)
	require.Len(t, subs, 1)
	assert.Equal(t, "redemption", subs[0].Source)
	assert.Equal(t, "active", subs[0].Status)
}

func TestRedeemMissingPlanDoesNotConsume(t *testing.T) {
	setupRedeemTables(t)
	user := createRedeemUser(t, "redeem-missing-plan")
	key := "10000000000000000000000000000016"
	require.NoError(t, DB.Select(
		"UserId", "Key", "Status", "Name", "Quota", "CreatedTime", "PlanId", "MaxUses", "MaxUsesPerUser", "UsedCount",
	).Create(&Redemption{
		Name:           "missing-plan",
		Key:            key,
		Status:         common.RedemptionCodeStatusEnabled,
		Quota:          0,
		PlanId:         999001,
		CreatedTime:    common.GetTimestamp(),
		MaxUses:        1,
		MaxUsesPerUser: 1,
	}).Error)

	_, err := Redeem(key, user.Id)
	require.ErrorIs(t, err, ErrRedeemFailed)

	var redemption Redemption
	require.NoError(t, DB.First(&redemption, "key = ?", key).Error)
	assert.Equal(t, common.RedemptionCodeStatusEnabled, redemption.Status)
	assert.Equal(t, 0, redemption.UsedCount)
}

func TestRedeemSubscriptionPurchaseLimitDoesNotConsume(t *testing.T) {
	setupRedeemTables(t)
	user := createRedeemUser(t, "redeem-limit")
	plan := &SubscriptionPlan{
		Title:              "Limited Plan",
		PriceAmount:        1,
		DurationUnit:       SubscriptionDurationMonth,
		DurationValue:      1,
		Enabled:            true,
		TotalAmount:        100,
		MaxPurchasePerUser: 1,
	}
	require.NoError(t, DB.Create(plan).Error)
	now := common.GetTimestamp()
	require.NoError(t, DB.Create(&UserSubscription{
		UserId:      user.Id,
		PlanId:      plan.Id,
		AmountTotal: 100,
		StartTime:   now,
		EndTime:     now + 3600,
		Status:      "active",
		Source:      "admin",
	}).Error)

	key := "10000000000000000000000000000017"
	require.NoError(t, (&Redemption{
		Name:           "limited-sub",
		Key:            key,
		Status:         common.RedemptionCodeStatusEnabled,
		Quota:          0,
		PlanId:         plan.Id,
		CreatedTime:    now,
		MaxUses:        1,
		MaxUsesPerUser: 1,
	}).Insert())

	_, err := Redeem(key, user.Id)
	require.ErrorIs(t, err, ErrRedeemFailed)

	var redemption Redemption
	require.NoError(t, DB.First(&redemption, "key = ?", key).Error)
	assert.Equal(t, common.RedemptionCodeStatusEnabled, redemption.Status)
	assert.Equal(t, 0, redemption.UsedCount)

	var count int64
	require.NoError(t, DB.Model(&UserSubscription{}).Where("user_id = ? AND plan_id = ?", user.Id, plan.Id).Count(&count).Error)
	assert.EqualValues(t, 1, count)
}

func TestBackfillRedemptionUsedCount(t *testing.T) {
	setupRedeemTables(t)
	require.NoError(t, DB.Create(&Redemption{
		Name:        "used-legacy",
		Key:         "10000000000000000000000000000018",
		Status:      common.RedemptionCodeStatusUsed,
		Quota:       10,
		UsedUserId:  7,
		UsedCount:   0,
		CreatedTime: common.GetTimestamp(),
	}).Error)

	require.NoError(t, backfillRedemptionUsedCount())

	var redemption Redemption
	require.NoError(t, DB.First(&redemption, "key = ?", "10000000000000000000000000000018").Error)
	assert.Equal(t, 1, redemption.UsedCount)
}
