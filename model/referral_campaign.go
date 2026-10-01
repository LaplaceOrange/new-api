package model

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ReferralDirectionInviter = "invitee_to_inviter"
	ReferralDirectionInvitee = "inviter_to_invitee"
	ReferralDirectionBoth    = "both"
	ReferralSpendWallet      = "wallet"
	ReferralSpendAll         = "all"
	ReferralSpendPaidWallet  = "paid_wallet"
)

type ReferralCampaign struct {
	Id                 int64  `json:"id" gorm:"primaryKey"`
	Name               string `json:"name" gorm:"type:varchar(128);not null"`
	Enabled            bool   `json:"enabled" gorm:"not null"`
	Priority           int    `json:"priority" gorm:"not null"`
	StartAt            int64  `json:"start_at" gorm:"index;not null"`
	EndAt              int64  `json:"end_at" gorm:"index;not null"`
	Direction          string `json:"direction" gorm:"type:varchar(32);not null"`
	SpendBasis         string `json:"spend_basis" gorm:"type:varchar(32);not null"`
	RequiredFriends    int    `json:"required_friends" gorm:"not null"`
	PaidThresholdQuota int64  `json:"paid_threshold_quota" gorm:"type:bigint;not null"`
	BaseBps            int    `json:"base_bps" gorm:"not null"`
	InviterBps         int    `json:"inviter_bps" gorm:"not null"`
	InviteePoolBps     int    `json:"invitee_pool_bps" gorm:"not null"`
}

type ReferralPolicy struct {
	Id               int   `json:"-" gorm:"primaryKey;autoIncrement:false"`
	MinTransferQuota int64 `json:"min_transfer_quota" gorm:"type:bigint;not null"`
	BucketMinutes    int   `json:"bucket_minutes" gorm:"not null"`
}

func DefaultReferralPolicy() ReferralPolicy {
	return ReferralPolicy{Id: 1, MinTransferQuota: int64(common.QuotaPerUnit), BucketMinutes: 60}
}

func (c *ReferralCampaign) Validate() error {
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" || utf8.RuneCountInString(c.Name) > 128 || c.StartAt <= 0 || c.EndAt <= c.StartAt ||
		c.EndAt > 253402300799 || c.Priority < math.MinInt32 || c.Priority > math.MaxInt32 ||
		c.RequiredFriends < 1 || c.RequiredFriends > math.MaxInt32 ||
		c.PaidThresholdQuota < 0 || c.PaidThresholdQuota > common.MaxWalletQuota {
		return errors.New("invalid referral campaign fields")
	}
	switch c.Direction {
	case ReferralDirectionBoth, ReferralDirectionInvitee, ReferralDirectionInviter:
	default:
		return errors.New("invalid referral direction")
	}
	switch c.SpendBasis {
	case ReferralSpendWallet, ReferralSpendAll, ReferralSpendPaidWallet:
	default:
		return errors.New("invalid referral spend basis")
	}
	if c.BaseBps < 0 || c.BaseBps > 10_000 || c.InviterBps < 0 || c.InviterBps > 10_000 ||
		c.InviteePoolBps < 0 || c.InviteePoolBps > 10_000 {
		return errors.New("referral rates must be between 0 and 10000 basis points")
	}
	return nil
}

func SaveReferralCampaign(c *ReferralCampaign) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if c.Id == 0 {
		// Creation never activates a campaign implicitly.
		c.Enabled = false
		return DB.Create(c).Error
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var existing ReferralCampaign
		if err := lockForUpdate(tx).First(&existing, c.Id).Error; err != nil {
			return err
		}
		if err := tx.Model(&existing).Select("*").Updates(c).Error; err != nil {
			return err
		}
		now := ReferralNow()
		if !c.Enabled || now < c.StartAt || now >= c.EndAt {
			return nil
		}
		var inviters []int
		if err := tx.Model(&ReferralFriend{}).Where("campaign_id = ? AND paid_quota > ?", c.Id, c.PaidThresholdQuota).
			Group("inviter_id").Having("COUNT(*) >= ?", c.RequiredFriends).Order("inviter_id asc").Pluck("inviter_id", &inviters).Error; err != nil {
			return err
		}
		for _, inviterId := range inviters {
			if _, err := referralUnlocked(tx, c, inviterId, now); err != nil {
				return err
			}
		}
		return nil
	})
}

