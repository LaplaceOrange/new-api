package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRevenueQueriesFilterOnlinePaymentsAndRankCustomers(t *testing.T) {
	truncateTables(t)

	for id := 1; id <= 7; id++ {
		require.NoError(t, DB.Create(&User{
			Id:       id,
			Username: fmt.Sprintf("revenue-user-%d", id),
			AffCode:  fmt.Sprintf("revenue-aff-%d", id),
			Status:   common.UserStatusEnabled,
		}).Error)
	}
	require.NoError(t, DB.Create(&User{
		Id:       8,
		Username: "revenue-legacy-epay",
		AffCode:  "revenue-aff-8",
		Status:   common.UserStatusEnabled,
	}).Error)

	now := int64(1_000_000)
	topUps := []TopUp{
		{UserId: 1, Money: 10, TradeNo: "revenue-1a", PaymentProvider: PaymentProviderEpay, PaymentMethod: "alipay", Status: common.TopUpStatusSuccess, CompleteTime: now - 10, CreateTime: now - 10},
		{UserId: 1, Money: 5, TradeNo: "revenue-1b", PaymentProvider: PaymentProviderEpay, PaymentMethod: "alipay", Status: common.TopUpStatusSuccess, CompleteTime: now - 9, CreateTime: now - 9},
		{UserId: 2, Money: 20, TradeNo: "revenue-2", PaymentProvider: PaymentProviderStripe, PaymentMethod: PaymentMethodStripe, Status: common.TopUpStatusSuccess, CompleteTime: now - 8, CreateTime: now - 8},
		{UserId: 3, Money: 7, TradeNo: "revenue-3", PaymentProvider: "", PaymentMethod: PaymentMethodStripe, Status: common.TopUpStatusSuccess, CompleteTime: now - 7, CreateTime: now - 7},
		{UserId: 4, Money: 100, TradeNo: "revenue-balance", PaymentProvider: PaymentProviderBalance, PaymentMethod: PaymentMethodBalance, Status: common.TopUpStatusSuccess, CompleteTime: now - 6, CreateTime: now - 6},
		{UserId: 5, Money: 100, TradeNo: "revenue-pending", PaymentProvider: PaymentProviderEpay, PaymentMethod: "alipay", Status: common.TopUpStatusPending, CompleteTime: now - 5, CreateTime: now - 5},
		{UserId: 6, Money: 100, TradeNo: "revenue-unknown", PaymentProvider: "unknown", PaymentMethod: PaymentMethodStripe, Status: common.TopUpStatusSuccess, CompleteTime: now - 4, CreateTime: now - 4},
		{UserId: 7, Money: 100, TradeNo: "revenue-old", PaymentProvider: PaymentProviderEpay, PaymentMethod: "alipay", Status: common.TopUpStatusSuccess, CompleteTime: now - 1_000, CreateTime: now - 1_000},
		{UserId: 8, Money: 3, TradeNo: "revenue-legacy-epay-order", PaymentProvider: "", PaymentMethod: "alipay", Status: common.TopUpStatusSuccess, CompleteTime: now - 3, CreateTime: now - 3},
	}
	for index := range topUps {
		require.NoError(t, DB.Create(&topUps[index]).Error)
	}

	totals, err := GetRevenueTotals(now-100, now)
	require.NoError(t, err)
	assert.Equal(t, float64(45), totals.Revenue)
	assert.Equal(t, int64(5), totals.Orders)
	assert.Equal(t, int64(4), totals.Customers)

	amountRankings, err := GetRevenueCustomerTotals(now-100, now, 10)
	require.NoError(t, err)
	require.Len(t, amountRankings, 4)
	assert.Equal(t, []int{2, 1, 3, 8}, []int{amountRankings[0].UserID, amountRankings[1].UserID, amountRankings[2].UserID, amountRankings[3].UserID})
	assert.Equal(t, float64(20), amountRankings[0].Revenue)
	assert.Equal(t, int64(2), amountRankings[1].Orders)

	orderRankings, err := GetRevenueCustomerOrderTotals(now-100, now, 10)
	require.NoError(t, err)
	require.Len(t, orderRankings, 4)
	assert.Equal(t, []int{1, 2, 3, 8}, []int{orderRankings[0].UserID, orderRankings[1].UserID, orderRankings[2].UserID, orderRankings[3].UserID})
	assert.Equal(t, int64(2), orderRankings[0].Orders)
	assert.Equal(t, float64(20), orderRankings[1].Revenue)
}
