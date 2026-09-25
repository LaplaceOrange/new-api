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

func (degradationMonitorHandler) Type() string { return model.SystemTaskTypeDegradationMonitor }

func (degradationMonitorHandler) Enabled() bool {
	cfg := degradation.LoadConfig()
	return cfg.Enabled && cfg.HasModels()
}

func (degradationMonitorHandler) Interval() time.Duration { return time.Minute }

func (degradationMonitorHandler) NewPayload() any { return nil }

func (degradationMonitorHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	if err := runOneDegradationCheck(ctx); err != nil {
		finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}
	finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusSucceeded, nil, nil)
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
	item := dueItem
	passed, scored, detected := runDegradationProbe(ctx, item.group, item.model, item.expected)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	verdict := degradation.Decide(time.Now(), due.Failures, cfg.RetryCount, cfg.Interval(), cfg.RetryInterval(), passed, scored, detected)
	target := model.DegradationTarget{
		GroupName:     item.group,
		ModelName:     item.model,
		Status:        verdict.Status,
		DetectedModel: verdict.Detected,
		Failures:      verdict.Failures,
		NextCheckAt:   verdict.Next.Unix(),
	}
	return model.SaveDegradationAttempt(target, model.DegradationEvent{
		GroupName:     item.group,
		ModelName:     item.model,
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
