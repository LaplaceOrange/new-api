package controller

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetChannelDefaultBaseURLsUsesBuiltInDefaults(t *testing.T) {
	originalBaseURLs := constant.ChannelBaseURLs
	constant.ChannelBaseURLs = append([]string(nil), originalBaseURLs...)
	constant.ChannelBaseURLs[constant.ChannelTypeDeepSeek] = "https://deepseek.server.example"
	t.Cleanup(func() {
		constant.ChannelBaseURLs = originalBaseURLs
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/channel/default_base_urls", nil)
	GetChannelDefaultBaseURLs(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool           `json:"success"`
		Data    map[int]string `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	assert.Equal(t, "https://deepseek.server.example", response.Data[constant.ChannelTypeDeepSeek])
	assert.Equal(t, "https://api.openai.com", response.Data[constant.ChannelTypeOpenAI])
	assert.NotContains(t, response.Data, constant.ChannelTypeAzure)
	assert.NotContains(t, response.Data, constant.ChannelTypeNewAPI)
	assert.NotContains(t, response.Data, constant.ChannelTypeTaskPlugin)
}

func TestValidateChannelProxy(t *testing.T) {
	tests := []struct {
		name    string
		proxy   string
		wantErr bool
	}{
		{name: "empty"},
		{name: "http", proxy: "http://proxy.example:8080"},
		{name: "https", proxy: "https://proxy.example:8443"},
		{name: "socks5", proxy: "socks5://proxy.example"},
		{name: "socks5h", proxy: "socks5h://proxy.example:1080/"},
		{name: "unsupported", proxy: "ftp://proxy.example", wantErr: true},
		{name: "path", proxy: "socks5://proxy.example:1080/path", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setting, err := common.Marshal(dto.ChannelSettings{Proxy: test.proxy})
			require.NoError(t, err)
			channel := &model.Channel{
				Type:    constant.ChannelTypeOpenAI,
				Setting: common.GetPointer(string(setting)),
			}

			err = validateChannel(channel, false)

			if test.wantErr {
				require.ErrorContains(t, err, "invalid channel proxy")
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCheckAndPersistChannelRateMultiplierUpdatesRemark(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	var requestedPath string
	var authorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		authorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"effective_rate_multiplier":1.25}`))
	}))
	t.Cleanup(server.Close)

	settings := dto.ChannelOtherSettings{
		UpstreamRateMultiplierCheckEnabled: true,
		UpstreamRateMultiplierCheckType:    dto.UpstreamRateMultiplierCheckTypeSub2API,
	}
	otherSettings, err := common.Marshal(settings)
	require.NoError(t, err)
	baseURL := server.URL
	remark := "0.8\nkeep this note"
	channel := &model.Channel{
		Type:          constant.ChannelTypeOpenAI,
		Name:          "rate multiplier test",
		Key:           "test-key\nsecond-key",
		BaseURL:       &baseURL,
		Remark:        &remark,
		OtherSettings: string(otherSettings),
		Models:        "gpt-test",
		Group:         "default",
	}
	require.NoError(t, db.Create(channel).Error)

	checkAndPersistChannelRateMultiplier(context.Background(), channel)

	var persisted model.Channel
	require.NoError(t, db.First(&persisted, channel.Id).Error)
	require.NotNil(t, persisted.Remark)
	assert.Equal(t, "/v1/sub2api/billing", requestedPath)
	assert.Equal(t, "Bearer test-key", authorization)
	assert.Equal(t, "1.25\nkeep this note", *persisted.Remark)
}

func newPriceMonitorTestChannel(t *testing.T, baseURL string, limit *float64) *model.Channel {
	t.Helper()
	channel := &model.Channel{
		Type: constant.ChannelTypeOpenAI, Name: "price monitor", Key: "test-key",
		BaseURL: &baseURL, Models: "gpt-4o-mini,secondary", Group: "default",
		Status: common.ChannelStatusEnabled, AutoBan: common.GetPointer(0),
		Remark: common.GetPointer("old\noperator note"),
	}
	channel.SetOtherSettings(dto.ChannelOtherSettings{
		UpstreamRateMultiplierCheckEnabled: true,
		UpstreamRateMultiplierCheckType:    dto.UpstreamRateMultiplierCheckTypeSub2API,
		UpstreamRateMultiplierLimit:        limit,
	})
	require.NoError(t, channel.Insert())
	return channel
}

