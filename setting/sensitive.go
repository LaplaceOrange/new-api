package setting

import "strings"

var CheckSensitiveEnabled = true
var CheckSensitiveOnPromptEnabled = true
var SensitiveWordAutoBanEnabled = false

// SensitiveWordUserBanThreshold is the number of prompt matches required to
// automatically disable a user when sensitive-word auto-ban is enabled.
var SensitiveWordUserBanThreshold = 2

// SensitiveWordIPUserBanThreshold is the number of distinct automatically
// banned users from one IP required to automatically ban that IP.
var SensitiveWordIPUserBanThreshold = 3

const SensitiveWordBanThresholdMax = 1_000_000

func SensitiveWordUserBanLimit() int {
	if SensitiveWordUserBanThreshold < 1 {
		return 2
	}
	if SensitiveWordUserBanThreshold > SensitiveWordBanThresholdMax {
		return SensitiveWordBanThresholdMax
	}
	return SensitiveWordUserBanThreshold
}

func SensitiveWordIPUserBanLimit() int {
	if SensitiveWordIPUserBanThreshold < 1 {
		return 3
	}
	if SensitiveWordIPUserBanThreshold > SensitiveWordBanThresholdMax {
		return SensitiveWordBanThresholdMax
	}
	return SensitiveWordIPUserBanThreshold
}

//var CheckSensitiveOnCompletionEnabled = true

// StopOnSensitiveEnabled 如果检测到敏感词，是否立刻停止生成，否则替换敏感词
var StopOnSensitiveEnabled = true

// StreamCacheQueueLength 流模式缓存队列长度，0表示无缓存
var StreamCacheQueueLength = 0

// SensitiveWords 敏感词
// var SensitiveWords []string
var SensitiveWords = []string{
	"test_sensitive",
}

func SensitiveWordsToString() string {
	return strings.Join(SensitiveWords, "\n")
}

func SensitiveWordsFromString(s string) {
	SensitiveWords = []string{}
	sw := strings.Split(s, "\n")
	for _, w := range sw {
		w = strings.TrimSpace(w)
		if w != "" {
			SensitiveWords = append(SensitiveWords, w)
		}
	}
}

func ShouldCheckPromptSensitive() bool {
	return CheckSensitiveEnabled && CheckSensitiveOnPromptEnabled
}

//func ShouldCheckCompletionSensitive() bool {
//	return CheckSensitiveEnabled && CheckSensitiveOnCompletionEnabled
//}
