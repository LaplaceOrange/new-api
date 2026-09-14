package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const (
	UserCleanupRulesOptionKey = "UserCleanup.Rules"

	UserCleanupConditionGroup    = "group"
	UserCleanupConditionIdleTime = "idle_time"

	UserCleanupActionInclude = "include"
	UserCleanupActionIgnore  = "ignore"

	UserCleanupOperatorGT  = "gt"
	UserCleanupOperatorGTE = "gte"
	UserCleanupOperatorEQ  = "eq"
	UserCleanupOperatorLT  = "lt"
	UserCleanupOperatorLTE = "lte"

	UserCleanupUnitHour  = "hour"
	UserCleanupUnitDay   = "day"
	UserCleanupUnitMonth = "month"

	userCleanupScanBatchSize = 200
)

type UserCleanupRule struct {
	ID        string `json:"id"`
	Condition string `json:"condition"`
	Action    string `json:"action"`
	Group     string `json:"group,omitempty"`
	Operator  string `json:"operator,omitempty"`
	Duration  int64  `json:"duration,omitempty"`
	Unit      string `json:"unit,omitempty"`
}

type UserCleanupRulesConfig struct {
	Rules            []UserCleanupRule `json:"rules"`
	UnmatchedAction  string            `json:"unmatched_action"`
	SkippedRuleCount int               `json:"skipped_rule_count,omitempty"`
}

type UserCleanupCandidate struct {
	Id        int    `json:"id"`
	ScanID    string `json:"scan_id" gorm:"type:varchar(64);uniqueIndex:uk_cleanup_scan_user;index"`
	UserID    int    `json:"user_id" gorm:"uniqueIndex:uk_cleanup_scan_user;index"`
	CreatedAt int64  `json:"created_at" gorm:"bigint"`
}

