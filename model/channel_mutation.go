package model

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

var errRateMultiplierObservationChanged = errors.New("channel rate multiplier configuration or observation changed")

func (channel *Channel) IsPriceMonitorDisabled() bool {
	info := channel.GetOtherInfo()
	reason, ok := info["price_monitor_disabled_reason"].(string)
	return channel.Status == common.ChannelStatusAutoDisabled && ok && reason != "" && info["status_reason"] == reason
}

func (channel *Channel) SameRateMultiplierSource(other *Channel) bool {
	return other != nil && channel.Type == other.Type && channel.Key == other.Key &&
		channel.GetBaseURL() == other.GetBaseURL() && channel.GetSetting().Proxy == other.GetSetting().Proxy
}

func (channel *Channel) HasObservedRateMultiplier(rateMultiplier float64) bool {
	if channel.Remark == nil {
		return false
	}
	firstLine, _, _ := strings.Cut(*channel.Remark, "\n")
	return firstLine == strconv.FormatFloat(rateMultiplier, 'f', -1, 64)
}

func UpdateChannelRateMultiplier(source *Channel, rateMultiplier float64, recoverPrice bool) (*Channel, bool, error) {
	if source == nil || math.IsNaN(rateMultiplier) || math.IsInf(rateMultiplier, 0) || rateMultiplier <= 0 {
		return nil, false, fmt.Errorf("invalid channel rate multiplier")
	}
	pollingLock := GetChannelPollingLock(source.Id)
	pollingLock.Lock()
	defer pollingLock.Unlock()

	statusChanged := false
	updated, err := UpdateChannelAtomically(source.Id, func(current *Channel) error {
		settings := current.GetOtherSettings()
		if !settings.UpstreamRateMultiplierCheckEnabled || !current.SameRateMultiplierSource(source) {
			return errRateMultiplierObservationChanged
		}
		if err := settings.ValidateUpstreamRateMultiplier(); err != nil {
			return err
		}
		if recoverPrice {
			if !current.HasObservedRateMultiplier(rateMultiplier) {
				return errRateMultiplierObservationChanged
			}
		} else {
			remark := ""
			if current.Remark != nil {
				remark = *current.Remark
			}
			_, remaining, multiline := strings.Cut(remark, "\n")
			remark = strconv.FormatFloat(rateMultiplier, 'f', -1, 64)
			if multiline {
				remark += "\n" + remaining
			}
			runes := []rune(remark)
			if len(runes) > 255 {
				remark = string(runes[:255])
			}
			current.Remark = &remark
		}
		if !settings.HasRateMultiplierLimit() || current.Status == common.ChannelStatusManuallyDisabled {
			return nil
		}
		info := current.GetOtherInfo()
		if rateMultiplier > *settings.UpstreamRateMultiplierLimit {
			if current.Status != common.ChannelStatusEnabled {
				return nil
			}
			reason := fmt.Sprintf("Upstream rate multiplier %g exceeds configured limit %g", rateMultiplier, *settings.UpstreamRateMultiplierLimit)
			info["price_monitor_disabled_reason"] = reason
			info["status_reason"] = reason
			info["status_time"] = common.GetTimestamp()
			current.Status = common.ChannelStatusAutoDisabled
			statusChanged = true
		} else if recoverPrice && current.IsPriceMonitorDisabled() {
			if current.ChannelInfo.IsMultiKey && !current.HasEnabledMultiKey() {
				return nil
			}
			current.Status = common.ChannelStatusEnabled
			delete(info, "price_monitor_disabled_reason")
			delete(info, "status_reason")
			delete(info, "status_time")
			statusChanged = true
		}
		current.SetOtherInfo(info)
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	CacheUpdateChannel(updated)
	return updated, statusChanged, nil
}

// ReplaceMultiKeyKeys preserves per-key health state for credentials that
// remain present after an append or replacement operation.
func (channel *Channel) ReplaceMultiKeyKeys(keys string) {
	if channel == nil {
		return
	}
	previous := *channel
	parsed := (&Channel{Key: keys}).GetKeys()
	cleanKeys := make([]string, 0, len(parsed))
	for _, key := range parsed {
		if IsUsableChannelKey(key) {
			cleanKeys = append(cleanKeys, strings.TrimSpace(key))
		}
	}
	channel.Key = strings.Join(cleanKeys, "\n")
	channel.Keys = nil
	remapMultiKeyStateByKey(channel, &previous)
}

// UpdateChannelAtomically applies an update intent to the latest persisted
// channel state and rebuilds its abilities in the same transaction.
func UpdateChannelAtomically(channelID int, apply func(*Channel) error) (*Channel, error) {
	if channelID <= 0 || apply == nil {
		return nil, fmt.Errorf("invalid channel update")
	}

	channel := &Channel{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		contribution, err := lockActiveChannelContributionTx(tx, channelID)
		if err != nil {
			return err
		}
		// SQLite has no SELECT FOR UPDATE. Acquire its single writer lock before
		// reading so a competing writer cannot commit between the read and write.
		if common.UsingMainDatabase(common.DatabaseTypeSQLite) {
			if err := tx.Model(&Channel{}).
				Where("id = ?", channelID).
				UpdateColumn("status", gorm.Expr("status")).Error; err != nil {
				return err
			}
		}
		if err := lockForUpdate(tx).Where("id = ?", channelID).First(channel).Error; err != nil {
			return err
		}
		before := *channel

		channel.Keys = nil
		if err := apply(channel); err != nil {
			return err
		}
		channel.Id = channelID
		channel.Keys = nil
		if contribution != nil && channelContributionReviewedFieldsChanged(&before, channel) {
			return ErrChannelContributionRequiresReview
		}
		channel.normalizeMultiKeyAvailability()
		configuredModels := channel.GetModels()
		for modelName := range channel.ChannelInfo.DisabledModels {
			if !slices.Contains(configuredModels, modelName) {
				delete(channel.ChannelInfo.DisabledModels, modelName)
			}
		}

		if err := tx.Model(&Channel{}).
			Where("id = ?", channelID).
			Select("*").
			Omit("id").
			Updates(channel).Error; err != nil {
			return err
		}
		if contribution != nil && channel.Status != before.Status {
			now := common.GetTimestamp()
			if channel.Status == common.ChannelStatusManuallyDisabled {
				if err := setLockedContributionHealthPausedTx(tx, contribution, channelID, true, now); err != nil {
					return err
				}
			} else if before.Status == common.ChannelStatusManuallyDisabled {
				if err := setLockedContributionHealthPausedTx(tx, contribution, channelID, false, now); err != nil {
					return err
				}
			}
		}
		return channel.UpdateAbilities(tx)
	})
	if err != nil {
		return nil, err
	}
	return channel, nil
}

// SetChannelModelsEnabled changes only the selected models on the latest row.
func SetChannelModelsEnabled(channelID int, models []string, enabled bool) (*Channel, error) {
	if len(models) == 0 {
		return nil, fmt.Errorf("at least one model is required")
	}
	pollingLock := GetChannelPollingLock(channelID)
	pollingLock.Lock()
	defer pollingLock.Unlock()

	channel, err := UpdateChannelAtomically(channelID, func(current *Channel) error {
		configured := current.GetModels()
		for _, modelName := range models {
			if modelName == "" || !slices.Contains(configured, modelName) {
				return fmt.Errorf("model %q is not configured on this channel", modelName)
			}
		}
		if current.ChannelInfo.DisabledModels == nil {
			current.ChannelInfo.DisabledModels = make(map[string]bool)
		}
		for _, modelName := range models {
			if enabled {
				delete(current.ChannelInfo.DisabledModels, modelName)
			} else {
				current.ChannelInfo.DisabledModels[modelName] = true
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	CacheUpdateChannel(channel)
	return channel, nil
}

func channelContributionReviewedFieldsChanged(before *Channel, after *Channel) bool {
	if before == nil || after == nil {
		return true
	}
	type reviewedFields struct {
		Name         string
		Type         int
		BaseURL      *string
		Key          string
		Group        string
		Models       string
		ModelMapping *string
	}
	left := reviewedFields{
		Name: before.Name, Type: before.Type, BaseURL: before.BaseURL, Key: before.Key,
		Group: before.Group, Models: before.Models, ModelMapping: before.ModelMapping,
	}
	right := reviewedFields{
		Name: after.Name, Type: after.Type, BaseURL: after.BaseURL, Key: after.Key,
		Group: after.Group, Models: after.Models, ModelMapping: after.ModelMapping,
	}
	return !reflect.DeepEqual(left, right)
}