func TestChannelPriceMonitorDatabaseMatrix(t *testing.T) {
	for _, dialect := range []struct{ kind, env string }{
		{"sqlite", ""}, {"mysql", "TEST_MYSQL_DSN"}, {"postgres", "TEST_POSTGRES_DSN"},
	} {
		t.Run(dialect.kind, func(t *testing.T) {
			if dialect.env != "" && os.Getenv(dialect.env) == "" {
				t.Skip("set " + dialect.env + " to run this database")
			}
			db := modelManagementDB(t, dialect.kind, os.Getenv(dialect.env))
			require.NoError(t, db.AutoMigrate(&model.ChannelContribution{}, &model.ChannelContributionRevision{}, &model.ChannelContributionModelHealth{}))
			for _, memoryCache := range []bool{false, true} {
				t.Run(fmt.Sprintf("memory_cache_%t", memoryCache), func(t *testing.T) {
					common.MemoryCacheEnabled = memoryCache
					model.InitChannelCache()
					rateMultiplier := 1.25
					billingFailed := false
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						if billingFailed {
							w.WriteHeader(http.StatusBadGateway)
							return
						}
						_, _ = fmt.Fprintf(w, `{"effective_rate_multiplier":%g}`, rateMultiplier)
					}))
					t.Cleanup(server.Close)
					channel := newPriceMonitorTestChannel(t, server.URL, common.GetPointer(1.0))
					channel.ChannelInfo.IsMultiKey = true
					channel.ChannelInfo.MultiKeyStatusList = map[int]int{1: common.ChannelStatusManuallyDisabled}
					channel.ChannelInfo.DisabledModels = map[string]bool{"secondary": true}
					channel.Key = "test-key\nsecond-key"
					updated, err := model.UpdateChannelAtomically(channel.Id, func(current *model.Channel) error {
						current.Key = channel.Key
						current.ChannelInfo = channel.ChannelInfo
						return nil
					})
					require.NoError(t, err)
					model.CacheUpdateChannel(updated)
					channel = updated

					result := checkAndPersistChannelRateMultiplier(context.Background(), channel)
					require.NotNil(t, result)
					assert.True(t, result.Disabled)
					assert.True(t, result.Exceeded)
					var stored model.Channel
					require.NoError(t, db.First(&stored, channel.Id).Error)
					assert.Equal(t, common.ChannelStatusAutoDisabled, stored.Status)
					assert.True(t, stored.IsPriceMonitorDisabled())
					assert.Equal(t, "1.25\noperator note", *stored.Remark)
					assert.Equal(t, channel.ChannelInfo.MultiKeyStatusList, stored.ChannelInfo.MultiKeyStatusList)
					assert.False(t, model.IsChannelEnabledForGroupModel("default", "gpt-4o-mini", channel.Id))
					cached, err := model.CacheGetChannel(channel.Id)
					require.NoError(t, err)
					assert.Equal(t, stored.Status, cached.Status)

					repeated := checkAndPersistChannelRateMultiplier(context.Background(), &stored)
					assert.False(t, repeated.Disabled)
					assert.Equal(t, stored.GetOtherInfo()["status_time"], repeated.source.GetOtherInfo()["status_time"])
					changed, err := model.EnableChannelAfterHealthCheck(channel.Id, "test-key", channel, result.RateMultiplier)
					require.NoError(t, err)
					assert.False(t, changed)

					billingFailed = true
					failed := checkAndPersistChannelRateMultiplier(context.Background(), &stored)
					assert.False(t, failed.Checked)
					recoverChannelPriceMonitor(failed)
					require.NoError(t, db.First(&stored, channel.Id).Error)
					assert.Equal(t, common.ChannelStatusAutoDisabled, stored.Status)
					assert.Equal(t, "1.25\noperator note", *stored.Remark)

					billingFailed = false
					rateMultiplier = 1
					safe := checkAndPersistChannelRateMultiplier(context.Background(), &stored)
					assert.False(t, safe.Exceeded)
					require.NoError(t, db.First(&stored, channel.Id).Error)
					assert.Equal(t, common.ChannelStatusAutoDisabled, stored.Status, "price drop alone does not establish health")
					recoverChannelPriceMonitor(safe)
					assert.True(t, safe.Enabled)
					require.NoError(t, db.First(&stored, channel.Id).Error)
					assert.Equal(t, common.ChannelStatusEnabled, stored.Status)
					assert.False(t, stored.IsPriceMonitorDisabled())
					assert.Equal(t, channel.ChannelInfo.MultiKeyStatusList, stored.ChannelInfo.MultiKeyStatusList)
					assert.True(t, model.IsChannelEnabledForGroupModel("default", "gpt-4o-mini", channel.Id))
					assert.False(t, model.IsChannelEnabledForGroupModel("default", "secondary", channel.Id))
					cached, err = model.CacheGetChannel(channel.Id)
					require.NoError(t, err)
					assert.Equal(t, stored.Status, cached.Status)

					rateMultiplier = 2
					assert.True(t, checkAndPersistChannelRateMultiplier(context.Background(), &stored).Disabled)
					changed, err = model.UpdateChannelStatusWithError(channel.Id, "", common.ChannelStatusEnabled, "manual operation")
					require.NoError(t, err)
					require.True(t, changed)
					assert.True(t, checkAndPersistChannelRateMultiplier(context.Background(), &stored).Disabled)
					_, err = model.UpdateChannelStatusWithError(channel.Id, "", common.ChannelStatusManuallyDisabled, "manual operation")
					require.NoError(t, err)
					rateMultiplier = 0.5
					manual := checkAndPersistChannelRateMultiplier(context.Background(), &stored)
					recoverChannelPriceMonitor(manual)
					assert.False(t, manual.Enabled)
					require.NoError(t, db.First(&stored, channel.Id).Error)
					assert.Equal(t, common.ChannelStatusManuallyDisabled, stored.Status)
				})
			}
		})
	}
}