func ListReferralCampaigns() ([]ReferralCampaign, error) {
	campaigns := make([]ReferralCampaign, 0)
	err := DB.Order("priority desc, start_at desc, id desc").Find(&campaigns).Error
	return campaigns, err
}

func activeReferralCampaign(tx *gorm.DB, at int64) (*ReferralCampaign, error) {
	var campaign ReferralCampaign
	err := tx.Where("enabled = ? AND start_at <= ? AND end_at > ?", true, at, at).
		Order("priority desc, start_at desc, id desc").First(&campaign).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &campaign, nil
}

func GetReferralPolicy() (ReferralPolicy, error) {
	var policy ReferralPolicy
	err := DB.First(&policy, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return DefaultReferralPolicy(), nil
	}
	return policy, err
}

func SaveReferralPolicy(policy *ReferralPolicy) error {
	if policy.MinTransferQuota < 1 || policy.MinTransferQuota > common.MaxWalletQuota ||
		policy.BucketMinutes < 1 || policy.BucketMinutes > math.MaxInt32/60 {
		return errors.New("invalid referral transfer policy")
	}
	policy.Id = 1
	return DB.Save(policy).Error
}

type ReferralFriend struct {
	UserId     int   `json:"user_id" gorm:"primaryKey;autoIncrement:false"`
	InviterId  int   `json:"inviter_id" gorm:"index;not null"`
	CampaignId int64 `json:"campaign_id" gorm:"index;not null"`
	JoinedAt   int64 `json:"joined_at" gorm:"not null"`
	PaidQuota  int64 `json:"paid_quota" gorm:"type:bigint;not null"`
}

func RegisterReferralFriend(userId, inviterId int, joinedAt int64) error {
	if userId <= 0 || inviterId <= 0 || userId == inviterId {
		return nil
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		return registerReferralFriendTx(tx, userId, inviterId, joinedAt)
	})
}

func recordReferralRegistrationTx(tx *gorm.DB, user *User, inviterId int) error {
	if inviterId <= 0 || inviterId == user.Id {
		return nil
	}
	if err := tx.Model(&User{}).Where("id = ?", inviterId).
		Update("aff_count", gorm.Expr("aff_count + ?", 1)).Error; err != nil {
		return err
	}
	if operation_setting.IsPaymentComplianceConfirmed() {
		return registerReferralFriendTx(tx, user.Id, inviterId, user.CreatedAt)
	}
	return nil
}

func registerReferralFriendTx(tx *gorm.DB, userId, inviterId int, joinedAt int64) error {
	if userId <= 0 || inviterId <= 0 || userId == inviterId {
		return nil
	}
	var inviter User
	if err := tx.Select("id").First(&inviter, inviterId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	campaign, err := activeReferralCampaign(tx, joinedAt)
	if err != nil || campaign == nil {
		return err
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&ReferralFriend{
		UserId: userId, InviterId: inviterId, CampaignId: campaign.Id, JoinedAt: joinedAt,
	}).Error
}

// RecordReferralPaidQuota is called after a gateway has credited a real payment.
// The order identity prevents callback replay from counting twice.
type ReferralPaidOrder struct {
	OrderKey   string `gorm:"primaryKey;type:varchar(64)"`
	TradeNo    string `gorm:"type:varchar(255);not null"`
	UserId     int    `gorm:"not null"`
	CampaignId int64  `gorm:"not null"`
	Quota      int64  `gorm:"type:bigint;not null"`
	Recorded   bool   `gorm:"not null"`
}

func RecordReferralPaidQuota(tradeNo string, userId int, creditedQuota int, completedAt int64) error {
	if tradeNo == "" || creditedQuota <= 0 {
		return nil
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		return recordReferralPaidQuotaTx(tx, tradeNo, userId, creditedQuota, completedAt)
	})
}

