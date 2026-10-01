package setting

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
)

type GroupType struct {
	Name       string   `json:"name"`
	Icon       string   `json:"icon"`
	CustomIcon string   `json:"custom_icon,omitempty"`
	Color      string   `json:"color"`
	Groups     []string `json:"groups"`
}

var groupTypeCustomIconPattern = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*(\.(Color|Avatar))?$`)

func ValidateGroupTypes(value string) error {
	var types []GroupType
	if err := common.UnmarshalJsonStr(value, &types); err != nil {
		return err
	}
	if types == nil {
		return errors.New("group types must be an array")
	}
	names := make(map[string]bool)
	groups := make(map[string]bool)
	for _, groupType := range types {
		name := strings.TrimSpace(groupType.Name)
		if name == "" || utf8.RuneCountInString(name) > 48 || names[name] {
			return fmt.Errorf("invalid or duplicate group type name: %q", groupType.Name)
		}
		names[name] = true
		if groupType.Groups == nil {
			return errors.New("group type groups must be an array")
		}
		switch groupType.Icon {
		case "layers", "sparkles", "zap", "shield", "star", "globe":
		case "custom":
			if len(groupType.CustomIcon) > 64 || !groupTypeCustomIconPattern.MatchString(groupType.CustomIcon) {
				return fmt.Errorf("invalid LobeHub icon ID: %q", groupType.CustomIcon)
			}
		default:
			return fmt.Errorf("invalid group type icon: %q", groupType.Icon)
		}
		color := groupType.Color
		if len(color) != 7 || color[0] != '#' {
			return fmt.Errorf("invalid group type color: %q", color)
		}
		for _, digit := range color[1:] {
			if !strings.ContainsRune("0123456789abcdefABCDEF", digit) {
				return fmt.Errorf("invalid group type color: %q", color)
			}
		}
		for _, group := range groupType.Groups {
			if strings.TrimSpace(group) == "" || groups[group] {
				return fmt.Errorf("invalid or duplicate group assignment: %q", group)
			}
			groups[group] = true
		}
	}
	return nil
}

func GetGroupTypes(availableGroups []string) []GroupType {
	common.OptionMapRWMutex.RLock()
	value := common.OptionMap["GroupTypes"]
	common.OptionMapRWMutex.RUnlock()
	var types []GroupType
	if err := ValidateGroupTypes(value); err != nil {
		return []GroupType{}
	}
	if err := common.UnmarshalJsonStr(value, &types); err != nil || types == nil {
		return []GroupType{}
	}
	for i := range types {
		groups := make([]string, 0, len(types[i].Groups))
		for _, group := range types[i].Groups {
			if slices.Contains(availableGroups, group) {
				groups = append(groups, group)
			}
		}
		types[i].Groups = groups
	}
	return types
}