func TestChannelPriceMonitorPreservesOtherDisableReasonsAndLatestConfiguration(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	channel := newPriceMonitorTestChannel(t, "https://billing.example", common.GetPointer(1.0))
	changed, err := model.UpdateChannelStatusWithError(channel.Id, "", common.ChannelStatusAutoDisabled, "upstream unhealthy")
	require.NoError(t, err)
	require.True(t, changed)
	updated, changed, err := model.UpdateChannelRateMultiplier(channel, 2, false)
	require.NoError(t, err)
	assert.False(t, changed)
	assert.False(t, updated.IsPriceMonitorDisabled())
	assert.Equal(t, "upstream unhealthy", updated.GetOtherInfo()["status_reason"])
	updated, _, err = model.UpdateChannelRateMultiplier(channel, 0.5, false)
	require.NoError(t, err)
	_, changed, err = model.UpdateChannelRateMultiplier(updated, 0.5, true)
	require.NoError(t, err)
	assert.False(t, changed)

	_, err = model.UpdateChannelStatusWithError(channel.Id, "", common.ChannelStatusEnabled, "manual operation")
	require.NoError(t, err)
	_, err = model.UpdateChannelAtomically(channel.Id, func(current *model.Channel) error {
		settings := current.GetOtherSettings()
		settings.UpstreamRateMultiplierLimit = common.GetPointer(3.0)
		current.SetOtherSettings(settings)
		return nil
	})
	require.NoError(t, err)
	updated, changed, err = model.UpdateChannelRateMultiplier(channel, 2, false)
	require.NoError(t, err)
	assert.False(t, changed, "use the latest limit instead of the test snapshot")
	assert.Equal(t, common.ChannelStatusEnabled, updated.Status)

	_, _, err = model.UpdateChannelRateMultiplier(channel, 4, false)
	require.NoError(t, err)
	updated, _, err = model.UpdateChannelRateMultiplier(channel, 2, false)
	require.NoError(t, err)
	_, _, err = model.UpdateChannelRateMultiplier(channel, 4, false)
	require.NoError(t, err)
	_, _, err = model.UpdateChannelRateMultiplier(updated, 2, true)
	require.Error(t, err, "an older healthy test cannot overwrite a newer price observation")

	_, err = model.UpdateChannelAtomically(channel.Id, func(current *model.Channel) error {
		current.BaseURL = common.GetPointer("https://replacement.example")
		return nil
	})
	require.NoError(t, err)
	_, _, err = model.UpdateChannelRateMultiplier(channel, 0.5, false)
	require.Error(t, err)
	var stored model.Channel
	require.NoError(t, db.First(&stored, channel.Id).Error)
	assert.Equal(t, "4\noperator note", *stored.Remark)
	assert.Equal(t, common.ChannelStatusAutoDisabled, stored.Status)
}

func TestChannelPriceMonitorWithoutLimitOnlyUpdatesRemark(t *testing.T) {
	setupModelListControllerTestDB(t)
	channel := newPriceMonitorTestChannel(t, "https://billing.example", nil)
	channel.Remark = nil
	_, err := model.UpdateChannelAtomically(channel.Id, func(current *model.Channel) error {
		current.Remark = nil
		return nil
	})
	require.NoError(t, err)
	updated, changed, err := model.UpdateChannelRateMultiplier(channel, 10, false)
	require.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, "10", *updated.Remark)
	assert.Equal(t, common.ChannelStatusEnabled, updated.Status)
}