func recordReferralPaidQuotaTx(tx *gorm.DB, tradeNo string, userId int, creditedQuota int, completedAt int64) error {
	var friend ReferralFriend
	if err := lockForUpdate(tx).Where("user_id = ?", userId).First(&friend).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	var campaign ReferralCampaign
	if err := tx.First(&campaign, friend.CampaignId).Error; err != nil {
		return err
	}
	if completedAt < campaign.StartAt || completedAt >= campaign.EndAt {
		return nil
	}
	order := ReferralPaidOrder{OrderKey: fmt.Sprintf("%x", sha256.Sum256([]byte(tradeNo))),
		TradeNo: tradeNo, UserId: userId, CampaignId: campaign.Id, Quota: int64(creditedQuota)}
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&order)
	if result.Error != nil {
		return result.Error
	}
	if err := lockForUpdate(tx).First(&order, "order_key = ?", order.OrderKey).Error; err != nil {
		return err
	}
	if order.Recorded {
		return nil
	}
	if order.UserId != userId || order.CampaignId != campaign.Id {
		return errors.New("referral payment owner mismatch")
	}
	if err := tx.Model(&order).Update("recorded", true).Error; err != nil {
		return err
	}
	if friend.PaidQuota > math.MaxInt64-int64(creditedQuota) {
		return errors.New("referral paid quota overflow")
	}
	if err := tx.Model(&friend).Update("paid_quota", friend.PaidQuota+int64(creditedQuota)).Error; err != nil {
		return err
	}
	_, err := referralUnlocked(tx, &campaign, friend.InviterId, completedAt)
	return err
}

type ReferralUnlock struct {
	InviterId  int   `gorm:"primaryKey;autoIncrement:false"`
	CampaignId int64 `gorm:"primaryKey;autoIncrement:false"`
	UnlockedAt int64 `gorm:"not null"`
}

func referralUnlocked(tx *gorm.DB, campaign *ReferralCampaign, inviterId int, at int64) (bool, error) {
	var unlock ReferralUnlock
	err := tx.Where("inviter_id = ? AND campaign_id = ?", inviterId, campaign.Id).First(&unlock).Error
	if err == nil {
		return true, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}
	var qualified int64
	if err := tx.Model(&ReferralFriend{}).Where("inviter_id = ? AND campaign_id = ? AND paid_quota > ?",
		inviterId, campaign.Id, campaign.PaidThresholdQuota).Count(&qualified).Error; err != nil {
		return false, err
	}
	if qualified < int64(campaign.RequiredFriends) {
		return false, nil
	}
	unlock = ReferralUnlock{InviterId: inviterId, CampaignId: campaign.Id, UnlockedAt: at}
	err = tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&unlock).Error
	return err == nil, err
}

type ReferralPaidWallet struct {
	UserId        int   `gorm:"primaryKey;autoIncrement:false"`
	PaidRemaining int64 `gorm:"type:bigint;not null"`
}

func creditReferralPaidWalletTx(tx *gorm.DB, userId, creditedQuota int) error {
	// The untracked pre-existing balance stays non-paid. Only successful online
	// top-ups after migration can increase this earmarked portion.
	account := ReferralPaidWallet{UserId: userId}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&account).Error; err != nil {
		return err
	}
	if err := lockForUpdate(tx).Where("user_id = ?", userId).First(&account).Error; err != nil {
		return err
	}
	var user User
	if err := tx.Select("quota").First(&user, userId).Error; err != nil {
		return err
	}
	// A payment that repays existing wallet debt cannot fund a future spend.
	creditedQuota = min(creditedQuota, max(0, user.Quota))
	if account.PaidRemaining > common.MaxWalletQuota-int64(creditedQuota) {
		return errors.New("paid wallet balance overflow")
	}
	return tx.Model(&account).Update("paid_remaining", account.PaidRemaining+int64(creditedQuota)).Error
}

type ReferralAccount struct {
	UserId         int   `json:"user_id" gorm:"primaryKey;autoIncrement:false"`
	Balance        int64 `json:"balance" gorm:"type:bigint;not null"`
	LifetimeEarned int64 `json:"lifetime_earned" gorm:"type:bigint;not null"`
}

