package model

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordSensitiveWordHitUsesConfiguredUserAndIPThresholds(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&SensitiveWordUserBan{}, &SensitiveWordUserIPBan{}, &SensitiveWordIPBan{}))

	oldUserThreshold := setting.SensitiveWordUserBanThreshold
	oldIPThreshold := setting.SensitiveWordIPUserBanThreshold
	setting.SensitiveWordUserBanThreshold = 2
	setting.SensitiveWordIPUserBanThreshold = 2
	t.Cleanup(func() {
		setting.SensitiveWordUserBanThreshold = oldUserThreshold
		setting.SensitiveWordIPUserBanThreshold = oldIPThreshold
		DB.Where("1 = 1").Delete(&SensitiveWordUserIPBan{})
		DB.Where("1 = 1").Delete(&SensitiveWordUserBan{})
		DB.Where("1 = 1").Delete(&SensitiveWordIPBan{})
	})

	first, err := RecordSensitiveWordHit(101, "192.0.2.10", 100)
	require.NoError(t, err)
	assert.False(t, first.UserBanned)
	assert.False(t, first.IPBanned)

	second, err := RecordSensitiveWordHit(101, "192.0.2.10", 101)
	require.NoError(t, err)
	assert.True(t, second.UserBanned)
	assert.False(t, second.IPBanned)

	third, err := RecordSensitiveWordHit(102, "192.0.2.10", 102)
	require.NoError(t, err)
	assert.False(t, third.UserBanned)
	assert.False(t, third.IPBanned)

	fourth, err := RecordSensitiveWordHit(102, "192.0.2.10", 103)
	require.NoError(t, err)
	assert.True(t, fourth.UserBanned)
	assert.True(t, fourth.IPBanned)

	var associationCount int64
	require.NoError(t, DB.Model(&SensitiveWordUserIPBan{}).Where("ip = ?", "192.0.2.10").Distinct("user_id").Count(&associationCount).Error)
	assert.Equal(t, int64(2), associationCount)
}

func TestRecordSensitiveWordLogStoresUniqueMatchedWords(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Log{}))
	request := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = request
	context.Set("id", 101)
	context.Set("channel_id", 7)
	context.Set("original_model", "test-model")
	context.Set("token_name", "test-token")
	context.Set("token_id", 9)
	context.Set("group", "default")
	context.Set(common.RequestIdKey, "sensitive-request")

	RecordSensitiveWordLog(context, []string{"xxx", "xxx", " yyy "}, 999)

	var log Log
	require.NoError(t, LOG_DB.Where("request_id = ?", "sensitive-request").First(&log).Error)
	assert.Equal(t, LogTypeError, log.Type)
	assert.Equal(t, "sensitive words detected", log.Content)
	other, err := common.StrToMap(log.Other)
	require.NoError(t, err)
	assert.Equal(t, float64(2), other["sensitive_word_count"])
	assert.Equal(t, []interface{}{"xxx", "yyy"}, other["sensitive_words"])
}
