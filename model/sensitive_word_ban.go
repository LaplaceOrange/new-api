package model

import (
	"errors"
	"net/netip"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	SensitiveWordUserBanSource = "sensitive_word"
)

type SensitiveWordUserBan struct {
	Id           int64  `json:"id" gorm:"primaryKey"`
	UserId       int    `json:"user_id" gorm:"not null;uniqueIndex"`
	HitCount     int    `json:"hit_count" gorm:"not null;default:0"`
	FirstHitAt   int64  `json:"first_hit_at" gorm:"not null"`
	LastHitAt    int64  `json:"last_hit_at" gorm:"not null"`
	BannedAt     int64  `json:"banned_at" gorm:"not null;index"`
	AutoDisabled bool   `json:"auto_disabled"`
	BanSource    string `json:"ban_source" gorm:"type:varchar(32);not null"`
	CreatedAt    int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (SensitiveWordUserBan) TableName() string { return "sensitive_word_user_bans" }

type SensitiveWordIPBan struct {
	Id         int64  `json:"id" gorm:"primaryKey"`
	IP         string `json:"ip" gorm:"type:varchar(128);not null;uniqueIndex"`
	HitCount   int    `json:"hit_count" gorm:"not null;default:0"`
	FirstHitAt int64  `json:"first_hit_at" gorm:"not null"`
	LastHitAt  int64  `json:"last_hit_at" gorm:"not null"`
	BannedAt   int64  `json:"banned_at" gorm:"not null;index"`
	CreatedAt  int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (SensitiveWordIPBan) TableName() string { return "sensitive_word_ip_bans" }

// SensitiveWordUserIPBan records which IP contributed to an automatic user
// ban. A separate table preserves multiple IP associations for one user.
type SensitiveWordUserIPBan struct {
	Id        int64  `json:"id" gorm:"primaryKey"`
	UserId    int    `json:"user_id" gorm:"not null;uniqueIndex:idx_sensitive_word_user_ip"`
	IP        string `json:"ip" gorm:"type:varchar(128);not null;uniqueIndex:idx_sensitive_word_user_ip"`
	BannedAt  int64  `json:"banned_at" gorm:"not null"`
	CreatedAt int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (SensitiveWordUserIPBan) TableName() string { return "sensitive_word_user_ip_bans" }

func NormalizeSensitiveWordIP(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}
	if strings.HasPrefix(ip, "[") && strings.HasSuffix(ip, "]") {
		ip = strings.TrimSuffix(strings.TrimPrefix(ip, "["), "]")
	}
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return ""
	}
	return addr.Unmap().WithZone("").String()
}

type SensitiveWordHitResult struct {
	UserHitCount int
	IPHitCount   int
	UserBanned   bool
	IPBanned     bool
}

// RecordSensitiveWordHit increments both counters in one transaction. Row
// locks are used where supported; SQLite serializes the surrounding write
// transaction instead.
func RecordSensitiveWordHit(userID int, ip string, now int64) (SensitiveWordHitResult, error) {
	if now <= 0 {
		now = time.Now().Unix()
	}
	ip = NormalizeSensitiveWordIP(ip)
	if userID <= 0 && ip == "" {
		return SensitiveWordHitResult{}, errors.New("sensitive word hit has no identity")
	}

	result := SensitiveWordHitResult{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if userID > 0 {
			var currentUser User
			userErr := tx.Select("status").First(&currentUser, userID).Error
			if userErr != nil && !errors.Is(userErr, gorm.ErrRecordNotFound) {
				return userErr
			}
			userEnabled := userErr == gorm.ErrRecordNotFound || currentUser.Status == common.UserStatusEnabled
			ban := SensitiveWordUserBan{
				UserId: userID, HitCount: 1, FirstHitAt: now, LastHitAt: now,
				BanSource: SensitiveWordUserBanSource,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "user_id"}},
				DoUpdates: clause.Assignments(map[string]any{
					"hit_count":   gorm.Expr("hit_count + ?", 1),
					"last_hit_at": now,
					"ban_source":  SensitiveWordUserBanSource,
				}),
			}).Create(&ban).Error; err != nil {
				return err
			}
			if err := lockForUpdate(tx).Where("user_id = ?", userID).First(&ban).Error; err != nil {
				return err
			}
			result.UserHitCount = ban.HitCount
			result.UserBanned = ban.BannedAt > 0
			if !result.UserBanned && userEnabled && ban.HitCount >= setting.SensitiveWordUserBanLimit() {
				if err := tx.Model(&SensitiveWordUserBan{}).
					Where("user_id = ? AND banned_at = 0", userID).
					Update("banned_at", now).Error; err != nil {
					return err
				}
				ban.BannedAt = now
				result.UserBanned = true
			}
			if result.UserBanned && userEnabled && ip != "" {
				association := SensitiveWordUserIPBan{UserId: userID, IP: ip, BannedAt: ban.BannedAt}
				if err := tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "user_id"}, {Name: "ip"}},
					DoNothing: true,
				}).Create(&association).Error; err != nil {
					return err
				}
			}
		}

		if ip != "" {
			ban := SensitiveWordIPBan{IP: ip, HitCount: 1, FirstHitAt: now, LastHitAt: now}
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "ip"}},
				DoUpdates: clause.Assignments(map[string]any{
					"hit_count":   gorm.Expr("hit_count + ?", 1),
					"last_hit_at": now,
				}),
			}).Create(&ban).Error; err != nil {
				return err
			}
			if err := lockForUpdate(tx).Where("ip = ?", ip).First(&ban).Error; err != nil {
				return err
			}
			result.IPHitCount = ban.HitCount
			result.IPBanned = ban.BannedAt > 0
			if !result.IPBanned {
				var bannedUsers int64
				if err := tx.Model(&SensitiveWordUserIPBan{}).
					Where("ip = ?", ip).
					Distinct("user_id").
					Count(&bannedUsers).Error; err != nil {
					return err
				}
				if bannedUsers >= int64(setting.SensitiveWordIPUserBanLimit()) {
					if err := tx.Model(&SensitiveWordIPBan{}).
						Where("ip = ? AND banned_at = 0", ip).
						Update("banned_at", now).Error; err != nil {
						return err
					}
					result.IPBanned = true
				}
			}
		}
		return nil
	})
	return result, err
}

