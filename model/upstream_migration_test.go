package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUpstreamSchemaMigratesTwiceOnSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	for range 2 {
		require.NoError(t, db.AutoMigrate(&Upstream{}, &UpstreamAddress{}, &UpstreamSnapshot{}, &UpstreamConfig{}))
	}
	require.True(t, db.Migrator().HasTable(&Upstream{}))
	require.True(t, db.Migrator().HasTable(&UpstreamAddress{}))
	require.True(t, db.Migrator().HasTable(&UpstreamSnapshot{}))
}
