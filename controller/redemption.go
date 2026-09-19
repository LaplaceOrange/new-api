package controller

import (
	"errors"
	"net/http"
	"strconv"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

type redemptionUpsertRequest struct {
	Id             int    `json:"id"`
	Name           string `json:"name"`
	Quota          int    `json:"quota"`
	Count          int    `json:"count"`
	ExpiredTime    int64  `json:"expired_time"`
	Status         int    `json:"status"`
	PlanId         int    `json:"plan_id"`
	MaxUses        *int   `json:"max_uses"`
	MaxUsesPerUser *int   `json:"max_uses_per_user"`
}

func (req redemptionUpsertRequest) usageLimits(defaultMaxUses, defaultMaxPerUser int) (int, int) {
	maxUses := defaultMaxUses
	if req.MaxUses != nil {
		maxUses = *req.MaxUses
	}
	maxPerUser := defaultMaxPerUser
	if req.MaxUsesPerUser != nil {
		maxPerUser = *req.MaxUsesPerUser
	}
	return maxUses, maxPerUser
}

func redemptionConfigError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, model.ErrRedemptionRewardInvalid) {
		common.ApiErrorI18n(c, i18n.MsgRedemptionRewardInvalid)
		return true
	}
	if errors.Is(err, model.ErrRedemptionInvalidPlan) {
		common.ApiErrorI18n(c, i18n.MsgRedemptionInvalidPlan)
		return true
	}
	if errors.Is(err, model.ErrRedemptionUsageLimitInvalid) {
		common.ApiErrorI18n(c, i18n.MsgRedemptionUsageLimitInvalid)
		return true
	}
	common.ApiError(c, err)
	return true
}

func GetAllRedemptions(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	redemptions, total, err := model.GetAllRedemptions(pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(redemptions)
	common.ApiSuccess(c, pageInfo)
	return
}

func SearchRedemptions(c *gin.Context) {
	keyword := c.Query("keyword")
	status := c.Query("status")
	pageInfo := common.GetPageQuery(c)
	redemptions, total, err := model.SearchRedemptions(keyword, status, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(redemptions)
	common.ApiSuccess(c, pageInfo)
	return
}

func GetRedemption(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	redemption, err := model.GetRedemptionById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    redemption,
	})
	return
}

func AddRedemption(c *gin.Context) {
	if !operation_setting.IsPaymentComplianceConfirmed() {
		common.ApiErrorI18n(c, i18n.MsgPaymentComplianceRequired)
		return
	}

	req := redemptionUpsertRequest{}
	err := c.ShouldBindJSON(&req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if utf8.RuneCountInString(req.Name) == 0 || utf8.RuneCountInString(req.Name) > 20 {
		common.ApiErrorI18n(c, i18n.MsgRedemptionNameLength)
		return
	}
	if req.Count <= 0 {
		common.ApiErrorI18n(c, i18n.MsgRedemptionCountPositive)
		return
	}
	if req.Count > 100 {
		common.ApiErrorI18n(c, i18n.MsgRedemptionCountMax)
		return
	}
	maxUses, maxUsesPerUser := req.usageLimits(1, 1)
	template := model.Redemption{
		Name:           req.Name,
		Quota:          req.Quota,
		ExpiredTime:    req.ExpiredTime,
		PlanId:         req.PlanId,
		MaxUses:        maxUses,
		MaxUsesPerUser: maxUsesPerUser,
	}
	if err := template.ValidateConfig(); err != nil {
		redemptionConfigError(c, err)
		return
	}
	if valid, msg := validateExpiredTime(c, req.ExpiredTime); !valid {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
		return
	}
	var keys []string
	for range req.Count {
		key := common.GetUUID()
		cleanRedemption := model.Redemption{
			UserId:         c.GetInt("id"),
			Name:           req.Name,
			Key:            key,
			CreatedTime:    common.GetTimestamp(),
			Quota:          req.Quota,
			ExpiredTime:    req.ExpiredTime,
			PlanId:         req.PlanId,
			MaxUses:        maxUses,
			MaxUsesPerUser: maxUsesPerUser,
		}
		err = cleanRedemption.Insert()
		if err != nil {
			common.SysError("failed to insert redemption: " + err.Error())
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": i18n.T(c, i18n.MsgRedemptionCreateFailed),
				"data":    keys,
			})
			return
		}
		keys = append(keys, key)
	}
	recordManageAudit(c, "redemption.create", map[string]any{
		"name":              req.Name,
		"count":             req.Count,
		"quota":             logger.LogQuota(req.Quota),
		"plan_id":           req.PlanId,
		"max_uses":          maxUses,
		"max_uses_per_user": maxUsesPerUser,
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    keys,
	})
	return
}

func DeleteRedemption(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := model.DeleteRedemptionById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func UpdateRedemption(c *gin.Context) {
	statusOnly := c.Query("status_only")
	req := redemptionUpsertRequest{}
	err := c.ShouldBindJSON(&req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanRedemption, err := model.GetRedemptionById(req.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if statusOnly == "" {
		maxUses, maxUsesPerUser := req.usageLimits(cleanRedemption.MaxUses, cleanRedemption.MaxUsesPerUser)
		if valid, msg := validateExpiredTime(c, req.ExpiredTime); !valid {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
			return
		}
		// If you add more fields, please also update redemption.Update()
		cleanRedemption.Name = req.Name
		cleanRedemption.Quota = req.Quota
		cleanRedemption.ExpiredTime = req.ExpiredTime
		cleanRedemption.PlanId = req.PlanId
		cleanRedemption.MaxUses = maxUses
		cleanRedemption.MaxUsesPerUser = maxUsesPerUser
	}
	if statusOnly != "" {
		cleanRedemption.Status = req.Status
	}
	err = cleanRedemption.Update()
	if err != nil {
		redemptionConfigError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    cleanRedemption,
	})
	return
}

func DeleteInvalidRedemption(c *gin.Context) {
	rows, err := model.DeleteInvalidRedemptions()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    rows,
	})
	return
}

func validateExpiredTime(c *gin.Context, expired int64) (bool, string) {
	if expired != 0 && expired < common.GetTimestamp() {
		return false, i18n.T(c, i18n.MsgRedemptionExpireTimeInvalid)
	}
	return true, ""
}

func DeleteRedemptionBatch(c *gin.Context) {
	var request struct {
		Ids []int `json:"ids" binding:"required,min=1,max=1000,dive,gt=0"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	count, err := model.BatchDeleteRedemptions(request.Ids)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "redemption.delete_batch", map[string]any{
		"count":                    count,
		"total":                    len(request.Ids),
		"requested_redemption_ids": request.Ids,
	})
	common.ApiSuccess(c, count)
}
