package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildUpstreamForecastStatusAndTopUp(t *testing.T) {
	tests := []struct {
		name          string
		balance       float64
		consumption   float64
		wantStatus    string
		wantRemaining any
		wantTopUp     float64
	}{
		{name: "exhausted", balance: 0, consumption: 5, wantStatus: "exhausted", wantRemaining: float64(0), wantTopUp: 5},
		{name: "critical", balance: 0.5, consumption: 7, wantStatus: "critical", wantRemaining: float64(0.5), wantTopUp: 6.5},
		{name: "warning", balance: 4, consumption: 7, wantStatus: "warning", wantRemaining: float64(4), wantTopUp: 3},
		{name: "healthy", balance: 10, consumption: 7, wantStatus: "healthy", wantRemaining: float64(10), wantTopUp: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			balance := tc.balance
			consumption := tc.consumption
			item := &model.Upstream{
				Balance: &balance,
				Snapshot: &model.UpstreamSnapshot{
					Days:        7,
					Consumption: &consumption,
					Complete:    true,
				},
			}
			forecast := buildUpstreamForecast(item)
			require.Equal(t, tc.wantStatus, forecast["status"])
			assert.Equal(t, tc.wantRemaining, forecast["remaining_days"])
			assert.Equal(t, tc.wantTopUp, forecast["suggested_topup"])
		})
	}
}

func TestBuildUpstreamForecastHandlesUnknownAndZeroUsage(t *testing.T) {
	balance := 10.0
	consumption := 0.0
	forecast := buildUpstreamForecast(&model.Upstream{
		Balance: &balance,
		Snapshot: &model.UpstreamSnapshot{
			Days:        7,
			Consumption: &consumption,
			Complete:    true,
		},
	})
	require.Equal(t, "no_usage", forecast["status"])
	assert.Nil(t, forecast["remaining_days"])

	incomplete := true
	unknown := buildUpstreamForecast(&model.Upstream{
		Balance: &balance,
		Snapshot: &model.UpstreamSnapshot{
			Days:        7,
			Consumption: &consumption,
			Complete:    !incomplete,
		},
	})
	require.Equal(t, "unknown", unknown["status"])
	assert.Nil(t, unknown["suggested_topup"])
}
