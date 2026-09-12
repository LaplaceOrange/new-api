package perfmetrics

import (
	"errors"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRecentSuccessRatesPreferringMaxChannel(t *testing.T) {
	bucket := int64(1_700_000_000)
	earlier := bucket - 3600
	aggregated := map[int64]counters{
		earlier: {requestCount: 20, successCount: 10},
		bucket:  {requestCount: 20, successCount: 10},
	}
	channelBuckets := map[int]map[int64]counters{
		1: {
			earlier: {requestCount: 10, successCount: 10},
			bucket:  {requestCount: 10, successCount: 8},
		},
		2: {
			earlier: {requestCount: 10, successCount: 5},
			bucket:  {requestCount: 10, successCount: 9},
		},
		3: {
			earlier: {requestCount: 10, successCount: 0},
			bucket:  {requestCount: 10, successCount: 0},
		},
	}
	available := map[int]struct{}{1: {}, 2: {}}

	rates := recentSuccessRatesPreferringMaxChannel(aggregated, channelBuckets, available, 3)
	require.Equal(t, []float64{100, 90}, rates)
}

func TestRecentSuccessRatesPreferringMaxChannelFallsBackWithoutChannelData(t *testing.T) {
	bucket := int64(1_700_000_000)
	aggregated := map[int64]counters{
		bucket: {requestCount: 10, successCount: 7},
	}

	rates := recentSuccessRatesPreferringMaxChannel(aggregated, nil, map[int]struct{}{1: {}}, 3)
	require.Equal(t, []float64{70}, rates)

	rates = recentSuccessRatesPreferringMaxChannel(aggregated, map[int]map[int64]counters{
		1: {bucket: {requestCount: 10, successCount: 10}},
	}, nil, 3)
	require.Equal(t, []float64{70}, rates)
}

func TestRecentSuccessRatesPreferringMaxChannelIgnoresUnavailableChannels(t *testing.T) {
	bucket := int64(1_700_000_000)
	aggregated := map[int64]counters{
		bucket: {requestCount: 20, successCount: 10},
	}
	channelBuckets := map[int]map[int64]counters{
		1: {bucket: {requestCount: 10, successCount: 5}},
		2: {bucket: {requestCount: 10, successCount: 10}},
	}
	available := map[int]struct{}{1: {}}

	rates := recentSuccessRatesPreferringMaxChannel(aggregated, channelBuckets, available, 3)
	require.Equal(t, []float64{50}, rates)
}

func TestQuerySummaryAllUsesBestAvailableChannelSuccessRate(t *testing.T) {
	setupPerfSummaryTestDB(t)
	bucket := time.Now().Unix()
	bucket = bucket - (bucket % 3600)

	require.NoError(t, model.DB.Create(&model.Channel{Id: 1, Name: "good", Key: "sk-good", Status: common.ChannelStatusEnabled, Models: "gpt-4", Group: "default"}).Error)
	require.NoError(t, model.DB.Create(&model.Channel{Id: 2, Name: "ok", Key: "sk-ok", Status: common.ChannelStatusEnabled, Models: "gpt-4", Group: "default"}).Error)
	require.NoError(t, model.DB.Create(&model.Channel{Id: 3, Name: "dead", Key: "sk-dead", Status: common.ChannelStatusAutoDisabled, Models: "gpt-4", Group: "default"}).Error)
	require.NoError(t, model.DB.Create(&model.Ability{Group: "default", Model: "gpt-4", ChannelId: 1, Enabled: true}).Error)
	require.NoError(t, model.DB.Create(&model.Ability{Group: "default", Model: "gpt-4", ChannelId: 2, Enabled: true}).Error)
	require.NoError(t, model.DB.Create(&model.Ability{Group: "default", Model: "gpt-4", ChannelId: 3, Enabled: true}).Error)

	require.NoError(t, model.DB.Create(&model.PerfMetric{
		ModelName:      "gpt-4",
		Group:          "default",
		BucketTs:       bucket,
		RequestCount:   30,
		SuccessCount:   15,
		TotalLatencyMs: 3000,
	}).Error)
	require.NoError(t, model.DB.Create(&model.PerfMetricChannel{ModelName: "gpt-4", Group: "default", ChannelId: 1, BucketTs: bucket, RequestCount: 10, SuccessCount: 10}).Error)
	require.NoError(t, model.DB.Create(&model.PerfMetricChannel{ModelName: "gpt-4", Group: "default", ChannelId: 2, BucketTs: bucket, RequestCount: 10, SuccessCount: 5}).Error)
	require.NoError(t, model.DB.Create(&model.PerfMetricChannel{ModelName: "gpt-4", Group: "default", ChannelId: 3, BucketTs: bucket, RequestCount: 10, SuccessCount: 0}).Error)

	result, err := QuerySummaryAll(24, []string{"default"}, true)
	require.NoError(t, err)
	require.Len(t, result.Models, 1)
	assert.Equal(t, "gpt-4", result.Models[0].ModelName)
	assert.Equal(t, 50.0, result.Models[0].SuccessRate)
	assert.Equal(t, []float64{100}, result.Models[0].RecentSuccessRates)
}

func TestQuerySummaryAllSingleGroupUsesGroupStatus(t *testing.T) {
	setupPerfSummaryTestDB(t)
	bucket := time.Now().Unix()
	bucket = bucket - (bucket % 3600)

	require.NoError(t, model.DB.Create(&model.Channel{Id: 1, Name: "good", Key: "sk-good", Status: common.ChannelStatusEnabled, Models: "gpt-4", Group: "default"}).Error)
	require.NoError(t, model.DB.Create(&model.Channel{Id: 2, Name: "ok", Key: "sk-ok", Status: common.ChannelStatusEnabled, Models: "gpt-4", Group: "default"}).Error)
	require.NoError(t, model.DB.Create(&model.Ability{Group: "default", Model: "gpt-4", ChannelId: 1, Enabled: true}).Error)
	require.NoError(t, model.DB.Create(&model.Ability{Group: "vip", Model: "gpt-4", ChannelId: 2, Enabled: true}).Error)

	require.NoError(t, model.DB.Create(&model.PerfMetric{
		ModelName:      "gpt-4",
		Group:          "default",
		BucketTs:       bucket,
		RequestCount:   10,
		SuccessCount:   5,
		TotalLatencyMs: 1000,
	}).Error)
	require.NoError(t, model.DB.Create(&model.PerfMetric{
		ModelName:      "gpt-4",
		Group:          "vip",
		BucketTs:       bucket,
		RequestCount:   10,
		SuccessCount:   10,
		TotalLatencyMs: 1000,
	}).Error)
	require.NoError(t, model.DB.Create(&model.PerfMetricChannel{ModelName: "gpt-4", Group: "default", ChannelId: 1, BucketTs: bucket, RequestCount: 10, SuccessCount: 5}).Error)
	require.NoError(t, model.DB.Create(&model.PerfMetricChannel{ModelName: "gpt-4", Group: "vip", ChannelId: 2, BucketTs: bucket, RequestCount: 10, SuccessCount: 10}).Error)

	allGroups, err := QuerySummaryAll(24, []string{"default", "vip"}, true)
	require.NoError(t, err)
	require.Len(t, allGroups.Models, 1)
	assert.Equal(t, 75.0, allGroups.Models[0].SuccessRate)
	assert.Equal(t, []float64{100}, allGroups.Models[0].RecentSuccessRates)

	defaultGroup, err := QuerySummaryAll(24, []string{"default"}, false)
	require.NoError(t, err)
	require.Len(t, defaultGroup.Models, 1)
	assert.Equal(t, 50.0, defaultGroup.Models[0].SuccessRate)
	assert.Equal(t, []float64{50}, defaultGroup.Models[0].RecentSuccessRates)

	vipGroup, err := QuerySummaryAll(24, []string{"vip"}, false)
	require.NoError(t, err)
	require.Len(t, vipGroup.Models, 1)
	assert.Equal(t, 100.0, vipGroup.Models[0].SuccessRate)
	assert.Equal(t, []float64{100}, vipGroup.Models[0].RecentSuccessRates)
}

func setupPerfSummaryTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}, &model.Model{}, &model.PerfMetric{}, &model.PerfMetricChannel{}))
	original := model.DB
	model.DB = db
	t.Cleanup(func() {
		model.DB = original
		hotBuckets.Range(func(key, _ any) bool {
			hotBuckets.Delete(key)
			return true
		})
	})
}
func TestUpsertPerfMetricChannelAddsCountsAndAutoMigrateIsIdempotent(t *testing.T) {
	setupPerfSummaryTestDB(t)
	require.NoError(t, model.DB.AutoMigrate(&model.PerfMetricChannel{}))

	metric := &model.PerfMetricChannel{
		ModelName:    "gpt-4",
		Group:        "default",
		ChannelId:    11,
		BucketTs:     1_700_000_000,
		RequestCount: 4,
		SuccessCount: 3,
	}
	require.NoError(t, model.UpsertPerfMetricChannel(metric))
	require.NoError(t, model.UpsertPerfMetricChannel(&model.PerfMetricChannel{
		ModelName:    "gpt-4",
		Group:        "default",
		ChannelId:    11,
		BucketTs:     1_700_000_000,
		RequestCount: 6,
		SuccessCount: 2,
	}))

	rows, err := model.GetPerfMetricChannelBucketsAll(1_699_000_000, 1_701_000_000, []string{"default"})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, 11, rows[0].ChannelId)
	assert.EqualValues(t, 10, rows[0].RequestCount)
	assert.EqualValues(t, 5, rows[0].SuccessCount)
}

