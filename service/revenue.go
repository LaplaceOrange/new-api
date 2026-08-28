package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/model"
)

const (
	revenueCacheTTL      = 5 * time.Minute
	revenueCustomerLimit = 10
)

type RevenueResponse struct {
	Summary               RevenueSummary    `json:"summary"`
	Trend                 []RevenuePoint    `json:"trend"`
	Providers             []RevenueProvider `json:"providers"`
	Customers             []RevenueCustomer `json:"customers"`
	CustomerOrderRankings []RevenueCustomer `json:"customer_order_rankings"`
}

type RevenueSummary struct {
	Revenue          float64 `json:"revenue"`
	Orders           int64   `json:"orders"`
	Customers        int64   `json:"customers"`
	AverageOrder     float64 `json:"average_order"`
	RevenueGrowthPct float64 `json:"revenue_growth_pct"`
	OrderGrowthPct   float64 `json:"order_growth_pct"`
}

type RevenuePoint struct {
	Ts        string  `json:"ts"`
	Label     string  `json:"label"`
	Revenue   float64 `json:"revenue"`
	Orders    int64   `json:"orders"`
	Customers int64   `json:"customers"`
}

type RevenueProvider struct {
	Provider  string  `json:"provider"`
	Revenue   float64 `json:"revenue"`
	Orders    int64   `json:"orders"`
	Customers int64   `json:"customers"`
	Share     float64 `json:"share"`
}

type RevenueCustomer struct {
	Rank        int     `json:"rank"`
	UserID      int     `json:"user_id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name,omitempty"`
	Revenue     float64 `json:"revenue"`
	Orders      int64   `json:"orders"`
}

type revenuePeriodConfig struct {
	id          string
	duration    time.Duration
	bucketSize  int64
	labelLayout string
	hasPrevious bool
}

type revenueCacheItem struct {
	expiresAt time.Time
	data      *RevenueResponse
}

var (
	revenueCacheMu sync.Mutex
	revenueCache   = map[string]revenueCacheItem{}
)

func GetRevenueSnapshot(period string) (*RevenueResponse, error) {
	config, err := revenueConfig(period)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	revenueCacheMu.Lock()
	if item, ok := revenueCache[config.id]; ok && now.Before(item.expiresAt) {
		revenueCacheMu.Unlock()
		return item.data, nil
	}
	revenueCacheMu.Unlock()

	data, err := buildRevenueSnapshot(config, now)
	if err != nil {
		return nil, err
	}
	revenueCacheMu.Lock()
	revenueCache[config.id] = revenueCacheItem{expiresAt: now.Add(revenueCacheTTL), data: data}
	revenueCacheMu.Unlock()
	return data, nil
}

func InvalidateRevenueCache() {
	revenueCacheMu.Lock()
	revenueCache = map[string]revenueCacheItem{}
	revenueCacheMu.Unlock()
}

func revenueConfig(period string) (revenuePeriodConfig, error) {
	switch period {
	case "today":
		return revenuePeriodConfig{id: "today", duration: 24 * time.Hour, bucketSize: 3600, labelLayout: "15:04", hasPrevious: true}, nil
	case "", "week":
		return revenuePeriodConfig{id: "week", duration: 7 * 24 * time.Hour, bucketSize: 24 * 3600, labelLayout: "Jan 2", hasPrevious: true}, nil
	case "month":
		return revenuePeriodConfig{id: "month", duration: 30 * 24 * time.Hour, bucketSize: 24 * 3600, labelLayout: "Jan 2", hasPrevious: true}, nil
	case "year":
		return revenuePeriodConfig{id: "year", duration: 365 * 24 * time.Hour, bucketSize: 30 * 24 * 3600, labelLayout: "Jan 2006", hasPrevious: true}, nil
	case "all":
		return revenuePeriodConfig{id: "all", bucketSize: 30 * 24 * 3600, labelLayout: "Jan 2006", hasPrevious: false}, nil
	default:
		return revenuePeriodConfig{}, fmt.Errorf("invalid revenue period: %s", period)
	}
}