type ReferralTransfer struct {
	Id             int64  `json:"id" gorm:"primaryKey"`
	UserId         int    `json:"user_id" gorm:"index;not null"`
	Amount         int64  `json:"amount" gorm:"type:bigint;not null"`
	LegacyPart     int64  `json:"legacy_part" gorm:"type:bigint;not null"`
	CreatedAt      int64  `json:"created_at" gorm:"index;not null"`
	SourceSnapshot string `json:"source_snapshot" gorm:"type:text;not null"`
}

type ReferralTransferSource struct {
	LedgerId   int64  `json:"ledger_id,omitempty"`
	CampaignId int64  `json:"campaign_id,omitempty"`
	SourceKind string `json:"source_kind"`
	SourceId   string `json:"source_id,omitempty"`
	Amount     int64  `json:"amount"`
}

type ReferralLedger struct {
	Id               int64  `json:"id" gorm:"primaryKey"`
	UserId           int    `json:"user_id" gorm:"index;uniqueIndex:uk_referral_entry,priority:4;not null"`
	CampaignId       int64  `json:"campaign_id" gorm:"index;not null"`
	SourceKind       string `json:"source_kind" gorm:"type:varchar(32);uniqueIndex:uk_referral_entry,priority:1;not null"`
	SourceId         string `json:"source_id" gorm:"type:varchar(120);uniqueIndex:uk_referral_entry,priority:2;not null"`
	EntryKind        string `json:"entry_kind" gorm:"type:varchar(32);uniqueIndex:uk_referral_entry,priority:3;not null"`
	Amount           int64  `json:"amount" gorm:"type:bigint;not null"`
	SourceQuota      int64  `json:"source_quota" gorm:"type:bigint;not null"`
	RateBps          int    `json:"rate_bps" gorm:"not null"`
	PoolRecipients   int    `json:"pool_recipients" gorm:"not null"`
	PoolIndex        int    `json:"pool_index" gorm:"not null"`
	CreatedAt        int64  `json:"created_at" gorm:"index;not null"`
	CampaignSnapshot string `json:"campaign_snapshot" gorm:"type:text;not null"`
	Available        int64  `json:"-" gorm:"type:bigint;not null"`
}

type ReferralSettlement struct {
	SourceKind     string `gorm:"primaryKey;type:varchar(32)"`
	SourceId       string `gorm:"primaryKey;type:varchar(120)"`
	UserId         int    `gorm:"not null"`
	Quota          int64  `gorm:"type:bigint;not null"`
	RemainingQuota int64  `gorm:"type:bigint;not null"`
	CreatedAt      int64  `gorm:"not null"`
	PaidQuota      int64  `gorm:"type:bigint;not null"`
	PaidBasis      bool   `gorm:"not null"`
	Processed      bool   `gorm:"not null"`
}

func referralReward(quota int64, bps int) (int64, error) {
	if quota <= 0 || bps <= 0 {
		return 0, nil
	}
	amount := decimal.NewFromInt(quota).Mul(decimal.NewFromInt(int64(bps))).
		Div(decimal.NewFromInt(10_000)).Truncate(0)
	if amount.IsZero() {
		return 0, nil
	}
	if !amount.IsPositive() || !amount.IsInteger() || amount.GreaterThan(decimal.NewFromInt(common.MaxQuota)) {
		return 0, errors.New("referral reward outside single-request quota domain")
	}
	value, err := common.QuotaFromDecimalStrict(amount)
	return int64(value), err
}

