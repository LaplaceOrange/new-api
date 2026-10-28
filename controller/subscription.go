package controller

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ---- Shared types ----

type SubscriptionPlanDTO struct {
	Plan  model.SubscriptionPlan `json:"plan"`
	Quote map[string]any         `json:"quote,omitempty"`
}

type BillingPreferenceRequest struct {
	BillingPreference string `json:"billing_preference"`
}

type SubscriptionPriorityRequest struct {
	SubscriptionId int    `json:"subscription_id"`
	Move           string `json:"move"` // up | down
}

type SubscriptionBalancePayRequest struct {
	PlanId int `json:"plan_id"`
}

// normalizeSubscriptionPlanTokenPackage validates and normalizes the plan's token-package
// fields (amount / counting mode / deductible group & model scopes). Returns a non-empty
// user-facing error message when invalid.
func normalizeSubscriptionPlanTokenPackage(plan *model.SubscriptionPlan) string {
	if plan.TokenAmount < 0 || plan.TokenAmount > model.SubscriptionTokenAmountMax {
		return "套餐Token数量无效"
	}
	plan.ContentType = strings.TrimSpace(plan.ContentType)
	switch plan.ContentType {
	case model.SubscriptionContentQuota:
		plan.TokenAmount = 0
		plan.TokenGroups = ""
		plan.TokenModels = ""
	case model.SubscriptionContentToken:
		plan.TotalAmount = 0
	case "":
		// Preserve legacy plans that contain both benefits until an administrator
		// explicitly chooses one of the new content types.
	default:
		return "套餐内容类型无效"
	}
	if plan.TokenCountMode == "" {
		plan.TokenCountMode = model.SubscriptionTokenModeShared
	}
	if plan.TokenCountMode != model.SubscriptionTokenModeShared && plan.TokenCountMode != model.SubscriptionTokenModePerModel {
		return "套餐Token计数模式无效"
	}
	if plan.MaxConcurrency < 0 || plan.MaxConcurrency > 100000 {
		return "套餐并发限制必须在 0 到 100000 之间"
	}
	groupRatios := ratio_setting.GetGroupRatioCopy()
	normalizeScope := func(raw string, groupScope bool) (string, string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return "", ""
		}
		var items []string
		if err := common.UnmarshalJsonStr(raw, &items); err != nil {
			return "", "套餐Token抵扣范围格式错误"
		}
		seen := make(map[string]struct{}, len(items))
		normalized := make([]string, 0, len(items))
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if _, ok := seen[item]; ok {
				continue
			}
			seen[item] = struct{}{}
			if groupScope {
				if _, ok := groupRatios[item]; !ok {
					return "", "套餐Token可抵扣分组不存在: " + item
				}
			}
			normalized = append(normalized, item)
		}
		if len(normalized) == 0 {
			return "", ""
		}
		data, err := common.Marshal(normalized)
		if err != nil {
			return "", "套餐Token抵扣范围序列化失败"
		}
		return string(data), ""
	}
	tokenGroups, errMsg := normalizeScope(plan.TokenGroups, true)
	if errMsg != "" {
		return errMsg
	}
	tokenModels, errMsg := normalizeScope(plan.TokenModels, false)
	if errMsg != "" {
		return errMsg
	}
	plan.TokenGroups = tokenGroups
	plan.TokenModels = tokenModels
	return ""
}

// ---- User APIs ----

func GetSubscriptionPlans(c *gin.Context) {
	if !operation_setting.IsPaymentComplianceConfirmed() {
		common.ApiSuccess(c, []SubscriptionPlanDTO{})
		return
	}

	var plans []model.SubscriptionPlan
	if err := model.DB.Where("enabled = ?", true).Order("sort_order desc, id desc").Find(&plans).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	result := make([]SubscriptionPlanDTO, 0, len(plans))
	for _, p := range plans {
		p.NormalizeDefaults()
		quote, err := quotePaymentFloat(p.PriceAmount)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		result = append(result, SubscriptionPlanDTO{
			Plan:  p,
			Quote: quote.APIData(),
		})
	}
	common.ApiSuccess(c, result)
}

