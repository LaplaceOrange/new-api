package controller

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRelayErrorLoggingConfiguration(t *testing.T) {
	if dialect := os.Getenv("RELAY_ERROR_LOG_TEST_DIALECT"); dialect != "" {
		common.InitEnv()
		gin.SetMode(gin.TestMode)
		common.RedisEnabled = false

		mainDB, _ := newAuditTestDatabase(t, "sqlite", "")
		logDB, _ := newAuditTestDatabase(t, dialect, os.Getenv("RELAY_ERROR_LOG_TEST_DSN"))
		for _, database := range []*gorm.DB{mainDB, logDB} {
			sqlDB, err := database.DB()
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
		}
		model.DB, model.LOG_DB = mainDB, logDB
		logType := common.DatabaseTypeSQLite
		switch dialect {
		case "mysql":
			logType = common.DatabaseTypeMySQL
		case "postgres":
			logType = common.DatabaseTypePostgreSQL
		}
		common.SetDatabaseTypes(common.DatabaseTypeSQLite, logType)
		require.NoError(t, mainDB.AutoMigrate(&model.User{}))
		require.NoError(t, logDB.AutoMigrate(&model.Log{}))
		require.NoError(t, mainDB.Create(&model.User{Id: 7, Username: "log-owner", Group: "default", Quota: 1000}).Error)
		versionQuery := "SELECT VERSION()"
		if dialect == "sqlite" {
			versionQuery = "SELECT sqlite_version()"
		}
		var version string
		require.NoError(t, logDB.Raw(versionQuery).Scan(&version).Error)
		t.Logf("log database: %s %s", dialect, version)

		enabled, err := strconv.ParseBool(os.Getenv("RELAY_ERROR_LOG_TEST_ENABLED"))
		require.NoError(t, err)
		for _, status := range []int{http.StatusBadRequest, http.StatusTooManyRequests, http.StatusBadGateway} {
			t.Run(strconv.Itoa(status), func(t *testing.T) {
				recorder := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(recorder)
				ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
				ctx.Set("id", 7)
				ctx.Set("username", "log-owner")
				ctx.Set("token_id", 11)
				ctx.Set("token_name", "test-token")
				ctx.Set("group", "default")
				ctx.Set("original_model", "test-model")
				ctx.Set(common.RequestIdKey, fmt.Sprintf("failed-request-%d", status))
				common.SetContextKey(ctx, constant.ContextKeyRequestStartTime, time.Now().Add(-time.Second))
				apiErr := service.RelayErrorHandler(ctx, &http.Response{
					StatusCode: status,
					Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"provider rejected request","type":"upstream_error","code":"provider_failure"}}`)),
				}, false)
				processChannelError(ctx, types.ChannelError{ChannelId: 202, AutoBan: false}, apiErr, nil)
				logs, total, err := model.GetUserLogs(7, model.LogTypeUnknown, 0, 0, "", "", 0, 10, "", ctx.GetString(common.RequestIdKey), "")
				require.NoError(t, err)
				if !enabled {
					assert.Zero(t, total)
					assert.Empty(t, logs)
					return
				}
				require.EqualValues(t, 1, total)
				require.Len(t, logs, 1)
				entry := logs[0]
				assert.Equal(t, model.LogTypeError, entry.Type)
				assert.Equal(t, 202, entry.ChannelId)
				assert.Equal(t, "test-model", entry.ModelName)
				assert.Equal(t, "test-token", entry.TokenName)
				assert.Equal(t, 11, entry.TokenId)
				assert.Zero(t, entry.Quota)
				assert.Contains(t, entry.Content, "provider rejected request")
				other, err := common.StrToMap(entry.Other)
				require.NoError(t, err)
				assert.Equal(t, float64(status), other["status_code"])
				assert.Equal(t, "/v1/chat/completions", other["request_path"])
				assert.NotContains(t, other, "admin_info")
			})
		}
		return
	}

	// Environment initialization changes process-wide settings; isolate each case.
	for _, database := range []struct{ dialect, env string }{
		{"sqlite", ""}, {"mysql", "TEST_MYSQL_DSN"}, {"postgres", "TEST_POSTGRES_DSN"},
	} {
		t.Run(database.dialect, func(t *testing.T) {
			dsn := os.Getenv(database.env)
			if database.env != "" && dsn == "" {
				t.Skip(database.env + " not configured")
			}
			for _, config := range []struct {
				name, value string
				enabled     bool
			}{{"default", "", true}, {"enabled", "true", true}, {"disabled", "false", false}} {
				t.Run(config.name, func(t *testing.T) {
					t.Setenv("ERROR_LOG_ENABLED", config.value)
					t.Setenv("RELAY_ERROR_LOG_TEST_DIALECT", database.dialect)
					t.Setenv("RELAY_ERROR_LOG_TEST_DSN", dsn)
					t.Setenv("RELAY_ERROR_LOG_TEST_ENABLED", strconv.FormatBool(config.enabled))
					command := exec.Command(os.Args[0], "-test.run=^TestRelayErrorLoggingConfiguration$", "-test.v", "-log-dir="+t.TempDir())
					output, err := command.CombinedOutput()
					t.Log(string(output))
					require.NoError(t, err)
				})
			}
		})
	}
}

func TestProcessChannelErrorUsesSnapshotWithoutLeakingChannelMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousRedisEnabled := common.RedisEnabled
	previousMainDatabaseType := common.MainDatabaseType()
	previousLogDatabaseType := common.LogDatabaseType()
	previousErrorLogEnabled := constant.ErrorLogEnabled

	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(&model.User{}, &model.Log{}))
	model.DB, model.LOG_DB = database, database
	common.RedisEnabled = false
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	constant.ErrorLogEnabled = true
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.RedisEnabled = previousRedisEnabled
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		constant.ErrorLogEnabled = previousErrorLogEnabled
		require.NoError(t, sqlDB.Close())
	})

	require.NoError(t, database.Create(&model.User{Id: 7, Username: "log-owner", Group: "default"}).Error)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Set("id", 7)
	ctx.Set("username", "log-owner")
	ctx.Set("token_name", "test-token")
	ctx.Set("token_id", 11)
	ctx.Set("original_model", "gpt-test")
	ctx.Set("group", "default")
	ctx.Set("channel_id", 202)
	ctx.Set("channel_name", "mutable-context-channel")
	ctx.Set("channel_type", 9)
	ctx.Set("use_channel", []string{"101"})
	common.SetContextKey(ctx, constant.ContextKeyRequestStartTime, time.Now().Add(-time.Second))

	channelSnapshot := types.ChannelError{
		ChannelId:   101,
		ChannelType: 1,
		ChannelName: "snapshot-channel",
		AutoBan:     false,
	}
	apiErr := types.NewOpenAIError(errors.New("upstream failed"), types.ErrorCodeBadResponseStatusCode, http.StatusBadGateway)

	processChannelError(ctx, channelSnapshot, apiErr, nil)

	var stored model.Log
	require.NoError(t, database.First(&stored).Error)
	assert.Equal(t, channelSnapshot.ChannelId, stored.ChannelId)
	storedOther, err := common.StrToMap(stored.Other)
	require.NoError(t, err)
	assert.Equal(t, float64(http.StatusBadGateway), storedOther["status_code"])
	for _, key := range []string{"channel_id", "channel_name", "channel_type"} {
		assert.NotContains(t, storedOther, key)
	}
	adminInfo, ok := storedOther["admin_info"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []any{"101"}, adminInfo["use_channel"])

	logs, total, err := model.GetUserLogs(7, model.LogTypeError, 0, 0, "", "", 0, 10, "", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, logs, 1)
	assert.Equal(t, channelSnapshot.ChannelId, logs[0].ChannelId)
	assert.Empty(t, logs[0].ChannelName)
	userOther, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	assert.NotContains(t, userOther, "admin_info")
	for _, key := range []string{"channel_id", "channel_name", "channel_type"} {
		assert.NotContains(t, userOther, key)
	}
}
