package controller

import (
	"context"
	"crypto/sha256"
	"fmt"
	"slices"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/degradation"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

type degradationMonitorHandler struct{}

type degradationCheckHandler struct{}

func (degradationCheckHandler) Type() string { return model.SystemTaskTypeDegradationCheck }

func (degradationCheckHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	var payload degradationTestPayload
	err := task.DecodePayload(&payload)
	var result *degradationCheckResult
	if err == nil {
		if payload.ChannelID > 0 {
			channel, loadErr := model.GetChannelById(payload.ChannelID, true)
			err = loadErr
			if err == nil && (!slices.Contains(channel.GetModels(), payload.Model) || !slices.Contains(channel.GetGroups(), payload.Group)) {
				err = fmt.Errorf("model or group is no longer available on this channel")
			}
			if err == nil {
				result, err = detectChannelDegradation(ctx, channel, payload.Group, payload.Model, payload.Expected)
			}
		} else {
			err = runManualDegradationCheck(ctx, payload)
		}
	}
	if err != nil {
		finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}
	finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusSucceeded, result, nil)
}

func enqueueDegradationCheck(payload degradationTestPayload) (*model.SystemTask, bool, error) {
	scope := []any{payload.Group, payload.Model}
	if payload.ChannelID > 0 {
		scope = append(scope, payload.ChannelID)
	}
	raw, err := common.Marshal(scope)
	if err != nil {
		return nil, false, err
	}
	key := fmt.Sprintf("%x", sha256.Sum256(raw))
	return service.EnqueueSystemTaskWithKey(model.SystemTaskTypeDegradationCheck, key, payload)
}

type degradationTestPayload struct {
	Group     string `json:"group"`
	Model     string `json:"model"`
	Scheduled bool   `json:"scheduled,omitempty"`
	ChannelID int    `json:"channel_id,omitempty"`
	Expected  string `json:"expected,omitempty"`
}

func (degradationMonitorHandler) Type() string { return model.SystemTaskTypeDegradationMonitor }

func (degradationMonitorHandler) Enabled() bool {
	cfg := degradation.LoadConfig()
	return cfg.Enabled && cfg.HasModels()
}

func (degradationMonitorHandler) Interval() time.Duration { return time.Minute }

func (degradationMonitorHandler) NewPayload() any { return nil }

func (degradationMonitorHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	var payload degradationTestPayload
	if err := task.DecodePayload(&payload); err != nil {
		finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}
	var err error
	if payload.Group != "" || payload.Model != "" {
		err = runManualDegradationCheck(ctx, payload)
	} else {
		err = scheduleDegradationChecks(ctx)
	}
	if err != nil {
		finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}
	finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusSucceeded, nil, nil)
}

func runManualDegradationCheck(ctx context.Context, payload degradationTestPayload) error {
	cfg := degradation.LoadConfig()
	if payload.Scheduled && !cfg.Enabled {
		return nil
	}
	for _, group := range cfg.Groups {
		if group.Group != payload.Group || !ratio_setting.ContainsGroupRatio(group.Group) {
			continue
		}
		if !enabledModelSet(group.Group)[payload.Model] {
			break
		}
		for _, item := range group.Models {
			if item.Model == payload.Model {
				targets, err := model.ListDegradationTargets()
				if err != nil {
					return err
				}
				target := model.DegradationTarget{GroupName: payload.Group, ModelName: payload.Model}
				for _, existing := range targets {
					if existing.GroupName == payload.Group && existing.ModelName == payload.Model {
						target = existing
						break
					}
				}
				if payload.Scheduled && target.NextCheckAt > time.Now().Unix() {
					return nil
				}
				return executeDegradationCheck(ctx, cfg, target, item.Expected)
			}
		}
	}
	return fmt.Errorf("model %s is not monitored in group %s", payload.Model, payload.Group)
}

