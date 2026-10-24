package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	hostdto "github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type listModelsResponse struct {
	Success bool               `json:"success"`
	Data    []dto.OpenAIModels `json:"data"`
	Object  string             `json:"object"`
}

type userModelsResponse struct {
	Success bool     `json:"success"`
	Data    []string `json:"data"`
}

func setupModelListControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	initModelListColumnNames(t)

	gin.SetMode(gin.TestMode)
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db

	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Channel{},
		&model.Ability{},
		&model.Model{},
		&model.Vendor{},
		&model.Option{},
		&model.ChannelContribution{},
		&model.ChannelContributionRevision{},
		&model.ChannelContributionModelHealth{},
	))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func initModelListColumnNames(t *testing.T) {
	t.Helper()

	originalIsMasterNode := common.IsMasterNode
	originalSQLitePath := common.SQLitePath
	originalMainDatabaseType := common.MainDatabaseType()
	originalLogDatabaseType := common.LogDatabaseType()
	originalSQLDSN, hadSQLDSN := os.LookupEnv("SQL_DSN")
	defer func() {
		common.IsMasterNode = originalIsMasterNode
		common.SQLitePath = originalSQLitePath
		common.SetDatabaseTypes(originalMainDatabaseType, originalLogDatabaseType)
		if hadSQLDSN {
			require.NoError(t, os.Setenv("SQL_DSN", originalSQLDSN))
		} else {
			require.NoError(t, os.Unsetenv("SQL_DSN"))
		}
	}()

	common.IsMasterNode = false
	common.SQLitePath = fmt.Sprintf("file:%s_init?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	require.NoError(t, os.Setenv("SQL_DSN", "local"))

	require.NoError(t, model.InitDB())
	if model.DB != nil {
		sqlDB, err := model.DB.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}
}

func withTieredBillingConfig(t *testing.T, modes map[string]string, exprs map[string]string) {
	t.Helper()

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		if strings.HasPrefix(key, "billing_setting.") {
			saved[key] = value
		}
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
		model.InvalidatePricingCache()
	})

	modeBytes, err := common.Marshal(modes)
	require.NoError(t, err)
	exprBytes, err := common.Marshal(exprs)
	require.NoError(t, err)

	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": string(modeBytes),
		"billing_setting.billing_expr": string(exprBytes),
	}))
	model.InvalidatePricingCache()
}

func withSelfUseModeDisabled(t *testing.T) {
	t.Helper()

	original := operation_setting.SelfUseModeEnabled
	operation_setting.SelfUseModeEnabled = false
	t.Cleanup(func() {
		operation_setting.SelfUseModeEnabled = original
	})
}

func withSelfUseModeEnabled(t *testing.T) {
	t.Helper()

	original := operation_setting.SelfUseModeEnabled
	operation_setting.SelfUseModeEnabled = true
	t.Cleanup(func() {
		operation_setting.SelfUseModeEnabled = original
	})
}

func decodeListModelsPayload(t *testing.T, recorder *httptest.ResponseRecorder) listModelsResponse {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload listModelsResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	require.Equal(t, "list", payload.Object)
	return payload
}

func decodeListModelsResponse(t *testing.T, recorder *httptest.ResponseRecorder) map[string]struct{} {
	t.Helper()

	payload := decodeListModelsPayload(t, recorder)
	ids := make(map[string]struct{}, len(payload.Data))
	for _, item := range payload.Data {
		ids[item.Id] = struct{}{}
	}
	return ids
}

func pricingByModelName(pricings []model.Pricing) map[string]model.Pricing {
	byName := make(map[string]model.Pricing, len(pricings))
	for _, pricing := range pricings {
		byName[pricing.ModelName] = pricing
	}
	return byName
}

func decodeUserModelsResponse(t *testing.T, recorder *httptest.ResponseRecorder) []string {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload userModelsResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	return payload.Data
}

