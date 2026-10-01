package controller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	hostdto "github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/degradation"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const degradationHistoryWindow = 30 * 24 * time.Hour

type degradationEventView struct {
	Status        string `json:"status"`
	DetectedModel string `json:"detected_model"`
	CreatedAt     int64  `json:"created_at"`
	ChannelID     int    `json:"channel_id,omitempty"`
}

type degradationModelView struct {
	Model         string                 `json:"model"`
	Sort          int                    `json:"sort"`
	Status        string                 `json:"status"`
	DetectedModel string                 `json:"detected_model"`
	NextCheckAt   int64                  `json:"next_check_at"`
	Timeline      []degradationEventView `json:"timeline"`
}

type degradationGroupView struct {
	Group  string                 `json:"group"`
	Sort   int                    `json:"sort"`
	Models []degradationModelView `json:"models"`
}

type degradationPage struct {
	Groups []degradationGroupView `json:"groups"`
	Config *degradation.Config    `json:"config,omitempty"`
}

func GetDegradationPage(c *gin.Context) {
	isAdmin := c.GetInt("role") >= common.RoleAdminUser
	cfg := degradation.LoadConfig()
	targets, err := model.ListDegradationTargets()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	targetByKey := make(map[string]model.DegradationTarget, len(targets))
	for _, target := range targets {
		targetByKey[target.GroupName+"\n"+target.ModelName] = target
	}
	since := time.Now().Add(-degradationHistoryWindow).Unix()
	groups := make([]degradationGroupView, 0, len(cfg.Groups))
	for _, group := range cfg.Groups {
		enabled := enabledModelSet(group.Group)
		groupKnown := ratio_setting.ContainsGroupRatio(group.Group)
		models := make([]degradationModelView, 0, len(group.Models))
		for _, item := range group.Models {
			view := degradationModelView{
				Model:    item.Model,
				Sort:     item.Sort,
				Status:   degradation.StatusPending,
				Timeline: []degradationEventView{},
			}
			if target, ok := targetByKey[group.Group+"\n"+item.Model]; ok {
				view.Status = target.Status
				view.DetectedModel = target.DetectedModel
				view.NextCheckAt = target.NextCheckAt
			}
			if !groupKnown || !enabled[item.Model] {
				view.Status = degradation.StatusUnavailable
			}
			events, eventErr := model.ListDegradationEvents(group.Group, item.Model, since)
			if eventErr != nil {
				common.ApiError(c, eventErr)
				return
			}
			for _, event := range events {
				eventView := degradationEventView{
					Status:        event.Status,
					DetectedModel: event.DetectedModel,
					CreatedAt:     event.CreatedAt,
				}
				if isAdmin {
					eventView.ChannelID = event.ChannelID
				}
				view.Timeline = append(view.Timeline, eventView)
			}
			models = append(models, view)
		}
		groups = append(groups, degradationGroupView{Group: group.Group, Sort: group.Sort, Models: models})
	}
	page := degradationPage{Groups: groups}
	if isAdmin {
		page.Config = &cfg
	}
	common.ApiSuccess(c, page)
}

func GetDegradationGroupModels(c *gin.Context) {
	group := strings.TrimSpace(c.Query("group"))
	if !ratio_setting.ContainsGroupRatio(group) {
		common.ApiErrorMsg(c, "unknown group")
		return
	}
	names := model.GetGroupEnabledModels(group)
	slices.Sort(names)
	common.ApiSuccess(c, names)
}

func UpdateDegradationConfig(c *gin.Context) {
	var input degradation.Config
	if err := common.DecodeJson(c.Request.Body, &input); err != nil {
		common.ApiError(c, err)
		return
	}
	knownGroups := map[string]struct{}{}
	for name := range ratio_setting.GetGroupRatioCopy() {
		knownGroups[name] = struct{}{}
	}
	models := map[string]map[string]struct{}{}
	for _, group := range input.Groups {
		name := strings.TrimSpace(group.Group)
		if _, ok := models[name]; ok {
			continue
		}
		set := map[string]struct{}{}
		if _, ok := knownGroups[name]; ok {
			for _, modelName := range model.GetGroupEnabledModels(name) {
				set[modelName] = struct{}{}
			}
		}
		models[name] = set
	}
	cfg, err := degradation.Normalize(input, knownGroups, models)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	raw, err := common.Marshal(cfg)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err = model.UpdateOption(degradation.OptionKey, string(raw)); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, cfg)
}

