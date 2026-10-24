package common

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	hostreasoning "github.com/QuantumNous/new-api/setting/reasoning"
)

// ResolveChannelModel follows aliases using the same base-model fallback as relay.
func ResolveChannelModel(origin, mapping string) (string, error) {
	if mapping == "" || mapping == "{}" {
		return origin, nil
	}
	var aliases map[string]string
	if err := common.UnmarshalJsonStr(mapping, &aliases); err != nil {
		return "", fmt.Errorf("unmarshal_model_mapping_failed")
	}
	current := origin
	visited := map[string]bool{current: true}
	for {
		next := aliases[current]
		if next == "" {
			next = aliases[hostreasoning.BaseModelName(current)]
		}
		if next == "" || next == current {
			return current, nil
		}
		if visited[next] {
			return "", errors.New("model_mapping_contains_cycle")
		}
		visited[next] = true
		current = next
	}
}
