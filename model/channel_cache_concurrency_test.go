package model

import (
	"context"
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupChannelConcurrencySelectionTest(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Channel{}, &Ability{}))

	originalDB := DB
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	originalRedisEnabled, originalRDB := common.RedisEnabled, common.RDB
	channelSyncLock.Lock()
	originalRoutes := group2model2channels
	originalChannels := channelsIDM
	originalAdvanced := channel2advancedCustomConfig
	originalGeneration := channelCacheGeneration
	channelSyncLock.Unlock()

	DB = db
	common.MemoryCacheEnabled = true
	common.RedisEnabled = false
	common.RDB = nil

	t.Cleanup(func() {
		DB = originalDB
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		common.RedisEnabled, common.RDB = originalRedisEnabled, originalRDB
		channelSyncLock.Lock()
		group2model2channels = originalRoutes
		channelsIDM = originalChannels
		channel2advancedCustomConfig = originalAdvanced
		channelCacheGeneration = originalGeneration
		channelSyncLock.Unlock()
		sqlDB, closeErr := db.DB()
		if closeErr == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func createConcurrencySelectionChannel(t *testing.T, db *gorm.DB, id int, priority int64, limit *int) {
	t.Helper()
	weight := uint(100)
	require.NoError(t, db.Create(&Channel{
		Id:               id,
		Type:             constant.ChannelTypeOpenAI,
		Key:              fmt.Sprintf("key-%d", id),
		Status:           common.ChannelStatusEnabled,
		Name:             fmt.Sprintf("channel-%d", id),
		Models:           "concurrency-model",
		Group:            "default",
		Priority:         &priority,
		Weight:           &weight,
		ConcurrencyLimit: limit,
	}).Error)
	require.NoError(t, db.Create(&Ability{
		Group:     "default",
		Model:     "concurrency-model",
		ChannelId: id,
		Enabled:   true,
		Priority:  &priority,
		Weight:    weight,
	}).Error)
}

func TestGetRandomSatisfiedChannelWithConcurrencyFallsBackWithinAndAcrossPriorities(t *testing.T) {
	db := setupChannelConcurrencySelectionTest(t)
	limit := 1
	createConcurrencySelectionChannel(t, db, 3001, 10, &limit)
	createConcurrencySelectionChannel(t, db, 3002, 10, &limit)
	createConcurrencySelectionChannel(t, db, 3003, 5, nil)
	InitChannelCache()

	occupied, acquired, err := common.AcquireChannelConcurrency(context.Background(), 3001, &limit)
	require.NoError(t, err)
	require.True(t, acquired)
	defer occupied.Release()

	selected, lease, err := GetRandomSatisfiedChannelWithConcurrency(context.Background(), "default", "concurrency-model", "/v1/chat/completions")
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.NotNil(t, lease)
	assert.Equal(t, 3002, selected.Id, "an available channel at the same priority wins before lowering priority")
	lease.Release()

	occupied2, acquired, err := common.AcquireChannelConcurrency(context.Background(), 3002, &limit)
	require.NoError(t, err)
	require.True(t, acquired)
	defer occupied2.Release()
	selected, lease, err = GetRandomSatisfiedChannelWithConcurrency(context.Background(), "default", "concurrency-model", "/v1/chat/completions")
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.NotNil(t, lease)
	assert.Equal(t, 3003, selected.Id, "when the whole highest priority is full, selection lowers immediately")
	lease.Release()
}

func TestGetRandomSatisfiedChannelWithConcurrencyRescansHighestPriorityOnRetry(t *testing.T) {
	db := setupChannelConcurrencySelectionTest(t)
	limit := 1
	createConcurrencySelectionChannel(t, db, 3011, 10, &limit)
	createConcurrencySelectionChannel(t, db, 3012, 5, nil)
	InitChannelCache()

	occupied, acquired, err := common.AcquireChannelConcurrency(context.Background(), 3011, &limit)
	require.NoError(t, err)
	require.True(t, acquired)

	selected, lease, err := GetRandomSatisfiedChannelWithConcurrency(context.Background(), "default", "concurrency-model", "/v1/chat/completions")
	require.NoError(t, err)
	require.NotNil(t, selected)
	assert.Equal(t, 3012, selected.Id)
	require.NotNil(t, lease)
	lease.Release()
	occupied.Release()

	selected, lease, err = GetRandomSatisfiedChannelWithConcurrency(context.Background(), "default", "concurrency-model", "/v1/chat/completions")
	require.NoError(t, err)
	require.NotNil(t, selected)
	assert.Equal(t, 3011, selected.Id)
	require.NotNil(t, lease)
	lease.Release()
}

func TestGetRandomSatisfiedChannelWithConcurrencyReturnsNilWhenAllCandidatesAreFull(t *testing.T) {
	db := setupChannelConcurrencySelectionTest(t)
	limit := 1
	createConcurrencySelectionChannel(t, db, 3021, 10, &limit)
	createConcurrencySelectionChannel(t, db, 3022, 5, &limit)
	InitChannelCache()

	first, acquired, err := common.AcquireChannelConcurrency(context.Background(), 3021, &limit)
	require.NoError(t, err)
	require.True(t, acquired)
	second, acquired, err := common.AcquireChannelConcurrency(context.Background(), 3022, &limit)
	require.NoError(t, err)
	require.True(t, acquired)
	defer first.Release()
	defer second.Release()

	selected, lease, err := GetRandomSatisfiedChannelWithConcurrency(context.Background(), "default", "concurrency-model", "/v1/chat/completions")
	require.NoError(t, err)
	assert.Nil(t, selected)
	assert.Nil(t, lease)
}
