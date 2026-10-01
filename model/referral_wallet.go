package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrReferralWalletInsufficient = errors.New("wallet quota insufficient")

// ReferralWalletSpend retains the funding split through reservation, settlement
// and refund, including asynchronous tasks that complete on another process.
type ReferralWalletSpend struct {
	SourceId  string `gorm:"primaryKey;type:varchar(120)"`
	UserId    int    `gorm:"index;not null"`
	Quota     int64  `gorm:"type:bigint;not null"`
	PaidQuota int64  `gorm:"type:bigint;not null"`
}

// SetReferralWalletSpend reconciles a source to a target net charge atomically.
// Old mixed balances are non-paid; only the tracked online top-up portion is
// paid. Refunds restore the paid portion before the non-paid portion.
func SetReferralWalletSpend(userId int, sourceId string, target int64, reserve bool) (int64, error) {
	if userId <= 0 || sourceId == "" || len(sourceId) > 120 || target < 0 || target > common.MaxQuota {
		return 0, errors.New("invalid wallet spend")
	}
	var delta, paid int64
	err := DB.Transaction(func(tx *gorm.DB) error {
		var err error
		paid, delta, err = reconcileReferralWalletSpendTx(tx, userId, sourceId, target, reserve)
		if err == nil && target == 0 {
			err = refundReferralSpendTx(tx, "wallet", sourceId, 0)
		}
		return err
	})
	if err != nil {
		return 0, err
	}
	if delta != 0 {
		if err := cacheIncrUserQuota(userId, -delta); err != nil {
			common.SysError(fmt.Sprintf("sync tracked wallet spend user=%d source=%s: %v", userId, sourceId, err))
		}
	}
	return paid, nil
}

func AdjustReferralWalletSpend(userId int, sourceId string, adjustment int) error {
	if sourceId == "" || len(sourceId) > 120 || adjustment < -common.MaxQuota || adjustment > common.MaxQuota {
		return errors.New("invalid wallet adjustment")
	}
	var delta int64
	err := DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := lockForUpdate(tx).Select("id").First(&user, userId).Error; err != nil {
			return err
		}
		spend := ReferralWalletSpend{SourceId: sourceId, UserId: userId}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&spend).Error; err != nil {
			return err
		}
		if err := lockForUpdate(tx).First(&spend, "source_id = ?", sourceId).Error; err != nil {
			return err
		}
		target := spend.Quota + int64(adjustment)
		if target < 0 || target > common.MaxQuota {
			return errors.New("wallet adjustment exceeds source charge")
		}
		var err error
		_, delta, err = reconcileReferralWalletSpendTx(tx, userId, sourceId, target, false)
		return err
	})
	if err == nil && delta != 0 {
		if cacheErr := cacheIncrUserQuota(userId, -delta); cacheErr != nil {
			common.SysError(fmt.Sprintf("sync wallet adjustment user=%d source=%s: %v", userId, sourceId, cacheErr))
		}
	}
	return err
}

func reconcileReferralWalletSpendTx(tx *gorm.DB, userId int, sourceId string, target int64, reserve bool) (int64, int64, error) {
	var user User
	if err := lockForUpdate(tx).Select("id", "quota").First(&user, userId).Error; err != nil {
		return 0, 0, err
	}
	if user.Quota > common.MaxWalletQuota || user.Quota < -common.MaxWalletQuota {
		return 0, 0, ErrWalletQuotaLimitExceeded
	}
	account := ReferralPaidWallet{UserId: userId}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&account).Error; err != nil {
		return 0, 0, err
	}
	if err := lockForUpdate(tx).First(&account, "user_id = ?", userId).Error; err != nil {
		return 0, 0, err
	}
	spend := ReferralWalletSpend{SourceId: sourceId, UserId: userId}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&spend).Error; err != nil {
		return 0, 0, err
	}
	if err := lockForUpdate(tx).First(&spend, "source_id = ?", sourceId).Error; err != nil {
		return 0, 0, err
	}
	if spend.UserId != userId || spend.Quota < 0 || spend.Quota > common.MaxQuota ||
		spend.PaidQuota < 0 || spend.PaidQuota > spend.Quota ||
		account.PaidRemaining < 0 || account.PaidRemaining > common.MaxWalletQuota {
		return 0, 0, errors.New("invalid wallet source state")
	}
	delta := target - spend.Quota
	paid := spend.PaidQuota
	if delta == 0 {
		return paid, 0, nil
	}
	if delta > 0 && reserve && int64(user.Quota) < delta {
		return 0, 0, ErrReferralWalletInsufficient
	}
	after := int64(user.Quota) - delta
	if after > common.MaxWalletQuota || after < -common.MaxWalletQuota {
		return 0, 0, ErrWalletQuotaLimitExceeded
	}
	if delta > 0 {
		nonPaid := max(0, int64(user.Quota)-account.PaidRemaining)
		paidDelta := min(account.PaidRemaining, max(0, delta-nonPaid))
		account.PaidRemaining -= paidDelta
		paid += paidDelta
	} else {
		paidRefund := min(paid, -delta)
		if account.PaidRemaining > common.MaxWalletQuota-paidRefund {
			return 0, 0, ErrWalletQuotaLimitExceeded
		}
		account.PaidRemaining += paidRefund
		paid -= paidRefund
	}
	result := tx.Model(&user).Where("quota = ?", user.Quota).Update("quota", after)
	if result.Error != nil {
		return 0, 0, result.Error
	}
	if result.RowsAffected != 1 {
		return 0, 0, errors.New("wallet changed concurrently")
	}
	if err := tx.Model(&account).Update("paid_remaining", account.PaidRemaining).Error; err != nil {
		return 0, 0, err
	}
	err := tx.Model(&spend).Updates(map[string]any{"quota": target, "paid_quota": paid}).Error
	return paid, delta, err
}

func GetReferralWalletPaidQuota(sourceId string) (int64, error) {
	var spend ReferralWalletSpend
	err := DB.First(&spend, "source_id = ?", sourceId).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	return spend.PaidQuota, err
}

// consumeReferralPaidWalletTx accounts for non-relay wallet debits (for example
// subscription purchases and administrator deductions). They do not earn rebates.
func consumeReferralPaidWalletTx(tx *gorm.DB, userId int, before, debit int64) error {
	if debit <= 0 {
		return nil
	}
	var account ReferralPaidWallet
	err := lockForUpdate(tx).First(&account, "user_id = ?", userId).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	nonPaid := max(0, before-account.PaidRemaining)
	paidDebit := min(account.PaidRemaining, max(0, debit-nonPaid))
	return tx.Model(&account).Update("paid_remaining", account.PaidRemaining-paidDebit).Error
}