// subscriptionTokenUsageDTO describes a subscription's token-package usage for the user UI.
type subscriptionTokenUsageDTO struct {
	Enabled      bool                 `json:"enabled"`
	QuotaEnabled bool                 `json:"quota_enabled"`
	Mode         string               `json:"mode"`
	Total        int64                `json:"total"`
	Used         int64                `json:"used"`
	PerModel     []tokenModelUsageDTO `json:"per_model,omitempty"`
	Groups       []string             `json:"groups,omitempty"`
	Models       []string             `json:"models,omitempty"`
}

type tokenModelUsageDTO struct {
	Model string `json:"model"`
	Used  int64  `json:"used"`
}

type subscriptionConcurrencyDTO struct {
	Current    int  `json:"current"`
	Limit      int  `json:"limit"`
	Known      bool `json:"known"`
	LimitKnown bool `json:"limit_known"`
}

func GetSubscriptionSelf(c *gin.Context) {
	userId := c.GetInt("id")
	settingMap, _ := model.GetUserSetting(userId, false)
	pref := common.NormalizeBillingPreference(settingMap.BillingPreference)

	// Get all subscriptions (including expired)
	allSubscriptions, err := model.GetAllUserSubscriptions(userId)
	if err != nil {
		allSubscriptions = []model.SubscriptionSummary{}
	}

	// Get active subscriptions for backward compatibility
	activeSubscriptions, err := model.GetAllActiveUserSubscriptions(userId)
	if err != nil {
		activeSubscriptions = []model.SubscriptionSummary{}
	}

	// Token-package usage per subscription (keyed by subscription id)
	tokenUsages := buildSubscriptionTokenUsages(allSubscriptions)
	concurrencyUsages := buildSubscriptionConcurrencyUsages(c, allSubscriptions)

	common.ApiSuccess(c, gin.H{
		"billing_preference": pref,
		"subscriptions":      activeSubscriptions, // all active subscriptions
		"all_subscriptions":  allSubscriptions,    // all subscriptions including expired
		"token_usages":       tokenUsages,
		"concurrency_usages": concurrencyUsages,
	})
}

func buildSubscriptionConcurrencyUsages(c *gin.Context, subscriptions []model.SubscriptionSummary) map[int]*subscriptionConcurrencyDTO {
	result := make(map[int]*subscriptionConcurrencyDTO)
	for _, summary := range subscriptions {
		sub := summary.Subscription
		if sub == nil {
			continue
		}
		usage := &subscriptionConcurrencyDTO{}
		if plan, err := model.GetSubscriptionPlanById(sub.PlanId); err == nil && plan != nil {
			usage.Limit = plan.MaxConcurrency
			usage.LimitKnown = true
		} else if err != nil {
			common.SysError(fmt.Sprintf("failed to load subscription plan concurrency: subscription_id=%d err=%v", sub.Id, err))
		}
		count, known, err := common.GetSubscriptionConcurrency(c.Request.Context(), sub.Id)
		if err != nil {
			common.SysError(fmt.Sprintf("failed to load subscription concurrency: subscription_id=%d err=%v", sub.Id, err))
			usage.Known = false
		} else {
			usage.Current = count
			usage.Known = known
		}
		result[sub.Id] = usage
	}
	return result
}

