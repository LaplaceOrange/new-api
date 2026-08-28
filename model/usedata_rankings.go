package model

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type RankingQuotaTotal struct {
	ModelName   string `json:"model_name"`
	TotalTokens int64  `json:"total_tokens"`
}

type RankingQuotaBucket struct {
	ModelName string `json:"model_name"`
	Bucket    int64  `json:"bucket"`
	Tokens    int64  `json:"tokens"`
}

type RankingUserTotal struct {
	UserID      int    `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	TotalTokens int64  `json:"total_tokens"`
}

func GetRankingQuotaTotals(startTime int64, endTime int64) ([]RankingQuotaTotal, error) {
	var rows []RankingQuotaTotal
	query := DB.Table("quota_data").
		Select("model_name, sum(token_used) as total_tokens").
		Where("model_name <> ''").
		Group("model_name").
		Having("sum(token_used) > 0").
		Order("total_tokens DESC")
	query = applyRankingQuotaTimeRange(query, startTime, endTime)
	err := query.Find(&rows).Error
	return rows, err
}

func GetRankingQuotaBuckets(startTime int64, endTime int64, bucketSize int64) ([]RankingQuotaBucket, error) {
	if bucketSize <= 0 {
		bucketSize = 3600
	}
	bucketExpr := rankingBucketExpr(bucketSize)
	var rows []RankingQuotaBucket
	query := DB.Table("quota_data").
		Select(fmt.Sprintf("model_name, %s as bucket, sum(token_used) as tokens", bucketExpr)).
		Where("model_name <> ''").
		Group(fmt.Sprintf("model_name, %s", bucketExpr)).
		Having("sum(token_used) > 0").
		Order("bucket ASC")
	query = applyRankingQuotaTimeRange(query, startTime, endTime)
	err := query.Find(&rows).Error
	return rows, err
}

func GetRankingUserTotals(startTime int64, endTime int64, limit int) ([]RankingUserTotal, error) {
	var rows []RankingUserTotal
	query := DB.Table("quota_data AS q").
		Select("q.user_id, COALESCE(NULLIF(MAX(u.username), ''), MAX(q.username), '') AS username, COALESCE(MAX(u.display_name), '') AS display_name, sum(q.token_used) AS total_tokens").
		Joins("LEFT JOIN users AS u ON u.id = q.user_id").
		Where("q.user_id > 0").
		Group("q.user_id").
		Having("sum(q.token_used) > 0").
		Order("total_tokens DESC, q.user_id ASC")
	query = applyRankingQuotaTimeRangeColumn(query, "q.created_at", startTime, endTime)
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&rows).Error
	return rows, err
}

func rankingBucketExpr(bucketSize int64) string {
	if common.UsingMainDatabase(common.DatabaseTypeMySQL) {
		return fmt.Sprintf("FLOOR(created_at / %d) * %d", bucketSize, bucketSize)
	}
	return fmt.Sprintf("(created_at / %d) * %d", bucketSize, bucketSize)
}

func applyRankingQuotaTimeRange(query *gorm.DB, startTime int64, endTime int64) *gorm.DB {
	return applyRankingQuotaTimeRangeColumn(query, "created_at", startTime, endTime)
}

func applyRankingQuotaTimeRangeColumn(query *gorm.DB, column string, startTime int64, endTime int64) *gorm.DB {
	if startTime > 0 {
		query = query.Where(column+" >= ?", startTime)
	}
	if endTime > 0 {
		query = query.Where(column+" <= ?", endTime)
	}
	return query
}