func TestGetUserModelsFiltersByRequestedGroup(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	originalPolicies := setting.GetGroupModelPoliciesCopy()
	t.Cleanup(func() {
		raw, err := common.Marshal(originalPolicies)
		require.NoError(t, err)
		require.NoError(t, setting.UpdateGroupModelPoliciesByJSONString(string(raw)))
	})
	require.NoError(t, db.Create(&model.User{
		Id:       1002,
		Username: "playground-model-user",
		Password: "password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "default", Model: "zz-default-only-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-disabled-model", ChannelId: 1, Enabled: false},
	}).Error)

	defaultRecorder := httptest.NewRecorder()
	defaultContext, _ := gin.CreateTestContext(defaultRecorder)
	defaultContext.Request = httptest.NewRequest(http.MethodGet, "/api/user/models?group=default", nil)
	defaultContext.Set("id", 1002)

	GetUserModels(defaultContext)

	defaultModels := decodeUserModelsResponse(t, defaultRecorder)
	require.ElementsMatch(t, []string{"zz-default-only-model"}, defaultModels)

	require.NoError(t, setting.UpdateGroupModelPoliciesByJSONString(`{
		"default": {
			"mode": "blacklist",
			"blacklist": ["zz-default-only-model"],
			"whitelist": []
		}
	}`))
	restrictedRecorder := httptest.NewRecorder()
	restrictedContext, _ := gin.CreateTestContext(restrictedRecorder)
	restrictedContext.Request = httptest.NewRequest(http.MethodGet, "/api/user/models?group=default", nil)
	restrictedContext.Set("id", 1002)

	GetUserModels(restrictedContext)

	require.Empty(t, decodeUserModelsResponse(t, restrictedRecorder))

	vipRecorder := httptest.NewRecorder()
	vipContext, _ := gin.CreateTestContext(vipRecorder)
	vipContext.Request = httptest.NewRequest(http.MethodGet, "/api/user/models?group=vip", nil)
	vipContext.Set("id", 1002)

	GetUserModels(vipContext)

	require.Empty(t, decodeUserModelsResponse(t, vipRecorder))
}

