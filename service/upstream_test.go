package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type upstreamTestTransport func(*http.Request) (*http.Response, error)

func (transport upstreamTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return transport(req)
}

func setupUpstreamAccountTest(t *testing.T, access string) *model.Upstream {
	t.Helper()
	t.Setenv("CRYPTO_SECRET", "01234567890123456789012345678901")
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "upstream.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Upstream{}, &model.UpstreamAddress{}, &model.UpstreamSnapshot{}))
	previous := model.DB
	previousType := common.MainDatabaseType()
	model.DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		model.DB = previous
		common.SetMainDatabaseType(previousType)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
	})
	item := &model.Upstream{
		Name: "account", PrimaryURL: "https://upstream.example",
		Addresses: []string{"https://upstream.example"}, AuthMode: model.UpstreamAuthPassword,
		UserAgent: "configured-agent",
	}
	email, password := "account@example.com", "  significant whitespace  "
	require.NoError(t, model.SaveUpstream(item, model.UpstreamCredentials{AccountEmail: &email, AccountPassword: &password}))
	if access != "" {
		cipher, err := common.EncryptUpstreamCredential(access)
		require.NoError(t, err)
		require.NoError(t, db.Model(item).Update("access_cipher", cipher).Error)
	}
	return item
}

func stubUpstreamHTTP(t *testing.T, transport upstreamTestTransport) {
	t.Helper()
	previous := strictSSRFProtectedHTTPClient
	strictSSRFProtectedHTTPClient = &http.Client{Transport: transport}
	t.Cleanup(func() { strictSSRFProtectedHTTPClient = previous })
}

func upstreamTestResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}

func TestUpstreamAccountLoginObtainsAndRenewsJWT(t *testing.T) {
	for _, access := range []string{"", "expired"} {
		t.Run("previous_token_"+access, func(t *testing.T) {
			item := setupUpstreamAccountTest(t, access)
			var logins int
			var accountRequests int
			stubUpstreamHTTP(t, func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "configured-agent", req.Header.Get("User-Agent"))
				assert.Empty(t, req.Header.Get("X-Forwarded-For"))
				if req.URL.Path == "/api/v1/auth/login" {
					logins++
					assert.Equal(t, http.MethodPost, req.Method)
					assert.Empty(t, req.Header.Get("Authorization"))
					var account model.UpstreamAccountCredentials
					require.NoError(t, common.DecodeJson(req.Body, &account))
					assert.Equal(t, "account@example.com", account.Email)
					assert.Equal(t, "  significant whitespace  ", account.Password)
					return upstreamTestResponse(200, `{"code":0,"data":{"access_token":"fresh","refresh_token":"unused"}}`), nil
				}
				assert.Equal(t, "/api/v1/auth/me", req.URL.Path)
				accountRequests++
				if req.Header.Get("Authorization") == "Bearer expired" {
					return upstreamTestResponse(401, `{"code":401}`), nil
				}
				assert.Equal(t, "Bearer fresh", req.Header.Get("Authorization"))
				return upstreamTestResponse(200, `{"data":{"balance":12.5,"frozen_balance":2}}`), nil
			})
			require.NoError(t, RefreshUpstream(context.Background(), item.ID))
			require.Equal(t, 1, logins)
			if access == "" {
				assert.Equal(t, 1, accountRequests)
			} else {
				assert.Equal(t, 2, accountRequests)
			}
			saved, err := model.GetUpstream(item.ID)
			require.NoError(t, err)
			require.NotNil(t, saved.Balance)
			assert.Equal(t, 10.5, *saved.Balance)
			assert.NotEqual(t, "fresh", saved.AccessCipher)
			token, err := common.DecryptUpstreamCredential(saved.AccessCipher)
			require.NoError(t, err)
			assert.Equal(t, "fresh", token)
			require.NoError(t, RefreshUpstream(context.Background(), item.ID))
			assert.Equal(t, 1, logins, "valid cached JWT must not cause another login")
		})
	}
}

func TestUpstreamAccountLoginFailureStopsRepeatedAuthentication(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"wrong password", 401, `{"message":"echoed-password-secret"}`},
		{"captcha required", 403, `{"message":"captcha"}`},
		{"MFA required", 200, `{"code":0,"data":{"requires_2fa":true,"temp_token":"secret"}}`},
		{"business error", 200, `{"code":401,"data":{"access_token":"must-not-use"}}`},
		{"malformed response", 200, `{"data":{"access_token":`},
		{"redirect", 307, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			item := setupUpstreamAccountTest(t, "")
			var calls int
			stubUpstreamHTTP(t, func(req *http.Request) (*http.Response, error) {
				calls++
				assert.Equal(t, "/api/v1/auth/login", req.URL.Path)
				resp := upstreamTestResponse(tc.status, tc.body)
				resp.Header.Set("Location", "https://another.example/steal")
				return resp, nil
			})
			require.Error(t, RefreshUpstream(context.Background(), item.ID))
			require.Error(t, RefreshUpstream(context.Background(), item.ID))
			assert.Equal(t, 1, calls)
			saved, err := model.GetUpstream(item.ID)
			require.NoError(t, err)
			assert.True(t, saved.CredentialBlocked)
			assert.Empty(t, saved.AccessCipher)
			assert.NotContains(t, saved.LastError, "secret")
			assert.Nil(t, saved.Balance)
		})
	}
}

