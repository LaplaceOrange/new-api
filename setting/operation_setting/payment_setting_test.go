package operation_setting

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTopUpGroupUpgradeRules(t *testing.T) {
	valid, err := ParseTopUpGroupUpgradeRules(`[
		{"type":"single","amount":"100.00","group":"vip"},
		{"type":"cumulative","amount":"500","group":"svip"}
	]`)
	require.NoError(t, err)
	require.Equal(t, []TopUpGroupUpgradeRule{
		{Type: TopUpGroupUpgradeRuleTypeSingle, Amount: "100.00", Group: "vip"},
		{Type: TopUpGroupUpgradeRuleTypeCumulative, Amount: "500", Group: "svip"},
	}, valid)

	tooMany := make([]string, maxTopUpGroupUpgradeRules+1)
	for index := range tooMany {
		tooMany[index] = `{"type":"single","amount":"1","group":"vip"}`
	}

	for name, value := range map[string]string{
		"object instead of array": `{}`,
		"null":                    `null`,
		"unknown type":            `[{"type":"other","amount":"1","group":"vip"}]`,
		"empty amount":            `[{"type":"single","amount":"","group":"vip"}]`,
		"zero amount":             `[{"type":"single","amount":"0","group":"vip"}]`,
		"negative amount":         `[{"type":"single","amount":"-1","group":"vip"}]`,
		"more than two decimals":  `[{"type":"single","amount":"1.001","group":"vip"}]`,
		"empty group":             `[{"type":"single","amount":"1","group":""}]`,
		"too many rules":          "[" + strings.Join(tooMany, ",") + "]",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseTopUpGroupUpgradeRules(value)
			assert.Error(t, err)
		})
	}
}
