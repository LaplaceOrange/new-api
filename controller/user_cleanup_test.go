package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func performUserCleanupRequest(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	c.Request = httptest.NewRequest(method, path, reader)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", 9999)
	c.Set("role", common.RoleRootUser)
	c.Set("username", "root-operator")
	switch {
	case method == http.MethodGet && path == "/api/user/cleanup/rules":
		GetUserCleanupRules(c)
	case method == http.MethodPut && strings.HasPrefix(path, "/api/user/cleanup/rules"):
		UpdateUserCleanupRules(c)
	case method == http.MethodPost && path == "/api/user/cleanup/scan":
		StartUserCleanupScan(c)
	case method == http.MethodGet && path == "/api/user/cleanup/scan":
		GetUserCleanupScan(c)
	case method == http.MethodGet && path == "/api/user/cleanup/candidates":
		GetUserCleanupCandidates(c)
	case method == http.MethodGet && path == "/api/user/cleanup/candidates/ids":
		GetUserCleanupCandidateIDs(c)
	case method == http.MethodPost && path == "/api/user/cleanup/apply":
		ApplyUserCleanup(c)
	case method == http.MethodGet && strings.HasPrefix(path, "/api/user/cleanup/history"):
		GetUserCleanupHistory(c)
	default:
		t.Fatalf("unhandled path %s %s", method, path)
	}
	return recorder
}

func TestUserCleanupRulesAndApply(t *testing.T) {
	db := setupManageUserTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.SystemTask{}, &model.SystemTaskLock{}, &model.UserCleanupCandidate{}, &model.UserCleanupRecord{}))
	model.InitOptionMap()

	recorder := performUserCleanupRequest(t, http.MethodGet, "/api/user/cleanup/rules", "")
	assert.Contains(t, recorder.Body.String(), `"success":true`)
	assert.Contains(t, recorder.Body.String(), `"unmatched_action":"ignore"`)

	recorder = performUserCleanupRequest(t, http.MethodPut, "/api/user/cleanup/rules", `{"unmatched_action":"include","rules":[{"id":"r1","condition":"group","group":"default","action":"ignore"}]}`)
	assert.Contains(t, recorder.Body.String(), `"success":true`)
	assert.Contains(t, recorder.Body.String(), `"unmatched_action":"include"`)

	user := model.User{Username: "cleanup-target", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1, AffCode: "cleanup-aff", Quota: 42}
	admin := model.User{Username: "cleanup-admin", Password: "password", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1, AffCode: "cleanup-admin-aff"}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&admin).Error)
	require.NoError(t, model.AppendUserCleanupCandidates("scan-apply", []int{user.Id, admin.Id}))
	task, err := model.CreateSystemTask(model.SystemTaskTypeUserCleanupScan, service.UserCleanupScanPayload{}, service.UserCleanupScanState{ScanID: "scan-apply"})
	require.NoError(t, err)
	require.NoError(t, db.Model(task).Updates(map[string]any{"status": model.SystemTaskStatusSucceeded, "task_id": "scan-apply", "active_key": nil}).Error)

	recorder = performUserCleanupRequest(t, http.MethodPost, "/api/user/cleanup/apply", fmt.Sprintf(`{"user_ids":[%d,%d]}`, user.Id, admin.Id))
	assert.Contains(t, recorder.Body.String(), `"success":true`)
	assert.Contains(t, recorder.Body.String(), `"disabled_count":1`)

	var reloaded model.User
	require.NoError(t, db.Where("id = ?", user.Id).First(&reloaded).Error)
	assert.Equal(t, common.UserStatusDisabled, reloaded.Status)
	var reloadedAdmin model.User
	require.NoError(t, db.Where("id = ?", admin.Id).First(&reloadedAdmin).Error)
	assert.Equal(t, common.UserStatusEnabled, reloadedAdmin.Status)

	recorder = performUserCleanupRequest(t, http.MethodGet, "/api/user/cleanup/history?p=1&page_size=10", "")
	assert.Contains(t, recorder.Body.String(), `"success":true`)
	assert.Contains(t, recorder.Body.String(), "cleanup-target")
}

func TestStartUserCleanupScanReturnsExistingActiveTask(t *testing.T) {
	db := setupManageUserTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.SystemTask{}, &model.SystemTaskLock{}, &model.UserCleanupCandidate{}, &model.UserCleanupRecord{}))
	model.InitOptionMap()
	first, err := service.StartUserCleanupScanTask(model.DefaultUserCleanupRulesConfig(), 100)
	require.NoError(t, err)
	second, err := service.StartUserCleanupScanTask(model.DefaultUserCleanupRulesConfig(), 100)
	require.NoError(t, err)
	assert.Equal(t, first.TaskID, second.TaskID)
	var count int64
	require.NoError(t, db.Model(&model.SystemTask{}).Count(&count).Error)
	assert.EqualValues(t, 1, count)
}