func buildSubscriptionTokenUsages(subscriptions []model.SubscriptionSummary) map[int]*subscriptionTokenUsageDTO {
	result := make(map[int]*subscriptionTokenUsageDTO)
	subIds := make([]int, 0, len(subscriptions))
	for _, summary := range subscriptions {
		if summary.Subscription != nil {
			subIds = append(subIds, summary.Subscription.Id)
		}
	}
	if len(subIds) == 0 {
		return result
	}
	perModelRows, err := model.GetUserSubscriptionTokenUsages(subIds)
	if err != nil {
		common.SysError("failed to load subscription token usages: " + err.Error())
		perModelRows = nil
	}
	perModelBySub := make(map[int][]tokenModelUsageDTO)
	for _, row := range perModelRows {
		perModelBySub[row.UserSubscriptionId] = append(perModelBySub[row.UserSubscriptionId], tokenModelUsageDTO{
			Model: row.ModelName,
			Used:  row.Used,
		})
	}
	for _, summary := range subscriptions {
		sub := summary.Subscription
		if sub == nil {
			continue
		}
		plan, err := model.GetSubscriptionPlanById(sub.PlanId)
		if err != nil || plan == nil {
			continue
		}
		usage := &subscriptionTokenUsageDTO{
			Enabled:      plan.HasTokenBenefit(),
			QuotaEnabled: plan.HasQuotaBenefit(),
			Total:        plan.TokenAmount,
			Used:         sub.TokenUsed,
		}
		if plan.TokenCountMode == model.SubscriptionTokenModePerModel {
			usage.Mode = model.SubscriptionTokenModePerModel
			usage.PerModel = perModelBySub[sub.Id]
			if usage.PerModel == nil {
				usage.PerModel = []tokenModelUsageDTO{}
			}
		} else {
			usage.Mode = model.SubscriptionTokenModeShared
		}
		if groups := model.ParseSubscriptionTokenScope(plan.TokenGroups); len(groups) > 0 {
			usage.Groups = groups
		}
		if models := model.ParseSubscriptionTokenScope(plan.TokenModels); len(models) > 0 {
			usage.Models = models
		}
		result[sub.Id] = usage
	}
	return result
}

func UpdateSubscriptionSelfPriority(c *gin.Context) {
	userId := c.GetInt("id")
	var req SubscriptionPriorityRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.SubscriptionId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if req.Move != "up" && req.Move != "down" {
		common.ApiErrorMsg(c, "移动方向无效")
		return
	}
	if err := model.UpdateUserSubscriptionPriority(userId, req.SubscriptionId, req.Move == "up"); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func UpdateSubscriptionPreference(c *gin.Context) {
	userId := c.GetInt("id")
	var req BillingPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	pref := common.NormalizeBillingPreference(req.BillingPreference)

	user, err := model.GetUserById(userId, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	current := user.GetSetting()
	current.BillingPreference = pref
	if err := model.UpdateUserSetting(user.Id, current); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"billing_preference": pref})
}

func SubscriptionRequestBalancePay(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}

	userId := c.GetInt("id")
	var req SubscriptionBalancePayRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	if err := model.PurchaseSubscriptionWithBalance(userId, req.PlanId); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

// ---- Admin APIs ----

func AdminListSubscriptionPlans(c *gin.Context) {
	var plans []model.SubscriptionPlan
	if err := model.DB.Order("sort_order desc, id desc").Find(&plans).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	result := make([]SubscriptionPlanDTO, 0, len(plans))
	for _, p := range plans {
		p.NormalizeDefaults()
		result = append(result, SubscriptionPlanDTO{
			Plan: p,
		})
	}
	common.ApiSuccess(c, result)
}

type AdminUpsertSubscriptionPlanRequest struct {
	Plan model.SubscriptionPlan `json:"plan"`
}

