package degradation

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDecideSchedulesPassRetryAndTerminalStates(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	interval := time.Hour
	retry := 10 * time.Minute

	passed := Decide(now, 0, 2, interval, retry, true, true, "gpt-4o")
	require.Equal(t, StatusPassed, passed.Status)
	require.Equal(t, 0, passed.Failures)
	require.Equal(t, now.Add(interval), passed.Next)

	retrying := Decide(now, 0, 2, interval, retry, false, true, "other")
	require.Equal(t, StatusRetrying, retrying.Status)
	require.Equal(t, 1, retrying.Failures)
	require.Equal(t, now.Add(retry), retrying.Next)
	require.Equal(t, "other", retrying.Detected)

	recovered := Decide(now, 1, 2, interval, retry, true, true, "gpt-4o")
	require.Equal(t, StatusPassed, recovered.Status)
	require.Equal(t, now.Add(interval), recovered.Next)

	suspected := Decide(now, 2, 2, interval, retry, false, true, "other")
	require.Equal(t, StatusSuspected, suspected.Status)
	require.Equal(t, "other", suspected.Detected)
	require.Equal(t, now.Add(interval), suspected.Next)
	require.Equal(t, 0, suspected.Failures)

	failed := Decide(now, 2, 2, interval, retry, false, false, "")
	require.Equal(t, StatusFailed, failed.Status)
	require.Empty(t, failed.Detected)

	immediate := Decide(now, 0, 0, interval, retry, false, true, "other")
	require.Equal(t, StatusSuspected, immediate.Status)
}

func TestMatchUsesRequestNameWhenExpectedIsBlank(t *testing.T) {
	passed, detected, scored := Match(ExpectedName("gpt-4o", "  "), "gpt-4o", "GPT-4o", "ranked")
	require.True(t, passed)
	require.True(t, scored)
	require.Equal(t, "GPT-4o", detected)

	passed, detected, scored = Match("custom-id", "custom-id", "Other", "partial")
	require.True(t, passed)
	require.Equal(t, "Other", detected)

	passed, detected, scored = Match("gpt-4o", "other", "Other", "unscorable")
	require.False(t, passed)
	require.False(t, scored)
	require.Empty(t, detected)
}

func TestEmbeddedDetectorReturnsPrediction(t *testing.T) {
	challenges, err := GenerateChallenges()
	require.NoError(t, err)
	require.Len(t, challenges, 3)

	samples := make([]Sample, 0, len(challenges))
	for _, challenge := range challenges {
		require.NotEmpty(t, challenge.Prompt)
		require.Positive(t, challenge.ExpectedCount)
		numbers := make([]string, challenge.ExpectedCount)
		for i := range numbers {
			numbers[i] = strconv.Itoa((i*17)%355 + 1)
		}
		samples = append(samples, Sample{Text: strings.Join(numbers, ", "), ExpectedCount: challenge.ExpectedCount})
	}

	analysis, err := Analyze(samples)
	require.NoError(t, err)
	require.NotEmpty(t, analysis.Decision)
	require.NotEmpty(t, analysis.Prediction)
}