func TestChannelPriceMonitorRecoveryRequiresActiveLimitAndUsableKeys(t *testing.T) {
	setupModelListControllerTestDB(t)
	for _, scenario := range []string{"check disabled", "limit removed", "no usable key"} {
		t.Run(scenario, func(t *testing.T) {
			channel := newPriceMonitorTestChannel(t, "https://billing.example", common.GetPointer(1.0))
			_, _, err := model.UpdateChannelRateMultiplier(channel, 2, false)
			require.NoError(t, err)
			safe, _, err := model.UpdateChannelRateMultiplier(channel, 0.5, false)
			require.NoError(t, err)
			_, err = model.UpdateChannelAtomically(channel.Id, func(current *model.Channel) error {
				settings := current.GetOtherSettings()
				switch scenario {
				case "check disabled":
					settings.UpstreamRateMultiplierCheckEnabled = false
				case "limit removed":
					settings.UpstreamRateMultiplierLimit = nil
				case "no usable key":
					current.ChannelInfo.IsMultiKey = true
					current.ChannelInfo.MultiKeyStatusList = map[int]int{0: common.ChannelStatusManuallyDisabled}
				}
				current.SetOtherSettings(settings)
				return nil
			})
			require.NoError(t, err)
			_, changed, err := model.UpdateChannelRateMultiplier(safe, 0.5, true)
			if scenario == "check disabled" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.False(t, changed)
			stored, err := model.GetChannelById(channel.Id, true)
			require.NoError(t, err)
			assert.Equal(t, common.ChannelStatusAutoDisabled, stored.Status)
			changed, err = model.EnableChannelAfterHealthCheck(channel.Id, "", safe, nil)
			require.NoError(t, err)
			assert.False(t, changed)
		})
	}
}

