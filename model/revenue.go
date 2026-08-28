package model

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

var onlinePaymentProviders = []string{
	PaymentProviderEpay,
	PaymentProviderStripe,
	PaymentProviderCreem,
	PaymentProviderWaffo,
	PaymentProviderWaffoPancake,
}

var legacyOfflinePaymentMethods = []string{
	PaymentMethodBalance,
}

type RevenueTotals struct {
	Revenue   float64 `json:"revenue"`
	Orders    int64   `json:"orders"`
	Customers int64   `json:"customers"`
}

type RevenueBucket struct {
	Bucket    int64   `json:"bucket"`
	Revenue   float64 `json:"revenue"`
	Orders    int64   `json:"orders"`
	Customers int64   `json:"customers"`
}

type RevenueProviderTotal struct {
	Provider  string  `json:"provider"`
	Revenue   float64 `json:"revenue"`
	Orders    int64   `json:"orders"`
	Customers int64   `json:"customers"`
}

type RevenueCustomerTotal struct {
	UserID      int     `json:"user_id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	Revenue     float64 `json:"revenue"`
	Orders      int64   `json:"orders"`
}

func GetRevenueTotals(startTime int64, endTime int64) (RevenueTotals, error) {
	row := RevenueTotals{}
	query := applyOnlineRevenueFilter(DB.Table("top_ups"), "top_ups.").
		Select("COALESCE(sum(money), 0) AS revenue, count(*) AS orders, count(DISTINCT user_id) AS customers")
	query = applyRevenueTimeRange(query, startTime, endTime)
	err := query.Scan(&row).Error
	return row, err
}

func GetRevenueBuckets(startTime int64, endTime int64, bucketSize int64) ([]RevenueBucket, error) {
	if bucketSize <= 0 {
		bucketSize = 3600
	}
	timeExpr := revenueTimeExpr("")
	bucketExpr := revenueBucketExpr(timeExpr, bucketSize)
	rows := make([]RevenueBucket, 0)
	query := applyOnlineRevenueFilter(DB.Table("top_ups"), "top_ups.").
		Select(fmt.Sprintf("%s AS bucket, COALESCE(sum(money), 0) AS revenue, count(*) AS orders, count(DISTINCT user_id) AS customers", bucketExpr)).
		Group(bucketExpr).
		Order("bucket ASC")
	query = applyRevenueTimeRange(query, startTime, endTime)
	err := query.Find(&rows).Error
	return rows, err
}

func GetRevenueProviderTotals(startTime int64, endTime int64) ([]RevenueProviderTotal, error) {
	providerExpr := "CASE WHEN payment_provider = '' THEN payment_method ELSE payment_provider END"
	rows := make([]RevenueProviderTotal, 0)
	query := applyOnlineRevenueFilter(DB.Table("top_ups"), "top_ups.").
		Select(providerExpr + " AS provider, COALESCE(sum(money), 0) AS revenue, count(*) AS orders, count(DISTINCT user_id) AS customers").
		Group(providerExpr).
		Order("revenue DESC, provider ASC")
	query = applyRevenueTimeRange(query, startTime, endTime)
	err := query.Find(&rows).Error
	return rows, err
}

func GetRevenueCustomerTotals(startTime int64, endTime int64, limit int) ([]RevenueCustomerTotal, error) {
	return getRevenueCustomerTotals(startTime, endTime, limit, "COALESCE(SUM(t.money), 0) DESC, t.user_id ASC")
}

func GetRevenueCustomerOrderTotals(startTime int64, endTime int64, limit int) ([]RevenueCustomerTotal, error) {
	return getRevenueCustomerTotals(startTime, endTime, limit, "COUNT(*) DESC, COALESCE(SUM(t.money), 0) DESC, t.user_id ASC")
}

func getRevenueCustomerTotals(startTime int64, endTime int64, limit int, order string) ([]RevenueCustomerTotal, error) {
	rows := make([]RevenueCustomerTotal, 0)
	query := applyOnlineRevenueFilter(DB.Table("top_ups AS t"), "t.").
		Select("t.user_id, COALESCE(MAX(u.username), '') AS username, COALESCE(MAX(u.display_name), '') AS display_name, COALESCE(sum(t.money), 0) AS revenue, count(*) AS orders").
		Joins("LEFT JOIN users AS u ON u.id = t.user_id").
		Group("t.user_id").
		Order(order)
	query = applyRevenueTimeRangeColumn(query, revenueTimeExpr("t."), startTime, endTime)
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&rows).Error
	return rows, err
}

func applyOnlineRevenueFilter(query *gorm.DB, prefix string) *gorm.DB {
	if prefix == "" {
		prefix = "top_ups."
	}
	return query.
		Where(prefix+"status = ?", common.TopUpStatusSuccess).
		Where(prefix+"money > 0").
		Where("("+prefix+"payment_provider IN ? OR ("+prefix+"payment_provider = '' AND "+prefix+"payment_method <> '' AND "+prefix+"payment_method NOT IN ?))", onlinePaymentProviders, legacyOfflinePaymentMethods)
}

func applyRevenueTimeRange(query *gorm.DB, startTime int64, endTime int64) *gorm.DB {
	return applyRevenueTimeRangeColumn(query, revenueTimeExpr(""), startTime, endTime)
}

func applyRevenueTimeRangeColumn(query *gorm.DB, columnExpr string, startTime int64, endTime int64) *gorm.DB {
	if startTime > 0 {
		query = query.Where(columnExpr+" >= ?", startTime)
	}
	if endTime > 0 {
		query = query.Where(columnExpr+" <= ?", endTime)
	}
	return query
}

func revenueTimeExpr(prefix string) string {
	return "CASE WHEN " + prefix + "complete_time > 0 THEN " + prefix + "complete_time ELSE " + prefix + "create_time END"
}

func revenueBucketExpr(timeExpr string, bucketSize int64) string {
	if common.UsingMainDatabase(common.DatabaseTypeMySQL) {
		return fmt.Sprintf("FLOOR((%s) / %d) * %d", timeExpr, bucketSize, bucketSize)
	}
	return fmt.Sprintf("((%s) / %d) * %d", timeExpr, bucketSize, bucketSize)
}