func buildRevenueSnapshot(config revenuePeriodConfig, now time.Time) (*RevenueResponse, error) {
	startTime, endTime := revenueTimeRange(config, now)
	totals, err := model.GetRevenueTotals(startTime, endTime)
	if err != nil {
		return nil, err
	}
	buckets, err := model.GetRevenueBuckets(startTime, endTime, config.bucketSize)
	if err != nil {
		return nil, err
	}
	providers, err := model.GetRevenueProviderTotals(startTime, endTime)
	if err != nil {
		return nil, err
	}
	customers, err := model.GetRevenueCustomerTotals(startTime, endTime, revenueCustomerLimit)
	if err != nil {
		return nil, err
	}
	customerOrderRankings, err := model.GetRevenueCustomerOrderTotals(startTime, endTime, revenueCustomerLimit)
	if err != nil {
		return nil, err
	}

	previous := model.RevenueTotals{}
	if config.hasPrevious {
		previousStart, previousEnd := previousRevenueTimeRange(config, startTime)
		previous, err = model.GetRevenueTotals(previousStart, previousEnd)
		if err != nil {
			return nil, err
		}
	}

	return &RevenueResponse{
		Summary:               buildRevenueSummary(totals, previous, config.hasPrevious),
		Trend:                 buildRevenueTrend(buckets, config),
		Providers:             buildRevenueProviders(providers, totals.Revenue),
		Customers:             buildRevenueCustomers(customers),
		CustomerOrderRankings: buildRevenueCustomers(customerOrderRankings),
	}, nil
}

func revenueTimeRange(config revenuePeriodConfig, now time.Time) (int64, int64) {
	endTime := now.Unix()
	if config.duration <= 0 {
		return 0, endTime
	}
	return now.Add(-config.duration).Unix(), endTime
}

func previousRevenueTimeRange(config revenuePeriodConfig, currentStart int64) (int64, int64) {
	previousEnd := currentStart - 1
	return time.Unix(currentStart, 0).Add(-config.duration).Unix(), previousEnd
}

func buildRevenueSummary(current model.RevenueTotals, previous model.RevenueTotals, showGrowth bool) RevenueSummary {
	averageOrder := 0.0
	if current.Orders > 0 {
		averageOrder = current.Revenue / float64(current.Orders)
	}
	summary := RevenueSummary{
		Revenue:      roundRankingFloat(current.Revenue),
		Orders:       current.Orders,
		Customers:    current.Customers,
		AverageOrder: roundRankingFloat(averageOrder),
	}
	if showGrowth {
		summary.RevenueGrowthPct = revenueGrowthPct(current.Revenue, previous.Revenue)
		summary.OrderGrowthPct = rankingGrowthPct(current.Orders, previous.Orders)
	}
	return summary
}

func buildRevenueTrend(rows []model.RevenueBucket, config revenuePeriodConfig) []RevenuePoint {
	points := make([]RevenuePoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, RevenuePoint{
			Ts:        rankingBucketTs(row.Bucket),
			Label:     time.Unix(row.Bucket, 0).Format(config.labelLayout),
			Revenue:   roundRankingFloat(row.Revenue),
			Orders:    row.Orders,
			Customers: row.Customers,
		})
	}
	return points
}

func buildRevenueProviders(rows []model.RevenueProviderTotal, totalRevenue float64) []RevenueProvider {
	providers := make([]RevenueProvider, 0, len(rows))
	for _, row := range rows {
		providers = append(providers, RevenueProvider{
			Provider:  row.Provider,
			Revenue:   roundRankingFloat(row.Revenue),
			Orders:    row.Orders,
			Customers: row.Customers,
			Share:     revenueShare(row.Revenue, totalRevenue),
		})
	}
	return providers
}

func buildRevenueCustomers(rows []model.RevenueCustomerTotal) []RevenueCustomer {
	customers := make([]RevenueCustomer, 0, len(rows))
	for idx, row := range rows {
		username := row.Username
		if username == "" {
			username = fmt.Sprintf("User %d", row.UserID)
		}
		customers = append(customers, RevenueCustomer{
			Rank:        idx + 1,
			UserID:      row.UserID,
			Username:    username,
			DisplayName: row.DisplayName,
			Revenue:     roundRankingFloat(row.Revenue),
			Orders:      row.Orders,
		})
	}
	return customers
}

func revenueGrowthPct(current float64, previous float64) float64 {
	if previous <= 0 {
		if current > 0 {
			return 100
		}
		return 0
	}
	return roundRankingFloat((current - previous) / previous * 100)
}

func revenueShare(value float64, total float64) float64 {
	if value <= 0 || total <= 0 {
		return 0
	}
	return roundRankingFloat(value / total)
}
