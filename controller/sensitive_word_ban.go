package controller

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type sensitiveWordBanLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type sensitiveWordBanCodeRequest struct {
	FlowToken string `json:"flow_token"`
	Code      string `json:"code"`
}

type sensitiveWordBanConfirmRequest struct {
	FlowToken string `json:"flow_token"`
}

type sensitiveWordBanFlowPayload struct {
	IP          string `json:"ip"`
	AuthVersion int64  `json:"auth_version"`
	UserID      int    `json:"user_id"`
}

func currentSensitiveWordBanIP(c *gin.Context) string {
	return model.NormalizeSensitiveWordIP(c.ClientIP())
}

func createSensitiveWordBanConfirmFlow(ip string, user *model.User) (string, error) {
	payload, err := common.Marshal(sensitiveWordBanFlowPayload{
		IP:          ip,
		AuthVersion: user.AuthVersion,
		UserID:      user.Id,
	})
	if err != nil {
		return "", err
	}
	token, _, err := model.CreateAuthFlow(model.AuthFlowCreate{
		Purpose:   model.AuthFlowPurposeIPBanConfirm,
		UserId:    user.Id,
		Payload:   string(payload),
		ExpiresAt: time.Now().Add(5 * time.Minute),
	})
	return token, err
}

func SensitiveWordBanLogin(c *gin.Context) {
	banned, err := model.IsSensitiveWordIPBanned(currentSensitiveWordBanIP(c))
	if err != nil || !banned {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "登录失败"})
		return
	}
	var request sensitiveWordBanLoginRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "登录失败"})
		return
	}
	user := &model.User{Username: strings.TrimSpace(request.Username), Password: request.Password}
	if err := user.ValidateAndFill(); err != nil || user.Role < common.RoleAdminUser {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "登录失败"})
		return
	}
	ip := currentSensitiveWordBanIP(c)
	twoFAEnabled, err := model.IsTwoFAEnabled(user.Id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "登录失败"})
		return
	}
	if twoFAEnabled {
		payload, marshalErr := common.Marshal(sensitiveWordBanFlowPayload{
			IP:          ip,
			AuthVersion: user.AuthVersion,
			UserID:      user.Id,
		})
		if marshalErr != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "登录失败"})
			return
		}
		flowToken, _, flowErr := model.CreateAuthFlow(model.AuthFlowCreate{
			Purpose:   model.AuthFlowPurposeIPBan2FA,
			UserId:    user.Id,
			Payload:   string(payload),
			ExpiresAt: time.Now().Add(5 * time.Minute),
		})
		if flowErr != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "登录失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"require_2fa": true, "flow_token": flowToken}})
		return
	}
	confirmToken, err := createSensitiveWordBanConfirmFlow(ip, user)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "登录失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"confirm_token": confirmToken}})
}

func SensitiveWordBanVerify2FA(c *gin.Context) {
	banned, err := model.IsSensitiveWordIPBanned(currentSensitiveWordBanIP(c))
	if err != nil || !banned {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "登录失败"})
		return
	}
	var request sensitiveWordBanCodeRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil || request.FlowToken == "" || request.Code == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "登录失败"})
		return
	}
	flow, err := model.GetAuthFlow(request.FlowToken, model.AuthFlowMatch{Purpose: model.AuthFlowPurposeIPBan2FA})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "登录失败"})
		return
	}
	var payload sensitiveWordBanFlowPayload
	if common.UnmarshalJsonStr(flow.Payload, &payload) != nil || payload.UserID <= 0 || payload.IP != currentSensitiveWordBanIP(c) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "登录失败"})
		return
	}
	user, err := model.GetUserById(payload.UserID, false)
	if err != nil || user.Role < common.RoleAdminUser || user.Status != common.UserStatusEnabled || user.AuthVersion != payload.AuthVersion {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "登录失败"})
		return
	}
	twoFA, err := model.GetTwoFAByUserId(user.Id)
	if err != nil || twoFA == nil || !twoFA.IsEnabled {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "登录失败"})
		return
	}
	cleanCode, codeErr := common.ValidateNumericCode(request.Code)
	valid := false
	if codeErr == nil {
		valid, _ = twoFA.ValidateTOTPAndUpdateUsage(cleanCode)
	}
	if !valid {
		valid, _ = twoFA.ValidateBackupCodeAndUpdateUsage(request.Code)
	}
	if !valid {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "登录失败"})
		return
	}
	if _, err := model.ConsumeAuthFlow(request.FlowToken, model.AuthFlowMatch{Purpose: model.AuthFlowPurposeIPBan2FA, UserId: user.Id}); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "登录失败"})
		return
	}
	confirmToken, err := createSensitiveWordBanConfirmFlow(payload.IP, user)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "登录失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"confirm_token": confirmToken}})
}