func TestGetGroupModelCandidatesOnlyReturnsEnabledChannels(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	enabledChannel := &model.Channel{
		Name:   "candidate-enabled",
		Key:    "candidate-enabled-key",
		Status: common.ChannelStatusEnabled,
		Type:   constant.ChannelTypeOpenAI,
	}
	disabledChannel := &model.Channel{
		Name:   "candidate-disabled",
		Key:    "candidate-disabled-key",
		Status: common.ChannelStatusManuallyDisabled,
		Type:   constant.ChannelTypeOpenAI,
	}
	require.NoError(t, db.Create(enabledChannel).Error)
	require.NoError(t, db.Create(disabledChannel).Error)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "default", Model: "enabled-candidate", ChannelId: enabledChannel.Id, Enabled: true},
		{Group: "default", Model: "disabled-channel-candidate", ChannelId: disabledChannel.Id, Enabled: true},
		{Group: "default", Model: "disabled-ability-candidate", ChannelId: enabledChannel.Id, Enabled: false},
		{Group: "vip", Model: "other-group-candidate", ChannelId: enabledChannel.Id, Enabled: true},
	}).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/group/models?group=default", nil)

	GetGroupModelCandidates(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload struct {
		Data []string `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	assert.Equal(t, []string{"enabled-candidate"}, payload.Data)
}

func TestFilterPricingByGroupModelPoliciesKeepsModelsAllowedByAnyGroup(t *testing.T) {
	originalPolicies := setting.GetGroupModelPoliciesCopy()
	t.Cleanup(func() {
		raw, err := common.Marshal(originalPolicies)
		require.NoError(t, err)
		require.NoError(t, setting.UpdateGroupModelPoliciesByJSONString(string(raw)))
	})
	require.NoError(t, setting.UpdateGroupModelPoliciesByJSONString(`{
		"default": {
			"mode": "blacklist",
			"blacklist": ["shared-model"],
			"whitelist": []
		},
		"vip": {
			"mode": "whitelist",
			"blacklist": [],
			"whitelist": ["shared-model"]
		}
	}`))

	pricing := filterPricingByGroupModelPolicies([]model.Pricing{
		{ModelName: "shared-model", EnableGroup: []string{"default", "vip"}},
		{ModelName: "default-only-model", EnableGroup: []string{"default"}},
		{ModelName: "all-groups-model", EnableGroup: []string{"all"}},
		{ModelName: "shared-model", EnableGroup: []string{"default"}},
		{ModelName: "vip-denied-model", EnableGroup: []string{"vip"}},
	}, map[string]string{"default": "Default", "vip": "VIP"})

	require.Len(t, pricing, 3)
	assert.Equal(t, "shared-model", pricing[0].ModelName)
	assert.Equal(t, []string{"vip"}, pricing[0].EnableGroup)
	assert.Equal(t, "default-only-model", pricing[1].ModelName)
	assert.Equal(t, []string{"default"}, pricing[1].EnableGroup)
	assert.Equal(t, "all-groups-model", pricing[2].ModelName)
	assert.Equal(t, []string{"default"}, pricing[2].EnableGroup)
}

func TestGroupModelPoliciesDatabaseMatrix(t *testing.T) {
	require.NoError(t, i18n.Init())
	gin.SetMode(gin.TestMode)
	withSelfUseModeEnabled(t)
	originalPolicies := setting.GroupModelPolicies2JSONString()
	originalUsableGroups := setting.UserUsableGroups2JSONString()
	affinitySetting := operation_setting.GetChannelAffinitySetting()
	originalAffinity := *affinitySetting
	*affinitySetting = operation_setting.ChannelAffinitySetting{
		Enabled: true, DefaultTTLSeconds: 60,
		Rules: []operation_setting.ChannelAffinityRule{{
			Name: "group-model-policy-test", ModelRegex: []string{"^policy-model$"},
			KeySources:      []operation_setting.ChannelAffinityKeySource{{Type: "request_header", Key: "X-Policy-Session"}},
			IncludeRuleName: true, IncludeUsingGroup: true, IncludeModelName: true,
		}},
	}
	t.Cleanup(func() {
		_, err := service.ClearChannelAffinityCacheByRuleName("group-model-policy-test")
		require.NoError(t, err)
		*affinitySetting = originalAffinity
		require.NoError(t, setting.UpdateGroupModelPoliciesByJSONString(originalPolicies))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups))
	})

	for _, dialect := range []struct{ kind, env string }{{"sqlite", ""}, {"mysql", "TEST_MYSQL_DSN"}, {"postgres", "TEST_POSTGRES_DSN"}} {
		t.Run(dialect.kind, func(t *testing.T) {
			if dialect.env != "" && os.Getenv(dialect.env) == "" {
				t.Skip("set " + dialect.env + " to run this database")
			}
			db := modelManagementDB(t, dialect.kind, os.Getenv(dialect.env))
			require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP"}`))
			priority, weight := int64(0), uint(100)
			for index, group := range []string{"default", "vip"} {
				channel := model.Channel{
					Id: index + 1, Name: group, Key: "policy-test-key",
					Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled,
					Models: "policy-model", Group: group, Priority: &priority, Weight: &weight,
				}
				require.NoError(t, db.Create(&channel).Error)
				require.NoError(t, db.Create(&model.Ability{
					ChannelId: channel.Id, Model: "policy-model", Group: group,
					Enabled: true, Priority: &priority, Weight: weight,
				}).Error)
			}

			for _, mode := range []setting.GroupModelPolicyMode{
				setting.GroupModelPolicyModeBlacklist,
				setting.GroupModelPolicyModeWhitelist,
				setting.GroupModelPolicyModeAll,
				setting.GroupModelPolicyModeBlacklist,
			} {
				policies := map[string]setting.GroupModelPolicy{
					"default": {Mode: mode, Blacklist: []string{"policy-model"}, Whitelist: []string{"saved-offline-model"}},
				}
				raw, err := common.Marshal(policies)
				require.NoError(t, err)
				require.NoError(t, model.UpdateOption(setting.GroupModelPoliciesOptionKey, string(raw)))
				require.Error(t, model.UpdateOption(setting.GroupModelPoliciesOptionKey, `{"default":{"mode":"invalid"}}`))
				var stored model.Option
				require.NoError(t, db.Where(&model.Option{Key: setting.GroupModelPoliciesOptionKey}).First(&stored).Error)
				assert.Equal(t, string(raw), stored.Value)
				for range 2 {
					require.NoError(t, setting.UpdateGroupModelPoliciesByJSONString(`{}`))
					model.InitOptionMap()
					assert.Equal(t, policies["default"], setting.GetGroupModelPolicy("default"))
					assert.Equal(t, mode == setting.GroupModelPolicyModeAll, service.IsModelAllowedForGroup("default", "policy-model"))
				}
			}

			candidates, err := model.GetGroupModelsOnEnabledChannels("default")
			require.NoError(t, err)
			assert.Equal(t, []string{"policy-model"}, candidates, "policy-denied models must remain editable candidates")

			t.Run("model lists respect explicit and auto policies", func(t *testing.T) {
				for _, scenario := range []struct {
					name, group, policies string
					visible               bool
				}{
					{"explicit denied", "default", `{"default":{"mode":"whitelist"}}`, false},
					{"explicit allowed", "vip", `{"default":{"mode":"whitelist"}}`, true},
					{"auto union", "auto", `{"default":{"mode":"whitelist"}}`, true},
					{"auto all denied", "auto", `{"default":{"mode":"whitelist"},"vip":{"mode":"whitelist"}}`, false},
				} {
					t.Run(scenario.name, func(t *testing.T) {
						require.NoError(t, model.UpdateOption(setting.GroupModelPoliciesOptionKey, scenario.policies))
						for _, limited := range []bool{false, true} {
							recorder := httptest.NewRecorder()
							c, _ := gin.CreateTestContext(recorder)
							c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
							common.SetContextKey(c, constant.ContextKeyUserGroup, "default")
							common.SetContextKey(c, constant.ContextKeyTokenGroup, scenario.group)
							common.SetContextKey(c, constant.ContextKeyTokenAutoGroups, []string{"default", "vip"})
							common.SetContextKey(c, constant.ContextKeyTokenModelLimitEnabled, limited)
							common.SetContextKey(c, constant.ContextKeyTokenModelLimit, map[string]bool{"policy-model": true})
							ListModels(c, constant.ChannelTypeOpenAI)
							ids := decodeListModelsResponse(t, recorder)
							assert.Equal(t, scenario.visible, len(ids) == 1, recorder.Body.String())
							if scenario.visible {
								assert.Contains(t, ids, "policy-model")
							}
						}
					})
				}
			})

			for _, scenario := range []struct {
				name, group, policies string
				pin                   int
				affinity              bool
				status                int
				selectedGroup         string
			}{
				{"explicit denied", "default", `{"default":{"mode":"blacklist","blacklist":["policy-model"]}}`, 0, false, 503, ""},
				{"explicit empty whitelist", "default", `{"default":{"mode":"whitelist","whitelist":[]}}`, 0, false, 503, ""},
				{"unconfigured group", "default", `{}`, 0, false, 200, "default"},
				{"auto skips denied", "auto", `{"default":{"mode":"blacklist","blacklist":["policy-model"]}}`, 0, false, 200, "vip"},
				{"auto all denied", "auto", `{"default":{"mode":"whitelist"},"vip":{"mode":"whitelist"}}`, 0, false, 503, ""},
				{"fixed channel denied", "default", `{"default":{"mode":"blacklist","blacklist":["policy-model"]}}`, 1, false, 503, ""},
				{"fixed auto selects allowed group", "auto", `{"default":{"mode":"whitelist"}}`, 2, false, 200, "vip"},
				{"fixed auto denied", "auto", `{"default":{"mode":"whitelist"}}`, 1, false, 503, ""},
				{"affinity explicit denied", "default", `{"default":{"mode":"whitelist"}}`, 0, true, 503, ""},
				{"affinity auto skips denied", "auto", `{"default":{"mode":"whitelist"}}`, 0, true, 200, "vip"},
				{"affinity auto all denied", "auto", `{"default":{"mode":"whitelist"},"vip":{"mode":"whitelist"}}`, 0, true, 503, ""},
			} {
				t.Run(scenario.name, func(t *testing.T) {
					require.NoError(t, model.UpdateOption(setting.GroupModelPoliciesOptionKey, scenario.policies))
					sessionID := dialect.kind + "/" + scenario.name
					if scenario.affinity {
						seed, _ := gin.CreateTestContext(httptest.NewRecorder())
						seed.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
						seed.Request.Header.Set("X-Policy-Session", sessionID)
						service.GetPreferredChannelByAffinity(seed, "policy-model", scenario.group)
						service.RecordChannelAffinity(seed, 1)
						channelID, found := service.GetPreferredChannelByAffinity(seed, "policy-model", scenario.group)
						require.True(t, found)
						require.Equal(t, 1, channelID)
					}
					recorder := httptest.NewRecorder()
					router := gin.New()
					router.POST("/v1/chat/completions", func(c *gin.Context) {
						common.SetContextKey(c, constant.ContextKeyUserGroup, "default")
						common.SetContextKey(c, constant.ContextKeyUsingGroup, scenario.group)
						common.SetContextKey(c, constant.ContextKeyTokenAutoGroups, []string{"default", "vip"})
						if scenario.pin != 0 {
							service.GetChannelConstraints(c).AddPin(hostdto.ChannelPin{
								ChannelId: scenario.pin, Source: hostdto.PinSourceToken,
								Rank: hostdto.PinRankToken, RetryMode: hostdto.PinRetrySingleAttempt,
							})
						}
						c.Next()
					}, middleware.Distribute(), func(c *gin.Context) {
						selectedGroup := common.GetContextKeyString(c, constant.ContextKeyAutoGroup)
						if selectedGroup == "" {
							selectedGroup = common.GetContextKeyString(c, constant.ContextKeyUsingGroup)
						}
						assert.Equal(t, scenario.selectedGroup, selectedGroup)
						c.Status(http.StatusOK)
					})
					request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"policy-model"}`))
					request.Header.Set("Content-Type", "application/json")
					if scenario.affinity {
						request.Header.Set("X-Policy-Session", sessionID)
					}
					router.ServeHTTP(recorder, request)
					require.Equal(t, scenario.status, recorder.Code, recorder.Body.String())
					if scenario.status != http.StatusOK {
						var response struct {
							Error struct {
								Code string `json:"code"`
							} `json:"error"`
						}
						require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
						assert.Equal(t, "model_not_found", response.Error.Code)
					}
				})
			}
		})
	}
}

func TestGetUserModelsExpandsAutoGroupsInConfiguredOrder(t *testing.T) {
	originalAutoGroups := setting.AutoGroups2JsonString()
	originalUsableGroups := setting.UserUsableGroups2JSONString()
	originalSpecialGroups := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup.ReadAll()
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateAutoGroupsByJsonString(originalAutoGroups))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups))
		specialGroups := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup
		specialGroups.Clear()
		specialGroups.AddAll(originalSpecialGroups)
	})

	require.NoError(t, setting.UpdateAutoGroupsByJsonString(`["vip","default","unavailable"]`))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"auto":"自动分组","default":"默认分组","unavailable":"不可用分组"}`))
	specialGroups := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup
	specialGroups.Clear()
	specialGroups.Set("default", map[string]string{
		"+:vip":         "VIP 分组",
		"-:unavailable": "",
	})

	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:       1003,
		Username: "playground-auto-model-user",
		Password: "password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "vip", Model: "zz-vip-model", ChannelId: 1, Enabled: true},
		{Group: "vip", Model: "zz-shared-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-default-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-shared-model", ChannelId: 2, Enabled: true},
		{Group: "unavailable", Model: "zz-unavailable-model", ChannelId: 1, Enabled: true},
	}).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/user/models?group=auto", nil)
	context.Set("id", 1003)

	GetUserModels(context)

	models := decodeUserModelsResponse(t, recorder)
	require.Len(t, models, 3)
	assert.ElementsMatch(t, []string{"zz-vip-model", "zz-shared-model"}, models[:2])
	assert.Equal(t, "zz-default-model", models[2])
}

