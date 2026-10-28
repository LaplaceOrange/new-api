package model

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func seedTokenPlan(t *testing.T, plan *SubscriptionPlan) {
	t.Helper()
	require.NoError(t, DB.Create(plan).Error)
	// 测试库会反复删除重建同 id 的套餐，先失效计划缓存避免读到旧配置。
	InvalidateSubscriptionPlanCache(plan.Id)
}

func seedTokenSub(t *testing.T, sub *UserSubscription) {
	t.Helper()
	require.NoError(t, DB.Create(sub).Error)
}

func getTokenSub(t *testing.T, id int) UserSubscription {
	t.Helper()
	var sub UserSubscription
	require.NoError(t, DB.Where("id = ?", id).First(&sub).Error)
	return sub
}

func TestPreConsumeSubscriptionTokensAccumulatesByPriority(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	lowPlan := &SubscriptionPlan{Id: 9401, Title: "Low", TokenAmount: 400}
	highPlan := &SubscriptionPlan{Id: 9402, Title: "High", TokenAmount: 300}
	seedTokenPlan(t, lowPlan)
	seedTokenPlan(t, highPlan)
	// The low-priority subscription expires sooner (old tie-break), but Priority wins.
	seedTokenSub(t, &UserSubscription{Id: 9411, UserId: 301, PlanId: lowPlan.Id, StartTime: now - 100, EndTime: now + 1000, Status: "active"})
	seedTokenSub(t, &UserSubscription{Id: 9412, UserId: 301, PlanId: highPlan.Id, StartTime: now - 100, EndTime: now + 30*24*3600, Status: "active", Priority: 5})

	res, err := PreConsumeSubscriptionTokens("tok-req-1", 301, "default", "gpt-4o", 600)
	require.NoError(t, err)
	require.EqualValues(t, 600, res.ReservedTokens)
	require.Len(t, res.Details, 2)
	assert.Equal(t, 9412, res.Details[0].UserSubscriptionId)
	assert.EqualValues(t, 300, res.Details[0].Reserved)
	assert.Equal(t, 9411, res.Details[1].UserSubscriptionId)
	assert.EqualValues(t, 300, res.Details[1].Reserved)

	assert.EqualValues(t, 300, getTokenSub(t, 9412).TokenUsed)
	assert.EqualValues(t, 300, getTokenSub(t, 9411).TokenUsed)

	// Idempotent replay reports the same reservation without further deduction.
	res2, err := PreConsumeSubscriptionTokens("tok-req-1", 301, "default", "gpt-4o", 600)
	require.NoError(t, err)
	assert.EqualValues(t, 600, res2.ReservedTokens)
	assert.EqualValues(t, 300, getTokenSub(t, 9411).TokenUsed)
	assert.EqualValues(t, 300, getTokenSub(t, 9412).TokenUsed)
}

func TestPreConsumeSubscriptionTokensScopeMatching(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	groups, err := common.Marshal([]string{"vip"})
	require.NoError(t, err)
	models, err := common.Marshal([]string{"gpt-4o"})
	require.NoError(t, err)
	scoped := &SubscriptionPlan{Id: 9403, Title: "Scoped", TokenAmount: 1000, TokenGroups: string(groups), TokenModels: string(models)}
	open := &SubscriptionPlan{Id: 9404, Title: "Open", TokenAmount: 1000}
	seedTokenPlan(t, scoped)
	seedTokenPlan(t, open)
	seedTokenSub(t, &UserSubscription{Id: 9421, UserId: 302, PlanId: scoped.Id, StartTime: now - 100, EndTime: now + 1000, Status: "active"})
	seedTokenSub(t, &UserSubscription{Id: 9422, UserId: 303, PlanId: open.Id, StartTime: now - 100, EndTime: now + 1000, Status: "active"})

	// Group not in scope.
	res, err := PreConsumeSubscriptionTokens("tok-scope-1", 302, "default", "gpt-4o", 100)
	require.NoError(t, err)
	assert.Zero(t, res.ReservedTokens)
	// Model not in scope.
	res, err = PreConsumeSubscriptionTokens("tok-scope-2", 302, "vip", "claude-3", 100)
	require.NoError(t, err)
	assert.Zero(t, res.ReservedTokens)
	// Group + model both in scope.
	res, err = PreConsumeSubscriptionTokens("tok-scope-3", 302, "vip", "gpt-4o", 100)
	require.NoError(t, err)
	assert.EqualValues(t, 100, res.ReservedTokens)
	// Empty scopes mean no restriction.
	res, err = PreConsumeSubscriptionTokens("tok-scope-4", 303, "any-group", "any-model", 100)
	require.NoError(t, err)
	assert.EqualValues(t, 100, res.ReservedTokens)
}

