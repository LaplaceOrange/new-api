package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func GetUserCleanupRules(c *gin.Context) {
	config, err := model.GetUserCleanupRulesConfig()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, config)
}

func UpdateUserCleanupRules(c *gin.Context) {
	var config model.UserCleanupRulesConfig
	if err := common.DecodeJson(c.Request.Body, &config); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if err := model.SaveUserCleanupRulesConfig(config); err != nil {
		common.ApiError(c, err)
		return
	}
	saved, err := model.GetUserCleanupRulesConfig()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, saved)
}

func StartUserCleanupScan(c *gin.Context) {
	config, err := model.GetUserCleanupRulesConfig()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	task, err := service.StartUserCleanupScanTask(config, 0)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, task.ToResponse())
}

func GetUserCleanupScan(c *gin.Context) {
	task, err := model.GetActiveSystemTask(model.SystemTaskTypeUserCleanupScan)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if task == nil {
		task, err = model.GetLatestSystemTask(model.SystemTaskTypeUserCleanupScan)
		if err != nil {
			common.ApiError(c, err)
			return
		}
	}
	if task == nil {
		common.ApiSuccess(c, nil)
		return
	}
	common.ApiSuccess(c, task.ToResponse())
}

func GetUserCleanupCandidates(c *gin.Context) {
	scanID, err := currentUserCleanupScanID()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo := common.GetPageQuery(c)
	if scanID == "" {
		pageInfo.SetTotal(0)
		pageInfo.SetItems([]*model.User{})
		common.ApiSuccess(c, pageInfo)
		return
	}
	users, total, err := model.ListUserCleanupCandidateUsers(scanID, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(users)
	common.ApiSuccess(c, pageInfo)
}

func GetUserCleanupCandidateIDs(c *gin.Context) {
	scanID, err := currentUserCleanupScanID()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if scanID == "" {
		common.ApiSuccess(c, []int{})
		return
	}
	ids, err := model.ListUserCleanupCandidateIDs(scanID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, ids)
}

type applyUserCleanupRequest struct {
	UserIDs []int `json:"user_ids"`
}

func ApplyUserCleanup(c *gin.Context) {
	var req applyUserCleanupRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if len(req.UserIDs) == 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	task, err := model.GetLatestSystemTask(model.SystemTaskTypeUserCleanupScan)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if task == nil || task.Status == model.SystemTaskStatusPending || task.Status == model.SystemTaskStatusRunning {
		c.JSON(200, gin.H{
			"success": false,
			"message": "scan is not complete",
		})
		return
	}
	if len(req.UserIDs) > 2000 {
		req.UserIDs = req.UserIDs[:2000]
	}
	candidateIDs, err := model.FilterUserCleanupCandidateIDs(task.TaskID, req.UserIDs)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	now := common.GetTimestamp()
	operatorID := c.GetInt("id")
	disabled := make([]model.User, 0, len(candidateIDs))
	for _, userID := range candidateIDs {
		batch, err := model.DisableUsersForCleanup([]int{userID})
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if len(batch) == 0 {
			continue
		}
		user := batch[0]
		if err := model.InsertUserCleanupRecords([]model.UserCleanupRecord{{
			UserId:      user.Id,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Status:      user.Status,
			Group:       user.Group,
			Quota:       user.Quota,
			CleanedAt:   now,
			OperatorId:  operatorID,
		}}); err != nil {
			common.ApiError(c, err)
			return
		}
		recordManageAuditFor(c, user.Id, "user.manage", map[string]any{
			"action":   "disable",
			"username": user.Username,
			"id":       user.Id,
			"source":   "user_cleanup",
		})
		disabled = append(disabled, user)
	}
	if err := model.ClearUserCleanupCandidates(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"disabled_count": len(disabled),
	})
}

func GetUserCleanupHistory(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	records, total, err := model.ListUserCleanupRecords(pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(records)
	common.ApiSuccess(c, pageInfo)
}

func currentUserCleanupScanID() (string, error) {
	task, err := model.GetActiveSystemTask(model.SystemTaskTypeUserCleanupScan)
	if err != nil {
		return "", err
	}
	if task == nil {
		task, err = model.GetLatestSystemTask(model.SystemTaskTypeUserCleanupScan)
		if err != nil {
			return "", err
		}
	}
	if task == nil {
		return "", nil
	}
	return task.TaskID, nil
}