func TestGetAvailableModelChannelIDsMatchesNameRules(t *testing.T) {
	setupPerfSummaryTestDB(t)

	require.NoError(t, model.DB.Create(&model.Channel{Id: 1, Name: "prefix", Key: "sk-prefix", Status: common.ChannelStatusEnabled, Models: "gpt-4-turbo", Group: "default"}).Error)
	require.NoError(t, model.DB.Create(&model.Channel{Id: 2, Name: "contains", Key: "sk-contains", Status: common.ChannelStatusEnabled, Models: "claude-3-opus", Group: "default"}).Error)
	require.NoError(t, model.DB.Create(&model.Channel{Id: 3, Name: "suffix", Key: "sk-suffix", Status: common.ChannelStatusEnabled, Models: "llama-instruct", Group: "default"}).Error)
	require.NoError(t, model.DB.Create(&model.Channel{Id: 4, Name: "dead", Key: "sk-dead", Status: common.ChannelStatusAutoDisabled, Models: "gpt-4-mini", Group: "default"}).Error)
	require.NoError(t, model.DB.Create(&model.Ability{Group: "default", Model: "gpt-4-turbo", ChannelId: 1, Enabled: true}).Error)
	require.NoError(t, model.DB.Create(&model.Ability{Group: "default", Model: "claude-3-opus", ChannelId: 2, Enabled: true}).Error)
	require.NoError(t, model.DB.Create(&model.Ability{Group: "default", Model: "llama-instruct", ChannelId: 3, Enabled: true}).Error)
	require.NoError(t, model.DB.Create(&model.Ability{Group: "default", Model: "gpt-4-mini", ChannelId: 4, Enabled: true}).Error)
	require.NoError(t, model.DB.Create(&model.Model{ModelName: "gpt-4", NameRule: model.NameRulePrefix}).Error)
	require.NoError(t, model.DB.Create(&model.Model{ModelName: "opus", NameRule: model.NameRuleContains}).Error)
	require.NoError(t, model.DB.Create(&model.Model{ModelName: "instruct", NameRule: model.NameRuleSuffix}).Error)

	available, err := model.GetAvailableModelChannelIDs([]string{"default"})
	require.NoError(t, err)
	assert.Equal(t, map[int]struct{}{1: {}}, available["gpt-4-turbo"])
	assert.Equal(t, map[int]struct{}{1: {}}, available["gpt-4"])
	assert.Equal(t, map[int]struct{}{2: {}}, available["opus"])
	assert.Equal(t, map[int]struct{}{3: {}}, available["instruct"])
	_, hasDisabled := available["gpt-4-mini"]
	assert.False(t, hasDisabled)
}