func TestSubscriptionTokensPerModelBuckets(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	plan := &SubscriptionPlan{Id: 9405, Title: "PerModel", TokenAmount: 500, TokenCountMode: SubscriptionTokenModePerModel}
	seedTokenPlan(t, plan)
	seedTokenSub(t, &UserSubscription{Id: 9431, UserId: 304, PlanId: plan.Id, StartTime: now - 100, EndTime: now + 1000, Status: "active"})

	_, err := PreConsumeSubscriptionTokens("tok-pm-1", 304, "default", "model-a", 300)
	require.NoError(t, err)
	// model-a bucket has 200 left; a 300-token request only gets 200.
	res, err := PreConsumeSubscriptionTokens("tok-pm-2", 304, "default", "model-a", 300)
	require.NoError(t, err)
	assert.EqualValues(t, 200, res.ReservedTokens)
	// model-b has an independent full bucket.
	res, err = PreConsumeSubscriptionTokens("tok-pm-3", 304, "default", "model-b", 400)
	require.NoError(t, err)
	assert.EqualValues(t, 400, res.ReservedTokens)

	// Shared-mode counter is not used in per_model mode.
	assert.Zero(t, getTokenSub(t, 9431).TokenUsed)
	rows, err := GetUserSubscriptionTokenUsages([]int{9431})
	require.NoError(t, err)
	usageByModel := map[string]int64{}
	for _, row := range rows {
		usageByModel[row.ModelName] = row.Used
	}
	assert.EqualValues(t, 500, usageByModel["model-a"])
	assert.EqualValues(t, 400, usageByModel["model-b"])
}

func TestSettleSubscriptionTokensAdjustsToActual(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	plan := &SubscriptionPlan{Id: 9406, Title: "Settle", TokenAmount: 1000}
	seedTokenPlan(t, plan)
	seedTokenSub(t, &UserSubscription{Id: 9441, UserId: 305, PlanId: plan.Id, StartTime: now - 100, EndTime: now + 1000, Status: "active"})

	// Over-reserved: settle returns the excess to the package.
	_, err := PreConsumeSubscriptionTokens("tok-settle-1", 305, "default", "gpt-4o", 600)
	require.NoError(t, err)
	covered, details, err := SettleSubscriptionTokens("tok-settle-1", 300)
	require.NoError(t, err)
	assert.EqualValues(t, 300, covered)
	require.Len(t, details, 1)
	assert.EqualValues(t, 300, details[0].Consumed)
	assert.EqualValues(t, 300, getTokenSub(t, 9441).TokenUsed)

	// Idempotent re-settle.
	covered, _, err = SettleSubscriptionTokens("tok-settle-1", 300)
	require.NoError(t, err)
	assert.EqualValues(t, 300, covered)
	assert.EqualValues(t, 300, getTokenSub(t, 9441).TokenUsed)

	// Under-reserved: settle may consume beyond the reservation within package capacity.
	_, err = PreConsumeSubscriptionTokens("tok-settle-2", 305, "default", "gpt-4o", 200)
	require.NoError(t, err)
	covered, _, err = SettleSubscriptionTokens("tok-settle-2", 500)
	require.NoError(t, err)
	assert.EqualValues(t, 500, covered)
	assert.EqualValues(t, 800, getTokenSub(t, 9441).TokenUsed)

	// Zero actual usage refunds the whole reservation.
	_, err = PreConsumeSubscriptionTokens("tok-settle-3", 305, "default", "gpt-4o", 100)
	require.NoError(t, err)
	covered, _, err = SettleSubscriptionTokens("tok-settle-3", 0)
	require.NoError(t, err)
	assert.Zero(t, covered)
	assert.EqualValues(t, 800, getTokenSub(t, 9441).TokenUsed)
}