// CreditReferralSpend uses one transaction for idempotency and all recipients.
// Each recipient is uniquely identified for a source and direction.
func CreditReferralSpend(sourceKind, sourceId string, spenderId int, quota, paidQuota int64, at int64) error {
	if sourceId == "" || spenderId <= 0 || quota <= 0 {
		return nil
	}
	if len(sourceId) > 120 || quota > common.MaxQuota || paidQuota < 0 || paidQuota > quota ||
		(sourceKind != "wallet" && sourceKind != "subscription") {
		return errors.New("referral source quota outside single-request limit")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		settlement := ReferralSettlement{SourceKind: sourceKind, SourceId: sourceId,
			UserId: spenderId, Quota: quota, RemainingQuota: quota, CreatedAt: at}
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&settlement)
		if result.Error != nil {
			return result.Error
		}
		if err := lockForUpdate(tx).First(&settlement, "source_kind = ? AND source_id = ?", sourceKind, sourceId).Error; err != nil {
			return err
		}
		if settlement.Processed {
			return nil
		}
		if settlement.UserId != spenderId {
			return errors.New("referral spend owner mismatch")
		}
		if err := tx.Model(&settlement).Update("processed", true).Error; err != nil {
			return err
		}
		campaign, err := activeReferralCampaign(tx, at)
		if err != nil || campaign == nil {
			return err
		}
		if campaign.SpendBasis == ReferralSpendWallet && sourceKind != "wallet" ||
			campaign.SpendBasis == ReferralSpendPaidWallet && sourceKind != "wallet" {
			return nil
		}
		if campaign.SpendBasis == ReferralSpendPaidWallet {
			quota = paidQuota
			if err := tx.Model(&settlement).Updates(map[string]any{"paid_quota": paidQuota, "paid_basis": true}).Error; err != nil {
				return err
			}
		}
		if quota <= 0 {
			return nil
		}
		snapshot, err := common.Marshal(campaign)
		if err != nil {
			return err
		}
		var friend ReferralFriend
		err = tx.Where("user_id = ? AND campaign_id = ?", spenderId, campaign.Id).First(&friend).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil && (campaign.Direction == ReferralDirectionInviter || campaign.Direction == ReferralDirectionBoth) {
			rate := campaign.BaseBps
			unlocked, err := referralUnlocked(tx, campaign, friend.InviterId, at)
			if err != nil {
				return err
			}
			if unlocked {
				rate = 0
				if friend.PaidQuota > campaign.PaidThresholdQuota {
					rate = campaign.InviterBps
				}
			}
			amount, err := referralReward(quota, rate)
			if err != nil {
				return err
			}
			if err := creditReferralEntry(tx, ReferralLedger{
				UserId: friend.InviterId, CampaignId: campaign.Id, SourceKind: sourceKind,
				SourceId: sourceId, EntryKind: "inviter", Amount: amount,
				SourceQuota: quota, RateBps: rate, CreatedAt: at, CampaignSnapshot: string(snapshot),
			}); err != nil {
				return err
			}
		}
		if campaign.Direction != ReferralDirectionInvitee && campaign.Direction != ReferralDirectionBoth {
			return nil
		}
		var friends []ReferralFriend
		if err := tx.Where("inviter_id = ? AND campaign_id = ?", spenderId, campaign.Id).
			Order("user_id asc").Find(&friends).Error; err != nil {
			return err
		}
		if len(friends) == 0 {
			return nil
		}
		qualified := make([]ReferralFriend, 0, len(friends))
		for _, candidate := range friends {
			if candidate.PaidQuota > campaign.PaidThresholdQuota {
				qualified = append(qualified, candidate)
			}
		}
		rate := campaign.BaseBps
		unlocked, err := referralUnlocked(tx, campaign, spenderId, at)
		if err != nil {
			return err
		}
		if unlocked {
			friends = qualified
			rate = campaign.InviteePoolBps
		}
		if len(friends) == 0 {
			return nil
		}
		pool, err := referralReward(quota, rate)
		if err != nil {
			return err
		}
		for i, recipient := range friends {
			amount := pool / int64(len(friends))
			if int64(i) < pool%int64(len(friends)) {
				amount++
			}
			if err := creditReferralEntry(tx, ReferralLedger{
				UserId: recipient.UserId, CampaignId: campaign.Id, SourceKind: sourceKind,
				SourceId: sourceId, EntryKind: "invitee", Amount: amount,
				SourceQuota: quota, RateBps: rate, CreatedAt: at, CampaignSnapshot: string(snapshot),
				PoolRecipients: len(friends),
				PoolIndex:      i,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func creditReferralEntry(tx *gorm.DB, entry ReferralLedger) error {
	if entry.Amount == 0 {
		return nil
	}
	result := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "source_kind"}, {Name: "source_id"}, {Name: "entry_kind"}, {Name: "user_id"}},
		DoNothing: true,
	}).Create(&entry)
	if result.Error != nil || result.RowsAffected == 0 {
		return result.Error
	}
	account := ReferralAccount{UserId: entry.UserId}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&account).Error; err != nil {
		return err
	}
	if err := lockForUpdate(tx).First(&account, "user_id = ?", entry.UserId).Error; err != nil {
		return err
	}
	if entry.Amount > 0 && (account.Balance > math.MaxInt64-entry.Amount ||
		account.LifetimeEarned > math.MaxInt64-entry.Amount) ||
		entry.Amount < 0 && account.Balance < math.MinInt64-entry.Amount {
		return errors.New("referral account overflow")
	}
	if entry.Amount > 0 {
		available := max(0, account.Balance+entry.Amount) - max(0, account.Balance)
		if err := tx.Model(&entry).Update("available", available).Error; err != nil {
			return err
		}
	}
	updates := map[string]any{"balance": account.Balance + entry.Amount}
	if entry.Amount > 0 {
		updates["lifetime_earned"] = account.LifetimeEarned + entry.Amount
	}
	return tx.Model(&account).Updates(updates).Error
}

