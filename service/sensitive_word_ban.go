package service

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

type SensitiveWordEnforcementResult struct {
	UserHitCount int
	IPHitCount   int
	UserBanned   bool
	IPBanned     bool
}

func HandleSensitiveWordHit(c *gin.Context, words []string) (SensitiveWordEnforcementResult, error) {
	if !setting.SensitiveWordAutoBanEnabled || !setting.ShouldCheckPromptSensitive() {
		return SensitiveWordEnforcementResult{}, nil
	}
	if c == nil || len(words) == 0 {
		return SensitiveWordEnforcementResult{}, errors.New("invalid sensitive word hit context")
	}
	if recorded, ok := c.Get("sensitive_word_hit_recorded"); ok {
		if result, ok := recorded.(SensitiveWordEnforcementResult); ok {
			return result, nil
		}
		return SensitiveWordEnforcementResult{}, nil
	}
	result, err := model.RecordSensitiveWordHit(c.GetInt("id"), c.ClientIP(), common.GetTimestamp())
	if err != nil {
		return SensitiveWordEnforcementResult{}, err
	}
	if result.UserBanned && c.GetInt("id") > 0 {
		if err := model.DisableUserForSensitiveWords(c.GetInt("id")); err != nil {
			return SensitiveWordEnforcementResult{}, err
		}
	}
	enforcement := SensitiveWordEnforcementResult{
		UserHitCount: result.UserHitCount,
		IPHitCount:   result.IPHitCount,
		UserBanned:   result.UserBanned,
		IPBanned:     result.IPBanned,
	}
	c.Set("sensitive_word_hit_recorded", enforcement)
	return enforcement, nil
}

func SensitiveWordBlockMessage(result SensitiveWordEnforcementResult) string {
	if result.UserBanned || result.IPBanned {
		return "此用户或 IP 已因敏感词命中被封禁，请联系管理员"
	}
	return "敏感词命中"
}