func TestSettleSubscriptionTokensSplitsAcrossPackages(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	highPlan := &SubscriptionPlan{Id: 9407, Title: "High", TokenAmount: 300}
	lowPlan := &SubscriptionPlan{Id: 9408, Title: "Low", TokenAmount: 1000}
	seedTokenPlan(t, highPlan)
	seedTokenPlan(t, lowPlan)
	seedTokenSub(t, &UserSubscription{Id: 9451, UserId: 306, PlanId: lowPlan.Id, StartTime: now - 100, EndTime: now + 1000, Status: "active"})
	seedTokenSub(t, &UserSubscription{Id: 9452, UserId: 306, PlanId: highPlan.Id, StartTime: now - 100, EndTime: now + 30*24*3600, Status: "active", Priority: 5})

	_, err := PreConsumeSubscriptionTokens("tok-split-1", 306, "default", "gpt-4o", 500)
	require.NoError(t, err)
	// Actual 450: high-priority package keeps its 300, the rest goes to the low one.
	covered, details, err := SettleSubscriptionTokens("tok-split-1", 450)
	require.NoError(t, err)
	assert.EqualValues(t, 450, covered)
	require.Len(t, details, 2)
	assert.Equal(t, 9452, details[0].UserSubscriptionId)
	assert.EqualValues(t, 300, details[0].Consumed)
	assert.EqualValues(t, 150, details[1].Consumed)
	assert.EqualValues(t, 150, getTokenSub(t, 9451).TokenUsed)
}

func TestRefundSubscriptionTokensIdempotent(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	plan := &SubscriptionPlan{Id: 9409, Title: "Refund", TokenAmount: 1000}
	seedTokenPlan(t, plan)
	seedTokenSub(t, &UserSubscription{Id: 9461, UserId: 307, PlanId: plan.Id, StartTime: now - 100, EndTime: now + 1000, Status: "active"})

	res, err := PreConsumeSubscriptionTokens("tok-refund-1", 307, "default", "gpt-4o", 400)
	require.NoError(t, err)
	assert.EqualValues(t, 400, res.ReservedTokens)

	require.NoError(t, RefundSubscriptionTokens("tok-refund-1"))
	assert.Zero(t, getTokenSub(t, 9461).TokenUsed)
	// Second refund is a no-op.
	require.NoError(t, RefundSubscriptionTokens("tok-refund-1"))
	assert.Zero(t, getTokenSub(t, 9461).TokenUsed)
	// Pre-consuming for the same request after refund is rejected.
	_, err = PreConsumeSubscriptionTokens("tok-refund-1", 307, "default", "gpt-4o", 400)
	require.Error(t, err)
}

