package model

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
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

type upstreamBeforeAccountLogin struct {
	ID                int    `gorm:"primaryKey"`
	Name              string `gorm:"type:varchar(128);not null"`
	PrimaryURL        string `gorm:"type:text"`
	UserAgent         string `gorm:"type:text"`
	AutoRefreshToken  bool
	AccessCipher      string `gorm:"type:text"`
	RefreshCipher     string `gorm:"type:text"`
	RefreshPending    bool
	CredentialBlocked bool
	Balance           *float64
	BalanceUpdatedAt  int64  `gorm:"bigint"`
	LastAttemptAt     int64  `gorm:"bigint"`
	LastError         string `gorm:"type:text"`
	LeaseOwner        string `gorm:"type:varchar(64)"`
	LeaseUntil        int64  `gorm:"bigint"`
	Revision          int64  `gorm:"bigint"`
	CreatedAt         int64  `gorm:"bigint"`
	UpdatedAt         int64  `gorm:"bigint"`
}

func TestUpstreamAccountSchemaAndCredentials(t *testing.T) {
	t.Setenv("CRYPTO_SECRET", "01234567890123456789012345678901")
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			var dsn string
			switch dialect {
			case "sqlite":
				dsn = "local"
				previous := common.SQLitePath
				common.SQLitePath = filepath.Join(t.TempDir(), "upstreams.db")
				t.Cleanup(func() { common.SQLitePath = previous })
			case "mysql":
				dsn = os.Getenv("TEST_MYSQL_DSN")
			case "postgres":
				dsn = os.Getenv("TEST_POSTGRES_DSN")
			}
			if dsn == "" {
				t.Skip("test database DSN is not configured")
			}
			t.Setenv("UPSTREAM_TEST_DSN", dsn)
			db, dbType, err := chooseDB("UPSTREAM_TEST_DSN", false)
			require.NoError(t, err)
			naming := db.Config.NamingStrategy
			db.Config.NamingStrategy = schema.NamingStrategy{TablePrefix: "upstream_account_test_"}
			previousDB, previousType := DB, common.MainDatabaseType()
			DB = db
			common.SetMainDatabaseType(dbType)
			t.Cleanup(func() {
				DB = previousDB
				common.SetMainDatabaseType(previousType)
				db.Config.NamingStrategy = naming
				sqlDB, err := db.DB()
				require.NoError(t, err)
				require.NoError(t, sqlDB.Close())
			})
			for _, state := range []string{"fresh", "upgrade"} {
				t.Run(state, func(t *testing.T) {
					// The DSNs must refer to isolated test databases, never production.
					require.NoError(t, db.Migrator().DropTable(&UpstreamSnapshot{}, &UpstreamAddress{}, &Upstream{}))
					t.Cleanup(func() {
						require.NoError(t, db.Migrator().DropTable(&UpstreamSnapshot{}, &UpstreamAddress{}, &Upstream{}))
					})
					if state == "upgrade" {
						require.NoError(t, db.Table("upstream_account_test_upstreams").AutoMigrate(&upstreamBeforeAccountLogin{}))
						require.NoError(t, db.AutoMigrate(&UpstreamAddress{}, &UpstreamSnapshot{}))
						require.NoError(t, db.Table("upstream_account_test_upstreams").Create(&upstreamBeforeAccountLogin{
							ID: 999, Name: "legacy", AccessCipher: "old-jwt-cipher",
						}).Error)
					}
					require.NoError(t, db.AutoMigrate(&Upstream{}, &UpstreamAddress{}, &UpstreamSnapshot{}))
					recorder := &migrationSQLRecorder{}
					require.NoError(t, db.Session(&gorm.Session{Logger: recorder}).AutoMigrate(&Upstream{}, &UpstreamAddress{}, &UpstreamSnapshot{}))
					assert.Empty(t, recorder.schemaMutations(), "second startup must not alter the schema")
					if state == "upgrade" {
						legacy, err := GetUpstream(999)
						require.NoError(t, err)
						assert.Equal(t, UpstreamAuthJWT, legacy.AuthMode)
						assert.Equal(t, "old-jwt-cipher", legacy.AccessCipher)
					}
					email, password := "account@example.com", "  exact password  "
					item := &Upstream{Name: "password", PrimaryURL: "https://upstream.example", Addresses: []string{"https://upstream.example"}, AuthMode: UpstreamAuthPassword}
					require.NoError(t, SaveUpstream(item, UpstreamCredentials{AccountEmail: &email, AccountPassword: &password}))
					saved, err := GetUpstream(item.ID)
					require.NoError(t, err)
					assert.NotContains(t, saved.AccountCipher, email)
					assert.NotContains(t, saved.AccountCipher, password)
					plain, err := common.DecryptUpstreamCredential(saved.AccountCipher)
					require.NoError(t, err)
					var account UpstreamAccountCredentials
					require.NoError(t, common.UnmarshalJsonStr(plain, &account))
					assert.Equal(t, password, account.Password)
					saved.Name = "renamed"
					require.NoError(t, SaveUpstream(saved, UpstreamCredentials{}))
					require.Equal(t, item.AccountCipher, saved.AccountCipher)
					conflict := &Upstream{Name: "conflict", PrimaryURL: item.PrimaryURL, Addresses: item.Addresses, AuthMode: UpstreamAuthPassword}
					require.Error(t, SaveUpstream(conflict, UpstreamCredentials{AccountEmail: &email, AccountPassword: &password}))
					saved.AuthMode = UpstreamAuthJWT
					require.Error(t, SaveUpstream(saved, UpstreamCredentials{}), "mode changes must not reuse a generated token")
					jwt := "manual-jwt"
					require.NoError(t, SaveUpstream(saved, UpstreamCredentials{AccessToken: &jwt}))
					assert.Empty(t, saved.AccountCipher, "JWT mode must remove password credentials")
					saved.AuthMode = UpstreamAuthPassword
					require.Error(t, SaveUpstream(saved, UpstreamCredentials{}))
					require.NoError(t, SaveUpstream(saved, UpstreamCredentials{AccountEmail: &email, AccountPassword: &password}))
					assert.Empty(t, saved.AccessCipher)
					assert.Empty(t, saved.RefreshCipher)
				})
			}
		})
	}
}
