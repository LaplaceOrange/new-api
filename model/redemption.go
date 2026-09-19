package model

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"gorm.io/gorm"
)

type Redemption struct {
	Id             int            `json:"id"`
	UserId         int            `json:"user_id"`
	Key            string         `json:"key" gorm:"type:char(32);uniqueIndex"`
	Status         int            `json:"status" gorm:"default:1"`
	Name           string         `json:"name" gorm:"index"`
	Quota          int            `json:"quota" gorm:"default:100"`
	CreatedTime    int64          `json:"created_time" gorm:"bigint"`
	RedeemedTime   int64          `json:"redeemed_time" gorm:"bigint"`
	Count          int            `json:"count" gorm:"-:all"` // only for api request
	UsedUserId     int            `json:"used_user_id"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`
	ExpiredTime    int64          `json:"expired_time" gorm:"bigint"` // 过期时间，0 表示不过期
	PlanId         int            `json:"plan_id" gorm:"type:int;default:0;index"`
	PlanTitle      string         `json:"plan_title" gorm:"-"`
	MaxUses        int            `json:"max_uses" gorm:"type:int;default:1"`
	MaxUsesPerUser int            `json:"max_uses_per_user" gorm:"type:int;default:1"`
	UsedCount      int            `json:"used_count" gorm:"type:int;default:0"`
}

type RedemptionUsage struct {
	Id            int   `json:"id"`
	RedemptionId  int   `json:"redemption_id" gorm:"uniqueIndex:uk_redemption_user;index"`
	UserId        int   `json:"user_id" gorm:"uniqueIndex:uk_redemption_user;index"`
	UsedCount     int   `json:"used_count" gorm:"type:int;default:0"`
	FirstUsedTime int64 `json:"first_used_time" gorm:"bigint"`
	LastUsedTime  int64 `json:"last_used_time" gorm:"bigint"`
}

type RedeemResult struct {
	Quota     int    `json:"quota"`
	PlanId    int    `json:"plan_id"`
	PlanTitle string `json:"plan_title,omitempty"`
	id        int
}

func GetAllRedemptions(startIdx int, num int) (redemptions []*Redemption, total int64, err error) {
	// 开始事务
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 获取总数
	err = tx.Model(&Redemption{}).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// 获取分页数据
	err = tx.Order("id desc").Limit(num).Offset(startIdx).Find(&redemptions).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// 提交事务
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	attachRedemptionPlanTitles(redemptions)
	return redemptions, total, nil
}

func SearchRedemptions(keyword string, status string, startIdx int, num int) (redemptions []*Redemption, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&Redemption{})

	if keyword != "" {
		if id, err := strconv.Atoi(keyword); err == nil {
			query = query.Where("id = ? OR name LIKE ?", id, keyword+"%")
		} else {
			query = query.Where("name LIKE ?", keyword+"%")
		}
	}

	if status != "" {
		now := common.GetTimestamp()
		switch status {
		case "expired":
			query = query.Where(
				"status = ? AND expired_time != 0 AND expired_time < ?",
				common.RedemptionCodeStatusEnabled,
				now,
			)
		case strconv.Itoa(common.RedemptionCodeStatusEnabled):
			query = query.Where(
				"status = ? AND (expired_time = 0 OR expired_time >= ?)",
				common.RedemptionCodeStatusEnabled,
				now,
			)
		case strconv.Itoa(common.RedemptionCodeStatusDisabled):
			query = query.Where("status = ?", common.RedemptionCodeStatusDisabled)
		case strconv.Itoa(common.RedemptionCodeStatusUsed):
			query = query.Where("status = ?", common.RedemptionCodeStatusUsed)
		}
	}

	// Get total count
	err = query.Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Get paginated data
	err = query.Order("id desc").Limit(num).Offset(startIdx).Find(&redemptions).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	attachRedemptionPlanTitles(redemptions)
	return redemptions, total, nil
}

func GetRedemptionById(id int) (*Redemption, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	redemption := Redemption{Id: id}
	var err error = nil
	err = DB.First(&redemption, "id = ?", id).Error
	if err != nil {
		return &redemption, err
	}
	attachRedemptionPlanTitles([]*Redemption{&redemption})
	return &redemption, nil
}

