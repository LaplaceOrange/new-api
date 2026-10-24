package setting

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupModelPolicyModesKeepIndependentLists(t *testing.T) {
	original := GetGroupModelPoliciesCopy()
	t.Cleanup(func() {
		raw, err := common.Marshal(original)
		require.NoError(t, err)
		require.NoError(t, UpdateGroupModelPoliciesByJSONString(string(raw)))
	})

	require.NoError(t, UpdateGroupModelPoliciesByJSONString(`{
		"default": {
			"mode": "blacklist",
			"blacklist": ["gpt-4o"],
			"whitelist": ["claude-3-7-sonnet"]
		}
	}`))

	assert.False(t, IsModelAllowedByGroupPolicy("default", "gpt-4o"))
	assert.True(t, IsModelAllowedByGroupPolicy("default", "claude-3-7-sonnet"))

	require.NoError(t, UpdateGroupModelPoliciesByJSONString(`{
		"default": {
			"mode": "whitelist",
			"blacklist": ["gpt-4o"],
			"whitelist": ["claude-3-7-sonnet"]
		}
	}`))
	policy := GetGroupModelPolicy("default")
	assert.Equal(t, GroupModelPolicyModeWhitelist, policy.Mode)
	assert.Equal(t, []string{"gpt-4o"}, policy.Blacklist)
	assert.Equal(t, []string{"claude-3-7-sonnet"}, policy.Whitelist)
	assert.False(t, IsModelAllowedByGroupPolicy("default", "gpt-4o"))
	assert.True(t, IsModelAllowedByGroupPolicy("default", "claude-3-7-sonnet"))
	assert.False(t, IsModelAllowedByGroupPolicy("default", "unknown"))

	require.NoError(t, UpdateGroupModelPoliciesByJSONString(`{
		"default": {
			"mode": "all",
			"blacklist": ["gpt-4o"],
			"whitelist": ["claude-3-7-sonnet"]
		}
	}`))
	assert.True(t, IsModelAllowedByGroupPolicy("default", "gpt-4o"))
	assert.True(t, IsModelAllowedByGroupPolicy("default", "unknown"))
}

func TestGroupModelPolicyDefaultsAndValidation(t *testing.T) {
	original := GetGroupModelPoliciesCopy()
	t.Cleanup(func() {
		raw, err := common.Marshal(original)
		require.NoError(t, err)
		require.NoError(t, UpdateGroupModelPoliciesByJSONString(string(raw)))
	})

	require.NoError(t, UpdateGroupModelPoliciesByJSONString(`{
		"restricted": {
			"mode": "whitelist",
			"blacklist": [],
			"whitelist": []
		}
	}`))
	assert.True(t, IsModelAllowedByGroupPolicy("unconfigured", "gpt-4o"))
	assert.False(t, IsModelAllowedByGroupPolicy("restricted", "gpt-4o"))

	assert.Error(t, ValidateGroupModelPolicies(`{"default":{"mode":"invalid"}}`))
	assert.Error(t, ValidateGroupModelPolicies(`{"default":{"mode":"blacklist","blacklist":[""]}}`))
	assert.Error(t, ValidateGroupModelPolicies(`{"default":{"mode":"blacklist","blacklist":["gpt-4o","gpt-4o"]}}`))
	assert.Error(t, ValidateGroupModelPolicies(`[]`))
}
