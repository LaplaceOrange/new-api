package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTopUpGroupUpgradeRulesRequiresExistingGroup(t *testing.T) {
	previousGroups := ratio_setting.GetGroupRatioCopy()
	t.Cleanup(func() {
		previousGroupsJSON, err := common.Marshal(previousGroups)
		require.NoError(t, err)
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(string(previousGroupsJSON)))
	})

	groupsJSON, err := common.Marshal(map[string]float64{"default": 1, "vip": 2})
	require.NoError(t, err)
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(string(groupsJSON)))

	assert.NoError(t, validateOptionValue(operation_setting.TopUpGroupUpgradeRulesOptionKey,
		`[{"type":"single","amount":"100.00","group":"vip"}]`))
	assert.Error(t, validateOptionValue(operation_setting.TopUpGroupUpgradeRulesOptionKey,
		`[{"type":"single","amount":"100.00","group":"missing"}]`))
}
