package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ReconcileReferralTaskQuota commits funding, task quota, and rebate reversals
// together. A repeated callback sees the durable quota and returns no delta.
func ReconcileReferralTaskQuota(task *Task, target int) (int, error) {
	if target < 0 || target > common.MaxQuota || task.ID <= 0 || task.PrivateData.ReferralSourceId == "" {
		return 0, errors.New("invalid referral task settlement")
	}
	var delta int
	err := DB.Transaction(func(tx *gorm.DB) error {
		var stored Task
		if err := lockForUpdate(tx).First(&stored, task.ID).Error; err != nil {
			return err
		}
		if stored.UserId != task.UserId || stored.PrivateData.ReferralSourceId != task.PrivateData.ReferralSourceId {
			return errors.New("task funding owner mismatch")
		}
		if stored.Quota < 0 || stored.Quota > common.MaxQuota {
			return errors.New("invalid stored task quota")
		}
		delta = target - stored.Quota
		source := stored.PrivateData.BillingSource
		if source == "" {
			source = "wallet"
		}
		if delta != 0 {
			if source == "subscription" {
				if err := postConsumeUserSubscriptionDeltaTx(tx, stored.PrivateData.SubscriptionId, int64(delta)); err != nil {
					return err
				}
			} else {
				// Legacy/custom submitters may already have charged the task
				// without a source record. Preserve that charge as non-paid.
				spend := ReferralWalletSpend{SourceId: stored.PrivateData.ReferralSourceId, UserId: stored.UserId, Quota: int64(stored.Quota)}
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&spend).Error; err != nil {
					return err
				}
				if _, _, err := reconcileReferralWalletSpendTx(tx, stored.UserId, spend.SourceId, int64(target), false); err != nil {
					return err
				}
			}
			if err := tx.Model(&stored).Update("quota", target).Error; err != nil {
				return err
			}
		}
		if target <= stored.Quota {
			return refundReferralSpendTx(tx, source, stored.PrivateData.ReferralSourceId, int64(target))
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	task.Quota = target
	if delta != 0 && task.PrivateData.BillingSource != "subscription" {
		if err := cacheIncrUserQuota(task.UserId, -int64(delta)); err != nil {
			common.SysError(fmt.Sprintf("sync task wallet user=%d task=%s: %v", task.UserId, task.TaskID, err))
		}
	}
	return delta, nil
}