func TestChannelPriceMonitorRejectsInvalidLimitsAndUpstreamValues(t *testing.T) {
	for _, limit := range []float64{0, -1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		settings := dto.ChannelOtherSettings{UpstreamRateMultiplierLimit: &limit}
		require.Error(t, settings.ValidateUpstreamRateMultiplier())
	}
	channel := &model.Channel{Type: constant.ChannelTypeOpenAI}
	channel.OtherSettings = `{"upstream_rate_multiplier_limit":0}`
	require.ErrorContains(t, validateChannel(channel, false), "finite positive number")
	for _, payload := range []string{
		`{}`, `{"effective_rate_multiplier":0}`, `{"effective_rate_multiplier":-1}`,
		`{"effective_rate_multiplier":true}`, `{"effective_rate_multiplier":null}`,
		`{"effective_rate_multiplier":"NaN"}`, `{"effective_rate_multiplier":"Inf"}`,
		`{"effective_rate_multiplier":[]}`, `{"effective_rate_multiplier":1`,
	} {
		t.Run(payload, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(payload))
			}))
			t.Cleanup(server.Close)
			channel := &model.Channel{BaseURL: common.GetPointer(server.URL)}
			_, err := fetchSub2APIRateMultiplier(context.Background(), channel, server.Client())
			require.Error(t, err)
		})
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"effective_rate_multiplier":"1.25"}`))
	}))
	t.Cleanup(server.Close)
	channel.BaseURL = common.GetPointer(server.URL)
	value, err := fetchSub2APIRateMultiplier(context.Background(), channel, server.Client())
	require.NoError(t, err)
	assert.Equal(t, 1.25, value)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = fetchSub2APIRateMultiplier(ctx, channel, server.Client())
	require.Error(t, err)
}

func TestChannelPriceMonitorManualAndAutomaticHealthTests(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Log{}))
	previousDisable, previousEnable := common.AutomaticDisableChannelEnabled, common.AutomaticEnableChannelEnabled
	previousRatios := ratio_setting.ModelRatio2JSONString()
	common.AutomaticDisableChannelEnabled, common.AutomaticEnableChannelEnabled = false, false
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"gpt-4o-mini":1}`))
	t.Cleanup(func() {
		common.AutomaticDisableChannelEnabled, common.AutomaticEnableChannelEnabled = previousDisable, previousEnable
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(previousRatios))
	})
	user := &model.User{Username: "price-test-root", Role: common.RoleRootUser, Status: common.UserStatusEnabled, Group: "default"}
	require.NoError(t, db.Create(user).Error)
	rateMultiplier := 2.0
	healthy := true
	billingHealthy := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/v1/sub2api/billing" {
			if !billingHealthy {
				w.WriteHeader(http.StatusBadGateway)
				return
			}
			_, _ = fmt.Fprintf(w, `{"effective_rate_multiplier":%g}`, rateMultiplier)
			return
		}
		if !healthy {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":{"message":"unhealthy","type":"upstream_error"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"health","object":"chat.completion","model":"gpt-4o-mini","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	t.Cleanup(server.Close)
	channel := newPriceMonitorTestChannel(t, server.URL, common.GetPointer(1.0))
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", user.Id)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(channel.Id)}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/channel/test/"+strconv.Itoa(channel.Id), nil)
	TestChannel(ctx)
	var response struct {
		Success      bool                      `json:"success"`
		PriceMonitor channelPriceMonitorResult `json:"price_monitor"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, recorder.Body.String())
	assert.True(t, response.PriceMonitor.Disabled)
	assert.True(t, response.PriceMonitor.Exceeded)
	require.NoError(t, db.First(channel, channel.Id).Error)
	assert.Equal(t, common.ChannelStatusAutoDisabled, channel.Status)

	rateMultiplier = 0.5
	billingHealthy = false
	common.AutomaticEnableChannelEnabled = true
	summary := testChannelForHealthCheck(context.Background(), channel, user.Id, true, 10000000)
	assert.Equal(t, 1, summary.Succeeded)
	assert.Zero(t, summary.Enabled, "successful connectivity cannot bypass a failed price check")
	require.NoError(t, db.First(channel, channel.Id).Error)
	assert.Equal(t, common.ChannelStatusAutoDisabled, channel.Status)
	assert.Equal(t, "2\noperator note", *channel.Remark)

	billingHealthy = true
	common.AutomaticEnableChannelEnabled = false
	healthy = false
	summary = testChannelForHealthCheck(context.Background(), channel, user.Id, true, 10000000)
	assert.Equal(t, 1, summary.Failed)
	assert.Zero(t, summary.Enabled)
	require.NoError(t, db.First(channel, channel.Id).Error)
	assert.Equal(t, common.ChannelStatusAutoDisabled, channel.Status)

	healthy = true
	common.AutomaticDisableChannelEnabled = true
	summary = testChannelForHealthCheck(context.Background(), channel, user.Id, true, -1)
	assert.Equal(t, 1, summary.Failed)
	assert.Zero(t, summary.Enabled, "price recovery must wait for the response-time health check too")
	require.NoError(t, db.First(channel, channel.Id).Error)
	assert.Equal(t, common.ChannelStatusAutoDisabled, channel.Status)

	common.AutomaticDisableChannelEnabled = false
	summary = testChannelForHealthCheck(context.Background(), channel, user.Id, true, 10000000)
	assert.Equal(t, 1, summary.Succeeded)
	assert.Equal(t, 1, summary.Enabled)
	require.NoError(t, db.First(channel, channel.Id).Error)
	assert.Equal(t, common.ChannelStatusEnabled, channel.Status)

	rateMultiplier = 2
	summary = testChannelForHealthCheck(context.Background(), channel, user.Id, false, 10000000)
	assert.Equal(t, 1, summary.Disabled, "price protection also applies in passive recovery mode")
	require.NoError(t, db.First(channel, channel.Id).Error)
	assert.Equal(t, common.ChannelStatusAutoDisabled, channel.Status)
}

func TestSelectChannelsForAutomaticTestIncludesPriceProtectionWithoutAutoBan(t *testing.T) {
	setupModelListControllerTestDB(t)
	channel := &model.Channel{Id: 1, Status: common.ChannelStatusEnabled, AutoBan: common.GetPointer(0)}
	channel.SetOtherSettings(dto.ChannelOtherSettings{
		UpstreamRateMultiplierCheckEnabled: true,
		UpstreamRateMultiplierCheckType:    dto.UpstreamRateMultiplierCheckTypeSub2API,
		UpstreamRateMultiplierLimit:        common.GetPointer(1.0),
	})
	for _, mode := range []string{operation_setting.ChannelTestModeAutoBanOnly, operation_setting.ChannelTestModePassiveRecovery} {
		assert.Equal(t, []*model.Channel{channel}, selectChannelsForAutomaticTest([]*model.Channel{channel}, mode))
	}
	channel.Status = common.ChannelStatusManuallyDisabled
	assert.Empty(t, selectChannelsForAutomaticTest([]*model.Channel{channel}, operation_setting.ChannelTestModeScheduledAll))
}

func TestChannelTestResponseRecorderLimitsStringWrites(t *testing.T) {
	recorder := newChannelTestResponseRecorder(4)
	written, err := recorder.WriteString("streamed")

	require.NoError(t, err)
	assert.Equal(t, len("streamed"), written)
	assert.Equal(t, "stre", recorder.Body.String())
	assert.True(t, recorder.exceeded)
}

func TestFetchModelsResponseBodyRejectsOversizedPayload(t *testing.T) {
	payload := bytes.Repeat([]byte("x"), int(service.StrictSSRFProtectedResponseBodyLimitBytes)+1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(payload)
	}))
	t.Cleanup(server.Close)

	body, err := getFetchModelsResponseBody(http.MethodGet, server.URL, nil, nil, fetchChannelModelsOptions{
		HTTPClient: server.Client(),
	})

	require.ErrorContains(t, err, "model list response exceeds")
	assert.Nil(t, body)
}

func TestValidateChannelRequiresNewAPIBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL *string
		wantErr bool
	}{
		{name: "missing", wantErr: true},
		{name: "blank", baseURL: common.GetPointer("  "), wantErr: true},
		{name: "configured", baseURL: common.GetPointer("https://new-api.example")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			channel := &model.Channel{
				Type:    constant.ChannelTypeNewAPI,
				BaseURL: test.baseURL,
			}

			err := validateChannel(channel, false)

			if test.wantErr {
				require.ErrorContains(t, err, "New API channel base URL cannot be empty")
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestNewAPIChannelRegistration(t *testing.T) {
	apiType, ok := common.ChannelType2APIType(constant.ChannelTypeNewAPI)

	require.True(t, ok)
	assert.Equal(t, constant.APITypeNewAPI, apiType)
	assert.Equal(t, "New API", constant.GetChannelTypeName(constant.ChannelTypeNewAPI))
	require.Greater(t, len(constant.ChannelBaseURLs), constant.ChannelTypeNewAPI)
	assert.Empty(t, constant.ChannelBaseURLs[constant.ChannelTypeNewAPI])
}

func TestResponsesCompactChannelSupport(t *testing.T) {
	tests := []struct {
		name        string
		channelType int
		apiType     int
		want        bool
	}{
		{name: "OpenAI", channelType: constant.ChannelTypeOpenAI, apiType: constant.APITypeOpenAI, want: true},
		{name: "Azure", channelType: constant.ChannelTypeAzure, apiType: constant.APITypeOpenAI, want: true},
		{name: "Codex", channelType: constant.ChannelTypeCodex, apiType: constant.APITypeCodex, want: true},
		{name: "Advanced Custom", channelType: constant.ChannelTypeAdvancedCustom, apiType: constant.APITypeAdvancedCustom, want: true},
		{name: "Sub2API", channelType: constant.ChannelTypeSub2API, apiType: constant.APITypeSub2API, want: true},
		{name: "New API", channelType: constant.ChannelTypeNewAPI, apiType: constant.APITypeNewAPI, want: true},
		{name: "Anthropic", channelType: constant.ChannelTypeAnthropic, apiType: constant.APITypeAnthropic, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, common.SupportsResponsesCompact(test.channelType, test.apiType))
		})
	}
}

func TestMultiprotocolGatewayEndpointTypes(t *testing.T) {
	want := []constant.EndpointType{
		constant.EndpointTypeOpenAI,
		constant.EndpointTypeOpenAIResponse,
		constant.EndpointTypeOpenAIResponseCompact,
		constant.EndpointTypeAnthropic,
		constant.EndpointTypeGemini,
		constant.EndpointTypeOpenAIAlphaSearch,
	}

	assert.Equal(t, want, common.GetEndpointTypesByChannelType(constant.ChannelTypeNewAPI, "gpt-5"))
	assert.Equal(t, want, common.GetEndpointTypesByChannelType(constant.ChannelTypeSub2API, "gpt-5"))
}

func TestCopyChannelRejectsInvalidLegacyProxySettings(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	settingBytes, err := common.Marshal(dto.ChannelSettings{
		Proxy: "socks5://proxy.example/legacy-path",
	})
	require.NoError(t, err)
	setting := string(settingBytes)
	origin := &model.Channel{
		Type:    constant.ChannelTypeOpenAI,
		Name:    "legacy proxy channel",
		Key:     "test-key",
		Models:  "gpt-test",
		Group:   "default",
		Setting: &setting,
	}
	require.NoError(t, db.Create(origin).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", origin.Id)}}
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/channel/copy", nil)

	CopyChannel(ctx)

	assert.Contains(t, recorder.Body.String(), "invalid channel settings")
	var channelCount int64
	require.NoError(t, db.Model(&model.Channel{}).Count(&channelCount).Error)
	assert.Equal(t, int64(1), channelCount)
}

func TestDeleteChannelResetsProxyCacheWhenPreReadFails(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Log{}, &model.AuditLog{}))
	service.ResetProxyClientCache()
	t.Cleanup(service.ResetProxyClientCache)

	proxyURL := "http://proxy.example:8080"
	beforeDelete, err := service.GetHttpClientWithProxy(proxyURL)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: "999999"}}
	ctx.Request = httptest.NewRequest(http.MethodDelete, "/api/channel/999999", nil)

	DeleteChannel(ctx)

	assert.Contains(t, recorder.Body.String(), `"success":true`)
	afterDelete, err := service.GetHttpClientWithProxy(proxyURL)
	require.NoError(t, err)
	assert.NotSame(t, beforeDelete, afterDelete)
}

func TestDeleteChannelBatchReportsAndAuditsActualDeletedCount(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Log{}, &model.AuditLog{}))
	channel := &model.Channel{Name: "existing", Key: "test-key"}
	require.NoError(t, db.Create(channel).Error)

	requestBody, err := common.Marshal(ChannelBatch{Ids: []int{channel.Id, 999999}})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodDelete, "/api/channel/batch", bytes.NewReader(requestBody))
	ctx.Request.Header.Set("Content-Type", "application/json")

	DeleteChannelBatch(ctx)

	var response struct {
		Success bool  `json:"success"`
		Data    int64 `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, int64(1), response.Data)

	var auditLog model.AuditLog
	require.NoError(t, db.Order("id desc").First(&auditLog).Error)
	var auditData struct {
		Operation struct {
			Params map[string]any `json:"params"`
		} `json:"op"`
	}
	encodedAudit, err := common.Marshal(auditLog.Other)
	require.NoError(t, err)
	require.NoError(t, common.Unmarshal(encodedAudit, &auditData))
	assert.Equal(t, float64(1), auditData.Operation.Params["count"])
}