func TestGetAvailableModelChannelIDsKeepsExactNameAheadOfPrefix(t *testing.T) {
	setupPerfSummaryTestDB(t)

	require.NoError(t, model.DB.Create(&model.Channel{Id: 1, Name: "exact", Key: "sk-exact", Status: common.ChannelStatusEnabled, Models: "gpt-4", Group: "default"}).Error)
	require.NoError(t, model.DB.Create(&model.Channel{Id: 2, Name: "prefix", Key: "sk-prefix", Status: common.ChannelStatusEnabled, Models: "gpt-4-turbo", Group: "default"}).Error)
	require.NoError(t, model.DB.Create(&model.Ability{Group: "default", Model: "gpt-4", ChannelId: 1, Enabled: true}).Error)
	require.NoError(t, model.DB.Create(&model.Ability{Group: "default", Model: "gpt-4-turbo", ChannelId: 2, Enabled: true}).Error)
	require.NoError(t, model.DB.Create(&model.Model{ModelName: "gpt-4", NameRule: model.NameRulePrefix}).Error)

	available, err := model.GetAvailableModelChannelIDs([]string{"default"})
	require.NoError(t, err)
	assert.Equal(t, map[int]struct{}{1: {}}, available["gpt-4"])
	assert.Equal(t, map[int]struct{}{2: {}}, available["gpt-4-turbo"])
}