func AdminCreateSubscriptionPlan(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}

	var req AdminUpsertSubscriptionPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	req.Plan.Id = 0
	if strings.TrimSpace(req.Plan.Title) == "" {
		common.ApiErrorMsg(c, "套餐标题不能为空")
		return
	}
	if req.Plan.PriceAmount < 0 {
		common.ApiErrorMsg(c, "价格不能为负数")
		return
	}
	if req.Plan.PriceAmount > 9999 {
		common.ApiErrorMsg(c, "价格不能超过9999")
		return
	}
	if req.Plan.Currency == "" {
		req.Plan.Currency = "USD"
	}
	req.Plan.Currency = "USD"
	if req.Plan.AllowBalancePay == nil {
		req.Plan.AllowBalancePay = common.GetPointer(true)
	}
	if req.Plan.AllowWalletOverflow == nil {
		req.Plan.AllowWalletOverflow = common.GetPointer(true)
	}
	if req.Plan.DurationUnit == "" {
		req.Plan.DurationUnit = model.SubscriptionDurationMonth
	}
	if req.Plan.DurationValue <= 0 && req.Plan.DurationUnit != model.SubscriptionDurationCustom {
		req.Plan.DurationValue = 1
	}
	if req.Plan.MaxPurchasePerUser < 0 {
		common.ApiErrorMsg(c, "购买上限不能为负数")
		return
	}
	if req.Plan.TotalAmount < 0 {
		common.ApiErrorMsg(c, "总额度不能为负数")
		return
	}
	req.Plan.UpgradeGroup = strings.TrimSpace(req.Plan.UpgradeGroup)
	if req.Plan.UpgradeGroup != "" {
		if _, ok := ratio_setting.GetGroupRatioCopy()[req.Plan.UpgradeGroup]; !ok {
			common.ApiErrorMsg(c, "升级分组不存在")
			return
		}
	}
	req.Plan.DowngradeGroup = strings.TrimSpace(req.Plan.DowngradeGroup)
	if req.Plan.DowngradeGroup != "" {
		if _, ok := ratio_setting.GetGroupRatioCopy()[req.Plan.DowngradeGroup]; !ok {
			common.ApiErrorMsg(c, "降级分组不存在")
			return
		}
	}
	req.Plan.QuotaResetPeriod = model.NormalizeResetPeriod(req.Plan.QuotaResetPeriod)
	if req.Plan.QuotaResetPeriod == model.SubscriptionResetCustom && req.Plan.QuotaResetCustomSeconds <= 0 {
		common.ApiErrorMsg(c, "自定义重置周期需大于0秒")
		return
	}
	if err := model.ValidateSubscriptionPlanDuration(&req.Plan); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	if errMsg := normalizeSubscriptionPlanTokenPackage(&req.Plan); errMsg != "" {
		common.ApiErrorMsg(c, errMsg)
		return
	}
	err := model.DB.Create(&req.Plan).Error
	if err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidateSubscriptionPlanCache(req.Plan.Id)
	common.ApiSuccess(c, req.Plan)
}

