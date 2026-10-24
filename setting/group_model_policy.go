package setting

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

const GroupModelPoliciesOptionKey = "GroupModelPolicies"

type GroupModelPolicyMode string

const (
	GroupModelPolicyModeAll       GroupModelPolicyMode = "all"
	GroupModelPolicyModeBlacklist GroupModelPolicyMode = "blacklist"
	GroupModelPolicyModeWhitelist GroupModelPolicyMode = "whitelist"
)

type GroupModelPolicy struct {
	Mode      GroupModelPolicyMode `json:"mode"`
	Blacklist []string             `json:"blacklist"`
	Whitelist []string             `json:"whitelist"`
}

var (
	groupModelPoliciesMu sync.RWMutex
	groupModelPolicies   = map[string]GroupModelPolicy{}
)

func defaultGroupModelPolicy() GroupModelPolicy {
	return GroupModelPolicy{
		Mode:      GroupModelPolicyModeAll,
		Blacklist: []string{},
		Whitelist: []string{},
	}
}

func GetGroupModelPoliciesCopy() map[string]GroupModelPolicy {
	groupModelPoliciesMu.RLock()
	defer groupModelPoliciesMu.RUnlock()

	policies := make(map[string]GroupModelPolicy, len(groupModelPolicies))
	for group, policy := range groupModelPolicies {
		policies[group] = cloneGroupModelPolicy(policy)
	}
	return policies
}

func GetGroupModelPolicy(group string) GroupModelPolicy {
	groupModelPoliciesMu.RLock()
	policy, ok := groupModelPolicies[group]
	groupModelPoliciesMu.RUnlock()
	if !ok {
		return defaultGroupModelPolicy()
	}
	return cloneGroupModelPolicy(policy)
}

func IsModelAllowedByGroupPolicy(group, model string) bool {
	policy, configured := getGroupModelPolicy(group)
	if !configured || policy.Mode == GroupModelPolicyModeAll {
		return true
	}

	model = strings.TrimSpace(model)
	if model == "" {
		return false
	}
	modelNames := []string{model}
	if formatted := ratio_setting.FormatMatchingModelName(model); formatted != "" && formatted != model {
		modelNames = append(modelNames, formatted)
	}
	if routed := ratio_setting.RoutingMatchModelName(model); routed != "" && routed != model && routed != modelNames[len(modelNames)-1] {
		modelNames = append(modelNames, routed)
	}
	if policy.Mode == GroupModelPolicyModeWhitelist {
		return containsAnyModel(policy.Whitelist, modelNames)
	}
	return !containsAnyModel(policy.Blacklist, modelNames)
}

func GroupModelPolicies2JSONString() string {
	data, err := common.Marshal(GetGroupModelPoliciesCopy())
	if err != nil {
		return "{}"
	}
	return string(data)
}

func UpdateGroupModelPoliciesByJSONString(value string) error {
	policies, err := parseGroupModelPolicies(value)
	if err != nil {
		return err
	}

	groupModelPoliciesMu.Lock()
	groupModelPolicies = policies
	groupModelPoliciesMu.Unlock()
	return nil
}

func ValidateGroupModelPolicies(value string) error {
	_, err := parseGroupModelPolicies(value)
	return err
}

func parseGroupModelPolicies(value string) (map[string]GroupModelPolicy, error) {
	var policies map[string]GroupModelPolicy
	if err := common.UnmarshalJsonStr(value, &policies); err != nil {
		return nil, err
	}
	if policies == nil {
		return nil, fmt.Errorf("group model policies must be a JSON object")
	}

	normalized := make(map[string]GroupModelPolicy, len(policies))
	for rawGroup, policy := range policies {
		group := strings.TrimSpace(rawGroup)
		if group == "" {
			return nil, fmt.Errorf("group model policy group must not be empty")
		}
		if _, exists := normalized[group]; exists {
			return nil, fmt.Errorf("duplicate group model policy for %q", group)
		}
		if policy.Mode != GroupModelPolicyModeAll &&
			policy.Mode != GroupModelPolicyModeBlacklist &&
			policy.Mode != GroupModelPolicyModeWhitelist {
			return nil, fmt.Errorf("invalid group model policy mode for %q", group)
		}
		blacklist, err := normalizeModelList(policy.Blacklist, group, "blacklist")
		if err != nil {
			return nil, err
		}
		whitelist, err := normalizeModelList(policy.Whitelist, group, "whitelist")
		if err != nil {
			return nil, err
		}
		normalized[group] = GroupModelPolicy{
			Mode:      policy.Mode,
			Blacklist: blacklist,
			Whitelist: whitelist,
		}
	}
	return normalized, nil
}

func normalizeModelList(models []string, group, listName string) ([]string, error) {
	if models == nil {
		return []string{}, nil
	}
	normalized := make([]string, 0, len(models))
	seen := make(map[string]struct{}, len(models))
	for _, rawModel := range models {
		model := strings.TrimSpace(rawModel)
		if model == "" {
			return nil, fmt.Errorf("group model policy %s for %q contains an empty model", listName, group)
		}
		if _, ok := seen[model]; ok {
			return nil, fmt.Errorf("group model policy %s for %q contains duplicate model %q", listName, group, model)
		}
		seen[model] = struct{}{}
		normalized = append(normalized, model)
	}
	return normalized, nil
}

func getGroupModelPolicy(group string) (GroupModelPolicy, bool) {
	groupModelPoliciesMu.RLock()
	policy, ok := groupModelPolicies[group]
	groupModelPoliciesMu.RUnlock()
	if !ok {
		return defaultGroupModelPolicy(), false
	}
	return policy, true
}

func cloneGroupModelPolicy(policy GroupModelPolicy) GroupModelPolicy {
	return GroupModelPolicy{
		Mode:      policy.Mode,
		Blacklist: append([]string{}, policy.Blacklist...),
		Whitelist: append([]string{}, policy.Whitelist...),
	}
}

func containsAnyModel(models, targets []string) bool {
	for _, target := range targets {
		if slices.Contains(models, target) {
			return true
		}
	}
	return false
}
