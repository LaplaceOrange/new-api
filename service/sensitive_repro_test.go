package service

import (
	"testing"

	"github.com/QuantumNous/new-api/setting"
	"github.com/stretchr/testify/require"
)

func TestSensitiveWordContainsDoesNotMatchUnrelatedChineseText(t *testing.T) {
	old := setting.SensitiveWords
	defer func() { setting.SensitiveWords = old }()
	setting.SensitiveWords = []string{"色情", "赌博", "暴力", "违禁词", "test_sensitive"}

	matched, words := SensitiveWordContains("你好")
	require.False(t, matched, "你好 unexpectedly matched: %v", words)
	require.Empty(t, words)

	matched, words = SensitiveWordContains("这是色情内容")
	require.True(t, matched)
	require.NotEmpty(t, words)
}

func TestAcSearchMatchesOnlyConfiguredWords(t *testing.T) {
	matched, words := AcSearch("你好", []string{"user", "色情", "赌博"}, true)
	require.False(t, matched, "unrelated text matched: %v", words)
	require.Empty(t, words)
	matched, words = AcSearch("这是色情内容", []string{"user", "色情", "赌博"}, true)
	require.True(t, matched)
	require.Contains(t, words, "色情")
}

func TestAcSearchCacheFollowsUpdatedWordList(t *testing.T) {
	matched, words := AcSearch("旧词", []string{"旧词", "其他"}, true)
	require.True(t, matched)
	require.Contains(t, words, "旧词")
	matched, words = AcSearch("旧词", []string{"新词", "其他"}, true)
	require.False(t, matched, "stale sensitive-word cache returned %v", words)
	matched, words = AcSearch("新词", []string{"其他", "新词"}, true)
	require.True(t, matched)
	require.Contains(t, words, "新词")
}