func TestSettleTestQuotaUsesTieredBilling(t *testing.T) {
	info := &relaycommon.RelayInfo{
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:   "tiered_expr",
			ExprString:    `param("stream") == true ? tier("stream", p * 3) : tier("base", p * 2)`,
			ExprHash:      billingexpr.ExprHashString(`param("stream") == true ? tier("stream", p * 3) : tier("base", p * 2)`),
			GroupRatio:    1,
			EstimatedTier: "stream",
			QuotaPerUnit:  common.QuotaPerUnit,
			ExprVersion:   1,
		},
		BillingRequestInput: &billingexpr.RequestInput{
			Body: []byte(`{"stream":true}`),
		},
	}

	quota, result := settleTestQuota(info, types.PriceData{
		ModelRatio:      1,
		CompletionRatio: 2,
	}, &dto.Usage{
		PromptTokens: 1000,
	})

	require.Equal(t, 1500, quota)
	require.NotNil(t, result)
	require.Equal(t, "stream", result.MatchedTier)
}

func TestBuildTestLogOtherInjectsTieredInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	info := &relaycommon.RelayInfo{
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode: "tiered_expr",
			ExprString:  `tier("base", p * 2)`,
		},
		ChannelMeta: &relaycommon.ChannelMeta{},
	}
	priceData := types.PriceData{
		GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
	}
	usage := &dto.Usage{
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 12,
		},
	}

	requestRules := []billingexpr.RequestRuleTrace{{
		Cond:       `param("service_tier") == "fast"`,
		Multiplier: 2,
		Matched:    true,
	}}
	other := buildTestLogOther(ctx, info, priceData, usage, &billingexpr.TieredResult{
		MatchedTier:  "base",
		RequestRules: requestRules,
	})

	fields := other.Snapshot()
	require.Equal(t, "tiered_expr", fields["billing_mode"])
	require.Equal(t, "base", fields["matched_tier"])
	require.Equal(t, requestRules, fields["request_rules"])
	require.NotEmpty(t, fields["expr_b64"])
}

