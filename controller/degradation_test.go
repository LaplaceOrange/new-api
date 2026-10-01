package controller

import (
	"fmt"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	task, err := model.GetActiveSystemTask(model.SystemTaskTypeDegradationCheck)
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

// Only dedicated test databases may be passed through TEST_*_DSN.
func TestDegradationTaskIsolationDatabases(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			db := openDegradationTestDatabase(t, dialect)
			require.NoError(t, db.AutoMigrate(&model.SystemTask{}, &model.SystemTaskLock{}))
			t.Cleanup(func() { require.NoError(t, db.Migrator().DropTable(&model.SystemTaskLock{}, &model.SystemTask{})) })
			first, created, err := enqueueDegradationCheck(degradationTestPayload{Group: "default", Model: "a"})
			require.NoError(t, err)
			require.True(t, created)
			second, created, err := enqueueDegradationCheck(degradationTestPayload{Group: "default", Model: "b"})
			require.NoError(t, err)
			require.True(t, created)
			duplicate, created, err := enqueueDegradationCheck(degradationTestPayload{Group: "default", Model: "a"})
			require.NoError(t, err)
			assert.False(t, created)
			assert.Equal(t, first.TaskID, duplicate.TaskID)
			pending, err := model.FindEarliestPendingSystemTasks([]string{model.SystemTaskTypeDegradationCheck})
			require.NoError(t, err)
			require.Len(t, pending, 2)
			for _, task := range []*model.SystemTask{first, second} {
				_, claimed, err := model.ClaimSystemTask(task.ID, task.Type, "runner", time.Now().Add(time.Minute).Unix())
				require.NoError(t, err)
				require.True(t, claimed)
			}
			require.NoError(t, model.FinishSystemTask(first.TaskID, "runner", model.SystemTaskStatusFailed, nil, "probe failed"))
			require.NoError(t, model.UpdateSystemTaskState(second.TaskID, "runner", map[string]int{"progress": 50}))
			require.NoError(t, model.RenewSystemTaskLock(second.TaskID, "runner", time.Now().Add(2*time.Minute).Unix()))
			require.NoError(t, model.FinishSystemTask(second.TaskID, "runner", model.SystemTaskStatusSucceeded, nil, ""))
			_, created, err = enqueueDegradationCheck(degradationTestPayload{Group: "default", Model: "a"})
			require.NoError(t, err)
			assert.True(t, created)
		})
	}
}

func TestDegradationSchedulerEnqueuesAllDueModels(t *testing.T) {
	db := setupManageUserTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.Ability{}, &model.SystemTask{}, &model.SystemTaskLock{}, &model.DegradationTarget{}, &model.DegradationEvent{}))
	model.InitOptionMap()
	cfg := degradation.DefaultConfig()
	cfg.Enabled = true
	cfg.Groups = []degradation.GroupConfig{{Group: "default", Models: []degradation.ModelConfig{{Model: "a"}, {Model: "b"}, {Model: "later"}}}}
	raw, err := common.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, model.UpdateOption(degradation.OptionKey, string(raw)))
	for _, name := range []string{"a", "b", "later"} {
		require.NoError(t, db.Create(&model.Ability{Group: "default", Model: name, ChannelId: 1, Enabled: true}).Error)
		next := time.Now().Add(-time.Minute).Unix()
		if name == "later" {
			next = time.Now().Add(time.Hour).Unix()
		}
		require.NoError(t, model.EnsureDegradationTarget("default", name, next))
	}
	require.NoError(t, scheduleDegradationChecks(t.Context()))
	require.NoError(t, scheduleDegradationChecks(t.Context()))
	tasks, err := model.FindPendingSystemTasks(model.SystemTaskTypeDegradationCheck, 10)
	require.NoError(t, err)
	require.Len(t, tasks, 2)
	var names []string
	for _, task := range tasks {
		var p degradationTestPayload
		require.NoError(t, task.DecodePayload(&p))
		assert.True(t, p.Scheduled)
		names = append(names, p.Model)
	}
	assert.ElementsMatch(t, []string{"a", "b"}, names)
}

