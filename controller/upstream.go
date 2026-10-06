package controller

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type upstreamInput struct {
	Name             string   `json:"name"`
	PrimaryURL       string   `json:"primary_url"`
	Addresses        []string `json:"addresses"`
	UserAgent        string   `json:"user_agent"`
	AutoRefreshToken bool     `json:"auto_refresh_token"`
	AccessToken      *string  `json:"access_token"`
	RefreshToken     *string  `json:"refresh_token"`
	AuthMode         string   `json:"auth_mode"`
	AccountEmail     *string  `json:"account_email"`
	AccountPassword  *string  `json:"account_password"`
}

func upstreamResponse(item *model.Upstream) gin.H {
	forecast := buildUpstreamForecast(item)
	return gin.H{
		"id": item.ID, "name": item.Name, "primary_url": item.PrimaryURL, "addresses": item.Addresses,
		"user_agent": item.UserAgent, "auto_refresh_token": item.AutoRefreshToken,
		"has_access_token": item.HasAccessToken, "has_refresh_token": item.HasRefreshToken,
		"auth_mode": item.AuthMode, "has_account_credentials": item.HasAccountCredentials,
		"credential_blocked": item.CredentialBlocked, "balance": item.Balance,
		"balance_updated_at": item.BalanceUpdatedAt, "last_attempt_at": item.LastAttemptAt,
		"last_error": item.LastError, "refreshing": item.Refreshing, "snapshot": item.Snapshot,
		"channel_ids": item.ChannelIDs, "forecast": forecast,
	}
}

func buildUpstreamForecast(item *model.Upstream) gin.H {
	if item.Balance == nil || item.Snapshot == nil || item.Snapshot.Consumption == nil || !item.Snapshot.Complete {
		return gin.H{"status": "unknown", "daily_consumption": nil, "remaining_days": nil, "suggested_topup": nil, "stale": true}
	}
	consumption := *item.Snapshot.Consumption
	days := float64(item.Snapshot.Days)
	if days <= 0 || consumption <= 0 {
		return gin.H{"status": "no_usage", "daily_consumption": 0, "remaining_days": nil, "suggested_topup": 0, "stale": false}
	}
	daily := consumption / days
	remaining := *item.Balance / daily
	suggested := consumption - *item.Balance
	if suggested < 0 {
		suggested = 0
	}
	suggested = math.Ceil(suggested*100) / 100
	status := "healthy"
	if *item.Balance <= 0 {
		status = "exhausted"
	} else if remaining < 1 {
		status = "critical"
	} else if remaining < days {
		status = "warning"
	}
	return gin.H{"status": status, "daily_consumption": daily, "remaining_days": remaining, "suggested_topup": suggested, "stale": false}
}

func requireUpstreamCredentialProof(c *gin.Context) bool {
	if middleware.RequireSecurityProof(c, service.VerificationOperation{Scope: service.VerificationScopeUpstreamCredential}) == nil {
		return false
	}
	return true
}

func GetUpstreamConfig(c *gin.Context) {
	config, err := model.GetUpstreamConfig()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, config)
}

func UpdateUpstreamConfig(c *gin.Context) {
	if !requireUpstreamCredentialProof(c) {
		return
	}
	var config model.UpstreamConfig
	if err := common.DecodeJson(c.Request.Body, &config); err != nil {
		common.ApiErrorMsg(c, "Invalid upstream settings")
		return
	}
	if err := model.SaveUpstreamConfig(config); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, config)
}

func ListUpstreams(c *gin.Context) {
	days, mode, timezoneName := upstreamPeriod(c)
	items, err := model.ListUpstreams()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	for i := range items {
		ids, matchErr := model.MatchUpstreamChannels(items[i].Addresses)
		if matchErr != nil {
			common.ApiError(c, matchErr)
			return
		}
		items[i].ChannelIDs = ids
		var snapshot model.UpstreamSnapshot
		if model.DB.Where("upstream_id = ? AND days = ? AND mode = ?", items[i].ID, days, mode).Order("updated_at desc").First(&snapshot).Error == nil {
			items[i].Snapshot = &snapshot
		}
	}
	out := make([]gin.H, 0, len(items))
	for i := range items {
		out = append(out, upstreamResponse(&items[i]))
	}
	config, err := model.GetUpstreamConfig()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if timezoneName != "" {
		config.Timezone = timezoneName
	}
	common.ApiSuccess(c, gin.H{"items": out, "config": config})
}

func GetUpstream(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	item, err := model.GetUpstream(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	item.ChannelIDs, err = model.MatchUpstreamChannels(item.Addresses)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	days, mode, _ := upstreamPeriod(c)
	var snapshot model.UpstreamSnapshot
	if model.DB.Where("upstream_id = ? AND days = ? AND mode = ?", item.ID, days, mode).First(&snapshot).Error == nil {
		item.Snapshot = &snapshot
	}
	common.ApiSuccess(c, upstreamResponse(item))
}

func upstreamPeriod(c *gin.Context) (int, string, string) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	if days != 1 && days != 7 && days != 30 {
		days = 7
	}
	mode := c.DefaultQuery("mode", "rolling")
	if mode != "rolling" && mode != "calendar" {
		mode = "rolling"
	}
	config, _ := model.GetUpstreamConfig()
	return days, mode, config.Timezone
}

