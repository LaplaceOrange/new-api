package service

import (
	"context"
	"time"

	"github.com/QuantumNous/new-api/model"
)

type imageStudioCleanupHandler struct{}

func (imageStudioCleanupHandler) Type() string            { return "image_studio_cleanup" }
func (imageStudioCleanupHandler) Enabled() bool           { return true }
func (imageStudioCleanupHandler) Interval() time.Duration { return time.Hour }
func (imageStudioCleanupHandler) NewPayload() any         { return nil }

func (imageStudioCleanupHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	now := time.Now().UTC()
	processed := 0
	for {
		var records []model.ImageStudioRecord
		if err := model.DB.Where(
			"(expires_at < ? AND status <> ?) OR (status = ? AND created_at < ?) OR (status = ? AND created_at < ?)",
			now, "expired", "expired", now.Add(-180*24*time.Hour), "processing", now.Add(-time.Hour),
		).Order("id").Limit(100).Find(&records).Error; err != nil {
			failSystemTask(task, runnerID, err)
			return
		}
		if len(records) == 0 {
			break
		}
		for i := range records {
			if err := ctx.Err(); err != nil {
				failSystemTask(task, runnerID, err)
				return
			}
			record := &records[i]
			if record.Status == "processing" && now.Before(record.ExpiresAt) {
				if err := model.DB.Model(record).Updates(map[string]any{"status": "failed", "error": "generation interrupted"}).Error; err != nil {
					failSystemTask(task, runnerID, err)
					return
				}
				continue
			}
			if err := PurgeImageStudioRecord(ctx, record, now.After(record.CreatedAt.Add(180*24*time.Hour))); err != nil {
				failSystemTask(task, runnerID, err)
				return
			}
			processed++
		}
	}
	if err := model.FinishSystemTask(task.TaskID, runnerID, model.SystemTaskStatusSucceeded, map[string]int{"processed": processed}, ""); err != nil {
		logSystemTaskLockError(ctx, task, err)
	}
}

func init() {
	RegisterSystemTaskHandler(imageStudioCleanupHandler{})
}