func enabledModelSet(group string) map[string]bool {
	set := map[string]bool{}
	if !ratio_setting.ContainsGroupRatio(group) {
		return set
	}
	for _, name := range model.GetGroupEnabledModels(group) {
		set[name] = true
	}
	return set
}

func degradationChatText(body []byte) (string, error) {
	content := gjson.GetBytes(body, "choices.0.message.content")
	if content.Type == gjson.String {
		text := strings.TrimSpace(content.String())
		if text == "" {
			return "", errors.New("empty completion")
		}
		return text, nil
	}
	if content.IsArray() {
		var builder strings.Builder
		for _, part := range content.Array() {
			if part.Type == gjson.String {
				builder.WriteString(part.String())
				continue
			}
			builder.WriteString(part.Get("text").String())
		}
		text := strings.TrimSpace(builder.String())
		if text == "" {
			return "", errors.New("empty completion")
		}
		return text, nil
	}
	return "", errors.New("missing completion")
}

func probeDegradationChat(ctx context.Context, channel *model.Channel, groupName, modelName, prompt string) (degradation.Completion, int, error) {
	if err := ctx.Err(); err != nil {
		return degradation.Completion{}, 0, err
	}
	userID, err := resolveChannelTestUserID(nil)
	if err != nil {
		return degradation.Completion{}, 0, err
	}
	user, err := model.GetUserCache(userID)
	if err != nil {
		return degradation.Completion{}, 0, err
	}
	if user.Role != common.RoleRootUser || user.Status != common.UserStatusEnabled {
		return degradation.Completion{}, 0, errors.New("degradation monitor requires an enabled root user")
	}
	request := dto.GeneralOpenAIRequest{
		Model:     modelName,
		MaxTokens: common.GetPointer(uint(8192)),
		Messages:  []dto.Message{{Role: "user", Content: prompt}},
	}
	body, err := common.Marshal(request)
	if err != nil {
		return degradation.Completion{}, 0, err
	}
	w := newChannelTestResponseRecorder(int(service.StrictSSRFProtectedResponseBodyLimitBytes))
	channelID := 0
	router := gin.New()
	router.Use(middleware.RequestId(), middleware.BodyStorageCleanup())
	handlers := []gin.HandlerFunc{func(c *gin.Context) {
		user.WriteContext(c)
		c.Set("degradation_monitor", true)
		if err := middleware.SetupContextForToken(c, &model.Token{
			UserId: userID, Name: "degradation-monitor", Group: groupName, UnlimitedQuota: true,
		}); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		common.SetContextKey(c, constant.ContextKeyUsingGroup, groupName)
		// Explicit channel tests remain pinned, including disabled models.
		if channel != nil {
			service.GetChannelConstraints(c).AddPin(hostdto.ChannelPin{
				ChannelId: channel.Id, Source: hostdto.PinSourceToken,
				Rank: hostdto.PinRankToken, RetryMode: hostdto.PinRetrySingleAttempt,
			})
			common.SetContextKey(c, constant.ContextKeyRequestStartTime, time.Now())
			if setupErr := middleware.SetupContextForSelectedChannel(c, channel, modelName); setupErr != nil {
				c.AbortWithStatusJSON(setupErr.StatusCode, gin.H{"error": setupErr.ToOpenAIError()})
				return
			}
		}
		c.Next()
	}}
	if channel == nil {
		handlers = append(handlers, middleware.Distribute())
	}
	handlers = append(handlers, func(c *gin.Context) {
		Relay(c, types.RelayFormatOpenAI)
		channelID = common.GetContextKeyInt(c, constant.ContextKeyChannelId)
	})
	router.POST("/v1/chat/completions", handlers...)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if err := ctx.Err(); err != nil {
		return degradation.Completion{}, channelID, err
	}
	if w.Code != http.StatusOK {
		message := gjson.GetBytes(w.Body.Bytes(), "error.message").String()
		if message == "" {
			message = http.StatusText(w.Code)
		}
		return degradation.Completion{}, channelID, fmt.Errorf("degradation relay failed (status %d): %s", w.Code, message)
	}
	if w.exceeded {
		return degradation.Completion{}, channelID, errors.New("degradation relay response exceeds size limit")
	}
	text, err := degradationChatText(w.Body.Bytes())
	if err != nil {
		return degradation.Completion{}, channelID, err
	}
	return degradation.Completion{Text: text, FinishReason: gjson.GetBytes(w.Body.Bytes(), "choices.0.finish_reason").String()}, channelID, nil
}

