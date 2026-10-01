package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/degradation"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDegradationManualTestTargetsAndTaskStatus(t *testing.T) {
	db := setupManageUserTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.Ability{}, &model.SystemTask{}, &model.SystemTaskLock{}))
	model.InitOptionMap()
	cfg := degradation.DefaultConfig()
	cfg.Groups = []degradation.GroupConfig{{
		Group:  "default",
		Models: []degradation.ModelConfig{{Model: "example-model"}},
	}}
	raw, err := common.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, model.UpdateOption(degradation.OptionKey, string(raw)))
	require.NoError(t, db.Create(&model.Ability{Group: "default", Model: "example-model", ChannelId: 1, Enabled: true}).Error)

	request := func(body string) *httptest.ResponseRecorder {
		t.Helper()
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/degradation/test", strings.NewReader(body))
		StartDegradationTest(c)
		return recorder
	}

	invalid := request(`{"group":"default","model":"not-monitored"}`)
	assert.Contains(t, invalid.Body.String(), `"success":false`)
	var count int64
	require.NoError(t, db.Model(&model.SystemTask{}).Count(&count).Error)
	assert.Zero(t, count)

	started := request(`{"group":" default ","model":" example-model "}`)
	assert.Contains(t, started.Body.String(), `"success":true`)
	task, err := model.GetActiveSystemTask(model.SystemTaskTypeDegradationMonitor)
	require.NoError(t, err)
	require.NotNil(t, task)
	var payload degradationTestPayload
	require.NoError(t, task.DecodePayload(&payload))
	assert.Equal(t, degradationTestPayload{Group: "default", Model: "example-model"}, payload)

	busy := request(`{"group":"default","model":"example-model"}`)
	assert.Contains(t, busy.Body.String(), `"success":false`)
	require.NoError(t, db.Model(&model.SystemTask{}).Count(&count).Error)
	assert.EqualValues(t, 1, count)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = gin.Params{{Key: "task_id", Value: task.TaskID}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/degradation/test/"+task.TaskID, nil)
	GetDegradationTest(c)
	assert.Contains(t, recorder.Body.String(), `"status":"pending"`)
	assert.NotContains(t, recorder.Body.String(), `"payload"`)

	recorder = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(recorder)
	c.Params = gin.Params{{Key: "task_id", Value: "unknown"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/degradation/test/unknown", nil)
	GetDegradationTest(c)
	assert.Contains(t, recorder.Body.String(), `"success":false`)
}

func TestDegradationProbePreservesChannelMappingAndCompletionStatus(t *testing.T) {
	db := setupManageUserTestDB(t)
	previousRatios := ratio_setting.ModelRatio2JSONString()
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"gpt-4o-mini":1}`))
	t.Cleanup(func() { require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(previousRatios)) })
	user := model.User{Username: "degradation-probe", Group: "default", Status: common.UserStatusEnabled, Quota: 1_000_000}
	require.NoError(t, db.Create(&user).Error)
	service.InitHttpClient()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer channel-only-key", r.Header.Get("Authorization"))
		var request struct {
			Model     string                           `json:"model"`
			Stream    bool                             `json:"stream"`
			MaxTokens int                              `json:"max_tokens"`
			Messages  []struct{ Role, Content string } `json:"messages"`
		}
		if !assert.NoError(t, common.DecodeJson(r.Body, &request)) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		assert.Equal(t, "gpt-4o", request.Model)
		assert.False(t, request.Stream)
		assert.Equal(t, 8192, request.MaxTokens)
		if assert.Len(t, request.Messages, 1) {
			assert.Equal(t, "challenge from official CLI", request.Messages[0].Content)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"probe","object":"chat.completion","model":"gpt-4o","choices":[{"index":0,"message":{"role":"assistant","content":"1, 2, 3"},"finish_reason":"length"}],"usage":{"prompt_tokens":5,"completion_tokens":5,"total_tokens":10}}`))
	}))
	defer upstream.Close()
	mapping := `{"gpt-4o-mini":"gpt-4o"}`
	channel := &model.Channel{Id: 1, Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled,
		Key: "channel-only-key", BaseURL: &upstream.URL, ModelMapping: &mapping, Group: "default", Models: "gpt-4o-mini"}
	completion, err := probeDegradationChat(t.Context(), channel, user.Id, "default", "gpt-4o-mini", "challenge from official CLI")
	require.NoError(t, err)
	assert.Equal(t, degradation.Completion{Text: "1, 2, 3", FinishReason: "length"}, completion)
	var logs int64
	require.NoError(t, model.LOG_DB.Model(&model.Log{}).Count(&logs).Error)
	assert.Zero(t, logs, "degradation probes must continue to skip consume logs")
}