type UserCleanupRecord struct {
	Id          int    `json:"id"`
	UserId      int    `json:"user_id" gorm:"index"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Status      int    `json:"status"`
	Group       string `json:"group" gorm:"column:group"`
	Quota       int    `json:"quota"`
	CleanedAt   int64  `json:"cleaned_at" gorm:"bigint;index"`
	OperatorId  int    `json:"operator_id"`
}

func DefaultUserCleanupRulesConfig() UserCleanupRulesConfig {
	return UserCleanupRulesConfig{
		Rules:           []UserCleanupRule{},
		UnmatchedAction: UserCleanupActionIgnore,
	}
}

func UserCleanupRulesConfigJSON(config UserCleanupRulesConfig) string {
	normalized := NormalizeUserCleanupRulesConfig(config)
	data, err := common.Marshal(normalized)
	if err != nil {
		return `{"rules":[],"unmatched_action":"ignore"}`
	}
	return string(data)
}

func NormalizeUserCleanupRulesConfig(config UserCleanupRulesConfig) UserCleanupRulesConfig {
	if config.Rules == nil {
		config.Rules = []UserCleanupRule{}
	}
	if config.UnmatchedAction != UserCleanupActionInclude {
		config.UnmatchedAction = UserCleanupActionIgnore
	}
	for i := range config.Rules {
		config.Rules[i].Condition = strings.TrimSpace(config.Rules[i].Condition)
		config.Rules[i].Action = strings.TrimSpace(config.Rules[i].Action)
		config.Rules[i].Group = strings.TrimSpace(config.Rules[i].Group)
		config.Rules[i].Operator = strings.TrimSpace(config.Rules[i].Operator)
		config.Rules[i].Unit = strings.TrimSpace(config.Rules[i].Unit)
		config.Rules[i].ID = strings.TrimSpace(config.Rules[i].ID)
	}
	return config
}

func ParseUserCleanupRulesConfig(raw string) (UserCleanupRulesConfig, error) {
	config := DefaultUserCleanupRulesConfig()
	if strings.TrimSpace(raw) == "" {
		return config, nil
	}
	if err := common.UnmarshalJsonStr(raw, &config); err != nil {
		return DefaultUserCleanupRulesConfig(), err
	}
	return NormalizeUserCleanupRulesConfig(config), nil
}

func GetUserCleanupRulesConfig() (UserCleanupRulesConfig, error) {
	common.OptionMapRWMutex.RLock()
	raw := ""
	if common.OptionMap != nil {
		raw = common.OptionMap[UserCleanupRulesOptionKey]
	}
	common.OptionMapRWMutex.RUnlock()
	return ParseUserCleanupRulesConfig(raw)
}

func SaveUserCleanupRulesConfig(config UserCleanupRulesConfig) error {
	normalized := NormalizeUserCleanupRulesConfig(config)
	if normalized.UnmatchedAction != UserCleanupActionInclude && normalized.UnmatchedAction != UserCleanupActionIgnore {
		return errors.New("invalid unmatched action")
	}
	return UpdateOption(UserCleanupRulesOptionKey, UserCleanupRulesConfigJSON(normalized))
}

func (rule UserCleanupRule) IsComplete() bool {
	if rule.Action != UserCleanupActionInclude && rule.Action != UserCleanupActionIgnore {
		return false
	}
	switch rule.Condition {
	case UserCleanupConditionGroup:
		return rule.Group != ""
	case UserCleanupConditionIdleTime:
		return isUserCleanupOperator(rule.Operator) && isUserCleanupUnit(rule.Unit) && rule.Duration > 0
	default:
		return false
	}
}

func isUserCleanupOperator(operator string) bool {
	switch operator {
	case UserCleanupOperatorGT, UserCleanupOperatorGTE, UserCleanupOperatorEQ, UserCleanupOperatorLT, UserCleanupOperatorLTE:
		return true
	default:
		return false
	}
}

func isUserCleanupUnit(unit string) bool {
	switch unit {
	case UserCleanupUnitHour, UserCleanupUnitDay, UserCleanupUnitMonth:
		return true
	default:
		return false
	}
}

func userCleanupUnitSeconds(unit string) int64 {
	switch unit {
	case UserCleanupUnitHour:
		return 3600
	case UserCleanupUnitDay:
		return 86400
	case UserCleanupUnitMonth:
		return 30 * 86400
	default:
		return 0
	}
}

func UserLastActivityAt(lastLoginAt int64, lastTokenAccessedAt int64) int64 {
	if lastTokenAccessedAt > lastLoginAt {
		return lastTokenAccessedAt
	}
	return lastLoginAt
}

func MatchIdleTime(lastActivityAt int64, now int64, operator string, duration int64, unit string) bool {
	if !isUserCleanupOperator(operator) || duration <= 0 {
		return false
	}
	unitSeconds := userCleanupUnitSeconds(unit)
	if unitSeconds <= 0 {
		return false
	}
	if lastActivityAt <= 0 {
		switch operator {
		case UserCleanupOperatorGT, UserCleanupOperatorGTE:
			return true
		default:
			return false
		}
	}
	idle := now - lastActivityAt
	if idle < 0 {
		idle = 0
	}
	threshold := duration * unitSeconds
	switch operator {
	case UserCleanupOperatorGT:
		return idle > threshold
	case UserCleanupOperatorGTE:
		return idle >= threshold
	case UserCleanupOperatorLT:
		return idle < threshold
	case UserCleanupOperatorLTE:
		return idle <= threshold
	case UserCleanupOperatorEQ:
		return idle/unitSeconds == duration
	default:
		return false
	}
}

func IncompleteUserCleanupRuleCount(config UserCleanupRulesConfig) int {
	skipped := 0
	for _, rule := range config.Rules {
		if !rule.IsComplete() {
			skipped++
		}
	}
	return skipped
}

func EvaluateUserCleanupAction(config UserCleanupRulesConfig, group string, lastActivityAt int64, now int64) string {
	normalized := NormalizeUserCleanupRulesConfig(config)
	for _, rule := range normalized.Rules {
		if !rule.IsComplete() {
			continue
		}
		matched := false
		switch rule.Condition {
		case UserCleanupConditionGroup:
			matched = group == rule.Group
		case UserCleanupConditionIdleTime:
			matched = MatchIdleTime(lastActivityAt, now, rule.Operator, rule.Duration, rule.Unit)
		}
		if matched {
			return rule.Action
		}
	}
	return normalized.UnmatchedAction
}

func CountEnabledCommonUsers() (int64, error) {
	var total int64
	err := DB.Model(&User{}).
		Where("deleted_at IS NULL").
		Where("status = ?", common.UserStatusEnabled).
		Where("role < ?", common.RoleAdminUser).
		Count(&total).Error
	return total, err
}

func ListEnabledCommonUsersAfterID(afterID int, limit int) ([]User, error) {
	if limit <= 0 {
		limit = userCleanupScanBatchSize
	}
	var users []User
	err := DB.Model(&User{}).
		Where("deleted_at IS NULL").
		Where("status = ?", common.UserStatusEnabled).
		Where("role < ?", common.RoleAdminUser).
		Where("id > ?", afterID).
		Order("id asc").
		Limit(limit).
		Omit("password", "access_token").
		Find(&users).Error
	return users, err
}

func MaxTokenAccessedTimeByUserIDs(userIDs []int) (map[int]int64, error) {
	result := make(map[int]int64, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	type row struct {
		UserId       int   `gorm:"column:user_id"`
		AccessedTime int64 `gorm:"column:accessed_time"`
	}
	var rows []row
	err := DB.Model(&Token{}).
		Select("user_id, MAX(accessed_time) as accessed_time").
		Where("user_id IN ?", userIDs).
		Group("user_id").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, item := range rows {
		result[item.UserId] = item.AccessedTime
	}
	return result, nil
}

func ReplaceUserCleanupCandidates(scanID string, userIDs []int) error {
	if scanID == "" {
		return errors.New("scan id is required")
	}
	now := common.GetTimestamp()
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("scan_id = ?", scanID).Delete(&UserCleanupCandidate{}).Error; err != nil {
			return err
		}
		if len(userIDs) == 0 {
			return nil
		}
		rows := make([]UserCleanupCandidate, 0, len(userIDs))
		for _, userID := range userIDs {
			rows = append(rows, UserCleanupCandidate{
				ScanID:    scanID,
				UserID:    userID,
				CreatedAt: now,
			})
		}
		return tx.Create(&rows).Error
	})
}

func AppendUserCleanupCandidates(scanID string, userIDs []int) error {
	if scanID == "" {
		return errors.New("scan id is required")
	}
	if len(userIDs) == 0 {
		return nil
	}
	now := common.GetTimestamp()
	rows := make([]UserCleanupCandidate, 0, len(userIDs))
	for _, userID := range userIDs {
		rows = append(rows, UserCleanupCandidate{
			ScanID:    scanID,
			UserID:    userID,
			CreatedAt: now,
		})
	}
	return DB.Create(&rows).Error
}

func ClearUserCleanupCandidates() error {
	return DB.Where("1 = 1").Delete(&UserCleanupCandidate{}).Error
}

func ListUserCleanupCandidateIDs(scanID string) ([]int, error) {
	var ids []int
	err := DB.Model(&UserCleanupCandidate{}).
		Where("scan_id = ?", scanID).
		Order("user_id asc").
		Pluck("user_id", &ids).Error
	return ids, err
}

func ListUserCleanupCandidateUsers(scanID string, startIdx int, pageSize int) ([]*User, int64, error) {
	var total int64
	if err := DB.Model(&UserCleanupCandidate{}).Where("scan_id = ?", scanID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var ids []int
	if err := DB.Model(&UserCleanupCandidate{}).
		Where("scan_id = ?", scanID).
		Order("user_id asc").
		Limit(pageSize).
		Offset(startIdx).
		Pluck("user_id", &ids).Error; err != nil {
		return nil, 0, err
	}
	if len(ids) == 0 {
		return []*User{}, total, nil
	}
	var users []*User
	if err := DB.Omit("password", "access_token").Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	byID := make(map[int]*User, len(users))
	for _, user := range users {
		byID[user.Id] = user
	}
	ordered := make([]*User, 0, len(ids))
	for _, id := range ids {
		if user, ok := byID[id]; ok {
			ordered = append(ordered, user)
		}
	}
	return ordered, total, nil
}

func FilterUserCleanupCandidateIDs(scanID string, userIDs []int) ([]int, error) {
	if scanID == "" || len(userIDs) == 0 {
		return []int{}, nil
	}
	var ids []int
	err := DB.Model(&UserCleanupCandidate{}).
		Where("scan_id = ? AND user_id IN ?", scanID, userIDs).
		Pluck("user_id", &ids).Error
	return ids, err
}

func InsertUserCleanupRecords(records []UserCleanupRecord) error {
	if len(records) == 0 {
		return nil
	}
	return DB.Create(&records).Error
}

func ListUserCleanupRecords(startIdx int, pageSize int) ([]UserCleanupRecord, int64, error) {
	var total int64
	if err := DB.Model(&UserCleanupRecord{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var records []UserCleanupRecord
	err := DB.Order("cleaned_at desc").Order("id desc").Limit(pageSize).Offset(startIdx).Find(&records).Error
	return records, total, err
}

func DisableUsersForCleanup(userIDs []int) ([]User, error) {
	disabled := make([]User, 0, len(userIDs))
	for _, userID := range userIDs {
		user, err := GetUserById(userID, false)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return disabled, err
		}
		if user.Role >= common.RoleAdminUser || user.Status != common.UserStatusEnabled {
			continue
		}
		user.Status = common.UserStatusDisabled
		if err := user.Update(false); err != nil {
			return disabled, err
		}
		if err := InvalidateUserTokensCache(user.Id); err != nil {
			common.SysLog(fmt.Sprintf("failed to invalidate tokens cache for user %d: %s", user.Id, err.Error()))
		}
		disabled = append(disabled, *user)
	}
	return disabled, nil
}

func UserCleanupScanBatchSize() int {
	return userCleanupScanBatchSize
}
