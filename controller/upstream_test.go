package controller

import (
	"net/http"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpstreamAccountCredentialWritesRequireLiveRootProof(t *testing.T) {
	for _, state := range []string{"missing", "wrong_scope", "expired", "replayed", "demoted"} {
		t.Run(state, func(t *testing.T) {
			user, identity := setupSecurityEnrollmentTest(t)
			require.NoError(t, model.DB.Model(user).Update("role", common.RoleRootUser).Error)
			require.NoError(t, model.PublishUserAuthCache(user.Id))
			operation := service.VerificationOperation{Scope: service.VerificationScopeUpstreamCredential}
			proof := ""
			expected := "SECURITY_PROOF_REQUIRED"
			if state != "missing" {
				scope := operation
				if state == "wrong_scope" {
					scope.Scope = service.VerificationScopePasswordChange
				}
				proof = issueSecurityEnrollmentProof(t, identity, scope, service.VerificationMethodPassword)
			}
			switch state {
			case "wrong_scope":
				expected = "SECURITY_PROOF_SCOPE_MISMATCH"
			case "expired":
				require.NoError(t, model.DB.Model(&model.AuthFlow{}).Where("purpose = ?", model.AuthFlowPurposeSecurityProof).Update("expires_at", time.Now().Add(-time.Minute)).Error)
				expected = "SECURITY_PROOF_EXPIRED"
			case "replayed":
				_, err := service.ConsumeOperationProof(proof, identity, operation)
				require.NoError(t, err)
				expected = "SECURITY_PROOF_CONSUMED"
			case "demoted":
				require.NoError(t, model.DB.Model(user).Update("role", common.RoleAdminUser).Error)
				require.NoError(t, model.PublishUserAuthCache(user.Id))
				expected = "SECURITY_ACTION_FORBIDDEN"
			}
			response := securityEnrollmentRequest(
				http.MethodPost, "/api/upstream/",
				`{"auth_mode":"password","account_email":"account@example.com","account_password":"private-password"}`,
				proof, identity, CreateUpstream,
			)
			assert.Equal(t, http.StatusForbidden, response.Code)
			var result map[string]any
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
			assert.Equal(t, expected, result["code"])
			assert.NotContains(t, response.Body.String(), "private-password")
		})
	}
}

func TestUpstreamResponseDoesNotExposeAccountCredentials(t *testing.T) {
	item := &model.Upstream{
		AuthMode: model.UpstreamAuthPassword, AccountCipher: "account-secret",
		AccessCipher: "jwt-secret", RefreshCipher: "refresh-secret",
		HasAccountCredentials: true,
	}
	raw, err := common.Marshal(upstreamResponse(item))
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "account-secret")
	assert.NotContains(t, string(raw), "jwt-secret")
	assert.NotContains(t, string(raw), "refresh-secret")
	assert.Contains(t, string(raw), `"has_account_credentials":true`)
}

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