func TestUpdateUserSubscriptionPriority(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	plan := &SubscriptionPlan{Id: 9410, Title: "Prio", TokenAmount: 0}
	seedTokenPlan(t, plan)
	// All default priority: order is end_time asc (A, B, C); D is cancelled and excluded.
	seedTokenSub(t, &UserSubscription{Id: 9471, UserId: 308, PlanId: plan.Id, StartTime: now - 300, EndTime: now + 300, Status: "active"})
	seedTokenSub(t, &UserSubscription{Id: 9472, UserId: 308, PlanId: plan.Id, StartTime: now - 300, EndTime: now + 200, Status: "active"})
	seedTokenSub(t, &UserSubscription{Id: 9473, UserId: 308, PlanId: plan.Id, StartTime: now - 300, EndTime: now + 100, Status: "active"})
	seedTokenSub(t, &UserSubscription{Id: 9474, UserId: 308, PlanId: plan.Id, StartTime: now - 300, EndTime: now + 50, Status: "cancelled"})

	// Move 9471 (currently last) up: swaps with 9472; order becomes 9473, 9471, 9472.
	require.NoError(t, UpdateUserSubscriptionPriority(308, 9471, true))
	subs, err := GetAllActiveUserSubscriptions(308)
	require.NoError(t, err)
	require.Len(t, subs, 3)
	assert.Equal(t, 9473, subs[0].Subscription.Id)
	assert.Equal(t, 9471, subs[1].Subscription.Id)
	assert.Equal(t, 9472, subs[2].Subscription.Id)

	// Moving the top up fails.
	err = UpdateUserSubscriptionPriority(308, 9473, true)
	require.Error(t, err)
	// Moving the bottom down fails.
	err = UpdateUserSubscriptionPriority(308, 9472, false)
	require.Error(t, err)
	// Foreign subscription id is rejected.
	err = UpdateUserSubscriptionPriority(308, 9474, true)
	require.Error(t, err)
	// Other users' subscriptions are not visible.
	err = UpdateUserSubscriptionPriority(309, 9471, true)
	require.Error(t, err)
}

func TestPreConsumeUserSubscriptionPrefersPriorityOrder(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	plan := &SubscriptionPlan{Id: 9412, Title: "Quota", TotalAmount: 10_000}
	seedTokenPlan(t, plan)
	// soon expires but low priority vs later expiry with priority 7.
	seedTokenSub(t, &UserSubscription{Id: 9481, UserId: 310, PlanId: plan.Id, AmountTotal: 10_000, StartTime: now - 100, EndTime: now + 100, Status: "active"})
	seedTokenSub(t, &UserSubscription{Id: 9482, UserId: 310, PlanId: plan.Id, AmountTotal: 10_000, StartTime: now - 100, EndTime: now + 30*24*3600, Status: "active", Priority: 7})

	res, err := PreConsumeUserSubscription("quota-req-1", 310, "gpt-4o", 0, 100, nil)
	require.NoError(t, err)
	assert.Equal(t, 9482, res.UserSubscriptionId)
	assert.EqualValues(t, 100, getTokenSub(t, 9482).AmountUsed)
	assert.Zero(t, getTokenSub(t, 9481).AmountUsed)
}

