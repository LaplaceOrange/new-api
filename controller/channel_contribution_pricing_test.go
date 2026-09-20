package controller

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestBuildChannelContributionTestRunResponseUsesCurrentSystemPricing(t *testing.T) {
	previousDB := model.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	t.Cleanup(func() {
		model.DB = previousDB
	})
	require.NoError(t, db.AutoMigrate(
		&model.ChannelContributionRevision{},
		&model.ChannelContributionTestRun{},
		&model.ChannelContributionTestResult{},
	))

	previousPrices := ratio_setting.ModelPrice2JSONString()
	previousRatios := ratio_setting.ModelRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(previousPrices))
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(previousRatios))
	})
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{}`))
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"central-priced-model":1}`))

	revision := &model.ChannelContributionRevision{
		ContributionId: 1,
		Models:         "central-priced-model",
	}
	require.NoError(t, db.Create(revision).Error)
	run := &model.ChannelContributionTestRun{
		ContributionId: 1,
		RevisionId:     revision.Id,
		PricingReady:   false,
	}
	require.NoError(t, db.Create(run).Error)
	require.NoError(t, db.Create(&model.ChannelContributionTestResult{
		TestRunId:  run.Id,
		RevisionId: revision.Id,
		Model:      "central-priced-model",
		Success:    true,
	}).Error)

	response, err := buildChannelContributionTestRunResponse(run, true)
	require.NoError(t, err)
	assert.True(t, response.PricingReady)
	require.Len(t, response.Results, 1)
	assert.True(t, response.Results[0].PriceConfigured)

	run.PricingReady = true
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{}`))
	response, err = buildChannelContributionTestRunResponse(run, true)
	require.NoError(t, err)
	assert.False(t, response.PricingReady)
	require.Len(t, response.Results, 1)
	assert.False(t, response.Results[0].PriceConfigured)
}

func setupChannelContributionAdminApproveTest(t *testing.T, models string) *model.ChannelContribution {
	t.Helper()
	previousDB := model.DB
	previousLogDB := model.LOG_DB
	previousRedisEnabled := common.RedisEnabled
	previousMemoryCacheEnabled := common.MemoryCacheEnabled
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(
		&model.Channel{},
		&model.Ability{},
		&model.ChannelContribution{},
		&model.ChannelContributionRevision{},
		&model.ChannelContributionTestRun{},
		&model.ChannelContributionTestResult{},
		&model.ChannelContributionModelHealth{},
	))
	model.DB = db
	model.LOG_DB = db
	common.RedisEnabled = false
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		model.DB = previousDB
		model.LOG_DB = previousLogDB
		common.RedisEnabled = previousRedisEnabled
		common.MemoryCacheEnabled = previousMemoryCacheEnabled
	})

	revision := &model.ChannelContributionRevision{
		Name:         "pending upstream",
		Type:         constant.ChannelTypeOpenAI,
		BaseURL:      "https://example.com",
		Key:          "sk-test",
		Group:        "default",
		Models:       models,
		ModelMapping: "{}",
	}
	configHash, err := model.ComputeChannelContributionConfigHash(revision)
	require.NoError(t, err)
	revision.ConfigHash = configHash
	contribution := &model.ChannelContribution{
		UserId:   42,
		Username: "contributor",
		Status:   model.ChannelContributionStatusDraft,
	}
	require.NoError(t, model.CreateChannelContributionWithRevision(contribution, revision))
	require.NoError(t, model.SubmitChannelContribution(
		contribution.Id,
		contribution.UserId,
		revision.Id,
		revision.ConfigHash,
		"v1",
		"agreement",
		"agreement-hash",
		100,
	))
	contribution, err = model.GetChannelContributionById(contribution.Id)
	require.NoError(t, err)
	return contribution
}

func postApproveAdminChannelContribution(t *testing.T, id int) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Set("id", 7)
	context.Set("username", "reviewer")
	context.Params = gin.Params{{Key: "id", Value: strconv.Itoa(id)}}
	context.Request = httptest.NewRequest(http.MethodPost, "/api/channel-contributions/admin/"+strconv.Itoa(id)+"/approve", http.NoBody)
	ApproveAdminChannelContribution(context)
	return recorder
}

func TestApproveAdminChannelContributionAllowsMissingAndFailedTests(t *testing.T) {
	previousPrices := ratio_setting.ModelPrice2JSONString()
	previousRatios := ratio_setting.ModelRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(previousPrices))
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(previousRatios))
	})
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{}`))
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"gpt-test":1}`))

	untested := setupChannelContributionAdminApproveTest(t, "gpt-test")
	untestedRecorder := postApproveAdminChannelContribution(t, untested.Id)
	require.Equal(t, http.StatusOK, untestedRecorder.Code)
	assert.Contains(t, untestedRecorder.Body.String(), `"success":true`)
	untestedApproved, err := model.GetChannelContributionById(untested.Id)
	require.NoError(t, err)
	assert.Equal(t, model.ChannelContributionStatusApproved, untestedApproved.Status)
	require.NotNil(t, untestedApproved.ChannelId)

	failed := setupChannelContributionAdminApproveTest(t, "gpt-test")
	require.NotNil(t, failed.PendingRevisionId)
	revision, err := model.GetChannelContributionRevision(failed.Id, *failed.PendingRevisionId)
	require.NoError(t, err)
	require.NoError(t, model.DB.Create(&model.ChannelContributionTestRun{
		ContributionId: failed.Id,
		RevisionId:     revision.Id,
		ConfigHash:     revision.ConfigHash,
		ActorId:        7,
		ActorType:      model.ChannelContributionTestActorAdmin,
		Status:         model.ChannelContributionTestRunStatusFailed,
		Total:          1,
		Failed:         1,
		Error:          "probe failed",
	}).Error)
	failedRecorder := postApproveAdminChannelContribution(t, failed.Id)
	require.Equal(t, http.StatusOK, failedRecorder.Code)
	assert.Contains(t, failedRecorder.Body.String(), `"success":true`)
	failedApproved, err := model.GetChannelContributionById(failed.Id)
	require.NoError(t, err)
	assert.Equal(t, model.ChannelContributionStatusApproved, failedApproved.Status)

	unpriced := setupChannelContributionAdminApproveTest(t, "unpriced-model")
	denied := postApproveAdminChannelContribution(t, unpriced.Id)
	assert.Contains(t, denied.Body.String(), `"success":false`)
	assert.Contains(t, denied.Body.String(), "models without configured price")
}
