package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func GetReferralRewards(c *gin.Context) {
	page := common.GetPageQuery(c)
	if page.GetPage() < 1 || page.GetPage() > 21_474_836 || page.GetPageSize() < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid referral page"})
		return
	}
	summary, err := model.GetReferralSummary(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	entries, err := model.ListReferralLedger(c.GetInt("id"), page.GetStartIdx(), page.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	transfers, err := model.ListReferralTransfers(c.GetInt("id"), page.GetStartIdx(), page.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var entryTotal, transferTotal int64
	if err := model.DB.Model(&model.ReferralLedger{}).Where("user_id = ?", c.GetInt("id")).Count(&entryTotal).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DB.Model(&model.ReferralTransfer{}).Where("user_id = ?", c.GetInt("id")).Count(&transferTotal).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"summary": summary, "entries": entries, "transfers": transfers,
		"entry_total": entryTotal, "transfer_total": transferTotal})
}

func TransferReferralRewards(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	var input struct {
		Quota int64 `json:"quota"`
	}
	if err := common.DecodeJson(c.Request.Body, &input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid transfer"})
		return
	}
	transfer, err := model.TransferReferralBalance(c.GetInt("id"), input.Quota)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	common.ApiSuccess(c, transfer)
}

func GetReferralAdmin(c *gin.Context) {
	campaigns, err := model.ListReferralCampaigns()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	policy, err := model.GetReferralPolicy()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	total, err := model.GetReferralTransferTotal()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"campaigns": campaigns, "policy": policy, "total_transferred": total})
}

func CreateReferralCampaign(c *gin.Context) {
	var campaign model.ReferralCampaign
	if err := common.DecodeJson(c.Request.Body, &campaign); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid campaign"})
		return
	}
	campaign.Id = 0
	if err := campaign.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := model.SaveReferralCampaign(&campaign); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, campaign)
}

func UpdateReferralCampaign(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid campaign ID"})
		return
	}
	var campaign model.ReferralCampaign
	if err := common.DecodeJson(c.Request.Body, &campaign); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid campaign"})
		return
	}
	campaign.Id = id
	if campaign.Enabled && !requirePaymentCompliance(c) {
		return
	}
	if err := campaign.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := model.SaveReferralCampaign(&campaign); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, campaign)
}

func UpdateReferralPolicy(c *gin.Context) {
	var policy model.ReferralPolicy
	if err := common.DecodeJson(c.Request.Body, &policy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid policy"})
		return
	}
	if err := model.SaveReferralPolicy(&policy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	common.ApiSuccess(c, policy)
}

func GetReferralWithdrawalStats(c *gin.Context) {
	startAt, startErr := strconv.ParseInt(c.Query("start_at"), 10, 64)
	endAt, endErr := strconv.ParseInt(c.Query("end_at"), 10, 64)
	if startErr != nil || endErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid report range"})
		return
	}
	buckets, err := model.GetReferralTransferBuckets(startAt, endAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	common.ApiSuccess(c, buckets)
}