func TestTokenPackageResetsWithQuotaPeriod(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	plan := &SubscriptionPlan{Id: 9413, Title: "ResetDaily", TokenAmount: 1000, QuotaResetPeriod: SubscriptionResetDaily}
	seedTokenPlan(t, plan)
	// Reset is due: NextResetTime in the past.
	seedTokenSub(t, &UserSubscription{Id: 9491, UserId: 311, PlanId: plan.Id, AmountTotal: 0, AmountUsed: 77, TokenUsed: 500,
		StartTime: now - 2*24*3600, EndTime: now + 30*24*3600, Status: "active",
		LastResetTime: now - 2*24*3600, NextResetTime: now - 3600})
	require.NoError(t, DB.Create(&UserSubscriptionTokenUsage{UserSubscriptionId: 9491, ModelName: "gpt-4o", Used: 400}).Error)
	require.NoError(t, DB.Create(&SubscriptionPreConsumeRecord{
		RequestId:          "quota-reset-in-flight",
		UserId:             311,
		UserSubscriptionId: 9491,
		PreConsumed:        55,
		Status:             subscriptionPreConsumeStatusPending,
	}).Error)
	require.NoError(t, DB.Create(&SubscriptionTokenConsumeRecord{
		RequestId:          "token-reset-in-flight",
		UserId:             311,
		UserSubscriptionId: 9491,
		ModelName:          "gpt-4o",
		Reserved:           100,
		Status:             "consumed",
	}).Error)

	res, err := PreConsumeSubscriptionTokens("tok-reset-1", 311, "default", "gpt-4o", 600)
	require.NoError(t, err)
	assert.EqualValues(t, 600, res.ReservedTokens)

	sub := getTokenSub(t, 9491)
	// Reset drops settled usage but carries in-flight reservations into the new period.
	assert.EqualValues(t, 55, sub.AmountUsed)
	assert.EqualValues(t, 700, sub.TokenUsed)
	assert.Greater(t, sub.NextResetTime, now)
	// Per-model usage rows were cleared.
	rows, err := GetUserSubscriptionTokenUsages([]int{9491})
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestPreConsumeSubscriptionTokensSkipsConcurrencyFull(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	limited := &SubscriptionPlan{Id: 9414, Title: "Limited", TokenAmount: 1000, MaxConcurrency: 1}
	backup := &SubscriptionPlan{Id: 9415, Title: "Backup", TokenAmount: 1000}
	seedTokenPlan(t, limited)
	seedTokenPlan(t, backup)
	seedTokenSub(t, &UserSubscription{Id: 9501, UserId: 312, PlanId: limited.Id, StartTime: now - 100, EndTime: now + 30*24*3600, Status: "active", Priority: 5})
	seedTokenSub(t, &UserSubscription{Id: 9502, UserId: 312, PlanId: backup.Id, StartTime: now - 100, EndTime: now + 1000, Status: "active"})

	// First request takes the limited package's only concurrency slot.
	first, err := PreConsumeSubscriptionTokens("tok-conc-1", 312, "default", "gpt-4o", 100)
	require.NoError(t, err)
	require.EqualValues(t, 100, first.ReservedTokens)
	require.Equal(t, 9501, first.Details[0].UserSubscriptionId)
	require.Len(t, first.ConcurrencyLeases, 1)

	// Second in-flight request routes to the backup package.
	second, err := PreConsumeSubscriptionTokens("tok-conc-2", 312, "default", "gpt-4o", 100)
	require.NoError(t, err)
	require.EqualValues(t, 100, second.ReservedTokens)
	require.Equal(t, 9502, second.Details[0].UserSubscriptionId)

	// Releasing the first lease frees the slot again.
	for _, lease := range first.ConcurrencyLeases {
		lease.Release()
	}
	third, err := PreConsumeSubscriptionTokens("tok-conc-3", 312, "default", "gpt-4o", 100)
	require.NoError(t, err)
	require.EqualValues(t, 100, third.ReservedTokens)
	require.Equal(t, 9501, third.Details[0].UserSubscriptionId)
}

func TestPreConsumeUserSubscriptionConcurrencyFull(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	plan := &SubscriptionPlan{Id: 9416, Title: "Concurrent", TotalAmount: 10_000, MaxConcurrency: 1}
	seedTokenPlan(t, plan)
	seedTokenSub(t, &UserSubscription{Id: 9511, UserId: 313, PlanId: plan.Id, AmountTotal: 10_000, StartTime: now - 100, EndTime: now + 30*24*3600, Status: "active"})

	first, err := PreConsumeUserSubscription("quota-conc-1", 313, "gpt-4o", 0, 100, nil)
	require.NoError(t, err)
	require.Len(t, first.ConcurrencyLeases, 1)

	_, err = PreConsumeUserSubscription("quota-conc-2", 313, "gpt-4o", 0, 100, nil)
	require.ErrorIs(t, err, ErrSubscriptionConcurrencyFull)

	for _, lease := range first.ConcurrencyLeases {
		lease.Release()
	}
	again, err := PreConsumeUserSubscription("quota-conc-3", 313, "gpt-4o", 0, 100, nil)
	require.NoError(t, err)
	require.Len(t, again.ConcurrencyLeases, 1)
}

func TestPreConsumeSubscriptionTokensUnlimitedConcurrency(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	plan := &SubscriptionPlan{Id: 9417, Title: "Unlimited", ContentType: SubscriptionContentToken}
	seedTokenPlan(t, plan)
	seedTokenSub(t, &UserSubscription{Id: 9521, UserId: 314, PlanId: plan.Id, StartTime: now - 100, EndTime: now + 30*24*3600, Status: "active"})

	leases := make([]*common.ChannelConcurrencyLease, 0, 5)
	for i := range 5 {
		res, err := PreConsumeSubscriptionTokens("tok-unlimited-"+strconv.Itoa(i), 314, "default", "gpt-4o", 10)
		require.NoError(t, err)
		require.EqualValues(t, 10, res.ReservedTokens)
		require.Len(t, res.ConcurrencyLeases, 1)
		leases = append(leases, res.ConcurrencyLeases...)
	}
	count, known, err := common.GetSubscriptionConcurrency(nil, 9521)
	require.NoError(t, err)
	assert.True(t, known)
	assert.Equal(t, 5, count)
	for _, lease := range leases {
		lease.Release()
	}
	count, _, err = common.GetSubscriptionConcurrency(nil, 9521)
	require.NoError(t, err)
	assert.Zero(t, count)

	_, err = PreConsumeUserSubscription("tok-unlimited-quota", 314, "gpt-4o", 0, 10, nil)
	require.ErrorIs(t, err, ErrSubscriptionQuotaInsufficient)
	assert.Zero(t, getTokenSub(t, 9521).AmountUsed)

	quotaPlan := &SubscriptionPlan{Id: 9418, Title: "Quota", ContentType: SubscriptionContentQuota, TotalAmount: 1000}
	seedTokenPlan(t, quotaPlan)
	seedTokenSub(t, &UserSubscription{
		Id: 9522, UserId: 314, PlanId: quotaPlan.Id, AmountTotal: 1000,
		StartTime: now - 100, EndTime: now + 30*24*3600, Status: "active",
	})
	quota, err := PreConsumeUserSubscription("tok-unlimited-quota-fallback", 314, "gpt-4o", 0, 10, nil)
	require.NoError(t, err)
	assert.Equal(t, 9522, quota.UserSubscriptionId)
	assert.EqualValues(t, 10, getTokenSub(t, 9522).AmountUsed)
	for _, lease := range quota.ConcurrencyLeases {
		lease.Release()
	}
}

func TestMigrateLegacySubscriptionPreConsumeRecordsPreservesActiveTaskReservation(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	plan := &SubscriptionPlan{Id: 9420, Title: "Migration", TotalAmount: 1000}
	seedTokenPlan(t, plan)
	seedTokenSub(t, &UserSubscription{
		Id: 9421, UserId: 320, PlanId: plan.Id, AmountTotal: 1000, AmountUsed: 125,
		StartTime: now - 100, EndTime: now + 30*24*3600, Status: "active",
	})
	require.NoError(t, DB.Create(&SubscriptionPreConsumeRecord{
		RequestId: "migration-active", UserId: 320, UserSubscriptionId: 9421,
		PreConsumed: 100, Status: "consumed",
	}).Error)
	require.NoError(t, DB.Create(&SubscriptionPreConsumeRecord{
		RequestId: "migration-terminal", UserId: 320, UserSubscriptionId: 9421,
		PreConsumed: 25, Status: "consumed",
	}).Error)
	require.NoError(t, DB.Create(&Task{
		TaskID: "migration-active-task", UserId: 320, Quota: 125, Status: TaskStatusInProgress,
		PrivateData: TaskPrivateData{
			BillingSource:  "subscription",
			SubscriptionId: 9421,
			Execution:      &TaskExecutionSnapshot{RequestID: "migration-active"},
		},
	}).Error)
	require.NoError(t, DB.Create(&Task{
		TaskID: "migration-terminal-task", UserId: 320, Quota: 25, Status: TaskStatusSuccess,
		PrivateData: TaskPrivateData{
			BillingSource:  "subscription",
			SubscriptionId: 9421,
			Execution:      &TaskExecutionSnapshot{RequestID: "migration-terminal"},
		},
	}).Error)

	require.NoError(t, migrateLegacySubscriptionPreConsumeRecords(DB))

	var activeRecord, terminalRecord SubscriptionPreConsumeRecord
	require.NoError(t, DB.Where("request_id = ?", "migration-active").First(&activeRecord).Error)
	require.NoError(t, DB.Where("request_id = ?", "migration-terminal").First(&terminalRecord).Error)
	assert.Equal(t, subscriptionPreConsumeStatusPending, activeRecord.Status)
	assert.EqualValues(t, 125, activeRecord.PreConsumed)
	assert.Equal(t, subscriptionPreConsumeStatusSettled, terminalRecord.Status)

	require.NoError(t, SettleSubscriptionPreConsume("migration-active", 130))
	assert.EqualValues(t, 130, getTokenSub(t, 9421).AmountUsed)
	require.NoError(t, migrateLegacySubscriptionPreConsumeRecords(DB))
}

func TestSubscriptionPreConsumeTerminalStateTransitionsAreIdempotent(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	plan := &SubscriptionPlan{Id: 9430, Title: "Settlement", TotalAmount: 1000}
	seedTokenPlan(t, plan)
	seedTokenSub(t, &UserSubscription{
		Id: 9431, UserId: 321, PlanId: plan.Id, AmountTotal: 1000, AmountUsed: 100,
		StartTime: now - 100, EndTime: now + 30*24*3600, Status: "active",
	})
	require.NoError(t, DB.Create(&SubscriptionPreConsumeRecord{
		RequestId: "settlement-state", UserId: 321, UserSubscriptionId: 9431,
		PreConsumed: 100, Status: subscriptionPreConsumeStatusPending,
	}).Error)

	require.NoError(t, UpdateSubscriptionPreConsume("settlement-state", 150, false))
	require.NoError(t, UpdateSubscriptionPreConsume("settlement-state", 150, false))
	assert.EqualValues(t, 150, getTokenSub(t, 9431).AmountUsed)
	require.NoError(t, SettleSubscriptionPreConsume("settlement-state", 120))
	require.NoError(t, SettleSubscriptionPreConsume("settlement-state", 120))
	assert.EqualValues(t, 120, getTokenSub(t, 9431).AmountUsed)
	assert.Error(t, SettleSubscriptionPreConsume("settlement-state", 130))
	assert.Error(t, RefundSubscriptionPreConsume("settlement-state"))
	assert.EqualValues(t, 120, getTokenSub(t, 9431).AmountUsed)
}

func TestEnsureSubscriptionPlanTableSQLiteAddsContentTypeOnFreshAndLegacyTables(t *testing.T) {
	originalDB := DB
	originalDatabaseType := common.MainDatabaseType()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		DB = originalDB
		common.SetMainDatabaseType(originalDatabaseType)
	})

	require.NoError(t, db.Exec(`CREATE TABLE subscription_plans (
		id integer PRIMARY KEY,
		title varchar(128) NOT NULL,
		subtitle varchar(255) DEFAULT '',
		price_amount decimal(10,6) NOT NULL,
		currency varchar(8) NOT NULL DEFAULT 'USD',
		duration_unit varchar(16) NOT NULL DEFAULT 'month',
		duration_value integer NOT NULL DEFAULT 1,
		custom_seconds bigint NOT NULL DEFAULT 0,
		enabled numeric DEFAULT 1,
		sort_order integer DEFAULT 0,
		allow_balance_pay numeric DEFAULT 1,
		allow_wallet_overflow numeric DEFAULT 1,
		stripe_price_id varchar(128) DEFAULT '',
		creem_product_id varchar(128) DEFAULT '',
		waffo_pancake_product_id varchar(128) DEFAULT '',
		max_purchase_per_user integer DEFAULT 0,
		upgrade_group varchar(64) DEFAULT '',
		downgrade_group varchar(64) DEFAULT '',
		total_amount bigint NOT NULL DEFAULT 0,
		quota_reset_period varchar(16) DEFAULT 'never',
		quota_reset_custom_seconds bigint DEFAULT 0,
		token_amount bigint NOT NULL DEFAULT 0,
		token_count_mode varchar(16) NOT NULL DEFAULT 'shared',
		token_groups text,
		token_models text,
		max_concurrency integer NOT NULL DEFAULT 0,
		created_at bigint,
		updated_at bigint
	)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO subscription_plans (id, title, price_amount, total_amount, token_amount, max_concurrency)
		VALUES (1, 'Legacy', 5, 5000, 250, 3)`).Error)

	require.NoError(t, ensureSubscriptionPlanTableSQLite())
	require.NoError(t, ensureSubscriptionPlanTableSQLite())
	require.True(t, db.Migrator().HasColumn(&SubscriptionPlan{}, "content_type"))
	var plan SubscriptionPlan
	require.NoError(t, db.First(&plan, 1).Error)
	assert.Equal(t, "Legacy", plan.Title)
	assert.EqualValues(t, 5000, plan.TotalAmount)
	assert.EqualValues(t, 250, plan.TokenAmount)
	assert.EqualValues(t, 3, plan.MaxConcurrency)

	require.NoError(t, db.Migrator().DropTable(&SubscriptionPlan{}))
	require.NoError(t, ensureSubscriptionPlanTableSQLite())
	require.NoError(t, ensureSubscriptionPlanTableSQLite())
	require.True(t, db.Migrator().HasColumn(&SubscriptionPlan{}, "content_type"))
}