func openDegradationTestDatabase(t *testing.T, dialect string) *gorm.DB {
	t.Helper()
	var driver gorm.Dialector
	switch dialect {
	case "sqlite":
		driver = sqlite.Open(filepath.Join(t.TempDir(), "tasks.db"))
	case "mysql":
		dsn := os.Getenv("TEST_MYSQL_DSN")
		if dsn == "" {
			t.Skip("TEST_MYSQL_DSN not set")
		}
		driver = mysql.Open(dsn)
	case "postgres":
		dsn := os.Getenv("TEST_POSTGRES_DSN")
		if dsn == "" {
			t.Skip("TEST_POSTGRES_DSN not set")
		}
		driver = postgres.Open(dsn)
	}
	db, err := gorm.Open(driver, &gorm.Config{})
	require.NoError(t, err)
	previous := model.DB
	model.DB = db
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { model.DB = previous; _ = sqlDB.Close() })

	var version string
	query := "SELECT VERSION()"
	if dialect == "sqlite" {
		query = "SELECT sqlite_version()"
	}
	require.NoError(t, db.Raw(query).Scan(&version).Error)
	t.Logf("database: %s %s", dialect, version)
	return db
}

// Schema immediately before channel attribution was added. The latest release
// v1.0.0-rc.41 has no degradation tables, covered by the fresh case.
type degradationEventBeforeChannel struct {
	ID            int    `gorm:"primaryKey"`
	GroupName     string `gorm:"type:varchar(64);index:idx_degradation_event,priority:1"`
	ModelName     string `gorm:"type:varchar(255);index:idx_degradation_event,priority:2"`
	Status        string `gorm:"type:varchar(32)"`
	DetectedModel string `gorm:"type:varchar(255)"`
	CreatedAt     int64  `gorm:"bigint;index"`
}

func TestDegradationChannelHistoryDatabases(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			for _, upgrade := range []bool{false, true} {
				t.Run(fmt.Sprint("upgrade=", upgrade), func(t *testing.T) {
					db := openDegradationTestDatabase(t, dialect)
					t.Cleanup(func() {
						require.NoError(t, db.Migrator().DropTable(&model.DegradationEvent{}, &model.DegradationTarget{}))
					})
					if upgrade {
						require.NoError(t, db.Table("degradation_events").AutoMigrate(&degradationEventBeforeChannel{}))
						require.NoError(t, db.Table("degradation_events").Create(&degradationEventBeforeChannel{GroupName: "default", ModelName: "a", Status: "passed", DetectedModel: "a", CreatedAt: 1}).Error)
					}
					for range 2 {
						require.NoError(t, db.AutoMigrate(&model.DegradationEvent{}, &model.DegradationTarget{}))
					}
					require.NoError(t, model.SaveDegradationAttempt(model.DegradationTarget{GroupName: "default", ModelName: "a", Status: "passed"}, model.DegradationEvent{GroupName: "default", ModelName: "a", Status: "passed", DetectedModel: "a", ChannelID: 42}))
					require.NoError(t, model.SaveDegradationAttempt(model.DegradationTarget{GroupName: "default", ModelName: "a", Status: "failed"}, model.DegradationEvent{GroupName: "default", ModelName: "a", Status: "failed", ChannelID: 43}))
					events, err := model.ListDegradationEvents("default", "a", 0)
					require.NoError(t, err)
					expectedCount := 2
					if upgrade {
						expectedCount++
					}
					require.Len(t, events, expectedCount)
					assert.Equal(t, 43, events[0].ChannelID)
					assert.Equal(t, 42, events[1].ChannelID)
					if upgrade {
						assert.Zero(t, events[2].ChannelID)
						assert.Equal(t, "passed", events[2].Status)
						assert.Equal(t, "a", events[2].DetectedModel)
					}
					targets, err := model.ListDegradationTargets()
					require.NoError(t, err)
					require.Len(t, targets, 1)
					assert.Equal(t, "failed", targets[0].Status)
					assert.True(t, db.Migrator().HasIndex(&model.DegradationEvent{}, "idx_degradation_event"))
				})
			}
		})
	}
}
func TestDegradationChannelHistoryVisibleOnlyToAdmins(t *testing.T) {
	db := setupManageUserTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.Ability{}, &model.DegradationTarget{}, &model.DegradationEvent{}))
	model.InitOptionMap()
	cfg := degradation.DefaultConfig()
	cfg.Groups = []degradation.GroupConfig{{Group: "default", Models: []degradation.ModelConfig{{Model: "a"}}}}
	raw, err := common.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, model.UpdateOption(degradation.OptionKey, string(raw)))
	require.NoError(t, model.SaveDegradationAttempt(model.DegradationTarget{GroupName: "default", ModelName: "a"}, model.DegradationEvent{GroupName: "default", ModelName: "a", ChannelID: 42}))
	for _, role := range []int{0, common.RoleCommonUser, common.RoleAdminUser} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("role", role)
		GetDegradationPage(c)
		if role >= common.RoleAdminUser {
			assert.Contains(t, w.Body.String(), `"channel_id":42`)
		} else {
			assert.NotContains(t, w.Body.String(), `channel_id`)
		}
	}
}
