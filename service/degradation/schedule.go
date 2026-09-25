package degradation

import (
	"strings"
	"time"
)

const (
	StatusPending     = "pending"
	StatusPassed      = "passed"
	StatusRetrying    = "retrying"
	StatusSuspected   = "suspected"
	StatusFailed      = "failed"
	StatusUnavailable = "unavailable"

	OptionKey = "DegradationMonitor"
)

type Verdict struct {
	Status   string
	Detected string
	Next     time.Time
	Failures int
}

// Decide applies one finished attempt. retryCount is the number of extra
// attempts allowed after the first failure. A scored mismatch can become
// suspected only after those retries are exhausted. Transport errors and
// unscorable results never become suspected.
func Decide(now time.Time, failures, retryCount int, interval, retryInterval time.Duration, passed, scored bool, detected string) Verdict {
	if passed {
		return Verdict{Status: StatusPassed, Detected: detected, Next: now.Add(interval), Failures: 0}
	}
	nextFailures := failures + 1
	if nextFailures <= retryCount {
		return Verdict{Status: StatusRetrying, Detected: detected, Next: now.Add(retryInterval), Failures: nextFailures}
	}
	if scored && strings.TrimSpace(detected) != "" {
		return Verdict{Status: StatusSuspected, Detected: strings.TrimSpace(detected), Next: now.Add(interval), Failures: 0}
	}
	return Verdict{Status: StatusFailed, Detected: "", Next: now.Add(interval), Failures: 0}
}

func ExpectedName(modelName, expected string) string {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return strings.TrimSpace(modelName)
	}
	return expected
}

// Match reports whether the detector identity equals the expected name.
// Either the raw prediction id or the display name is accepted.
func Match(expected, prediction, predictionName, decision string) (passed bool, detected string, scored bool) {
	expected = strings.TrimSpace(expected)
	prediction = strings.TrimSpace(prediction)
	predictionName = strings.TrimSpace(predictionName)
	if decision == "unscorable" {
		return false, "", false
	}
	if predictionName == "" || predictionName == "Not scored" {
		detected = prediction
	} else {
		detected = predictionName
	}
	if prediction == "" && detected == "" {
		return false, "", false
	}
	passed = expected != "" && (expected == prediction || expected == predictionName)
	return passed, detected, true
}