func SensitiveWordBanConfirm(c *gin.Context) {
	banned, err := model.IsSensitiveWordIPBanned(currentSensitiveWordBanIP(c))
	if err != nil || !banned {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "解封失败"})
		return
	}
	var request sensitiveWordBanConfirmRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil || request.FlowToken == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "解封失败"})
		return
	}
	ip := currentSensitiveWordBanIP(c)
	var payload sensitiveWordBanFlowPayload
	flow, err := model.GetAuthFlow(request.FlowToken, model.AuthFlowMatch{Purpose: model.AuthFlowPurposeIPBanConfirm})
	if err != nil || common.UnmarshalJsonStr(flow.Payload, &payload) != nil || payload.IP != ip || payload.UserID <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "解封失败"})
		return
	}
	user, err := model.GetUserById(payload.UserID, false)
	if err != nil || user.Role < common.RoleAdminUser || user.Status != common.UserStatusEnabled || user.AuthVersion != payload.AuthVersion {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "解封失败"})
		return
	}
	_, err = model.ConsumeAuthFlowWithAction(request.FlowToken, model.AuthFlowMatch{Purpose: model.AuthFlowPurposeIPBanConfirm, UserId: user.Id}, func(tx *gorm.DB, _ *model.AuthFlow) error {
		var current model.User
		if err := tx.First(&current, user.Id).Error; err != nil {
			return err
		}
		if current.Role < common.RoleAdminUser || current.Status != common.UserStatusEnabled || current.AuthVersion != payload.AuthVersion {
			return fmt.Errorf("administrator authorization changed")
		}
		if err := tx.Where("ip = ?", ip).Delete(&model.SensitiveWordIPBan{}).Error; err != nil {
			return err
		}
		return tx.Where("ip = ?", ip).Delete(&model.SensitiveWordUserIPBan{}).Error
	})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "解封失败"})
		return
	}
	bundle, err := service.CreateLoginSessionAtAuthVersion(user.Id, user.AuthVersion, "ip_ban_unban", c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "解封成功，但登录失败"})
		return
	}
	service.WriteRefreshCookie(c, bundle.RefreshToken)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"access_token":      bundle.AccessToken,
			"token_type":        bundle.TokenType,
			"access_expires_at": bundle.AccessExpiresAt,
			"user":              buildSelfUserData(user),
			"session":           bundle.Session,
		},
	})
}

func GetSensitiveWordUserBans(c *gin.Context) {
	page := common.GetPageQuery(c)
	rows, total, err := model.ListSensitiveWordUserBans(c.Query("search"), page.GetStartIdx(), page.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		user, userErr := model.GetUserById(row.UserId, false)
		username := ""
		if userErr == nil && user != nil {
			username = user.Username
		}
		items = append(items, gin.H{"id": row.Id, "user_id": row.UserId, "username": username, "hit_count": row.HitCount, "first_hit_at": row.FirstHitAt, "last_hit_at": row.LastHitAt, "banned_at": row.BannedAt})
	}
	page.SetTotal(int(total))
	page.SetItems(items)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": page})
}

func UnbanSensitiveWordUser(c *gin.Context) {
	var userID int
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &userID); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if err := model.UnbanSensitiveWordUser(userID); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func GetSensitiveWordIPBans(c *gin.Context) {
	page := common.GetPageQuery(c)
	rows, total, err := model.ListSensitiveWordIPBans(c.Query("search"), page.GetStartIdx(), page.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	page.SetTotal(int(total))
	page.SetItems(rows)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": page})
}

func UnbanSensitiveWordIP(c *gin.Context) {
	if err := model.UnbanSensitiveWordIP(c.Param("ip")); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