func scheduleDegradationChecks(ctx context.Context) error {
	cfg := degradation.LoadConfig()
	if !cfg.Enabled {
		return nil
	}
	now := time.Now()
	if err := model.PruneDegradationEvents(now.Add(-degradationHistoryWindow).Unix()); err != nil {
		return err
	}
	keys := make([][2]string, 0)
	scheduled := map[string]bool{}
	for _, group := range cfg.Groups {
		if !ratio_setting.ContainsGroupRatio(group.Group) {
			continue
		}
		enabled := enabledModelSet(group.Group)
		for _, item := range group.Models {
			if !enabled[item.Model] {
				continue
			}
			keys = append(keys, [2]string{group.Group, item.Model})
			scheduled[group.Group+"\n"+item.Model] = true
			if err := model.EnsureDegradationTarget(group.Group, item.Model, now.Add(cfg.Interval()).Unix()); err != nil {
				return err
			}
		}
	}
	if err := model.DeleteDegradationTargetsExcept(keys); err != nil {
		return err
	}
	targets, err := model.ListDegradationTargets()
	if err != nil {
		return err
	}
	for _, target := range targets {
		if !scheduled[target.GroupName+"\n"+target.ModelName] || target.NextCheckAt > now.Unix() {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, _, err := enqueueDegradationCheck(degradationTestPayload{Group: target.GroupName, Model: target.ModelName, Scheduled: true}); err != nil {
			return err
		}
	}
	return nil
}

func executeDegradationCheck(ctx context.Context, cfg degradation.Config, target model.DegradationTarget, expected string) error {
	passed, scored, detected, channelID := runDegradationProbe(ctx, target.GroupName, target.ModelName, expected)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	verdict := degradation.Decide(time.Now(), target.Failures, cfg.RetryCount, cfg.Interval(), cfg.RetryInterval(), passed, scored, detected)
	return model.SaveDegradationAttempt(model.DegradationTarget{
		GroupName:     target.GroupName,
		ModelName:     target.ModelName,
		Status:        verdict.Status,
		DetectedModel: verdict.Detected,
		Failures:      verdict.Failures,
		NextCheckAt:   verdict.Next.Unix(),
	}, model.DegradationEvent{
		ChannelID:     channelID,
		GroupName:     target.GroupName,
		ModelName:     target.ModelName,
		Status:        verdict.Status,
		DetectedModel: verdict.Detected,
	})
}

func runDegradationProbe(ctx context.Context, groupName, modelName, expected string) (passed bool, scored bool, detected string, channelID int) {
	result, err := detectChannelDegradation(ctx, nil, groupName, modelName, expected)
	if err != nil {
		common.SysError("degradation analyze: " + err.Error())
		return false, false, "", result.ChannelID
	}
	return result.Status == degradation.StatusPassed, true, result.DetectedModel, result.ChannelID
}

type degradationCheckResult struct {
	Status        string `json:"status"`
	DetectedModel string `json:"detected_model"`
	ChannelID     int    `json:"channel_id"`
	Model         string `json:"model"`
}

func detectChannelDegradation(ctx context.Context, channel *model.Channel, groupName, modelName, expected string) (*degradationCheckResult, error) {
	result := &degradationCheckResult{Model: modelName}
	var lastChannelID atomic.Int64
	analysis, err := degradation.Detect(ctx, modelName, func(probeCtx context.Context, prompt string) (degradation.Completion, error) {
		completion, channelID, err := probeDegradationChat(probeCtx, channel, groupName, modelName, prompt)
		lastChannelID.Store(int64(channelID))
		return completion, err
	})
	result.ChannelID = int(lastChannelID.Load())
	if err != nil {
		return result, err
	}
	passed, detected, scored := degradation.Match(degradation.ExpectedName(modelName, expected), analysis.Prediction, analysis.PredictionName, analysis.Decision)
	if !scored {
		return result, fmt.Errorf("detector could not score the samples")
	}
	status := degradation.StatusSuspected
	if passed {
		status = degradation.StatusPassed
	}
	result.Status = status
	result.DetectedModel = detected
	return result, nil
}
