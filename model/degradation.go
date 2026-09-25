package model

import (
	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DegradationTarget struct {
	ID            int    `json:"id" gorm:"primaryKey"`
	GroupName     string `json:"group_name" gorm:"type:varchar(64);uniqueIndex:idx_degradation_target,priority:1"`
	ModelName     string `json:"model_name" gorm:"type:varchar(255);uniqueIndex:idx_degradation_target,priority:2"`
	Status        string `json:"status" gorm:"type:varchar(32);index"`
	DetectedModel string `json:"detected_model" gorm:"type:varchar(255)"`
	Failures      int    `json:"failures"`
	NextCheckAt   int64  `json:"next_check_at" gorm:"bigint;index"`
	UpdatedAt     int64  `json:"updated_at" gorm:"bigint"`
}

type DegradationEvent struct {
	ID            int    `json:"id" gorm:"primaryKey"`
	GroupName     string `json:"group_name" gorm:"type:varchar(64);index:idx_degradation_event,priority:1"`
	ModelName     string `json:"model_name" gorm:"type:varchar(255);index:idx_degradation_event,priority:2"`
	Status        string `json:"status" gorm:"type:varchar(32)"`
	DetectedModel string `json:"detected_model" gorm:"type:varchar(255)"`
	CreatedAt     int64  `json:"created_at" gorm:"bigint;index"`
}

func ListDegradationTargets() ([]DegradationTarget, error) {
	var targets []DegradationTarget
	err := DB.Order("group_name asc, model_name asc").Find(&targets).Error
	return targets, err
}

func ListDegradationEvents(groupName, modelName string, since int64) ([]DegradationEvent, error) {
	var events []DegradationEvent
	err := DB.Where("group_name = ? AND model_name = ? AND created_at >= ?", groupName, modelName, since).
		Order("created_at desc, id desc").
		Limit(200).
		Find(&events).Error
	return events, err
}

func PruneDegradationEvents(before int64) error {
	return DB.Where("created_at < ?", before).Delete(&DegradationEvent{}).Error
}

func SaveDegradationAttempt(target DegradationTarget, event DegradationEvent) error {
	now := common.GetTimestamp()
	target.UpdatedAt = now
	event.CreatedAt = now
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "group_name"}, {Name: "model_name"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"status", "detected_model", "failures", "next_check_at", "updated_at",
			}),
		}).Create(&target).Error; err != nil {
			return err
		}
		return tx.Create(&event).Error
	})
}

func EnsureDegradationTarget(groupName, modelName string, nextCheckAt int64) error {
	target := DegradationTarget{
		GroupName:   groupName,
		ModelName:   modelName,
		Status:      "pending",
		NextCheckAt: nextCheckAt,
		UpdatedAt:   common.GetTimestamp(),
	}
	return DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "group_name"}, {Name: "model_name"}},
		DoNothing: true,
	}).Create(&target).Error
}

func DeleteDegradationTargetsExcept(keys [][2]string) error {
	if len(keys) == 0 {
		return DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&DegradationTarget{}).Error
	}
	query := DB.Model(&DegradationTarget{})
	for i, key := range keys {
		if i == 0 {
			query = query.Where("NOT (group_name = ? AND model_name = ?)", key[0], key[1])
			continue
		}
		query = query.Where("NOT (group_name = ? AND model_name = ?)", key[0], key[1])
	}
	return query.Delete(&DegradationTarget{}).Error
}

func ClearDegradationHistory(groupName, modelName string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_name = ? AND model_name = ?", groupName, modelName).Delete(&DegradationEvent{}).Error; err != nil {
			return err
		}
		return tx.Model(&DegradationTarget{}).
			Where("group_name = ? AND model_name = ?", groupName, modelName).
			Updates(map[string]any{
				"status":         "pending",
				"detected_model": "",
				"failures":       0,
				"updated_at":     common.GetTimestamp(),
			}).Error
	})
}
