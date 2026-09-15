package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestMatchIdleTime(t *testing.T) {
	now := int64(1_000_000)
	day := int64(86400)
	t.Run("never active matches greater", func(t *testing.T) {
		assert.True(t, MatchIdleTime(0, now, UserCleanupOperatorGT, 1, UserCleanupUnitDay))
		assert.True(t, MatchIdleTime(0, now, UserCleanupOperatorGTE, 1, UserCleanupUnitDay))
		assert.False(t, MatchIdleTime(0, now, UserCleanupOperatorEQ, 1, UserCleanupUnitDay))
		assert.False(t, MatchIdleTime(0, now, UserCleanupOperatorLT, 1, UserCleanupUnitDay))
		assert.False(t, MatchIdleTime(0, now, UserCleanupOperatorLTE, 1, UserCleanupUnitDay))
	})
	t.Run("operators", func(t *testing.T) {
		last := now - 3*day
		assert.True(t, MatchIdleTime(last, now, UserCleanupOperatorGT, 2, UserCleanupUnitDay))
		assert.False(t, MatchIdleTime(last, now, UserCleanupOperatorGT, 3, UserCleanupUnitDay))
		assert.True(t, MatchIdleTime(last, now, UserCleanupOperatorGTE, 3, UserCleanupUnitDay))
		assert.True(t, MatchIdleTime(last, now, UserCleanupOperatorEQ, 3, UserCleanupUnitDay))
		assert.False(t, MatchIdleTime(last, now, UserCleanupOperatorEQ, 2, UserCleanupUnitDay))
		assert.True(t, MatchIdleTime(last, now, UserCleanupOperatorLT, 4, UserCleanupUnitDay))
		assert.True(t, MatchIdleTime(last, now, UserCleanupOperatorLTE, 3, UserCleanupUnitDay))
		assert.False(t, MatchIdleTime(last, now, UserCleanupOperatorLT, 3, UserCleanupUnitDay))
	})
	t.Run("hour unit equality uses floor", func(t *testing.T) {
		last := now - (2*3600 + 10)
		assert.True(t, MatchIdleTime(last, now, UserCleanupOperatorEQ, 2, UserCleanupUnitHour))
	})
}

func TestEvaluateUserCleanupAction(t *testing.T) {
	now := int64(1_000_000)
	config := UserCleanupRulesConfig{
		UnmatchedAction: UserCleanupActionIgnore,
		Rules: []UserCleanupRule{
			{Condition: UserCleanupConditionGroup, Group: "vip", Action: UserCleanupActionIgnore},
			{Condition: UserCleanupConditionIdleTime, Operator: UserCleanupOperatorGT, Duration: 7, Unit: UserCleanupUnitDay, Action: UserCleanupActionInclude},
			{Condition: UserCleanupConditionGroup, Action: UserCleanupActionInclude},
		},
	}
	assert.Equal(t, 1, IncompleteUserCleanupRuleCount(config))
	assert.Equal(t, UserCleanupActionIgnore, EvaluateUserCleanupAction(config, "vip", 0, now))
	assert.Equal(t, UserCleanupActionInclude, EvaluateUserCleanupAction(config, "default", 0, now))
	assert.Equal(t, UserCleanupActionIgnore, EvaluateUserCleanupAction(config, "default", now-3600, now))
	empty := UserCleanupRulesConfig{UnmatchedAction: UserCleanupActionInclude}
	assert.Equal(t, UserCleanupActionInclude, EvaluateUserCleanupAction(empty, "default", now, now))
}

func TestUserLastActivityAt(t *testing.T) {
	assert.EqualValues(t, 20, UserLastActivityAt(10, 20))
	assert.EqualValues(t, 10, UserLastActivityAt(10, 0))
	assert.EqualValues(t, 0, UserLastActivityAt(0, 0))
}

func setupUserCleanupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB := DB
	db, err := gorm.Open(sqlite.Open("file:user-cleanup?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	DB = db
	t.Cleanup(func() { DB = previousDB })
	require.NoError(t, db.AutoMigrate(&User{}, &Token{}, &UserCleanupCandidate{}, &UserCleanupRecord{}, &UserSession{}))
	return db
}

func TestUserCleanupScanAndDisable(t *testing.T) {
	db := setupUserCleanupTestDB(t)
	enabled := User{Username: "idle-user", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1, LastLoginAt: 1, AffCode: "idle-aff"}
	admin := User{Username: "admin-user", Password: "password", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1, AffCode: "admin-aff"}
	disabled := User{Username: "off-user", Password: "password", Role: common.RoleCommonUser, Status: common.UserStatusDisabled, Group: "default", AuthVersion: 1, AffCode: "off-aff"}
	require.NoError(t, db.Create(&enabled).Error)
	require.NoError(t, db.Create(&admin).Error)
	require.NoError(t, db.Create(&disabled).Error)
	require.NoError(t, db.Create(&Token{UserId: enabled.Id, Key: "token-idle", AccessedTime: 50}).Error)

	total, err := CountEnabledCommonUsers()
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)

	users, err := ListEnabledCommonUsersAfterID(0, 10)
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, enabled.Id, users[0].Id)

	accessed, err := MaxTokenAccessedTimeByUserIDs([]int{enabled.Id})
	require.NoError(t, err)
	assert.EqualValues(t, 50, accessed[enabled.Id])
	assert.EqualValues(t, 50, UserLastActivityAt(users[0].LastLoginAt, accessed[enabled.Id]))

	require.NoError(t, AppendUserCleanupCandidates("scan-1", []int{enabled.Id, admin.Id}))
	ids, err := FilterUserCleanupCandidateIDs("scan-1", []int{enabled.Id, admin.Id, 999})
	require.NoError(t, err)
	assert.ElementsMatch(t, []int{enabled.Id, admin.Id}, ids)

	disabledUsers, err := DisableUsersForCleanup([]int{enabled.Id, admin.Id})
	require.NoError(t, err)
	require.Len(t, disabledUsers, 1)
	assert.Equal(t, enabled.Id, disabledUsers[0].Id)
	assert.Equal(t, common.UserStatusDisabled, disabledUsers[0].Status)

	var reloaded User
	require.NoError(t, db.First(&reloaded, admin.Id).Error)
	assert.Equal(t, common.UserStatusEnabled, reloaded.Status)
}

func TestListUserCleanupCandidateUsersPages(t *testing.T) {
	db := setupUserCleanupTestDB(t)
	users := make([]User, 0, 3)
	for i := range 3 {
		user := User{
			Username: "page-user-" + string(rune('a'+i)), Password: "password",
			Role: common.RoleCommonUser, Status: common.UserStatusEnabled,
			Group: "default", AuthVersion: 1, AffCode: "page-aff-" + string(rune('a'+i)),
		}
		require.NoError(t, db.Create(&user).Error)
		users = append(users, user)
	}
	ids := []int{users[0].Id, users[1].Id, users[2].Id}
	require.NoError(t, AppendUserCleanupCandidates("scan-page", ids))

	page, total, err := ListUserCleanupCandidateUsers("scan-page", 1, 1)
	require.NoError(t, err)
	assert.EqualValues(t, 3, total)
	require.Len(t, page, 1)
	assert.Equal(t, ids[1], page[0].Id)
}

func TestListUserCleanupRecordsPages(t *testing.T) {
	db := setupUserCleanupTestDB(t)
	now := int64(1_700_000_000)
	for i := range 3 {
		require.NoError(t, db.Create(&UserCleanupRecord{
			UserId:    i + 1,
			Username:  "hist-user",
			Status:    common.UserStatusDisabled,
			Group:     "default",
			CleanedAt: now + int64(i),
		}).Error)
	}
	page, total, err := ListUserCleanupRecords(1, 1)
	require.NoError(t, err)
	assert.EqualValues(t, 3, total)
	require.Len(t, page, 1)
	assert.Equal(t, int64(now+1), page[0].CleanedAt)
}
