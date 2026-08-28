package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRankingUserTotalsReturnsTopTenByTokenUsage(t *testing.T) {
	truncateTables(t)

	for id := 1; id <= 11; id++ {
		require.NoError(t, DB.Create(&User{
			Id:          id,
			Username:    fmt.Sprintf("ranking-user-%d", id),
			DisplayName: fmt.Sprintf("Ranking User %d", id),
			AffCode:     fmt.Sprintf("ranking-aff-%d", id),
			Status:      common.UserStatusEnabled,
		}).Error)
		require.NoError(t, DB.Create(&QuotaData{
			UserID:    id,
			Username:  fmt.Sprintf("ranking-user-%d", id),
			ModelName: "ranking-model",
			CreatedAt: 100,
			TokenUsed: (12 - id) * 10,
		}).Error)
	}
	// Anonymous usage is not a user ranking row.
	require.NoError(t, DB.Create(&QuotaData{
		UserID:    0,
		ModelName: "ranking-model",
		CreatedAt: 100,
		TokenUsed: 10_000,
	}).Error)

	rows, err := GetRankingUserTotals(0, 200, 10)
	require.NoError(t, err)
	require.Len(t, rows, 10)
	assert.Equal(t, 1, rows[0].UserID)
	assert.Equal(t, "ranking-user-1", rows[0].Username)
	assert.Equal(t, "Ranking User 1", rows[0].DisplayName)
	assert.Equal(t, int64(110), rows[0].TotalTokens)
	assert.Equal(t, 10, rows[9].UserID)
	assert.NotContains(t, []int{rows[0].UserID, rows[1].UserID, rows[2].UserID, rows[3].UserID, rows[4].UserID, rows[5].UserID, rows[6].UserID, rows[7].UserID, rows[8].UserID, rows[9].UserID}, 11)
}
