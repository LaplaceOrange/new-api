package model

import (
	"fmt"
	"math"

	"github.com/QuantumNous/new-api/common"
)

const logStorageFixedBytesPerRow int64 = 128

var logRetentionDays = []int{1, 7, 30, 0}

type LogStorageRetention struct {
	Days           int   `json:"days"`
	Cutoff         int64 `json:"cutoff"`
	ClearableRows  int64 `json:"clearable_rows"`
	EstimatedBytes int64 `json:"estimated_bytes"`
}

type LogStorageStats struct {
	TotalRows  int64                 `json:"total_rows"`
	TotalBytes int64                 `json:"total_bytes"`
	Estimated  bool                  `json:"estimated"`
	Options    []LogStorageRetention `json:"options"`
}

type logStorageAggregate struct {
	TotalRows    int64 `gorm:"column:total_rows"`
	PayloadBytes int64 `gorm:"column:payload_bytes"`
	Rows1        int64 `gorm:"column:rows_1"`
	Bytes1       int64 `gorm:"column:bytes_1"`
	Rows7        int64 `gorm:"column:rows_7"`
	Bytes7       int64 `gorm:"column:bytes_7"`
	Rows30       int64 `gorm:"column:rows_30"`
	Bytes30      int64 `gorm:"column:bytes_30"`
}

func GetLogStorageStats(now int64) (LogStorageStats, error) {
	if now <= 0 {
		now = common.GetTimestamp()
	}
	payloadExpr := logStoragePayloadExpr()
	cutoff1 := LogRetentionCutoff(now, 1)
	cutoff7 := LogRetentionCutoff(now, 7)
	cutoff30 := LogRetentionCutoff(now, 30)
	selectExpr := fmt.Sprintf(`
		count(*) AS total_rows,
		COALESCE(sum(%s), 0) AS payload_bytes,
		COALESCE(sum(CASE WHEN created_at < ? THEN 1 ELSE 0 END), 0) AS rows_1,
		COALESCE(sum(CASE WHEN created_at < ? THEN %s ELSE 0 END), 0) AS bytes_1,
		COALESCE(sum(CASE WHEN created_at < ? THEN 1 ELSE 0 END), 0) AS rows_7,
		COALESCE(sum(CASE WHEN created_at < ? THEN %s ELSE 0 END), 0) AS bytes_7,
		COALESCE(sum(CASE WHEN created_at < ? THEN 1 ELSE 0 END), 0) AS rows_30,
		COALESCE(sum(CASE WHEN created_at < ? THEN %s ELSE 0 END), 0) AS bytes_30`,
		payloadExpr, payloadExpr, payloadExpr, payloadExpr,
	)

	aggregate := logStorageAggregate{}
	err := LOG_DB.Table("logs").Select(
		selectExpr,
		cutoff1, cutoff1,
		cutoff7, cutoff7,
		cutoff30, cutoff30,
	).Scan(&aggregate).Error
	if err != nil {
		return LogStorageStats{}, err
	}

	totalBytes, estimated := getPhysicalLogStorageBytes()
	if totalBytes <= 0 {
		totalBytes = aggregate.PayloadBytes
		estimated = true
	}
	stats := LogStorageStats{
		TotalRows:  aggregate.TotalRows,
		TotalBytes: totalBytes,
		Estimated:  estimated,
		Options:    make([]LogStorageRetention, 0, len(logRetentionDays)),
	}

	for _, days := range logRetentionDays {
		cutoff := LogRetentionCutoff(now, days)
		clearableRows := aggregate.TotalRows
		clearablePayload := aggregate.PayloadBytes
		switch days {
		case 1:
			clearableRows = aggregate.Rows1
			clearablePayload = aggregate.Bytes1
		case 7:
			clearableRows = aggregate.Rows7
			clearablePayload = aggregate.Bytes7
		case 30:
			clearableRows = aggregate.Rows30
			clearablePayload = aggregate.Bytes30
		}
		stats.Options = append(stats.Options, LogStorageRetention{
			Days:           days,
			Cutoff:         cutoff,
			ClearableRows:  clearableRows,
			EstimatedBytes: estimateLogStorageShare(totalBytes, aggregate.TotalRows, aggregate.PayloadBytes, clearableRows, clearablePayload),
		})
	}
	return stats, nil
}

func LogRetentionCutoff(now int64, days int) int64 {
	if days <= 0 {
		return 1<<63 - 1
	}
	return now - int64(days)*24*60*60
}

func IsAllowedLogRetentionDays(days int) bool {
	for _, allowed := range logRetentionDays {
		if days == allowed {
			return true
		}
	}
	return false
}

func logStoragePayloadExpr() string {
	columns := []string{"content", "username", "token_name", "model_name", "`group`", "ip", "request_id", "upstream_request_id", "other"}
	if common.UsingLogDatabase(common.DatabaseTypePostgreSQL) {
		columns[4] = `"group"`
	}
	lengthExpr := func(column string) string {
		switch {
		case common.UsingLogDatabase(common.DatabaseTypeClickHouse):
			return "COALESCE(length(" + column + "), 0)"
		case common.UsingLogDatabase(common.DatabaseTypeSQLite):
			return "COALESCE(LENGTH(CAST(" + column + " AS BLOB)), 0)"
		default:
			return "COALESCE(OCTET_LENGTH(" + column + "), 0)"
		}
	}

	result := fmt.Sprintf("%d", logStorageFixedBytesPerRow)
	for _, column := range columns {
		result += " + " + lengthExpr(column)
	}
	return result
}

func getPhysicalLogStorageBytes() (int64, bool) {
	var size int64
	var err error
	switch {
	case common.UsingLogDatabase(common.DatabaseTypeMySQL):
		err = LOG_DB.Raw("SELECT COALESCE(data_length + index_length, 0) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'logs'").Scan(&size).Error
	case common.UsingLogDatabase(common.DatabaseTypePostgreSQL):
		err = LOG_DB.Raw("SELECT COALESCE(pg_total_relation_size('logs'), 0)").Scan(&size).Error
	case common.UsingLogDatabase(common.DatabaseTypeSQLite):
		err = LOG_DB.Raw("SELECT COALESCE(sum(pgsize), 0) FROM dbstat WHERE name = 'logs' OR name IN (SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = 'logs')").Scan(&size).Error
	case common.UsingLogDatabase(common.DatabaseTypeClickHouse):
		err = LOG_DB.Raw("SELECT COALESCE(sum(bytes_on_disk), 0) FROM system.parts WHERE active AND database = currentDatabase() AND table = 'logs'").Scan(&size).Error
	default:
		return 0, true
	}
	if err != nil || size <= 0 {
		return 0, true
	}
	return size, false
}

func estimateLogStorageShare(totalBytes int64, totalRows int64, totalPayload int64, clearableRows int64, clearablePayload int64) int64 {
	if totalBytes <= 0 || clearableRows <= 0 {
		return 0
	}
	if clearableRows >= totalRows && totalRows > 0 {
		return totalBytes
	}
	ratio := 0.0
	if totalPayload > 0 {
		ratio = float64(clearablePayload) / float64(totalPayload)
	} else if totalRows > 0 {
		ratio = float64(clearableRows) / float64(totalRows)
	}
	if ratio <= 0 {
		return 0
	}
	if ratio >= 1 {
		return totalBytes
	}
	return int64(math.Round(float64(totalBytes) * ratio))
}