func TestListModelsIncludesTieredBillingModel(t *testing.T) {
	withSelfUseModeDisabled(t)
	withTieredBillingConfig(t, map[string]string{
		"zz-tiered-visible-model":      "tiered_expr",
		"zz-tiered-empty-expr-model":   "tiered_expr",
		"zz-tiered-missing-expr-model": "tiered_expr",
	}, map[string]string{
		"zz-tiered-visible-model":    `tier("base", p * 1 + c * 2)`,
		"zz-tiered-empty-expr-model": "   ",
	})

	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:       1001,
		Username: "model-list-user",
		Password: "password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "default", Model: "zz-tiered-visible-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-tiered-empty-expr-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-tiered-missing-expr-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-unpriced-model", ChannelId: 1, Enabled: true},
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	ctx.Set("id", 1001)

	ListModels(ctx, constant.ChannelTypeOpenAI)

	ids := decodeListModelsResponse(t, recorder)
	require.Contains(t, ids, "zz-tiered-visible-model")
	require.NotContains(t, ids, "zz-tiered-empty-expr-model")
	require.NotContains(t, ids, "zz-tiered-missing-expr-model")
	require.NotContains(t, ids, "zz-unpriced-model")

	pricingByName := pricingByModelName(model.GetPricing())
	visiblePricing, ok := pricingByName["zz-tiered-visible-model"]
	require.True(t, ok)
	require.Equal(t, "tiered_expr", visiblePricing.BillingMode)
	require.NotEmpty(t, visiblePricing.BillingExpr)

	emptyExprPricing, ok := pricingByName["zz-tiered-empty-expr-model"]
	require.True(t, ok)
	require.Empty(t, emptyExprPricing.BillingMode)
	require.Empty(t, emptyExprPricing.BillingExpr)

	missingExprPricing, ok := pricingByName["zz-tiered-missing-expr-model"]
	require.True(t, ok)
	require.Empty(t, missingExprPricing.BillingMode)
	require.Empty(t, missingExprPricing.BillingExpr)
}