func CreateUpstream(c *gin.Context) {
	if !requireUpstreamCredentialProof(c) {
		return
	}
	var input upstreamInput
	if err := common.DecodeJson(c.Request.Body, &input); err != nil {
		common.ApiErrorMsg(c, "Invalid upstream settings")
		return
	}
	item := &model.Upstream{Name: input.Name, PrimaryURL: input.PrimaryURL, Addresses: input.Addresses, UserAgent: input.UserAgent, AutoRefreshToken: input.AutoRefreshToken, AuthMode: input.AuthMode}
	if err := model.SaveUpstream(item, model.UpstreamCredentials{
		AccessToken: input.AccessToken, RefreshToken: input.RefreshToken,
		AccountEmail: input.AccountEmail, AccountPassword: input.AccountPassword,
	}); err != nil {
		common.ApiError(c, err)
		return
	}
	go func() {
		_ = service.RefreshUpstreamData(context.Background(), item.ID)
	}()
	recordManageAudit(c, "upstream.create", map[string]any{"id": item.ID, "name": item.Name})
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": gin.H{"id": item.ID}})
}

func UpdateUpstream(c *gin.Context) {
	if !requireUpstreamCredentialProof(c) {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var input upstreamInput
	if err := common.DecodeJson(c.Request.Body, &input); err != nil {
		common.ApiErrorMsg(c, "Invalid upstream settings")
		return
	}
	item, err := model.GetUpstream(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	item.Name, item.PrimaryURL, item.Addresses, item.UserAgent, item.AutoRefreshToken = input.Name, input.PrimaryURL, input.Addresses, input.UserAgent, input.AutoRefreshToken
	if input.AuthMode != "" {
		item.AuthMode = input.AuthMode
	}
	if err := model.SaveUpstream(item, model.UpstreamCredentials{
		AccessToken: input.AccessToken, RefreshToken: input.RefreshToken,
		AccountEmail: input.AccountEmail, AccountPassword: input.AccountPassword,
	}); err != nil {
		common.ApiError(c, err)
		return
	}
	go func() {
		_ = service.RefreshUpstreamData(context.Background(), id)
	}()
	recordManageAudit(c, "upstream.update", map[string]any{"id": id, "name": item.Name})
	common.ApiSuccess(c, gin.H{"id": id})
}

func DeleteUpstream(c *gin.Context) {
	if !requireUpstreamCredentialProof(c) {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DeleteUpstream(id); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "upstream.delete", map[string]any{"id": id})
	common.ApiSuccess(c, nil)
}

func RefreshUpstream(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	go func() {
		_ = service.RefreshUpstreamData(context.Background(), id)
	}()
	common.ApiSuccess(c, nil)
}

func RefreshAllUpstreams(c *gin.Context) {
	go func() {
		_ = service.RefreshAllUpstreams(context.Background())
	}()
	common.ApiSuccess(c, nil)
}

func GetUpstreamRankings(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	if days != 1 && days != 7 && days != 30 {
		common.ApiErrorMsg(c, "Days must be 1, 7, or 30")
		return
	}
	mode := c.DefaultQuery("mode", "rolling")
	if mode != "rolling" && mode != "calendar" {
		common.ApiErrorMsg(c, "Invalid period mode")
		return
	}
	dimension := c.DefaultQuery("dimension", "users")
	if dimension != "users" && dimension != "models" {
		common.ApiErrorMsg(c, "Invalid ranking dimension")
		return
	}
	metric := c.DefaultQuery("metric", "quota")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit != 10 && limit != 20 && limit != 50 {
		limit = 20
	}
	item, err := model.GetUpstream(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	ids, err := model.MatchUpstreamChannels(item.Addresses)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	config, configErr := model.GetUpstreamConfig()
	if configErr != nil {
		common.ApiError(c, configErr)
		return
	}
	location, locationErr := time.LoadLocation(config.Timezone)
	if locationErr != nil {
		config.Timezone = "Asia/Shanghai"
		location, _ = time.LoadLocation(config.Timezone)
	}
	now := time.Now().In(location)
	startTime, endTime := now.AddDate(0, 0, -days), now
	if mode == "calendar" {
		endTime = time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, location)
		startTime = endTime.AddDate(0, 0, -days)
	}
	start, end := startTime.Unix(), endTime.Unix()
	ranks, err := model.GetUpstreamRankings(c.Request.Context(), ids, start, end, dimension, metric, limit)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"items": ranks, "start_at": start, "end_at": end,
		"timezone": config.Timezone,
	})
}