func IsSensitiveWordIPBanned(ip string) (bool, error) {
	ip = NormalizeSensitiveWordIP(ip)
	if ip == "" {
		return false, nil
	}
	var count int64
	err := DB.Model(&SensitiveWordIPBan{}).Where("ip = ? AND banned_at > 0", ip).Count(&count).Error
	return count > 0, err
}

func DisableUserForSensitiveWords(userID int) error {
	if userID <= 0 {
		return gorm.ErrRecordNotFound
	}
	var user User
	var previousAuthVersion int64
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).First(&user, userID).Error; err != nil {
			return err
		}
		if user.Status == common.UserStatusDisabled {
			return nil
		}
		previousAuthVersion = user.AuthVersion
		user.Status = common.UserStatusDisabled
		if err := user.UpdateWithTx(tx, false); err != nil {
			return err
		}
		return tx.Model(&SensitiveWordUserBan{}).
			Where("user_id = ? AND ban_source = ? AND banned_at > 0", userID, SensitiveWordUserBanSource).
			Update("auto_disabled", true).Error
	})
	if err != nil {
		return err
	}
	if user.Status != common.UserStatusDisabled || user.AuthVersion == previousAuthVersion {
		return nil
	}
	if err := updateUserCache(user); err != nil {
		return err
	}
	_, err = RevokeAllUserSessions(userID, "sensitive_word_ban")
	return err
}

func UnbanSensitiveWordUser(userID int) error {
	if userID <= 0 {
		return gorm.ErrRecordNotFound
	}
	var user User
	var previousAuthVersion int64
	if err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).First(&user, userID).Error; err != nil {
			return err
		}
		var sensitiveBan SensitiveWordUserBan
		if err := lockForUpdate(tx).
			Where("user_id = ? AND banned_at > 0 AND ban_source = ?", userID, SensitiveWordUserBanSource).
			First(&sensitiveBan).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&SensitiveWordUserBan{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&SensitiveWordUserIPBan{}).Error; err != nil {
			return err
		}
		if user.Status == common.UserStatusDisabled && sensitiveBan.AutoDisabled {
			previousAuthVersion = user.AuthVersion
			user.Status = common.UserStatusEnabled
			return user.UpdateWithTx(tx, false)
		}
		return nil
	}); err != nil {
		return err
	}
	if user.Status == common.UserStatusEnabled && user.AuthVersion > previousAuthVersion {
		if err := updateUserCache(user); err != nil {
			return err
		}
		_, err := RevokeAllUserSessions(userID, "sensitive_word_unban")
		return err
	}
	return nil
}

func UnbanSensitiveWordIP(ip string) error {
	ip = NormalizeSensitiveWordIP(ip)
	if ip == "" {
		return gorm.ErrRecordNotFound
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("ip = ?", ip).Delete(&SensitiveWordUserIPBan{}).Error; err != nil {
			return err
		}
		return tx.Where("ip = ?", ip).Delete(&SensitiveWordIPBan{}).Error
	})
}

func ListSensitiveWordUserBans(search string, offset, limit int) ([]SensitiveWordUserBan, int64, error) {
	query := DB.Model(&SensitiveWordUserBan{}).Where("banned_at > 0 AND auto_disabled = ? AND ban_source = ?", true, SensitiveWordUserBanSource)
	search = strings.TrimSpace(search)
	if search != "" {
		query = query.Where("user_id = ? OR user_id IN (SELECT id FROM users WHERE username LIKE ? ESCAPE '!')", common.String2Int(search), "%"+escapeSensitiveWordLike(search)+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []SensitiveWordUserBan
	err := query.Order("banned_at DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

func ListSensitiveWordIPBans(search string, offset, limit int) ([]SensitiveWordIPBan, int64, error) {
	query := DB.Model(&SensitiveWordIPBan{}).Where("banned_at > 0")
	search = strings.TrimSpace(search)
	if search != "" {
		query = query.Where("ip LIKE ? ESCAPE '!'", "%"+escapeSensitiveWordLike(search)+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []SensitiveWordIPBan
	err := query.Order("banned_at DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

func escapeSensitiveWordLike(value string) string {
	value = strings.ReplaceAll(value, "!", "!!")
	value = strings.ReplaceAll(value, `%`, `!%`)
	return strings.ReplaceAll(value, `_`, `!_`)
}