func TestListModelsUsesAdvancedCustomEndpointTypesFromPricingCache(t *testing.T) {
	withSelfUseModeEnabled(t)
	db := setupModelListControllerTestDB(t)

	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	t.Cleanup(func() {
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		model.InvalidatePricingCache()
	})

	require.NoError(t, db.Create(&model.User{
		Id:       1003,
		Username: "advanced-custom-model-list-user",
		Password: "password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}).Error)

	channel := &model.Channel{
		Id:     701,
		Type:   constant.ChannelTypeAdvancedCustom,
		Key:    "advanced-custom-key",
		Status: common.ChannelStatusEnabled,
		Name:   "advanced-custom-channel",
		Group:  "default",
		Models: "gemini-3.5-flash",
	}
	channel.SetOtherSettings(dto.ChannelOtherSettings{
		AdvancedCustom: &dto.AdvancedCustomConfig{
			Routes: []dto.AdvancedCustomRoute{
				{
					IncomingPath: "/v1/chat/completions",
					UpstreamPath: "/v1/chat/completions",
				},
				{
					IncomingPath: "/v1/responses",
					UpstreamPath: "/v1beta/models/{model}:generateContent",
					Converter:    "openai_responses_to_gemini_generate_content",
					Models:       []string{"re:^gemini-"},
				},
			},
		},
	})
	require.NoError(t, db.Create(channel).Error)
	require.NoError(t, db.Create(&model.Ability{
		Group:     "default",
		Model:     "gemini-3.5-flash",
		ChannelId: 701,
		Enabled:   true,
	}).Error)

	model.InitChannelCache()
	model.GetPricing()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	ctx.Set("id", 1003)

	ListModels(ctx, constant.ChannelTypeOpenAI)

	payload := decodeListModelsPayload(t, recorder)
	require.Len(t, payload.Data, 1)
	require.Equal(t, "gemini-3.5-flash", payload.Data[0].Id)
	require.Equal(t, []constant.EndpointType{
		constant.EndpointTypeOpenAI,
		constant.EndpointTypeOpenAIResponse,
	}, payload.Data[0].SupportedEndpointTypes)
}

func TestListModelsTokenLimitIncludesTieredBillingModel(t *testing.T) {
	withSelfUseModeDisabled(t)
	withTieredBillingConfig(t, map[string]string{
		"zz-token-tiered-visible-model":      "tiered_expr",
		"zz-token-tiered-empty-expr-model":   "tiered_expr",
		"zz-token-tiered-missing-expr-model": "tiered_expr",
	}, map[string]string{
		"zz-token-tiered-visible-model":    `tier("base", p * 1 + c * 2)`,
		"zz-token-tiered-empty-expr-model": "",
	})
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "default", Model: "zz-token-tiered-visible-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-token-tiered-empty-expr-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-token-tiered-missing-expr-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-token-unpriced-model", ChannelId: 1, Enabled: true},
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	common.SetContextKey(ctx, constant.ContextKeyUserGroup, "default")
	common.SetContextKey(ctx, constant.ContextKeyTokenModelLimitEnabled, true)
	common.SetContextKey(ctx, constant.ContextKeyTokenModelLimit, map[string]bool{
		"zz-token-tiered-visible-model":      true,
		"zz-token-tiered-empty-expr-model":   true,
		"zz-token-tiered-missing-expr-model": true,
		"zz-token-unpriced-model":            true,
	})

	ListModels(ctx, constant.ChannelTypeOpenAI)

	ids := decodeListModelsResponse(t, recorder)
	require.Contains(t, ids, "zz-token-tiered-visible-model")
	require.NotContains(t, ids, "zz-token-tiered-empty-expr-model")
	require.NotContains(t, ids, "zz-token-tiered-missing-expr-model")
	require.NotContains(t, ids, "zz-token-unpriced-model")
}

func TestListModelsTokenLimitUsesResolvedCustomAutoGroups(t *testing.T) {
	withSelfUseModeEnabled(t)
	originalMax := setting.GetMaxTokenAutoGroups()
	originalUsableGroups := setting.UserUsableGroups2JSONString()
	originalRatios := ratio_setting.GroupRatio2JSONString()
	require.NoError(t, setting.UpdateMaxTokenAutoGroups("5"))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP"}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":1}`))
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateMaxTokenAutoGroups(fmt.Sprintf("%d", originalMax)))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalRatios))
	})

	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "vip", Model: "zz-vip-allowed", ChannelId: 1, Enabled: true},
		{Group: "vip", Model: "zz-vip-denied", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-default-outside-snapshot", ChannelId: 1, Enabled: true},
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	common.SetContextKey(ctx, constant.ContextKeyUserGroup, "default")
	common.SetContextKey(ctx, constant.ContextKeyTokenGroup, "auto")
	common.SetContextKey(ctx, constant.ContextKeyTokenAutoGroups, []string{"vip"})
	common.SetContextKey(ctx, constant.ContextKeyTokenModelLimitEnabled, true)
	common.SetContextKey(ctx, constant.ContextKeyTokenModelLimit, map[string]bool{
		"zz-vip-allowed":              true,
		"zz-default-outside-snapshot": true,
		"zz-not-enabled":              true,
	})

	ListModels(ctx, constant.ChannelTypeOpenAI)
	ids := decodeListModelsResponse(t, recorder)
	require.Equal(t, map[string]struct{}{"zz-vip-allowed": {}}, ids)

	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default"}`))
	emptyRecorder := httptest.NewRecorder()
	emptyCtx, _ := gin.CreateTestContext(emptyRecorder)
	emptyCtx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	common.SetContextKey(emptyCtx, constant.ContextKeyUserGroup, "default")
	common.SetContextKey(emptyCtx, constant.ContextKeyTokenGroup, "auto")
	common.SetContextKey(emptyCtx, constant.ContextKeyTokenAutoGroups, []string{"vip"})
	common.SetContextKey(emptyCtx, constant.ContextKeyTokenModelLimitEnabled, true)
	common.SetContextKey(emptyCtx, constant.ContextKeyTokenModelLimit, map[string]bool{"zz-vip-allowed": true})

	require.NotPanics(t, func() {
		ListModels(emptyCtx, constant.ChannelTypeAnthropic)
	})
	var anthropicResponse struct {
		Data    []dto.AnthropicModel `json:"data"`
		FirstID string               `json:"first_id"`
		LastID  string               `json:"last_id"`
	}
	require.NoError(t, common.Unmarshal(emptyRecorder.Body.Bytes(), &anthropicResponse))
	require.Empty(t, anthropicResponse.Data)
	require.Empty(t, anthropicResponse.FirstID)
	require.Empty(t, anthropicResponse.LastID)
}