// RefundReferralSpend reverses the original recipients and rates, never current
// campaign parameters. remainingQuota is a monotonically decreasing net charge.
func RefundReferralSpend(sourceKind, sourceId string, remainingQuota int64) error {
	if sourceId == "" {
		return nil
	}
	if remainingQuota < 0 {
		return errors.New("invalid referral refund")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		return refundReferralSpendTx(tx, sourceKind, sourceId, remainingQuota)
	})
}

func refundReferralSpendTx(tx *gorm.DB, sourceKind, sourceId string, remainingQuota int64) error {
	var settlement ReferralSettlement
	err := lockForUpdate(tx).Where("source_kind = ? AND source_id = ?", sourceKind, sourceId).First(&settlement).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if remainingQuota != 0 {
			return nil
		}
		tombstone := ReferralSettlement{SourceKind: sourceKind, SourceId: sourceId, Processed: true, CreatedAt: ReferralNow()}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&tombstone).Error; err != nil {
			return err
		}
		err = lockForUpdate(tx).Where("source_kind = ? AND source_id = ?", sourceKind, sourceId).First(&settlement).Error
	}
	if err != nil || remainingQuota >= settlement.RemainingQuota {
		return err
	}
	var entries []ReferralLedger
	if err := tx.Where("source_kind = ? AND source_id = ? AND amount > 0", sourceKind, sourceId).
		Order("user_id asc").Find(&entries).Error; err != nil {
		return err
	}
	for _, entry := range entries {
		basis := settlement.Quota
		oldRemaining, newRemaining := settlement.RemainingQuota, remainingQuota
		if settlement.PaidBasis {
			basis = settlement.PaidQuota
			oldRemaining = max(0, basis-(settlement.Quota-oldRemaining))
			newRemaining = max(0, basis-(settlement.Quota-newRemaining))
		}
		oldNet, err := referralReward(oldRemaining, entry.RateBps)
		if err != nil {
			return err
		}
		newNet, err := referralReward(newRemaining, entry.RateBps)
		if err != nil {
			return err
		}
		if entry.PoolRecipients > 0 {
			recipients := int64(entry.PoolRecipients)
			index := int64(entry.PoolIndex)
			oldPart, newPart := oldNet/recipients, newNet/recipients
			if index < oldNet%recipients {
				oldPart++
			}
			if index < newNet%recipients {
				newPart++
			}
			oldNet, newNet = oldPart, newPart
		}
		delta := newNet - oldNet
		refund := entry
		refund.Id = 0
		refund.Available = 0
		refund.Amount = delta
		refund.EntryKind = fmt.Sprintf("refund_%d", remainingQuota)
		refund.CreatedAt = ReferralNow()
		if err := creditReferralEntry(tx, refund); err != nil {
			return err
		}
		if err := lockForUpdate(tx).First(&entry, entry.Id).Error; err != nil {
			return err
		}
		if err := tx.Model(&entry).Update("available", max(0, entry.Available+delta)).Error; err != nil {
			return err
		}
	}
	return tx.Model(&settlement).Update("remaining_quota", remainingQuota).Error
}