func TestSameRequestTokenPackageAndQuotaShareConcurrencySlot(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	plan := &SubscriptionPlan{Id: 9418, Title: "PkgQuota", TokenAmount: 10_000, TotalAmount: 50_000, MaxConcurrency: 1}
	seedTokenPlan(t, plan)
	seedTokenSub(t, &UserSubscription{Id: 9531, UserId: 315, PlanId: plan.Id,
		AmountTotal: 50_000, StartTime: now - 100, EndTime: now + 30*24*3600, Status: "active"})

	// 同一请求先经 Token 包路径预占（已持有并发槽），额度路径不得自我争抢。
	tokenRes, err := PreConsumeSubscriptionTokens("tok-share-1", 315, "default", "gpt-4o", 400)
	require.NoError(t, err)
	require.EqualValues(t, 400, tokenRes.ReservedTokens)
	require.Len(t, tokenRes.ConcurrencyLeases, 1)

	owned := map[int]struct{}{}
	for _, d := range tokenRes.Details {
		owned[d.UserSubscriptionId] = struct{}{}
	}
	quotaRes, err := PreConsumeUserSubscription("tok-share-1", 315, "gpt-4o", 0, 100, owned)
	require.NoError(t, err)
	assert.Equal(t, 9531, quotaRes.UserSubscriptionId)
	assert.Empty(t, quotaRes.ConcurrencyLeases)
	assert.EqualValues(t, 100, getTokenSub(t, 9531).AmountUsed)
	assert.EqualValues(t, 400, getTokenSub(t, 9531).TokenUsed)

	// 未持有槽的其他请求仍会被并发限制拒绝（限流=1，槽被第一个请求占用）。
	_, err = PreConsumeUserSubscription("tok-share-2", 315, "gpt-4o", 0, 100, nil)
	require.ErrorIs(t, err, ErrSubscriptionConcurrencyFull)
}