func TestGroupTypesOptionIsValidatedAndPublishedToBothGroupViews(t *testing.T) {
	originalGroups := setting.UserUsableGroups2JSONString()
	originalRatios := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalGroups))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalRatios))
	})

	for _, dialect := range []struct{ kind, env string }{{"sqlite", ""}, {"mysql", "TEST_MYSQL_DSN"}, {"postgres", "TEST_POSTGRES_DSN"}} {
		t.Run(dialect.kind, func(t *testing.T) {
			if dialect.env != "" && os.Getenv(dialect.env) == "" {
				t.Skip("set " + dialect.env + " to run this database")
			}
			db := modelManagementDB(t, dialect.kind, os.Getenv(dialect.env))
			require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Standard","vip":"Priority"}`))
			require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":1}`))

			value := `[{"name":"Priority","icon":"star","color":"#059669","groups":["vip","private-group"]},{"name":"Restricted","icon":"shield","color":"#2563eb","groups":["admin-only"]},{"name":"Provider","icon":"custom","custom_icon":"DeepSeek.Color","color":"#059669","groups":["default"]}]`
			require.NoError(t, model.UpdateOption("GroupTypes", value))
			for _, invalid := range []string{
				`null`,
				`{}`,
				`[null]`,
				`[{"name":"","icon":"star","color":"#059669","groups":[]}]`,
				`[{"name":"A","icon":"invalid","color":"#059669","groups":[]}]`,
				`[{"name":"A","icon":"custom","color":"#059669","groups":[]}]`,
				`[{"name":"A","icon":"custom","custom_icon":"","color":"#059669","groups":[]}]`,
				`[{"name":"A","icon":"custom","custom_icon":" ","color":"#059669","groups":[]}]`,
				`[{"name":"A","icon":"custom","custom_icon":"https://example.com/icon.svg","color":"#059669","groups":[]}]`,
				`[{"name":"A","icon":"custom","custom_icon":"OpenAI.size={999}","color":"#059669","groups":[]}]`,
				`[{"name":"A","icon":"custom","custom_icon":"OpenAI.constructor","color":"#059669","groups":[]}]`,
				`[{"name":"A","icon":"custom","custom_icon":"<svg>","color":"#059669","groups":[]}]`,
				`[{"name":"A","icon":"custom","custom_icon":"` + strings.Repeat("A", 65) + `","color":"#059669","groups":[]}]`,
				`[{"name":"A","icon":"star","color":"red","groups":[]}]`,
				`[{"name":"A","icon":"star","color":"#059669","groups":null}]`,
				`[{"name":"A","icon":"star","color":"#gggggg","groups":[]}]`,
				`[{"name":"A","icon":"star","color":"#059669","groups":[" "]}]`,
				`[{"name":"A","icon":"star","color":"#059669","groups":["vip","vip"]}]`,
				`[{"name":"A","icon":"star","color":"#059669","groups":["vip"]},{"name":"B","icon":"star","color":"#059669","groups":["vip"]}]`,
				`[{"name":"A","icon":"star","color":"#059669","groups":[]},{"name":" A ","icon":"star","color":"#059669","groups":[]}]`,
			} {
				require.Error(t, model.UpdateOption("GroupTypes", invalid))
			}
			var persisted model.Option
			require.NoError(t, db.Where(&model.Option{Key: "GroupTypes"}).First(&persisted).Error)
			assert.Equal(t, value, persisted.Value)

			for range 2 {
				model.InitOptionMap()
				for _, handler := range []func(*gin.Context){GetPricing, GetUserGroups} {
					recorder := httptest.NewRecorder()
					ctx, _ := gin.CreateTestContext(recorder)
					ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
					handler(ctx)
					require.Equal(t, http.StatusOK, recorder.Code)
					var result struct {
						GroupTypes []setting.GroupType `json:"group_types"`
					}
					require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &result))
					assert.Equal(t, []setting.GroupType{
						{Name: "Priority", Icon: "star", Color: "#059669", Groups: []string{"vip"}},
						{Name: "Restricted", Icon: "shield", Color: "#2563eb", Groups: []string{}},
						{Name: "Provider", Icon: "custom", CustomIcon: "DeepSeek.Color", Color: "#059669", Groups: []string{"default"}},
					}, result.GroupTypes)
					assert.NotContains(t, recorder.Body.String(), "private-group")
					assert.NotContains(t, recorder.Body.String(), "admin-only")
				}
			}
			require.NoError(t, model.UpdateOption("GroupTypes", "[]"))
			model.InitOptionMap()
			assert.Empty(t, setting.GetGroupTypes([]string{"vip"}))
		})
	}
}

func TestSetupLoginDoesNotTouchPasswordWhenPasswordFieldOmitted(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Log{}, &model.AuditLog{}, &model.UserSession{}, &model.TwoFA{}, &model.PasskeyCredential{}))

	hashedPassword, err := common.Password2Hash("CurrentPassword123")
	require.NoError(t, err)
	user := &model.User{
		Username: "twofa-user",
		Password: hashedPassword,
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}
	require.NoError(t, db.Create(user).Error)

	router := gin.New()
	router.GET("/", func(c *gin.Context) {
		setupLogin(&model.User{
			Id:          user.Id,
			AuthVersion: user.AuthVersion,
			Username:    user.Username,
			Role:        user.Role,
			Status:      user.Status,
			Group:       user.Group,
		}, c)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	var stored model.User
	require.NoError(t, db.First(&stored, user.Id).Error)
	assert.Equal(t, hashedPassword, stored.Password)
}