func AdminUpdateSubscriptionPlan(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}

	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}
	var req AdminUpsertSubscriptionPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if strings.TrimSpace(req.Plan.Title) == "" {
		common.ApiErrorMsg(c, "套餐标题不能为空")
		return
	}
	if req.Plan.PriceAmount < 0 {
		common.ApiErrorMsg(c, "价格不能为负数")
		return
	}
	if req.Plan.PriceAmount > 9999 {
		common.ApiErrorMsg(c, "价格不能超过9999")
		return
	}
	req.Plan.Id = id
	if req.Plan.Currency == "" {
		req.Plan.Currency = "USD"
	}
	req.Plan.Currency = "USD"
	if req.Plan.DurationUnit == "" {
		req.Plan.DurationUnit = model.SubscriptionDurationMonth
	}
	if req.Plan.DurationValue <= 0 && req.Plan.DurationUnit != model.SubscriptionDurationCustom {
		req.Plan.DurationValue = 1
	}
	if req.Plan.MaxPurchasePerUser < 0 {
		common.ApiErrorMsg(c, "购买上限不能为负数")
		return
	}
	if req.Plan.TotalAmount < 0 {
		common.ApiErrorMsg(c, "总额度不能为负数")
		return
	}
	req.Plan.UpgradeGroup = strings.TrimSpace(req.Plan.UpgradeGroup)
	if req.Plan.UpgradeGroup != "" {
		if _, ok := ratio_setting.GetGroupRatioCopy()[req.Plan.UpgradeGroup]; !ok {
			common.ApiErrorMsg(c, "升级分组不存在")
			return
		}
	}
	req.Plan.DowngradeGroup = strings.TrimSpace(req.Plan.DowngradeGroup)
	if req.Plan.DowngradeGroup != "" {
		if _, ok := ratio_setting.GetGroupRatioCopy()[req.Plan.DowngradeGroup]; !ok {
			common.ApiErrorMsg(c, "降级分组不存在")
			return
		}
	}
	req.Plan.QuotaResetPeriod = model.NormalizeResetPeriod(req.Plan.QuotaResetPeriod)
	if req.Plan.QuotaResetPeriod == model.SubscriptionResetCustom && req.Plan.QuotaResetCustomSeconds <= 0 {
		common.ApiErrorMsg(c, "自定义重置周期需大于0秒")
		return
	}
	if err := model.ValidateSubscriptionPlanDuration(&req.Plan); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	if errMsg := normalizeSubscriptionPlanTokenPackage(&req.Plan); errMsg != "" {
		common.ApiErrorMsg(c, errMsg)
		return
	}

	err := model.DB.Transaction(func(tx *gorm.DB) error {
		// update plan (allow zero values updates with map)
		updateMap := map[string]any{
			"title":                      req.Plan.Title,
			"subtitle":                   req.Plan.Subtitle,
			"price_amount":               req.Plan.PriceAmount,
			"currency":                   req.Plan.Currency,
			"duration_unit":              req.Plan.DurationUnit,
			"duration_value":             req.Plan.DurationValue,
			"custom_seconds":             req.Plan.CustomSeconds,
			"enabled":                    req.Plan.Enabled,
			"sort_order":                 req.Plan.SortOrder,
			"stripe_price_id":            req.Plan.StripePriceId,
			"creem_product_id":           req.Plan.CreemProductId,
			"waffo_pancake_product_id":   req.Plan.WaffoPancakeProductId,
			"max_purchase_per_user":      req.Plan.MaxPurchasePerUser,
			"total_amount":               req.Plan.TotalAmount,
			"content_type":               req.Plan.ContentType,
			"upgrade_group":              req.Plan.UpgradeGroup,
			"downgrade_group":            req.Plan.DowngradeGroup,
			"quota_reset_period":         req.Plan.QuotaResetPeriod,
			"quota_reset_custom_seconds": req.Plan.QuotaResetCustomSeconds,
			"token_amount":               req.Plan.TokenAmount,
			"token_count_mode":           req.Plan.TokenCountMode,
			"token_groups":               req.Plan.TokenGroups,
			"token_models":               req.Plan.TokenModels,
			"max_concurrency":            req.Plan.MaxConcurrency,
			"updated_at":                 common.GetTimestamp(),
		}
		if req.Plan.AllowBalancePay != nil {
			updateMap["allow_balance_pay"] = *req.Plan.AllowBalancePay
		}
		if req.Plan.AllowWalletOverflow != nil {
			updateMap["allow_wallet_overflow"] = *req.Plan.AllowWalletOverflow
		}
		if err := tx.Model(&model.SubscriptionPlan{}).Where("id = ?", id).Updates(updateMap).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidateSubscriptionPlanCache(id)
	common.ApiSuccess(c, nil)
}

type AdminUpdateSubscriptionPlanStatusRequest struct {
	Enabled *bool `json:"enabled"`
}

func AdminUpdateSubscriptionPlanStatus(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}

	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}
	var req AdminUpdateSubscriptionPlanStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.DB.Model(&model.SubscriptionPlan{}).Where("id = ?", id).Update("enabled", *req.Enabled).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidateSubscriptionPlanCache(id)
	common.ApiSuccess(c, nil)
}

type AdminBindSubscriptionRequest struct {
	UserId int `json:"user_id"`
	PlanId int `json:"plan_id"`
}

func AdminBindSubscription(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}

	var req AdminBindSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.UserId <= 0 || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	msg, err := model.AdminBindSubscription(req.UserId, req.PlanId, "")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if msg != "" {
		common.ApiSuccess(c, gin.H{"message": msg})
		return
	}
	common.ApiSuccess(c, nil)
}

// ---- Admin: user subscription management ----

func AdminListUserSubscriptions(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Param("id"))
	if userId <= 0 {
		common.ApiErrorMsg(c, "无效的用户ID")
		return
	}
	subs, err := model.GetAllUserSubscriptions(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, subs)
}

type AdminCreateUserSubscriptionRequest struct {
	PlanId int `json:"plan_id"`
}

type AdminResetSubscriptionRequest struct {
	PlanId           int   `json:"plan_id"`
	AdvanceResetTime *bool `json:"advance_reset_time"`
}