func TestResolveChannelTestUserIDUsesRequestUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("id", 2)

	userID, err := resolveChannelTestUserID(ctx)

	require.NoError(t, err)
	require.Equal(t, 2, userID)
}

func TestSelectChannelsForAutomaticTestPassiveRecoveryOnlyUsesAutoDisabled(t *testing.T) {
	channels := []*model.Channel{
		{Id: 1, Status: common.ChannelStatusEnabled},
		{Id: 2, Status: common.ChannelStatusAutoDisabled},
		{Id: 3, Status: common.ChannelStatusManuallyDisabled},
	}

	selected := selectChannelsForAutomaticTest(channels, operation_setting.ChannelTestModePassiveRecovery)

	require.Len(t, selected, 1)
	require.Equal(t, 2, selected[0].Id)
}

func TestSelectChannelsForAutomaticTestScheduledSkipsManualDisabled(t *testing.T) {
	channels := []*model.Channel{
		{Id: 1, Status: common.ChannelStatusEnabled},
		{Id: 2, Status: common.ChannelStatusAutoDisabled},
		{Id: 3, Status: common.ChannelStatusManuallyDisabled},
	}

	selected := selectChannelsForAutomaticTest(channels, operation_setting.ChannelTestModeScheduledAll)

	require.Len(t, selected, 2)
	require.Equal(t, 1, selected[0].Id)
	require.Equal(t, 2, selected[1].Id)
}

