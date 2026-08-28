package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogStorageStatsReportsAllRetentionOptions(t *testing.T) {
	truncateTables(t)

	now := int64(1_000_000)
	for index, ageDays := range []int64{2, 10, 40} {
		require.NoError(t, LOG_DB.Create(&Log{
			UserId:    index + 1,
			CreatedAt: now - ageDays*24*60*60,
			Content:   "log entry",
		}).Error)
	}

	stats, err := GetLogStorageStats(now)
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.TotalRows)
	assert.Greater(t, stats.TotalBytes, int64(0))

	options := make(map[int]LogStorageRetention, len(stats.Options))
	for _, option := range stats.Options {
		options[option.Days] = option
	}
	require.Len(t, options, 4)
	assert.Equal(t, int64(3), options[1].ClearableRows)
	assert.Equal(t, int64(2), options[7].ClearableRows)
	assert.Equal(t, int64(1), options[30].ClearableRows)
	assert.Equal(t, int64(3), options[0].ClearableRows)
	assert.Equal(t, int64(1<<63-1), options[0].Cutoff)
	assert.True(t, IsAllowedLogRetentionDays(0))
	assert.False(t, IsAllowedLogRetentionDays(2))
}

func TestDeleteOldLogBatchDeletesOnlyRowsBeforeCutoff(t *testing.T) {
	truncateTables(t)

	require.NoError(t, LOG_DB.Create(&Log{CreatedAt: 99, Content: "old"}).Error)
	require.NoError(t, LOG_DB.Create(&Log{CreatedAt: 100, Content: "boundary"}).Error)
	require.NoError(t, LOG_DB.Create(&Log{CreatedAt: 101, Content: "new"}).Error)

	deleted, err := DeleteOldLogBatch(context.Background(), 100, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	var remaining []Log
	require.NoError(t, LOG_DB.Order("created_at ASC").Find(&remaining).Error)
	require.Len(t, remaining, 2)
	assert.Equal(t, int64(100), remaining[0].CreatedAt)
	assert.Equal(t, int64(101), remaining[1].CreatedAt)
}
