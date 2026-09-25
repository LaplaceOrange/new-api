package degradation

import (
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

type ModelConfig struct {
	Model    string `json:"model"`
	Expected string `json:"expected"`
	Sort     int    `json:"sort"`
}

type GroupConfig struct {
	Group  string        `json:"group"`
	Sort   int           `json:"sort"`
	Models []ModelConfig `json:"models"`
}

type Config struct {
	Enabled              bool          `json:"enabled"`
	IntervalMinutes      int           `json:"interval_minutes"`
	RetryCount           int           `json:"retry_count"`
	RetryIntervalMinutes int           `json:"retry_interval_minutes"`
	Groups               []GroupConfig `json:"groups"`
}

func DefaultConfig() Config {
	return Config{
		Enabled:              false,
		IntervalMinutes:      60,
		RetryCount:           2,
		RetryIntervalMinutes: 10,
		Groups:               []GroupConfig{},
	}
}

func LoadConfig() Config {
	cfg := DefaultConfig()
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[OptionKey]
	common.OptionMapRWMutex.RUnlock()
	if strings.TrimSpace(raw) == "" {
		return cfg
	}
	parsed := Config{}
	if err := common.UnmarshalJsonStr(raw, &parsed); err != nil {
		return cfg
	}
	if parsed.IntervalMinutes <= 0 {
		parsed.IntervalMinutes = cfg.IntervalMinutes
	}
	if parsed.RetryIntervalMinutes <= 0 {
		parsed.RetryIntervalMinutes = cfg.RetryIntervalMinutes
	}
	if parsed.Groups == nil {
		parsed.Groups = []GroupConfig{}
	}
	for i := range parsed.Groups {
		if parsed.Groups[i].Models == nil {
			parsed.Groups[i].Models = []ModelConfig{}
		}
	}
	return parsed
}

func (cfg Config) Interval() time.Duration {
	return time.Duration(cfg.IntervalMinutes) * time.Minute
}

func (cfg Config) RetryInterval() time.Duration {
	return time.Duration(cfg.RetryIntervalMinutes) * time.Minute
}

func (cfg Config) HasModels() bool {
	for _, group := range cfg.Groups {
		if len(group.Models) > 0 {
			return true
		}
	}
	return false
}

func Normalize(input Config, groups map[string]struct{}, models map[string]map[string]struct{}) (Config, error) {
	if input.IntervalMinutes < 1 || input.IntervalMinutes > 7*24*60 {
		return Config{}, fmt.Errorf("interval_minutes must be from 1 to 10080")
	}
	if input.RetryIntervalMinutes < 1 || input.RetryIntervalMinutes > 7*24*60 {
		return Config{}, fmt.Errorf("retry_interval_minutes must be from 1 to 10080")
	}
	if input.RetryCount < 0 || input.RetryCount > 20 {
		return Config{}, fmt.Errorf("retry_count must be from 0 to 20")
	}
	seenGroups := map[string]struct{}{}
	out := Config{
		Enabled:              input.Enabled,
		IntervalMinutes:      input.IntervalMinutes,
		RetryCount:           input.RetryCount,
		RetryIntervalMinutes: input.RetryIntervalMinutes,
		Groups:               make([]GroupConfig, 0, len(input.Groups)),
	}
	for _, group := range input.Groups {
		name := strings.TrimSpace(group.Group)
		if name == "" {
			return Config{}, fmt.Errorf("group is required")
		}
		if _, ok := groups[name]; !ok {
			return Config{}, fmt.Errorf("unknown group %s", name)
		}
		if _, ok := seenGroups[name]; ok {
			return Config{}, fmt.Errorf("duplicate group %s", name)
		}
		seenGroups[name] = struct{}{}
		allowed := models[name]
		seenModels := map[string]struct{}{}
		modelsOut := make([]ModelConfig, 0, len(group.Models))
		for _, item := range group.Models {
			modelName := strings.TrimSpace(item.Model)
			if modelName == "" {
				return Config{}, fmt.Errorf("model is required")
			}
			if _, ok := allowed[modelName]; !ok {
				return Config{}, fmt.Errorf("model %s is not enabled in group %s", modelName, name)
			}
			if _, ok := seenModels[modelName]; ok {
				return Config{}, fmt.Errorf("duplicate model %s in group %s", modelName, name)
			}
			seenModels[modelName] = struct{}{}
			expected := strings.TrimSpace(item.Expected)
			if len([]rune(expected)) > 256 {
				return Config{}, fmt.Errorf("expected name is too long")
			}
			modelsOut = append(modelsOut, ModelConfig{Model: modelName, Expected: expected, Sort: len(modelsOut)})
		}
		out.Groups = append(out.Groups, GroupConfig{Group: name, Sort: len(out.Groups), Models: modelsOut})
	}
	return out, nil
}
