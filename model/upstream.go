package model

import (
	"context"
	"encoding/hex"
	"errors"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Upstream struct {
	ID                int               `json:"id" gorm:"primaryKey"`
	Name              string            `json:"name" gorm:"type:varchar(128);not null"`
	PrimaryURL        string            `json:"primary_url" gorm:"type:text"`
	UserAgent         string            `json:"user_agent" gorm:"type:text"`
	AutoRefreshToken  bool              `json:"auto_refresh_token"`
	AccessCipher      string            `json:"-" gorm:"type:text"`
	RefreshCipher     string            `json:"-" gorm:"type:text"`
	RefreshPending    bool              `json:"-"`
	CredentialBlocked bool              `json:"credential_blocked"`
	Balance           *float64          `json:"balance"`
	BalanceUpdatedAt  int64             `json:"balance_updated_at" gorm:"bigint"`
	LastAttemptAt     int64             `json:"last_attempt_at" gorm:"bigint"`
	LastError         string            `json:"last_error" gorm:"type:text"`
	LeaseOwner        string            `json:"-" gorm:"type:varchar(64)"`
	LeaseUntil        int64             `json:"-" gorm:"bigint"`
	Revision          int64             `json:"-" gorm:"bigint"`
	CreatedAt         int64             `json:"created_at" gorm:"bigint"`
	UpdatedAt         int64             `json:"updated_at" gorm:"bigint"`
	Addresses         []string          `json:"addresses" gorm:"-"`
	HasAccessToken    bool              `json:"has_access_token" gorm:"-"`
	HasRefreshToken   bool              `json:"has_refresh_token" gorm:"-"`
	Refreshing        bool              `json:"refreshing" gorm:"-"`
	ChannelIDs        []int             `json:"channel_ids" gorm:"-"`
	Snapshot          *UpstreamSnapshot `json:"snapshot" gorm:"-"`
}

type UpstreamAddress struct {
	ID         int    `gorm:"primaryKey"`
	UpstreamID int    `gorm:"index;not null"`
	URL        string `gorm:"type:text;not null"`
	// A bounded hash avoids index length/collation differences across databases.
	URLHash string `gorm:"type:varchar(64);uniqueIndex;not null"`
}

type UpstreamSnapshot struct {
	ID          int      `json:"-" gorm:"primaryKey"`
	UpstreamID  int      `json:"-" gorm:"uniqueIndex:idx_upstream_window,priority:1"`
	Days        int      `json:"days" gorm:"uniqueIndex:idx_upstream_window,priority:2"`
	Mode        string   `json:"mode" gorm:"type:varchar(16);uniqueIndex:idx_upstream_window,priority:3"`
	StartAt     int64    `json:"start_at" gorm:"bigint"`
	EndAt       int64    `json:"end_at" gorm:"bigint"`
	Timezone    string   `json:"timezone" gorm:"type:varchar(64)"`
	Consumption *float64 `json:"consumption"`
	UpdatedAt   int64    `json:"updated_at" gorm:"bigint"`
	LastError   string   `json:"last_error" gorm:"type:text"`
	Complete    bool     `json:"complete"`
}

type UpstreamConfig struct {
	ID              int    `json:"-" gorm:"primaryKey;autoIncrement:false"`
	IntervalMinutes int    `json:"interval_minutes"`
	Timezone        string `json:"timezone" gorm:"type:varchar(64)"`
}

var ErrUpstreamBusy = errors.New("Upstream check is running; try again after it finishes.")

func NormalizeUpstreamURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" ||
		(u.Scheme != "https" && u.Scheme != "http") || len(raw) > 2048 {
		return "", errors.New("Enter a valid HTTP(S) base URL without credentials, query, or fragment.")
	}
	host, port := strings.ToLower(u.Hostname()), u.Port()
	if port != "" && !((u.Scheme == "https" && port == "443") || (u.Scheme == "http" && port == "80")) {
		host = net.JoinHostPort(host, port)
	} else if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	u.Host = host
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath = strings.TrimRight(u.RawPath, "/")
	return u.String(), nil
}

func GetUpstreamConfig() (UpstreamConfig, error) {
	config := UpstreamConfig{ID: 1, IntervalMinutes: 30, Timezone: "Asia/Shanghai"}
	err := DB.First(&config, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
	}
	return config, err
}

func SaveUpstreamConfig(config UpstreamConfig) error {
	if config.IntervalMinutes < 0 || config.IntervalMinutes > 1440 {
		return errors.New("Check interval must be between 0 and 1440 minutes.")
	}
	if _, err := time.LoadLocation(config.Timezone); err != nil || len(config.Timezone) > 64 {
		return errors.New("Enter a valid IANA time zone.")
	}
	config.ID = 1
	return DB.Transaction(func(tx *gorm.DB) error {
		old, err := GetUpstreamConfig()
		if err != nil {
			return err
		}
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&config).Error; err != nil {
			return err
		}
		if old.Timezone != config.Timezone {
			return tx.Model(&UpstreamSnapshot{}).Where("1 = 1").Updates(map[string]any{
				"complete": false, "last_error": "Statistics time zone changed; refresh required.",
			}).Error
		}
		return nil
	})
}