func TestQuerySummaryAllUsesNameRuleMatchedAvailableChannels(t *testing.T) {
	setupPerfSummaryTestDB(t)
	bucket := time.Now().Unix()
	bucket = bucket - (bucket % 3600)

	require.NoError(t, model.DB.Create(&model.Channel{Id: 1, Name: "good", Key: "sk-good", Status: common.ChannelStatusEnabled, Models: "gpt-4-turbo", Group: "default"}).Error)
	require.NoError(t, model.DB.Create(&model.Channel{Id: 2, Name: "ok", Key: "sk-ok", Status: common.ChannelStatusEnabled, Models: "gpt-4-mini", Group: "default"}).Error)
	require.NoError(t, model.DB.Create(&model.Ability{Group: "default", Model: "gpt-4-turbo", ChannelId: 1, Enabled: true}).Error)
	require.NoError(t, model.DB.Create(&model.Ability{Group: "default", Model: "gpt-4-mini", ChannelId: 2, Enabled: true}).Error)
	require.NoError(t, model.DB.Create(&model.Model{ModelName: "gpt-4", NameRule: model.NameRulePrefix}).Error)
	require.NoError(t, model.DB.Create(&model.PerfMetric{
		ModelName:      "gpt-4",
		Group:          "default",
		BucketTs:       bucket,
		RequestCount:   20,
		SuccessCount:   10,
		TotalLatencyMs: 2000,
	}).Error)
	require.NoError(t, model.DB.Create(&model.PerfMetricChannel{ModelName: "gpt-4", Group: "default", ChannelId: 1, BucketTs: bucket, RequestCount: 10, SuccessCount: 10}).Error)
	require.NoError(t, model.DB.Create(&model.PerfMetricChannel{ModelName: "gpt-4", Group: "default", ChannelId: 2, BucketTs: bucket, RequestCount: 10, SuccessCount: 5}).Error)

	result, err := QuerySummaryAll(24, []string{"default"}, true)
	require.NoError(t, err)
	require.Len(t, result.Models, 1)
	assert.Equal(t, "gpt-4", result.Models[0].ModelName)
	assert.Equal(t, []float64{100}, result.Models[0].RecentSuccessRates)
}

func TestUpsertPerfMetricPairRollsBackWhenChannelWriteFails(t *testing.T) {
	setupPerfSummaryTestDB(t)
	db := model.DB
	forcedErr := errors.New("forced channel upsert failure")
	callbackName := "test:fail-perf-metric-channel-create"
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == "perf_metric_channels" {
			tx.AddError(forcedErr)
		}
	}))
	t.Cleanup(func() { _ = db.Callback().Create().Remove(callbackName) })

	err := model.UpsertPerfMetricPair(&model.PerfMetric{
		ModelName:      "gpt-4",
		Group:          "default",
		BucketTs:       1_700_000_000,
		RequestCount:   4,
		SuccessCount:   3,
		TotalLatencyMs: 400,
	}, &model.PerfMetricChannel{
		ModelName:    "gpt-4",
		Group:        "default",
		ChannelId:    11,
		BucketTs:     1_700_000_000,
		RequestCount: 4,
		SuccessCount: 3,
	})
	require.ErrorIs(t, err, forcedErr)

	var metricCount int64
	require.NoError(t, model.DB.Model(&model.PerfMetric{}).Count(&metricCount).Error)
	assert.Zero(t, metricCount)
	var channelCount int64
	require.NoError(t, model.DB.Model(&model.PerfMetricChannel{}).Count(&channelCount).Error)
	assert.Zero(t, channelCount)
}

func TestRedisBucketKeyStaysAggregatedByModelGroup(t *testing.T) {
	key := bucketKey{model: "gpt-4", group: "default", channelId: 42, bucketTs: 1_700_000_000}
	assert.Equal(t, "perf:gpt-4:default:1700000000", redisBucketKey(key))
	assert.NotContains(t, redisBucketKey(key), "42")
}