func ReferralNow() int64 { return time.Now().Unix() }

func ReferralSourceId(requestId string, userId int) string {
	return fmt.Sprintf("%d:%s", userId, requestId)
}

type ReferralSummary struct {
	Balance          int64             `json:"balance"`
	LegacyBalance    int64             `json:"legacy_balance"`
	LifetimeEarned   int64             `json:"lifetime_earned"`
	MinTransferQuota int64             `json:"min_transfer_quota"`
	Campaign         *ReferralCampaign `json:"campaign"`
	InvitedCount     int64             `json:"invited_count"`
	QualifiedCount   int64             `json:"qualified_count"`
	Unlocked         bool              `json:"unlocked"`
}

func GetReferralSummary(userId int) (ReferralSummary, error) {
	summary := ReferralSummary{}
	policy, err := GetReferralPolicy()
	if err != nil {
		return summary, err
	}
	summary.MinTransferQuota = policy.MinTransferQuota
	var user User
	if err := DB.Select("id", "aff_quota", "aff_history").First(&user, userId).Error; err != nil {
		return summary, err
	}
	summary.LegacyBalance = int64(user.AffQuota)
	var account ReferralAccount
	if err := DB.First(&account, "user_id = ?", userId).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return summary, err
	}
	summary.Balance = summary.LegacyBalance + account.Balance
	summary.LifetimeEarned = int64(user.AffHistoryQuota) + account.LifetimeEarned
	summary.Campaign, err = activeReferralCampaign(DB, ReferralNow())
	if err != nil || summary.Campaign == nil {
		return summary, err
	}
	query := DB.Model(&ReferralFriend{}).Where("campaign_id = ? AND inviter_id = ?", summary.Campaign.Id, userId)
	if err := query.Count(&summary.InvitedCount).Error; err != nil {
		return summary, err
	}
	err = query.Where("paid_quota > ?", summary.Campaign.PaidThresholdQuota).Count(&summary.QualifiedCount).Error
	if err == nil {
		var count int64
		err = DB.Model(&ReferralUnlock{}).Where("inviter_id = ? AND campaign_id = ?", userId, summary.Campaign.Id).Count(&count).Error
		summary.Unlocked = count > 0 || summary.QualifiedCount >= int64(summary.Campaign.RequiredFriends)
	}
	return summary, err
}

func ListReferralLedger(userId, offset, limit int) ([]ReferralLedger, error) {
	entries := make([]ReferralLedger, 0)
	if limit < 1 || limit > 100 {
		limit = 20
	}
	err := DB.Where("user_id = ?", userId).Order("id desc").Offset(offset).Limit(limit).Find(&entries).Error
	return entries, err
}

