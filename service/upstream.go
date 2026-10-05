package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

type upstreamAccountResponse struct {
	Data struct {
		Balance       *float64 `json:"balance"`
		FrozenBalance *float64 `json:"frozen_balance"`
	} `json:"data"`
}

type upstreamUsageResponse struct {
	Data struct {
		TotalActualCost *float64 `json:"total_actual_cost"`
	} `json:"data"`
}

type upstreamUsageListResponse struct {
	Data struct {
		Items []struct {
			ActualCost *float64   `json:"actual_cost"`
			CreatedAt  *time.Time `json:"created_at"`
		} `json:"items"`
		Total    int64 `json:"total"`
		Page     int   `json:"page"`
		PageSize int   `json:"page_size"`
		Pages    int   `json:"pages"`
	} `json:"data"`
}

type upstreamRefreshResponse struct {
	Data struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	} `json:"data"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func upstreamHTTPClient() *http.Client {
	client := *GetStrictSSRFProtectedHTTPClient()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 0 {
			previous := via[len(via)-1].URL
			if !strings.EqualFold(previous.Scheme, req.URL.Scheme) ||
				!strings.EqualFold(previous.Host, req.URL.Host) {
				return errors.New("cross-origin upstream redirect blocked")
			}
		}
		return checkStrictProtectedFetchRedirect(req, via)
	}
	return &client
}

func readUpstreamBody(response *http.Response) ([]byte, error) {
	const maxBodySize = 1 << 20
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBodySize+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxBodySize {
		return nil, errors.New("upstream response is too large")
	}
	return body, nil
}

func upstreamRequest(ctx context.Context, baseURL, path, userAgent, accessToken string, body *strings.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	client := upstreamHTTPClient()
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("upstream returned HTTP %d", response.StatusCode)
	}
	return readUpstreamBody(response)
}

func RefreshUpstream(ctx context.Context, id int) error {
	return refreshUpstream(ctx, id, false)
}

func refreshUpstream(ctx context.Context, id int, refreshAttempted bool) error {
	owner := fmt.Sprintf("upstream-refresh-%d-%d", id, time.Now().UnixNano())
	claimed, err := model.ClaimUpstream(ctx, id, owner)
	if err != nil {
		return err
	}
	defer func() {
		_ = model.UpdateClaimedUpstream(context.Background(), id, owner, map[string]any{"lease_until": 0})
	}()
	accessToken, err := common.DecryptUpstreamCredential(claimed.AccessCipher)
	if err != nil {
		return recordUpstreamRefreshError(ctx, id, owner, err, true, false)
	}
	if accessToken == "" {
		return recordUpstreamRefreshError(ctx, id, owner, errors.New("upstream access credential is not configured"), true, false)
	}
	// sub2api exposes account balance and frozen balance from /api/v1/auth/me.
	body, err := upstreamRequest(ctx, claimed.PrimaryURL, "/api/v1/auth/me", claimed.UserAgent, accessToken, strings.NewReader(""))
	if err != nil {
		if claimed.AutoRefreshToken && claimed.RefreshCipher != "" && !refreshAttempted {
			nextAccess, refreshErr := refreshUpstreamAccess(ctx, claimed, owner)
			if refreshErr != nil {
				return recordUpstreamRefreshError(ctx, id, owner, refreshErr, true, true)
			}
			if nextAccess == "" {
				return recordUpstreamRefreshError(ctx, id, owner, errors.New("upstream refresh response did not contain an access token"), true, true)
			}
			body, err = upstreamRequest(ctx, claimed.PrimaryURL, "/api/v1/auth/me", claimed.UserAgent, nextAccess, strings.NewReader(""))
		}
		if err != nil {
			return recordUpstreamRefreshError(ctx, id, owner, err, false, false)
		}
	}

	var account upstreamAccountResponse
	if err := common.Unmarshal(body, &account); err != nil {
		return recordUpstreamRefreshError(ctx, id, owner, err, false, false)
	}
	if account.Data.Balance == nil || account.Data.FrozenBalance == nil {
		return recordUpstreamRefreshError(ctx, id, owner, errors.New("upstream balance fields are missing"), false, false)
	}
	if *account.Data.Balance < 0 {
		return recordUpstreamRefreshError(ctx, id, owner, errors.New("upstream balance is invalid"), false, false)
	}
	if *account.Data.FrozenBalance < 0 {
		return recordUpstreamRefreshError(ctx, id, owner, errors.New("upstream frozen balance is invalid"), false, false)
	}
	availableBalance := *account.Data.Balance - *account.Data.FrozenBalance
	if availableBalance < 0 {
		availableBalance = 0
	}
	return model.UpdateClaimedUpstream(ctx, id, owner, map[string]any{
		"balance": availableBalance, "balance_updated_at": time.Now().Unix(),
		"last_error": "", "credential_blocked": false, "lease_until": 0,
	})
}

func recordUpstreamRefreshError(ctx context.Context, id int, owner string, refreshErr error, credentialBlocked, refreshPending bool) error {
	values := map[string]any{"last_error": refreshErr.Error()}
	if credentialBlocked {
		values["credential_blocked"] = true
	}
	if refreshPending {
		values["refresh_pending"] = true
	}
	_ = model.UpdateClaimedUpstream(ctx, id, owner, values)
	return refreshErr
}

func refreshUpstreamAccess(ctx context.Context, item *model.Upstream, owner string) (string, error) {
	refreshToken, err := common.DecryptUpstreamCredential(item.RefreshCipher)
	if err != nil || refreshToken == "" {
		return "", errors.New("upstream refresh token is unavailable")
	}
	requestPayload, err := common.Marshal(map[string]string{"refresh_token": refreshToken})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(item.PrimaryURL, "/")+"/api/v1/auth/refresh", bytes.NewReader(requestPayload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if item.UserAgent != "" {
		req.Header.Set("User-Agent", item.UserAgent)
	}
	resp, err := upstreamHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("upstream token refresh returned HTTP %d", resp.StatusCode)
	}
	body, err := readUpstreamBody(resp)
	if err != nil {
		return "", err
	}
	var payload upstreamRefreshResponse
	if err := common.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	access := payload.Data.AccessToken
	if access == "" {
		access = payload.AccessToken
	}
	nextRefresh := payload.Data.RefreshToken
	if nextRefresh == "" {
		nextRefresh = payload.RefreshToken
	}
	if access == "" {
		return "", errors.New("upstream refresh response did not contain an access token")
	}
	accessCipher, err := common.EncryptUpstreamCredential(access)
	if err != nil {
		return "", err
	}
	values := map[string]any{"access_cipher": accessCipher, "credential_blocked": false, "refresh_pending": false}
	if nextRefresh != "" {
		values["refresh_cipher"], err = common.EncryptUpstreamCredential(nextRefresh)
		if err != nil {
			return "", err
		}
	}
	if err := model.UpdateClaimedUpstream(ctx, item.ID, owner, values); err != nil {
		return "", err
	}
	return access, nil
}

func RefreshAllUpstreams(ctx context.Context) error {
	items, err := model.ListUpstreams()
	if err != nil {
		return err
	}
	for i := range items {
		if err := RefreshUpstreamData(ctx, items[i].ID); err != nil {
			common.SysError(fmt.Sprintf("upstream refresh failed: id=%d error=%v", items[i].ID, err))
		}
	}
	return nil
}

func RefreshUpstreamData(ctx context.Context, id int) error {
	balanceErr := RefreshUpstream(ctx, id)
	config, configErr := model.GetUpstreamConfig()
	if configErr != nil {
		if balanceErr != nil {
			return balanceErr
		}
		return configErr
	}
	var statsErr error
	for _, days := range []int{1, 7, 30} {
		for _, mode := range []string{"rolling", "calendar"} {
			if err := UpdateUpstreamStatistics(ctx, id, days, mode, config.Timezone); err != nil {
				statsErr = err
				common.SysError(fmt.Sprintf("upstream statistics failed: id=%d days=%d mode=%s error=%v", id, days, mode, err))
			}
		}
	}
	if balanceErr != nil {
		return balanceErr
	}
	return statsErr
}

func UpdateUpstreamStatistics(ctx context.Context, id, days int, mode, timezoneName string) error {
	owner := fmt.Sprintf("upstream-stats-%d-%d", id, time.Now().UnixNano())
	item, err := model.ClaimUpstream(ctx, id, owner)
	if err != nil {
		return err
	}
	defer func() {
		_ = model.UpdateClaimedUpstream(context.Background(), id, owner, map[string]any{"lease_until": 0})
	}()
	location, err := time.LoadLocation(timezoneName)
	if err != nil {
		timezoneName = "Asia/Shanghai"
		location, _ = time.LoadLocation(timezoneName)
	}
	now := time.Now().In(location)
	start, end := upstreamPeriodBounds(now, days, mode, location)
	accessToken, err := common.DecryptUpstreamCredential(item.AccessCipher)
	if err != nil {
		return saveUpstreamStatisticsError(ctx, owner, id, days, mode, timezoneName, start, end, err)
	}
	var consumption float64
	if mode == "rolling" {
		consumption, err = fetchRollingUpstreamConsumption(ctx, item, accessToken, start, end, timezoneName)
	} else {
		consumption, err = fetchCalendarUpstreamConsumption(ctx, item, accessToken, start, end, timezoneName)
	}
	if err != nil {
		return saveUpstreamStatisticsError(ctx, owner, id, days, mode, timezoneName, start, end, err)
	}
	snapshot := &model.UpstreamSnapshot{
		UpstreamID: id, Days: days, Mode: mode, StartAt: start.Unix(), EndAt: end.Unix(),
		Timezone: timezoneName, Consumption: &consumption, UpdatedAt: time.Now().Unix(), Complete: true,
	}
	return model.SaveUpstreamSnapshot(ctx, owner, snapshot)
}

func upstreamPeriodBounds(now time.Time, days int, mode string, location *time.Location) (time.Time, time.Time) {
	current := now.In(location)
	if mode == "calendar" {
		end := time.Date(current.Year(), current.Month(), current.Day()+1, 0, 0, 0, 0, location)
		return end.AddDate(0, 0, -days), end
	}
	return current.AddDate(0, 0, -days), current
}

func upstreamDateQuery(start, end time.Time, timezoneName string) string {
	values := url.Values{}
	values.Set("start_date", start.Format("2006-01-02"))
	values.Set("end_date", end.Add(-time.Nanosecond).Format("2006-01-02"))
	values.Set("timezone", timezoneName)
	return values.Encode()
}

func fetchCalendarUpstreamConsumption(ctx context.Context, item *model.Upstream, accessToken string, start, end time.Time, timezoneName string) (float64, error) {
	// /usage/stats is used only for complete natural-day windows.
	body, err := upstreamRequest(ctx, item.PrimaryURL, "/api/v1/usage/stats?"+upstreamDateQuery(start, end, timezoneName), item.UserAgent, accessToken, strings.NewReader(""))
	if err != nil {
		return 0, err
	}
	var usage upstreamUsageResponse
	if err := common.Unmarshal(body, &usage); err != nil {
		return 0, err
	}
	if usage.Data.TotalActualCost == nil {
		return 0, errors.New("upstream usage stats did not return total_actual_cost")
	}
	return *usage.Data.TotalActualCost, nil
}

func fetchRollingUpstreamConsumption(ctx context.Context, item *model.Upstream, accessToken string, start, end time.Time, timezoneName string) (float64, error) {
	var total float64
	var seenItems int64
	const pageSize = 1000
	for page := 1; page <= 10000; page++ {
		values := url.Values{}
		values.Set("start_date", start.Format("2006-01-02"))
		values.Set("end_date", end.Add(-time.Nanosecond).Format("2006-01-02"))
		values.Set("timezone", timezoneName)
		values.Set("page", strconv.Itoa(page))
		values.Set("page_size", strconv.Itoa(pageSize))
		body, err := upstreamRequest(ctx, item.PrimaryURL, "/api/v1/usage?"+values.Encode(), item.UserAgent, accessToken, strings.NewReader(""))
		if err != nil {
			return 0, err
		}
		var response upstreamUsageListResponse
		if err := common.Unmarshal(body, &response); err != nil {
			return 0, err
		}
		if len(response.Data.Items) == 0 && response.Data.Total > 0 {
			return 0, errors.New("upstream usage pagination returned an empty page before completion")
		}
		seenItems += int64(len(response.Data.Items))
		for _, entry := range response.Data.Items {
			if entry.ActualCost == nil || entry.CreatedAt == nil {
				return 0, errors.New("upstream usage pagination returned an incomplete record")
			}
			if !entry.CreatedAt.Before(start) && entry.CreatedAt.Before(end) {
				total += *entry.ActualCost
			}
		}
		pages := response.Data.Pages
		if pages == 0 && response.Data.Total > 0 {
			pages = int((response.Data.Total + pageSize - 1) / pageSize)
		}
		if pages == 0 {
			if response.Data.Total == 0 {
				return total, nil
			}
			return 0, errors.New("upstream usage pagination metadata is incomplete")
		}
		if page >= pages {
			if response.Data.Total > 0 && seenItems < response.Data.Total {
				return 0, errors.New("upstream usage pagination did not return all records")
			}
			return total, nil
		}
	}
	return 0, errors.New("upstream usage pagination did not complete")
}

func saveUpstreamStatisticsError(ctx context.Context, owner string, id, days int, mode, timezoneName string, start, end time.Time, statsErr error) error {
	var previous model.UpstreamSnapshot
	_ = model.DB.Where("upstream_id = ? AND days = ? AND mode = ?", id, days, mode).First(&previous)
	snapshot := &model.UpstreamSnapshot{
		UpstreamID: id, Days: days, Mode: mode, Timezone: timezoneName,
		StartAt: start.Unix(), EndAt: end.Unix(),
		UpdatedAt: time.Now().Unix(), LastError: statsErr.Error(), Complete: false,
		Consumption: previous.Consumption,
	}
	if err := model.SaveUpstreamSnapshot(ctx, owner, snapshot); err != nil {
		return statsErr
	}
	return statsErr
}
