package service

import (
	"context"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
)

type userCleanupScanHandler struct{}

func (userCleanupScanHandler) Type() string { return model.SystemTaskTypeUserCleanupScan }

func (userCleanupScanHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	runUserCleanupScanTask(ctx, task, runnerID)
}

func init() {
	RegisterSystemTaskHandler(userCleanupScanHandler{})
}

type UserCleanupScanPayload struct {
	Rules model.UserCleanupRulesConfig `json:"rules"`
	Now   int64                        `json:"now"`
}

type UserCleanupScanState struct {
	Total            int64  `json:"total"`
	Processed        int64  `json:"processed"`
	Matched          int64  `json:"matched"`
	Progress         int    `json:"progress"`
	SkippedRuleCount int    `json:"skipped_rule_count"`
	ScanID           string `json:"scan_id"`
}

type UserCleanupScanResult struct {
	Matched          int64  `json:"matched"`
	SkippedRuleCount int    `json:"skipped_rule_count"`
	ScanID           string `json:"scan_id"`
}

func StartUserCleanupScanTask(rules model.UserCleanupRulesConfig, now int64) (*model.SystemTask, error) {
	activeTask, err := model.GetActiveSystemTask(model.SystemTaskTypeUserCleanupScan)
	if err != nil {
		return nil, err
	}
	if activeTask != nil {
		return activeTask, nil
	}
	if now <= 0 {
		now = common.GetTimestamp()
	}
	if err := model.ClearUserCleanupCandidates(); err != nil {
		return nil, err
	}
	payload := UserCleanupScanPayload{
		Rules: model.NormalizeUserCleanupRulesConfig(rules),
		Now:   now,
	}
	state := UserCleanupScanState{}
	task, err := model.CreateSystemTask(model.SystemTaskTypeUserCleanupScan, payload, state)
	if err != nil {
		activeTask, activeErr := model.GetActiveSystemTask(model.SystemTaskTypeUserCleanupScan)
		if activeErr == nil && activeTask != nil {
			return activeTask, nil
		}
		return nil, err
	}
	notifySystemTaskRunner()
	return task, nil
}

func runUserCleanupScanTask(ctx context.Context, task *model.SystemTask, runnerID string) {
	payload := UserCleanupScanPayload{}
	if err := task.DecodePayload(&payload); err != nil {
		failSystemTask(task, runnerID, err)
		return
	}
	now := payload.Now
	if now <= 0 {
		now = common.GetTimestamp()
	}
	skipped := model.IncompleteUserCleanupRuleCount(payload.Rules)
	total, err := model.CountEnabledCommonUsers()
	if err != nil {
		failSystemTask(task, runnerID, err)
		return
	}
	state := UserCleanupScanState{
		Total:            total,
		ScanID:           task.TaskID,
		SkippedRuleCount: skipped,
	}
	if err := model.UpdateSystemTaskState(task.TaskID, runnerID, state); err != nil {
		logSystemTaskLockError(ctx, task, err)
		return
	}
	if total == 0 {
		state.Progress = 100
		if err := model.UpdateSystemTaskState(task.TaskID, runnerID, state); err != nil {
			logSystemTaskLockError(ctx, task, err)
			return
		}
		finishUserCleanupScan(ctx, task, runnerID, state)
		return
	}

	afterID := 0
	for {
		if err := ctx.Err(); err != nil {
			failSystemTask(task, runnerID, err)
			return
		}
		users, err := model.ListEnabledCommonUsersAfterID(afterID, model.UserCleanupScanBatchSize())
		if err != nil {
			failSystemTask(task, runnerID, err)
			return
		}
		if len(users) == 0 {
			break
		}
		userIDs := make([]int, 0, len(users))
		for _, user := range users {
			userIDs = append(userIDs, user.Id)
		}
		accessed, err := model.MaxTokenAccessedTimeByUserIDs(userIDs)
		if err != nil {
			failSystemTask(task, runnerID, err)
			return
		}
		matchedIDs := make([]int, 0)
		for _, user := range users {
			lastActivity := model.UserLastActivityAt(user.LastLoginAt, accessed[user.Id])
			action := model.EvaluateUserCleanupAction(payload.Rules, user.Group, lastActivity, now)
			if action == model.UserCleanupActionInclude {
				matchedIDs = append(matchedIDs, user.Id)
			}
			afterID = user.Id
		}
		if err := model.AppendUserCleanupCandidates(task.TaskID, matchedIDs); err != nil {
			failSystemTask(task, runnerID, err)
			return
		}
		state.Processed += int64(len(users))
		state.Matched += int64(len(matchedIDs))
		state.Progress = userCleanupScanProgress(state.Processed, state.Total)
		if err := model.UpdateSystemTaskState(task.TaskID, runnerID, state); err != nil {
			logSystemTaskLockError(ctx, task, err)
			return
		}
		logger.LogInfo(ctx, fmt.Sprintf("user cleanup scan progress: processed=%d total=%d matched=%d", state.Processed, state.Total, state.Matched))
	}

	state.Progress = 100
	finishUserCleanupScan(ctx, task, runnerID, state)
}

func finishUserCleanupScan(ctx context.Context, task *model.SystemTask, runnerID string, state UserCleanupScanState) {
	if err := model.UpdateSystemTaskState(task.TaskID, runnerID, state); err != nil {
		logSystemTaskLockError(ctx, task, err)
		return
	}
	result := UserCleanupScanResult{
		Matched:          state.Matched,
		SkippedRuleCount: state.SkippedRuleCount,
		ScanID:           task.TaskID,
	}
	if err := model.FinishSystemTask(task.TaskID, runnerID, model.SystemTaskStatusSucceeded, result, ""); err != nil {
		logSystemTaskLockError(ctx, task, err)
	}
}

func userCleanupScanProgress(processed int64, total int64) int {
	if total <= 0 {
		return 100
	}
	if processed <= 0 {
		return 0
	}
	if processed >= total {
		return 100
	}
	return int(processed * 100 / total)
}
