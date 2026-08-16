package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateChannelContributionOptionRestrictsChannelTypesToSupportedSubset(t *testing.T) {
	assert.NoError(t, ValidateChannelContributionOption(
		ChannelContributionSettingPrefix+"allowed_channel_types",
		`[1,14,60]`,
	))
	assert.Error(t, ValidateChannelContributionOption(
		ChannelContributionSettingPrefix+"allowed_channel_types",
		`[3]`,
	))
	assert.Error(t, ValidateChannelContributionOption(
		ChannelContributionSettingPrefix+"allowed_channel_types",
		`[1,1]`,
	))
	assert.Error(t, ValidateChannelContributionOption(
		ChannelContributionSettingPrefix+"allowed_channel_types",
		`["1"]`,
	))
}

func TestValidateChannelContributionOptionRejectsInvalidGroupsAndDurations(t *testing.T) {
	assert.NoError(t, ValidateChannelContributionOption(
		ChannelContributionSettingPrefix+"allowed_groups",
		`["default","vip"]`,
	))
	assert.Error(t, ValidateChannelContributionOption(
		ChannelContributionSettingPrefix+"allowed_groups",
		`["default"," default "]`,
	))
	assert.Error(t, ValidateChannelContributionOption(
		ChannelContributionSettingPrefix+"allowed_groups",
		`[""]`,
	))
	assert.NoError(t, ValidateChannelContributionOption(
		ChannelContributionSettingPrefix+"unavailable_delete_hours",
		"48",
	))
	assert.Error(t, ValidateChannelContributionOption(
		ChannelContributionSettingPrefix+"unavailable_delete_hours",
		"0",
	))
}
