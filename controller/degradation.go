package controller

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/degradation"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"slices"
)

const degradationProbeTimeout = 180 * time.Second
const degradationHistoryWindow = 30 * 24 * time.Hour

type degradationEventView struct {
	Status        string `json:"status"`
	DetectedModel string `json:"detected_model"`
	CreatedAt     int64  `json:"created_at"`
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
				view.Timeline = append(view.Timeline, degradationEventView{
					Status:        event.Status,
					DetectedModel: event.DetectedModel,
					CreatedAt:     event.CreatedAt,
				})
			}
			models = append(models, view)
		}
		groups = append(groups, degradationGroupView{Group: group.Group, Sort: group.Sort, Models: models})
	}
	page := degradationPage{Groups: groups}
	if c.GetInt("role") >= common.RoleAdminUser {
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

func probeDegradationChat(ctx context.Context, channel *model.Channel, userID int, groupName, modelName, prompt string) (string, error) {
	result := testChannelWithOptions(ctx, channel, userID, modelName, string(constant.EndpointTypeOpenAI), false, channelTestOptions{
		SkipConsumeLog: true,
		GroupOverride:  groupName,
		Prompt:         prompt,
		ForceChat:      true,
		MaxTokens:      8192,
		Quiet:          true,
	})
	if result.localErr != nil {
		return "", result.localErr
	}
	return degradationChatText(result.body)
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
	var payload degradationTestPayload
	if err := common.DecodeJson(c.Request.Body, &payload); err != nil {
		common.ApiError(c, err)
		return
	}
	payload.Group = strings.TrimSpace(payload.Group)
	payload.Model = strings.TrimSpace(payload.Model)
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
	task, created, err := service.EnqueueSystemTask(model.SystemTaskTypeDegradationMonitor, payload)
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
	if task == nil || task.Type != model.SystemTaskTypeDegradationMonitor {
		common.ApiErrorMsg(c, "test not found")
		return
	}
	common.ApiSuccess(c, gin.H{"status": task.Status, "error": task.Error})
}