func GetUpstream(id int) (*Upstream, error) {
	var item Upstream
	if err := DB.First(&item, id).Error; err != nil {
		return nil, err
	}
	item.HasAccessToken, item.HasRefreshToken = item.AccessCipher != "", item.RefreshCipher != ""
	item.Refreshing = item.LeaseUntil > time.Now().Unix()
	item.Addresses = []string{}
	if err := DB.Model(&UpstreamAddress{}).Where("upstream_id = ?", id).Order("id").Pluck("url", &item.Addresses).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func ListUpstreams() ([]Upstream, error) {
	items := []Upstream{}
	if err := DB.Order("id").Find(&items).Error; err != nil {
		return nil, err
	}
	addresses := []UpstreamAddress{}
	if err := DB.Order("id").Find(&addresses).Error; err != nil {
		return nil, err
	}
	byID := map[int][]string{}
	for _, address := range addresses {
		byID[address.UpstreamID] = append(byID[address.UpstreamID], address.URL)
	}
	for i := range items {
		items[i].Addresses = byID[items[i].ID]
		if items[i].Addresses == nil {
			items[i].Addresses = []string{}
		}
		items[i].HasAccessToken, items[i].HasRefreshToken = items[i].AccessCipher != "", items[i].RefreshCipher != ""
		items[i].Refreshing = items[i].LeaseUntil > time.Now().Unix()
	}
	return items, nil
}

func SaveUpstream(item *Upstream, accessToken, refreshToken *string) error {
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" || len(item.Name) > 128 || len(item.Addresses) == 0 || len(item.Addresses) > 32 ||
		len(item.UserAgent) > 1024 || strings.ContainsAny(item.UserAgent, "\r\n") {
		return errors.New("Invalid upstream name, addresses, or User-Agent.")
	}
	primary, err := NormalizeUpstreamURL(item.PrimaryURL)
	if err != nil || !strings.HasPrefix(primary, "https://") {
		return errors.New("The primary account URL must use HTTPS.")
	}
	item.PrimaryURL = primary
	addresses := make([]UpstreamAddress, 0, len(item.Addresses))
	seen := map[string]bool{}
	for _, raw := range item.Addresses {
		normalized, err := NormalizeUpstreamURL(raw)
		if err != nil {
			return err
		}
		if seen[normalized] {
			return errors.New("Duplicate upstream address.")
		}
		seen[normalized] = true
		addresses = append(addresses, UpstreamAddress{URL: normalized, URLHash: hex.EncodeToString(common.Sha256Raw([]byte(normalized)))})
	}
	if !seen[primary] {
		return errors.New("The primary URL must be one of the upstream addresses.")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var old Upstream
		var oldAddresses []UpstreamAddress
		if item.ID > 0 {
			if err := lockForUpdate(tx).First(&old, item.ID).Error; err != nil {
				return err
			}
			if old.LeaseUntil > time.Now().Unix() {
				return ErrUpstreamBusy
			}
			if err := tx.Where("upstream_id = ?", item.ID).Order("id").Find(&oldAddresses).Error; err != nil {
				return err
			}
		}
		item.AccessCipher, item.RefreshCipher = old.AccessCipher, old.RefreshCipher
		for _, credential := range []struct {
			value       *string
			destination *string
		}{
			{accessToken, &item.AccessCipher}, {refreshToken, &item.RefreshCipher},
		} {
			if credential.value == nil {
				continue
			}
			value := strings.TrimSpace(*credential.value)
			if len(value) > 16384 || strings.ContainsAny(value, "\r\n") {
				return errors.New("Invalid upstream credential.")
			}
			encrypted, err := common.EncryptUpstreamCredential(value)
			if err != nil {
				return err
			}
			*credential.destination = encrypted
		}
		if item.AccessCipher == "" || (item.AutoRefreshToken && item.RefreshCipher == "") {
			return errors.New("An access JWT is required; automatic renewal also requires a refresh token.")
		}
		changed := old.PrimaryURL != item.PrimaryURL || old.UserAgent != item.UserAgent ||
			accessToken != nil || refreshToken != nil || old.AutoRefreshToken != item.AutoRefreshToken
		if len(oldAddresses) != len(addresses) {
			changed = true
		} else {
			oldAddressSet := make(map[string]bool, len(oldAddresses))
			for _, address := range oldAddresses {
				oldAddressSet[address.URL] = true
			}
			for _, address := range addresses {
				if !oldAddressSet[address.URL] {
					changed = true
					break
				}
			}
		}
		item.Revision = old.Revision + 1
		item.CreatedAt, item.UpdatedAt = old.CreatedAt, time.Now().Unix()
		if item.CreatedAt == 0 {
			item.CreatedAt = item.UpdatedAt
		}
		item.Balance, item.BalanceUpdatedAt = old.Balance, old.BalanceUpdatedAt
		item.CredentialBlocked, item.RefreshPending = old.CredentialBlocked, old.RefreshPending
		if changed {
			item.Balance, item.BalanceUpdatedAt = nil, 0
			if accessToken != nil || refreshToken != nil {
				item.CredentialBlocked, item.RefreshPending = false, false
			}
			if err := tx.Where("upstream_id = ?", item.ID).Delete(&UpstreamSnapshot{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Save(item).Error; err != nil {
			return err
		}
		if err := tx.Where("upstream_id = ?", item.ID).Delete(&UpstreamAddress{}).Error; err != nil {
			return err
		}
		for i := range addresses {
			addresses[i].UpstreamID = item.ID
		}
		if err := tx.Create(&addresses).Error; err != nil {
			return errors.New("An address is already assigned to another upstream.")
		}
		return nil
	})
}

func DeleteUpstream(id int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var item Upstream
		if err := lockForUpdate(tx).First(&item, id).Error; err != nil {
			return err
		}
		if item.LeaseUntil > time.Now().Unix() {
			return ErrUpstreamBusy
		}
		if err := tx.Where("upstream_id = ?", id).Delete(&UpstreamAddress{}).Error; err != nil {
			return err
		}
		if err := tx.Where("upstream_id = ?", id).Delete(&UpstreamSnapshot{}).Error; err != nil {
			return err
		}
		return tx.Delete(&item).Error
	})
}

func MatchUpstreamChannels(addresses []string) ([]int, error) {
	var channels []Channel
	if err := DB.Select("id", "type", "base_url").Find(&channels).Error; err != nil {
		return nil, err
	}
	match := map[string]bool{}
	for _, address := range addresses {
		match[address] = true
	}
	ids := []int{}
	for i := range channels {
		address, err := NormalizeUpstreamURL(channels[i].GetBaseURL())
		if err == nil && match[address] {
			ids = append(ids, channels[i].Id)
		}
	}
	return ids, nil
}

func ClaimUpstream(ctx context.Context, id int, owner string) (*Upstream, error) {
	now := time.Now().Unix()
	result := DB.WithContext(ctx).Model(&Upstream{}).Where("id = ? AND lease_until <= ?", id, now).
		Updates(map[string]any{"lease_owner": owner, "lease_until": now + 300, "last_attempt_at": now})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrUpstreamBusy
	}
	return GetUpstream(id)
}