func TransferReferralBalance(userId int, amount int64) (*ReferralTransfer, error) {
	if amount <= 0 || amount > common.MaxWalletQuota {
		return nil, errors.New("invalid referral transfer amount")
	}
	var transfer ReferralTransfer
	err := DB.Transaction(func(tx *gorm.DB) error {
		var policy ReferralPolicy
		if err := tx.First(&policy, 1).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			policy = DefaultReferralPolicy()
		}
		if amount < policy.MinTransferQuota {
			return fmt.Errorf("minimum referral transfer is %d quota", policy.MinTransferQuota)
		}
		var user User
		if err := lockForUpdate(tx).Select("id", "quota", "aff_quota").First(&user, userId).Error; err != nil {
			return err
		}
		account := ReferralAccount{UserId: userId}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&account).Error; err != nil {
			return err
		}
		if err := lockForUpdate(tx).First(&account, "user_id = ?", userId).Error; err != nil &&
			!errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if account.Balance > math.MaxInt64-int64(user.AffQuota) ||
			int64(user.AffQuota)+account.Balance < amount {
			return errors.New("insufficient referral balance")
		}
		if int64(user.Quota) > common.MaxWalletQuota-amount {
			return errors.New("user wallet would overflow")
		}
		legacy := min(int64(user.AffQuota), amount)
		if legacy > 0 {
			result := tx.Model(&User{}).Where("id = ? AND aff_quota >= ?", userId, legacy).
				Update("aff_quota", gorm.Expr("aff_quota - ?", legacy))
			if result.Error != nil || result.RowsAffected != 1 {
				return errors.New("legacy referral balance changed")
			}
		}
		remainder := amount - legacy
		if remainder > 0 {
			result := tx.Model(&ReferralAccount{}).Where("user_id = ? AND balance >= ?", userId, remainder).
				Update("balance", gorm.Expr("balance - ?", remainder))
			if result.Error != nil || result.RowsAffected != 1 {
				return errors.New("referral balance changed")
			}
		}
		result := tx.Model(&User{}).Where("id = ? AND quota <= ?", userId, common.MaxWalletQuota-amount).
			Update("quota", gorm.Expr("quota + ?", amount))
		if result.Error != nil || result.RowsAffected != 1 {
			return errors.New("wallet balance changed")
		}
		sources := make([]ReferralTransferSource, 0)
		if legacy > 0 {
			sources = append(sources, ReferralTransferSource{SourceKind: "legacy", Amount: legacy})
		}
		left := remainder
		var entries []ReferralLedger
		if left > 0 {
			if err := tx.Where("user_id = ? AND available > 0", userId).
				Order("id asc").Find(&entries).Error; err != nil {
				return err
			}
		}
		for _, entry := range entries {
			taken := min(left, entry.Available)
			if taken == 0 {
				break
			}
			sources = append(sources, ReferralTransferSource{LedgerId: entry.Id, CampaignId: entry.CampaignId,
				SourceKind: entry.SourceKind, SourceId: entry.SourceId, Amount: taken})
			if err := tx.Model(&entry).Update("available", entry.Available-taken).Error; err != nil {
				return err
			}
			left -= taken
		}
		if left > 0 {
			return errors.New("referral ledger reconciliation required")
		}
		snapshot, err := common.Marshal(sources)
		if err != nil {
			return err
		}
		transfer = ReferralTransfer{UserId: userId, Amount: amount, LegacyPart: legacy,
			CreatedAt: ReferralNow(), SourceSnapshot: string(snapshot)}
		return tx.Create(&transfer).Error
	})
	if err != nil {
		return nil, err
	}
	if err := cacheIncrUserQuota(userId, amount); err != nil {
		common.SysError(fmt.Sprintf("failed to update quota cache after referral transfer: user=%d err=%v", userId, err))
	}
	RecordLog(userId, LogTypeTopup, fmt.Sprintf("邀请返利划转 %s", logger.LogQuota(int(amount))))
	return &transfer, nil
}

func ListReferralTransfers(userId, offset, limit int) ([]ReferralTransfer, error) {
	transfers := make([]ReferralTransfer, 0)
	if limit < 1 || limit > 100 {
		limit = 20
	}
	err := DB.Where("user_id = ?", userId).Order("id desc").Offset(max(0, offset)).Limit(limit).Find(&transfers).Error
	return transfers, err
}

func GetReferralTransferTotal() (int64, error) {
	var total int64
	err := DB.Model(&ReferralTransfer{}).Select("COALESCE(SUM(amount), 0)").Scan(&total).Error
	return total, err
}

type ReferralTransferBucket struct {
	Bucket int64 `json:"bucket"`
	Amount int64 `json:"amount"`
	Count  int64 `json:"count"`
}

func GetReferralTransferBuckets(startAt, endAt int64) ([]ReferralTransferBucket, error) {
	policy, err := GetReferralPolicy()
	if err != nil {
		return nil, err
	}
	if startAt <= 0 || endAt <= startAt || endAt-startAt > 366*24*60*60 {
		return nil, errors.New("invalid referral report range")
	}
	bucketExpr := revenueBucketExpr("created_at", int64(policy.BucketMinutes)*60)
	buckets := make([]ReferralTransferBucket, 0)
	err = DB.Model(&ReferralTransfer{}).
		Select(bucketExpr+" AS bucket, COALESCE(SUM(amount), 0) AS amount, COUNT(*) AS count").
		Where("created_at >= ? AND created_at < ?", startAt, endAt).
		Group(bucketExpr).Order("bucket asc").Scan(&buckets).Error
	return buckets, err
}