func TestUpstreamServerFailureDoesNotTriggerPasswordLogin(t *testing.T) {
	item := setupUpstreamAccountTest(t, "valid")
	balance := 42.0
	require.NoError(t, model.DB.Model(item).Update("balance", balance).Error)
	stubUpstreamHTTP(t, func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, "/api/v1/auth/me", req.URL.Path)
		return upstreamTestResponse(500, ""), nil
	})
	require.Error(t, RefreshUpstream(context.Background(), item.ID))
	saved, err := model.GetUpstream(item.ID)
	require.NoError(t, err)
	require.NotNil(t, saved.Balance)
	assert.Equal(t, balance, *saved.Balance)
	assert.False(t, saved.CredentialBlocked)
}

func TestUpstreamAccountLoginLeasePreventsConcurrentCredentialUse(t *testing.T) {
	item := setupUpstreamAccountTest(t, "")
	started, resume := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	stubUpstreamHTTP(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/api/v1/auth/login" {
			calls.Add(1)
			close(started)
			<-resume
			return upstreamTestResponse(200, `{"data":{"access_token":"fresh"}}`), nil
		}
		return upstreamTestResponse(200, `{"data":{"balance":1,"frozen_balance":0}}`), nil
	})
	finished := make(chan error, 1)
	go func() { finished <- RefreshUpstream(context.Background(), item.ID) }()
	<-started
	secondErr := RefreshUpstream(context.Background(), item.ID)
	updateErr := model.SaveUpstream(item, model.UpstreamCredentials{})
	close(resume)
	require.NoError(t, <-finished)
	assert.ErrorIs(t, secondErr, model.ErrUpstreamBusy)
	assert.ErrorIs(t, updateErr, model.ErrUpstreamBusy)
	assert.EqualValues(t, 1, calls.Load())
}

func TestUpstreamLoginNetworkErrorDoesNotExposeSecrets(t *testing.T) {
	item := setupUpstreamAccountTest(t, "")
	stubUpstreamHTTP(t, func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network error containing credential-secret")
	})
	err := RefreshUpstream(context.Background(), item.ID)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "credential-secret")
	saved, err := model.GetUpstream(item.ID)
	require.NoError(t, err)
	assert.False(t, saved.CredentialBlocked, "transient failures should be retryable")
}

func TestUpstreamCredentialTransportAlwaysVerifiesTLS(t *testing.T) {
	previous := common.TLSInsecureSkipVerify
	common.TLSInsecureSkipVerify = true
	t.Cleanup(func() { common.TLSInsecureSkipVerify = previous })
	client := upstreamHTTPClient()
	protected, ok := client.Transport.(*ssrfProtectedRoundTripper)
	require.True(t, ok)
	transport := protected.newTransport(nil)
	assert.True(t, transport.TLSClientConfig == nil || !transport.TLSClientConfig.InsecureSkipVerify)
}

func TestUpstreamJWTModeRenewsOnlyExpiredTokensAndStopsUncertainReplay(t *testing.T) {
	for _, outcome := range []string{"success", "uncertain"} {
		t.Run(outcome, func(t *testing.T) {
			item := setupUpstreamAccountTest(t, "")
			item.AuthMode, item.AutoRefreshToken = model.UpstreamAuthJWT, true
			access, refresh := "expired", "one-time-refresh"
			require.NoError(t, model.SaveUpstream(item, model.UpstreamCredentials{AccessToken: &access, RefreshToken: &refresh}))
			var rotations int
			stubUpstreamHTTP(t, func(req *http.Request) (*http.Response, error) {
				if req.URL.Path == "/api/v1/auth/refresh" {
					rotations++
					var payload map[string]string
					require.NoError(t, common.DecodeJson(req.Body, &payload))
					assert.Equal(t, "one-time-refresh", payload["refresh_token"])
					pending, err := model.GetUpstream(item.ID)
					require.NoError(t, err)
					assert.True(t, pending.RefreshPending, "mark rotation before transmitting a single-use token")
					if outcome == "uncertain" {
						return nil, errors.New("response lost after token rotation")
					}
					return upstreamTestResponse(200, `{"code":0,"data":{"access_token":"fresh","refresh_token":"next-refresh"}}`), nil
				}
				assert.Equal(t, "/api/v1/auth/me", req.URL.Path)
				if req.Header.Get("Authorization") == "Bearer expired" {
					return upstreamTestResponse(401, ""), nil
				}
				assert.Equal(t, "Bearer fresh", req.Header.Get("Authorization"))
				return upstreamTestResponse(200, `{"data":{"balance":5,"frozen_balance":0}}`), nil
			})
			err := RefreshUpstream(context.Background(), item.ID)
			if outcome == "success" {
				require.NoError(t, err)
				saved, err := model.GetUpstream(item.ID)
				require.NoError(t, err)
				token, err := common.DecryptUpstreamCredential(saved.RefreshCipher)
				require.NoError(t, err)
				assert.Equal(t, "next-refresh", token)
				assert.False(t, saved.RefreshPending)
				require.NoError(t, RefreshUpstream(context.Background(), item.ID))
			} else {
				require.Error(t, err)
				require.Error(t, RefreshUpstream(context.Background(), item.ID))
				saved, err := model.GetUpstream(item.ID)
				require.NoError(t, err)
				assert.True(t, saved.RefreshPending)
				assert.True(t, saved.CredentialBlocked)
			}
			assert.Equal(t, 1, rotations)
		})
	}
}
