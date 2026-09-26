package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/degradation"
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
