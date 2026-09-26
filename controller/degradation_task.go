package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/degradation"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

type degradationMonitorHandler struct{}

type degradationTestPayload struct {
	Group string `json:"group"`
	Model string `json:"model"`
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
		err = runOneDegradationCheck(ctx)
	}
	if err != nil {
		finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}
	finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusSucceeded, nil, nil)
}

func runManualDegradationCheck(ctx context.Context, payload degradationTestPayload) error {
	cfg := degradation.LoadConfig()
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
				return executeDegradationCheck(ctx, cfg, target, item.Expected)
			}
		}
	}
	return fmt.Errorf("model %s is not monitored in group %s", payload.Model, payload.Group)
}

func runOneDegradationCheck(ctx context.Context) error {
	cfg := degradation.LoadConfig()
	if !cfg.Enabled {
		return nil
	}
	now := time.Now()
	if err := model.PruneDegradationEvents(now.Add(-degradationHistoryWindow).Unix()); err != nil {
		return err
	}
	keys := make([][2]string, 0)
	type scheduledModel struct {
		group    string
		model    string
		expected string
	}
	scheduled := map[string]scheduledModel{}
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
			scheduled[group.Group+"\n"+item.Model] = scheduledModel{group: group.Group, model: item.Model, expected: item.Expected}
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
	var due *model.DegradationTarget
	var dueItem scheduledModel
	for i := range targets {
		item, ok := scheduled[targets[i].GroupName+"\n"+targets[i].ModelName]
		if !ok || targets[i].NextCheckAt > now.Unix() {
			continue
		}
		if due == nil || targets[i].NextCheckAt < due.NextCheckAt {
			copyTarget := targets[i]
			due = &copyTarget
			dueItem = item
		}
	}
	if due == nil {
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return executeDegradationCheck(ctx, cfg, *due, dueItem.expected)
}

func executeDegradationCheck(ctx context.Context, cfg degradation.Config, target model.DegradationTarget, expected string) error {
	passed, scored, detected := runDegradationProbe(ctx, target.GroupName, target.ModelName, expected)
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
		GroupName:     target.GroupName,
		ModelName:     target.ModelName,
		Status:        verdict.Status,
		DetectedModel: verdict.Detected,
	})
}

func runDegradationProbe(ctx context.Context, groupName, modelName, expected string) (passed bool, scored bool, detected string) {
	channel, err := model.GetChannel(groupName, modelName, 0, nil)
	if err != nil || channel == nil || channel.Id == 0 {
		if err != nil {
			common.SysError(fmt.Sprintf("degradation route failed group=%s model=%s err=%v", groupName, modelName, err))
		}
		return false, false, ""
	}
	challenges, err := degradation.GenerateChallenges()
	if err != nil {
		common.SysError("degradation challenges: " + err.Error())
		return false, false, ""
	}
	userID, err := resolveChannelTestUserID(nil)
	if err != nil {
		common.SysError("degradation test user: " + err.Error())
		return false, false, ""
	}
	samples := make([]degradation.Sample, 0, len(challenges))
	for _, challenge := range challenges {
		if ctx.Err() != nil {
			return false, false, ""
		}
		reqCtx, cancel := context.WithTimeout(ctx, degradationProbeTimeout)
		text, probeErr := probeDegradationChat(reqCtx, channel, userID, groupName, modelName, challenge.Prompt)
		cancel()
		if probeErr != nil {
			common.SysError(fmt.Sprintf("degradation probe failed group=%s model=%s channel_id=%d err=%v", groupName, modelName, channel.Id, probeErr))
			return false, false, ""
		}
		samples = append(samples, degradation.Sample{Text: text, ExpectedCount: challenge.ExpectedCount})
	}
	analysis, err := degradation.Analyze(samples)
	if err != nil {
		common.SysError("degradation analyze: " + err.Error())
		return false, false, ""
	}
	passed, detected, scored = degradation.Match(degradation.ExpectedName(modelName, expected), analysis.Prediction, analysis.PredictionName, analysis.Decision)
	return passed, scored, detected
}