// Every checkpoint is fenced by the database lease, including token rotation.
func UpdateClaimedUpstream(ctx context.Context, id int, owner string, values map[string]any) error {
	values["updated_at"] = time.Now().Unix()
	result := DB.WithContext(ctx).Model(&Upstream{}).
		Where("id = ? AND lease_owner = ? AND lease_until > ?", id, owner, time.Now().Unix()).Updates(values)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		var held int64
		if err := DB.WithContext(ctx).Model(&Upstream{}).Where("id = ? AND lease_owner = ? AND lease_until > ?", id, owner, time.Now().Unix()).Count(&held).Error; err != nil {
			return err
		}
		if held == 0 {
			return ErrUpstreamBusy
		}
	}
	return nil
}

func SaveUpstreamSnapshot(ctx context.Context, owner string, snapshot *UpstreamSnapshot) error {
	return DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item Upstream
		if err := lockForUpdate(tx).Where("id = ? AND lease_owner = ? AND lease_until > ?", snapshot.UpstreamID, owner, time.Now().Unix()).First(&item).Error; err != nil {
			return err
		}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "upstream_id"}, {Name: "days"}, {Name: "mode"}},
			UpdateAll: true,
		}).Create(snapshot).Error
	})
}

type UpstreamRank struct {
	UserID   int    `json:"user_id,omitempty"`
	Name     string `json:"name"`
	Requests int64  `json:"requests"`
	Tokens   int64  `json:"tokens"`
	Quota    int64  `json:"quota"`
}

func GetUpstreamRankings(ctx context.Context, ids []int, start, end int64, dimension, metric string, limit int) ([]UpstreamRank, error) {
	rows := []UpstreamRank{}
	if len(ids) == 0 {
		return rows, nil
	}
	columns, group, tie := "model_name AS name", "model_name", "name"
	if dimension == "users" {
		columns, group, tie = "user_id, MAX(username) AS name", "user_id", "user_id"
	}
	order := "quota"
	if metric == "requests" || metric == "tokens" {
		order = metric
	}
	err := LOG_DB.WithContext(ctx).Model(&Log{}).Where("type = ? AND channel_id IN ? AND created_at >= ? AND created_at < ?", LogTypeConsume, ids, start, end).
		Select(columns + ", COUNT(*) AS requests, COALESCE(SUM(prompt_tokens), 0) + COALESCE(SUM(completion_tokens), 0) AS tokens, COALESCE(SUM(quota), 0) AS quota").
		Group(group).Order(order + " DESC, " + tie + " ASC").Limit(limit).Scan(&rows).Error
	return rows, err
}