func resolveAdvanceResetTime(value *bool) bool {
	if value == nil {
		return true
	}
	return *value
}

func recordSubscriptionResetUserLogs(c *gin.Context, result *model.SubscriptionResetResult, adminInfo *model.AuditAdminInfo) {
	if result == nil || result.ResetCount == 0 {
		return
	}
	content := fmt.Sprintf("管理员重置订阅套餐 %s（ID: %d）额度", result.PlanTitle, result.PlanId)
	for _, userId := range result.AffectedUserIds {
		model.RecordLogWithAdminInfo(userId, model.LogTypeManage, content, adminInfo, nil, c)
	}
}

// AdminCreateUserSubscription creates a new user subscription from a plan (no payment).
func AdminCreateUserSubscription(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}

	userId, _ := strconv.Atoi(c.Param("id"))
	if userId <= 0 {
		common.ApiErrorMsg(c, "无效的用户ID")
		return
	}
	var req AdminCreateUserSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	msg, err := model.AdminBindSubscription(userId, req.PlanId, "")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if msg != "" {
		common.ApiSuccess(c, gin.H{"message": msg})
		return
	}
	common.ApiSuccess(c, nil)
}

func AdminResetUserSubscriptionsByPlan(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Param("id"))
	if userId <= 0 {
		common.ApiErrorMsg(c, "无效的用户ID")
		return
	}
	var req AdminResetSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	advanceResetTime := resolveAdvanceResetTime(req.AdvanceResetTime)
	result, err := model.AdminResetUserSubscriptionsByPlan(userId, req.PlanId, advanceResetTime)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordSubscriptionResetUserLogs(c, result, auditOperatorInfo(c))
	recordManageAuditFor(c, userId, "subscription.user_plan_reset", map[string]any{
		"target_user_id":     userId,
		"plan_id":            result.PlanId,
		"plan_title":         result.PlanTitle,
		"reset_count":        result.ResetCount,
		"user_count":         result.UserCount,
		"advance_reset_time": result.AdvanceResetTime,
	})
	common.ApiSuccess(c, result)
}

func AdminResetPlanSubscriptions(c *gin.Context) {
	planId, _ := strconv.Atoi(c.Param("id"))
	if planId <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}
	var req AdminResetSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	advanceResetTime := resolveAdvanceResetTime(req.AdvanceResetTime)
	result, err := model.AdminResetPlanSubscriptions(planId, advanceResetTime)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordSubscriptionResetUserLogs(c, result, auditOperatorInfo(c))
	common.SysLog(fmt.Sprintf("admin reset subscription plan %d quota: reset_count=%d user_count=%d advance_reset_time=%t",
		result.PlanId, result.ResetCount, result.UserCount, result.AdvanceResetTime))
	recordManageAudit(c, "subscription.plan_reset", map[string]any{
		"plan_id":            result.PlanId,
		"plan_title":         result.PlanTitle,
		"reset_count":        result.ResetCount,
		"user_count":         result.UserCount,
		"advance_reset_time": result.AdvanceResetTime,
	})
	common.ApiSuccess(c, result)
}

// AdminInvalidateUserSubscription cancels a user subscription immediately.
func AdminInvalidateUserSubscription(c *gin.Context) {
	subId, _ := strconv.Atoi(c.Param("id"))
	if subId <= 0 {
		common.ApiErrorMsg(c, "无效的订阅ID")
		return
	}
	msg, err := model.AdminInvalidateUserSubscription(subId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if msg != "" {
		common.ApiSuccess(c, gin.H{"message": msg})
		return
	}
	common.ApiSuccess(c, nil)
}

// AdminDeleteUserSubscription hard-deletes a user subscription.
func AdminDeleteUserSubscription(c *gin.Context) {
	subId, _ := strconv.Atoi(c.Param("id"))
	if subId <= 0 {
		common.ApiErrorMsg(c, "无效的订阅ID")
		return
	}
	msg, err := model.AdminDeleteUserSubscription(subId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if msg != "" {
		common.ApiSuccess(c, gin.H{"message": msg})
		return
	}
	common.ApiSuccess(c, nil)
}