func Redeem(key string, userId int) (*RedeemResult, error) {
	if key == "" {
		return nil, errors.New("未提供兑换码")
	}
	if userId == 0 {
		return nil, errors.New("无效的 user id")
	}

	result := &RedeemResult{}
	groupChanged := false

	keyCol := "`key`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		keyCol = `"key"`
	}
	common.RandomSleep()
	err := DB.Transaction(func(tx *gorm.DB) error {
		redemption := &Redemption{}
		if err := lockForUpdate(tx).Where(keyCol+" = ?", key).First(redemption).Error; err != nil {
			return errors.New("无效的兑换码")
		}
		if redemption.Status != common.RedemptionCodeStatusEnabled {
			return errors.New("该兑换码已被使用")
		}
		if redemption.ExpiredTime != 0 && redemption.ExpiredTime < common.GetTimestamp() {
			return errors.New("该兑换码已过期")
		}
		if redemption.MaxUses > 0 && redemption.UsedCount >= redemption.MaxUses {
			return errors.New("该兑换码已被使用")
		}
		if (redemption.Quota > 0) == (redemption.PlanId > 0) {
			return ErrRedemptionRewardInvalid
		}

		var plan *SubscriptionPlan
		if redemption.PlanId > 0 {
			loaded, err := getSubscriptionPlanByIdTx(tx, redemption.PlanId)
			if err != nil {
				return ErrRedemptionInvalidPlan
			}
			plan = loaded
			var userRow User
			if err := lockForUpdate(tx).Select("id").Where("id = ?", userId).First(&userRow).Error; err != nil {
				return err
			}
		}

		now := common.GetTimestamp()
		var usage RedemptionUsage
		err := tx.Where("redemption_id = ? AND user_id = ?", redemption.Id, userId).First(&usage).Error
		if err == nil {
			if redemption.MaxUsesPerUser > 0 && usage.UsedCount >= redemption.MaxUsesPerUser {
				return errors.New("该兑换码已被使用")
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if redemption.Quota > 0 {
			if _, err := creditTopUpQuota(tx, &TopUp{UserId: userId}, redemption.Quota, nil); err != nil {
				return err
			}
		}
		if plan != nil {
			subscription, err := CreateUserSubscriptionFromPlanTx(tx, userId, plan, "redemption")
			if err != nil {
				return err
			}
			if subscription != nil && subscription.PrevUserGroup != "" {
				groupChanged = true
			}
			result.PlanId = plan.Id
			result.PlanTitle = plan.Title
		}

		claimed := tx.Model(&Redemption{}).
			Where("id = ? AND status = ? AND (max_uses = 0 OR used_count < max_uses)", redemption.Id, common.RedemptionCodeStatusEnabled).
			Updates(map[string]any{
				"used_count":    gorm.Expr("used_count + 1"),
				"redeemed_time": now,
				"used_user_id":  userId,
				"status":        gorm.Expr("CASE WHEN max_uses > 0 AND used_count + 1 >= max_uses THEN ? ELSE status END", common.RedemptionCodeStatusUsed),
			})
		if claimed.Error != nil {
			return claimed.Error
		}
		if claimed.RowsAffected == 0 {
			return errors.New("该兑换码已被使用")
		}

		if err == nil {
			usageResult := tx.Model(&RedemptionUsage{}).
				Where("id = ? AND (? = 0 OR used_count < ?)", usage.Id, redemption.MaxUsesPerUser, redemption.MaxUsesPerUser).
				Updates(map[string]any{
					"used_count":     gorm.Expr("used_count + 1"),
					"last_used_time": now,
				})
			if usageResult.Error != nil {
				return usageResult.Error
			}
			if usageResult.RowsAffected == 0 {
				return errors.New("该兑换码已被使用")
			}
		} else if err := tx.Create(&RedemptionUsage{
			RedemptionId:  redemption.Id,
			UserId:        userId,
			UsedCount:     1,
			FirstUsedTime: now,
			LastUsedTime:  now,
		}).Error; err != nil {
			return err
		}
		result.Quota = redemption.Quota
		result.id = redemption.Id
		return nil
	})
	if err != nil {
		common.SysError("redemption failed: " + err.Error())
		return nil, ErrRedeemFailed
	}
	if result.Quota > 0 {
		syncCreditUserQuotaCache(userId, result.Quota, "redemption")
		RecordLog(userId, LogTypeTopup, fmt.Sprintf("通过兑换码充值 %s，兑换码ID %d", logger.LogQuota(result.Quota), result.id))
	}
	if result.PlanId > 0 {
		if groupChanged {
			refreshSubscriptionUserGroupCache(userId, "redemption")
		}
		RecordLog(userId, LogTypeTopup, fmt.Sprintf("通过兑换码兑换订阅 %s，兑换码ID %d", result.PlanTitle, result.id))
	}
	return result, nil
}

func (redemption *Redemption) Insert() error {
	if err := redemption.ValidateConfig(); err != nil {
		return err
	}
	if redemption.Status == 0 {
		redemption.Status = common.RedemptionCodeStatusEnabled
	}
	return DB.Model(&Redemption{}).Create(map[string]any{
		"user_id":           redemption.UserId,
		"key":               redemption.Key,
		"status":            redemption.Status,
		"name":              redemption.Name,
		"quota":             redemption.Quota,
		"created_time":      redemption.CreatedTime,
		"redeemed_time":     redemption.RedeemedTime,
		"used_user_id":      redemption.UsedUserId,
		"expired_time":      redemption.ExpiredTime,
		"plan_id":           redemption.PlanId,
		"max_uses":          redemption.MaxUses,
		"max_uses_per_user": redemption.MaxUsesPerUser,
		"used_count":        redemption.UsedCount,
	}).Error
}

func (redemption *Redemption) SelectUpdate() error {
	// This can update zero values
	return DB.Model(redemption).Select("redeemed_time", "status").Updates(redemption).Error
}

// Update Make sure your token's fields is completed, because this will update non-zero values
func (redemption *Redemption) Update() error {
	if err := redemption.ValidateConfig(); err != nil {
		return err
	}
	if redemption.MaxUses > 0 && redemption.UsedCount >= redemption.MaxUses {
		redemption.Status = common.RedemptionCodeStatusUsed
	}
	return DB.Model(redemption).Select(
		"name",
		"status",
		"quota",
		"redeemed_time",
		"expired_time",
		"plan_id",
		"max_uses",
		"max_uses_per_user",
	).Updates(redemption).Error
}

func (redemption *Redemption) Delete() error {
	var err error
	err = DB.Delete(redemption).Error
	return err
}

func DeleteRedemptionById(id int) (err error) {
	if id == 0 {
		return errors.New("id 为空！")
	}
	redemption := Redemption{Id: id}
	err = DB.Where(redemption).First(&redemption).Error
	if err != nil {
		return err
	}
	return redemption.Delete()
}

func DeleteInvalidRedemptions() (int64, error) {
	now := common.GetTimestamp()
	result := DB.Where("status IN ? OR (status = ? AND expired_time != 0 AND expired_time < ?)", []int{common.RedemptionCodeStatusUsed, common.RedemptionCodeStatusDisabled}, common.RedemptionCodeStatusEnabled, now).Delete(&Redemption{})
	return result.RowsAffected, result.Error
}

// BatchDeleteRedemptions soft-deletes the selected codes in one statement.
func BatchDeleteRedemptions(ids []int) (int64, error) {
	if len(ids) == 0 || len(ids) > 1000 {
		return 0, errors.New("select between 1 and 1000 redemption codes")
	}
	for _, id := range ids {
		if id <= 0 {
			return 0, errors.New("redemption IDs must be positive")
		}
	}
	result := DB.Where("id IN ?", ids).Delete(&Redemption{})
	return result.RowsAffected, result.Error
}

func (redemption *Redemption) ValidateConfig() error {
	if redemption == nil {
		return ErrRedemptionRewardInvalid
	}
	if redemption.MaxUses < 0 || redemption.MaxUsesPerUser < 0 {
		return ErrRedemptionUsageLimitInvalid
	}
	if (redemption.Quota > 0) == (redemption.PlanId > 0) {
		return ErrRedemptionRewardInvalid
	}
	if redemption.Quota > 0 {
		if err := common.ValidateWalletQuota(redemption.Quota); err != nil {
			return err
		}
	}
	if redemption.PlanId > 0 {
		if _, err := GetSubscriptionPlanById(redemption.PlanId); err != nil {
			return ErrRedemptionInvalidPlan
		}
	}
	return nil
}

func attachRedemptionPlanTitles(redemptions []*Redemption) {
	if len(redemptions) == 0 {
		return
	}
	ids := make([]int, 0)
	seen := make(map[int]struct{})
	for _, redemption := range redemptions {
		if redemption == nil || redemption.PlanId <= 0 {
			continue
		}
		if _, ok := seen[redemption.PlanId]; ok {
			continue
		}
		seen[redemption.PlanId] = struct{}{}
		ids = append(ids, redemption.PlanId)
	}
	if len(ids) == 0 {
		return
	}
	var plans []SubscriptionPlan
	if err := DB.Select("id", "title").Where("id IN ?", ids).Find(&plans).Error; err != nil {
		return
	}
	titles := make(map[int]string, len(plans))
	for _, plan := range plans {
		titles[plan.Id] = plan.Title
	}
	for _, redemption := range redemptions {
		if redemption == nil {
			continue
		}
		redemption.PlanTitle = titles[redemption.PlanId]
	}
}

func backfillRedemptionUsedCount() error {
	return DB.Model(&Redemption{}).
		Where("status = ? AND used_count = 0 AND used_user_id != 0", common.RedemptionCodeStatusUsed).
		Update("used_count", 1).Error
}