func ClearDegradationHistory(c *gin.Context) {
	var input struct {
		Group string `json:"group"`
		Model string `json:"model"`
	}
	if err := common.DecodeJson(c.Request.Body, &input); err != nil {
		common.ApiError(c, err)
		return
	}
	groupName := strings.TrimSpace(input.Group)
	modelName := strings.TrimSpace(input.Model)
	if groupName == "" || modelName == "" {
		common.ApiErrorMsg(c, "group and model are required")
		return
	}
	if err := model.ClearDegradationHistory(groupName, modelName); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func StartDegradationTest(c *gin.Context) {
	var input struct {
		Group string `json:"group"`
		Model string `json:"model"`
	}
	if err := common.DecodeJson(c.Request.Body, &input); err != nil {
		common.ApiError(c, err)
		return
	}
	payload := degradationTestPayload{Group: strings.TrimSpace(input.Group), Model: strings.TrimSpace(input.Model)}
	if payload.Group == "" || payload.Model == "" {
		common.ApiErrorMsg(c, "group and model are required")
		return
	}
	cfg := degradation.LoadConfig()
	found := false
	for _, group := range cfg.Groups {
		if group.Group != payload.Group {
			continue
		}
		for _, item := range group.Models {
			if item.Model == payload.Model {
				found = true
				break
			}
		}
	}
	if !found || !ratio_setting.ContainsGroupRatio(payload.Group) || !enabledModelSet(payload.Group)[payload.Model] {
		common.ApiErrorMsg(c, "model is not available for monitoring")
		return
	}
	task, created, err := enqueueDegradationCheck(payload)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !created {
		common.ApiErrorMsg(c, "a degradation test is already running")
		return
	}
	common.ApiSuccess(c, gin.H{"task_id": task.TaskID})
}

func GetDegradationTest(c *gin.Context) {
	task, err := model.GetSystemTaskByTaskID(c.Param("task_id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if task == nil || (task.Type != model.SystemTaskTypeDegradationMonitor && task.Type != model.SystemTaskTypeDegradationCheck) {
		common.ApiErrorMsg(c, "test not found")
		return
	}
	var result *degradationCheckResult
	if task.Result != "" && task.Result != "null" {
		if err := common.UnmarshalJsonStr(task.Result, &result); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	common.ApiSuccess(c, gin.H{"status": task.Status, "error": task.Error, "result": result})
}

func StartChannelDegradationTest(c *gin.Context) {
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil || channelID <= 0 {
		common.ApiErrorMsg(c, "invalid channel ID")
		return
	}
	var input struct {
		Group    string  `json:"group"`
		Model    string  `json:"model"`
		Expected *string `json:"expected"`
	}
	if err := common.DecodeJson(c.Request.Body, &input); err != nil {
		common.ApiError(c, err)
		return
	}
	channel, err := model.GetChannelById(channelID, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	payload := degradationTestPayload{ChannelID: channelID, Group: strings.TrimSpace(input.Group), Model: strings.TrimSpace(input.Model)}
	if !slices.Contains(channel.GetModels(), payload.Model) || !slices.Contains(channel.GetGroups(), payload.Group) || !ratio_setting.ContainsGroupRatio(payload.Group) {
		common.ApiErrorMsg(c, "model or group is not available on this channel")
		return
	}
	if input.Expected != nil {
		payload.Expected = *input.Expected
	} else {
		for _, group := range degradation.LoadConfig().Groups {
			if group.Group != payload.Group {
				continue
			}
			for _, item := range group.Models {
				if item.Model == payload.Model {
					payload.Expected = item.Expected
					break
				}
			}
		}
	}
	payload.Expected, err = degradation.NormalizeExpectedNames(payload.Expected)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	task, created, err := enqueueDegradationCheck(payload)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !created {
		common.ApiErrorMsg(c, "a degradation test is already running")
		return
	}
	common.ApiSuccess(c, gin.H{"task_id": task.TaskID})
}