func TestSelectChannelsForAutomaticTestAutoBanOnlyUsesEligibleChannels(t *testing.T) {
	autoBanEnabled := 1
	autoBanDisabled := 0
	channels := []*model.Channel{
		{Id: 1, Status: common.ChannelStatusEnabled, AutoBan: &autoBanEnabled},
		{Id: 2, Status: common.ChannelStatusEnabled, AutoBan: &autoBanDisabled},
		{Id: 3, Status: common.ChannelStatusAutoDisabled, AutoBan: &autoBanEnabled},
		{Id: 4, Status: common.ChannelStatusManuallyDisabled, AutoBan: &autoBanEnabled},
		{Id: 5, Status: common.ChannelStatusEnabled},
	}

	selected := selectChannelsForAutomaticTest(channels, operation_setting.ChannelTestModeAutoBanOnly)

	require.Len(t, selected, 2)
	require.Equal(t, 1, selected[0].Id)
	require.Equal(t, 3, selected[1].Id)
}

func TestRunChannelTestWorkersHonorsConfiguredConcurrency(t *testing.T) {
	originalInterval := common.RequestInterval
	common.RequestInterval = 0
	t.Cleanup(func() { common.RequestInterval = originalInterval })

	channels := []*model.Channel{
		{Id: 1, Status: common.ChannelStatusEnabled},
		{Id: 2, Status: common.ChannelStatusEnabled},
		{Id: 3, Status: common.ChannelStatusEnabled},
		{Id: 4, Status: common.ChannelStatusEnabled},
	}
	started := make(chan struct{}, len(channels))
	release := make(chan struct{})
	var active atomic.Int32
	var maxActive atomic.Int32
	progress := make([]int, 0, len(channels)+1)
	summaryResult := make(chan channelTestSummary, 1)

	go func() {
		summaryResult <- runChannelTestWorkers(
			context.Background(),
			channels,
			2,
			func(_ context.Context, _ *model.Channel) channelTestSummary {
				current := active.Add(1)
				defer active.Add(-1)
				for {
					observed := maxActive.Load()
					if current <= observed || maxActive.CompareAndSwap(observed, current) {
						break
					}
				}
				started <- struct{}{}
				<-release
				return channelTestSummary{Tested: 1, Succeeded: 1}
			},
			func(processed, _ int) {
				progress = append(progress, processed)
			},
		)
	}()

	<-started
	<-started
	select {
	case <-started:
		t.Fatal("started more channel tests than the configured concurrency")
	default:
	}
	close(release)

	summary := <-summaryResult

	assert.Equal(t, int32(2), maxActive.Load())
	assert.Equal(t, channelTestSummary{Tested: 4, Succeeded: 4}, summary)
	assert.Equal(t, []int{0, 1, 2, 3, 4}, progress)
}

func TestRunChannelTestWorkersStopsAfterCancellation(t *testing.T) {
	originalInterval := common.RequestInterval
	common.RequestInterval = 0
	t.Cleanup(func() { common.RequestInterval = originalInterval })

	ctx, cancel := context.WithCancel(context.Background())
	channels := []*model.Channel{
		{Id: 1, Status: common.ChannelStatusEnabled},
		{Id: 2, Status: common.ChannelStatusEnabled},
		{Id: 3, Status: common.ChannelStatusEnabled},
		{Id: 4, Status: common.ChannelStatusEnabled},
	}
	started := make(chan struct{}, len(channels))
	progress := make([]int, 0, 1)
	summaryResult := make(chan channelTestSummary, 1)

	go func() {
		summaryResult <- runChannelTestWorkers(
			ctx,
			channels,
			2,
			func(ctx context.Context, _ *model.Channel) channelTestSummary {
				started <- struct{}{}
				<-ctx.Done()
				return channelTestSummary{Tested: 1, Succeeded: 1}
			},
			func(processed, _ int) {
				progress = append(progress, processed)
			},
		)
	}()

	<-started
	<-started
	cancel()

	summary := <-summaryResult

	select {
	case <-started:
		t.Fatal("started another channel test after cancellation")
	default:
	}
	assert.Equal(t, channelTestSummary{Tested: 2, Succeeded: 2}, summary)
	assert.Equal(t, []int{0}, progress)
}

func TestTestAllChannelsRejectsExistingActiveTask(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.SystemTask{}, &model.SystemTaskLock{}))

	existing, err := model.CreateSystemTask(model.SystemTaskTypeChannelTest, nil, nil)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/channel/test", nil)

	TestAllChannels(ctx)

	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Contains(t, recorder.Body.String(), existing.TaskID)
	require.Contains(t, recorder.Body.String(), "已有通道测试任务正在运行或等待中")
}
