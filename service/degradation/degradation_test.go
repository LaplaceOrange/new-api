package degradation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
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

// The test executable acts as a deterministic Bun/CLI process. This exercises
// real exec, environment isolation, cancellation and HTTP without requiring a
// network connection or Bun installation for ordinary Go tests.
func TestMain(m *testing.M) {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version":
			fmt.Print("1.4.2")
			return
		case "install":
			os.Exit(runFixtureInstall())
		default:
			if filepath.Base(os.Args[1]) == "fpd.js" {
				os.Exit(runFixtureCLI())
			}
		}
	}
	os.Exit(m.Run())
}

type cliFixture struct {
	Mode   string          `json:"mode"`
	Result detectionResult `json:"result"`
}

func validDetectionFixture() detectionResult {
	var result detectionResult
	_ = common.Unmarshal([]byte(`{"schema":"fpd-detection-v1","requested_rounds":1,"completed_rounds":1,"scored_rounds":1,"bank":{"models":2,"reference_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"rounds":[{"samples":[{"state":"complete"},{"state":"complete"},{"state":"complete"}]}],"analysis":{"prediction":"example","prediction_name":"Example","decision":"not_confirmed"}}`), &result)
	return result
}

func writeFixtureInstallation(dir, version string, fixture cliFixture) error {
	base := filepath.Join(dir, "node_modules", "lmfpd")
	if err := os.MkdirAll(filepath.Join(base, "bin"), 0700); err != nil {
		return err
	}
	manifest, err := common.Marshal(map[string]string{"name": "lmfpd", "version": version})
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(base, "package.json"), manifest, 0600); err != nil {
		return err
	}
	data, err := common.Marshal(fixture)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(base, "bin", "fpd.js"), data, 0600)
}

func runFixtureInstall() int {
	if strings.Join(os.Args[2:], " ") != "--ignore-scripts --no-cache --registry https://registry.npmjs.org" {
		return 1
	}
	var manifest struct{ Dependencies map[string]string }
	data, err := os.ReadFile("package.json")
	if err != nil || common.Unmarshal(data, &manifest) != nil {
		return 1
	}
	fixture := cliFixture{Result: validDetectionFixture()}
	if data, err := os.ReadFile(filepath.Join("..", "install-fixture.json")); err == nil {
		if common.Unmarshal(data, &fixture) != nil || fixture.Mode == "install-error" {
			return 1
		}
	}
	if writeFixtureInstallation(".", manifest.Dependencies["lmfpd"], fixture) != nil {
		return 1
	}
	return 0
}

func runFixtureCLI() int {
	data, err := os.ReadFile(os.Args[1])
	var fixture cliFixture
	if err != nil || common.Unmarshal(data, &fixture) != nil {
		return 1
	}
	if os.Getenv("SQL_DSN") != "" || os.Getenv("NODE_OPTIONS") != "" || os.Getenv("BUN_OPTIONS") != "" || os.Getenv("HTTPS_PROXY") != "" {
		return 1
	}
	switch fixture.Mode {
	case "exit":
		fmt.Fprint(os.Stderr, "upstream secret and "+os.Getenv("API_KEY"))
		return 1
	case "invalid":
		fmt.Print("not JSON")
		return 0
	case "overflow":
		fmt.Print(strings.Repeat("x", maxCLIOutput+1))
		return 0
	}
	if os.Getenv("BASE_URL") != "" {
		client := &http.Client{Transport: &http.Transport{Proxy: nil}}
		for i := range 3 {
			body, _ := common.Marshal(map[string]any{
				"model": os.Getenv("MODEL"), "stream": false,
				"messages": []any{map[string]string{"role": "user", "content": fmt.Sprintf("challenge %d", i)}},
			})
			request, _ := http.NewRequest(http.MethodPost, os.Getenv("BASE_URL")+"/chat/completions", strings.NewReader(string(body)))
			request.Header.Set("Authorization", "Bearer "+os.Getenv("API_KEY"))
			response, err := client.Do(request)
			if err != nil {
				return 1
			}
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			if response.StatusCode != http.StatusOK {
				return 1
			}
		}
	}
	raw, _ := common.Marshal(fixture.Result)
	fmt.Print(string(raw))
	return 0
}

type registryFixture func(*http.Request) (*http.Response, error)

func (f registryFixture) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func runtimeFixture(t *testing.T) *cliRuntime {
	t.Helper()
	bun, err := os.Executable()
	require.NoError(t, err)
	runtime := &cliRuntime{bun: bun, root: t.TempDir(), bundled: t.TempDir(), autoUpdate: true, updating: make(chan struct{}, 1)}
	require.NoError(t, writeFixtureInstallation(runtime.bundled, "1.0.0", cliFixture{Result: validDetectionFixture()}))
	runtime.client = &http.Client{Transport: registryFixture(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "https://registry.npmjs.org/lmfpd/latest", r.URL.String())
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"name":"lmfpd","version":"1.0.1"}`))}, nil
	})}
	return runtime
}

func TestDetectionResultContract(t *testing.T) {
	cases := []struct {
		name   string
		change func(*detectionResult)
	}{
		{"schema", func(r *detectionResult) { r.Schema = "fpd-detection-v2" }},
		{"cancelled", func(r *detectionResult) { r.Cancelled = true }},
		{"unscorable", func(r *detectionResult) { r.Analysis.Decision = "unscorable" }},
		{"no prediction", func(r *detectionResult) { r.Analysis.Prediction = "" }},
		{"round error", func(r *detectionResult) { r.Rounds[0].Error = "secret" }},
		{"partial transport failure", func(r *detectionResult) { r.Rounds[0].Samples[0].State = "failed" }},
		{"missing sample", func(r *detectionResult) { r.Rounds[0].Samples = r.Rounds[0].Samples[:2] }},
		{"bank identity", func(r *detectionResult) { r.Bank.ReferenceSHA256 = "bad" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := validDetectionFixture()
			tc.change(&result)
			raw, err := common.Marshal(result)
			require.NoError(t, err)
			_, err = parseDetection(raw)
			require.Error(t, err)
			assert.NotContains(t, err.Error(), "secret")
		})
	}
}

func TestProbeBridgeBoundaries(t *testing.T) {
	var calls int
	bridge := &probeBridge{model: "target", token: "private-capability", seen: map[string]bool{}, probe: func(ctx context.Context, prompt string) (Completion, error) {
		calls++
		_, bounded := ctx.Deadline()
		assert.True(t, bounded)
		return Completion{Text: prompt, FinishReason: "length"}, nil
	}}
	request := func(token, origin, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Origin", origin)
		response := httptest.NewRecorder()
		bridge.ServeHTTP(response, req)
		return response
	}
	valid := `{"model":"target","messages":[{"role":"user","content":"one"}],"stream":false}`
	assert.Equal(t, 401, request("wrong", "", valid).Code)
	assert.Equal(t, 401, request(bridge.token, "https://untrusted.invalid", valid).Code)
	assert.Equal(t, 400, request(bridge.token, "", strings.Replace(valid, "target", "other", 1)).Code)
	assert.Equal(t, 400, request(bridge.token, "", strings.Replace(valid, "false", "true", 1)).Code)
	assert.Equal(t, 400, request(bridge.token, "", strings.Replace(valid, "one", strings.Repeat("x", 64<<10), 1)).Code)
	assert.Equal(t, 400, request(bridge.token, "", valid+` {}`).Code)
	assert.Zero(t, calls)
	response := request(bridge.token, "", valid)
	assert.Equal(t, 200, response.Code)
	assert.Contains(t, response.Body.String(), `"finish_reason":"length"`)
	assert.Equal(t, 429, request(bridge.token, "", valid).Code) // replay
	for _, prompt := range []string{"two", "three"} {
		assert.Equal(t, 200, request(bridge.token, "", strings.Replace(valid, "one", prompt, 1)).Code)
	}
	assert.Equal(t, 429, request(bridge.token, "", strings.Replace(valid, "one", "four", 1)).Code)
	assert.Equal(t, 3, calls)
}

func TestCLIExecutionAndCancellation(t *testing.T) {
	runtime := runtimeFixture(t)
	installed, err := runtime.load(t.Context())
	require.NoError(t, err)
	t.Setenv("SQL_DSN", "database-secret")
	t.Setenv("NODE_OPTIONS", "--require=secret")
	t.Setenv("BUN_OPTIONS", "--preload=secret")
	t.Setenv("HTTPS_PROXY", "http://proxy-secret.invalid")
	calls := 0
	var address string
	probe := func(ctx context.Context, _ string) (Completion, error) {
		calls++
		address = ctx.Value(http.LocalAddrContextKey).(net.Addr).String()
		return Completion{Text: "numbers", FinishReason: "stop"}, nil
	}
	analysis, err := runtime.detect(t.Context(), installed, "example", probe)
	require.NoError(t, err)
	assert.Equal(t, "example", analysis.Prediction)
	assert.Equal(t, 3, calls)
	conn, err := net.DialTimeout("tcp", address, time.Second)
	if conn != nil {
		_ = conn.Close()
	}
	require.Error(t, err, "the temporary capability endpoint must expire with the detection")
	for _, mode := range []string{"exit", "invalid", "overflow"} {
		t.Run(mode, func(t *testing.T) {
			require.NoError(t, writeFixtureInstallation(installed.Directory, installed.Version, cliFixture{Mode: mode, Result: validDetectionFixture()}))
			_, err := runtime.detect(t.Context(), installed, "example", probe)
			require.Error(t, err)
			assert.NotContains(t, err.Error(), "secret")
		})
	}
	require.NoError(t, writeFixtureInstallation(installed.Directory, installed.Version, cliFixture{Result: validDetectionFixture()}))
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	_, err = runtime.detect(ctx, installed, "example", func(ctx context.Context, _ string) (Completion, error) {
		cancel()
		<-ctx.Done()
		return Completion{}, ctx.Err()
	})
	assert.ErrorIs(t, err, context.Canceled)
	expired, stop := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	defer stop()
	_, err = runtime.detect(expired, installed, "example", probe)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestPackageActivationAndConcurrentDetection(t *testing.T) {
	runtime := runtimeFixture(t)
	old, err := runtime.load(t.Context())
	require.NoError(t, err)
	var registryCalls atomic.Int32
	transport := runtime.client.Transport
	runtime.client.Transport = registryFixture(func(r *http.Request) (*http.Response, error) {
		registryCalls.Add(1)
		return transport.RoundTrip(r)
	})
	started, resume := make(chan struct{}), make(chan struct{})
	probeErr := make(chan error, 1)
	go func() {
		first := true
		_, err := runtime.detect(t.Context(), old, "example", func(context.Context, string) (Completion, error) {
			if first {
				first = false
				close(started)
				<-resume
			}
			return Completion{Text: "numbers", FinishReason: "stop"}, nil
		})
		probeErr <- err
	}()
	select {
	case <-started:
	case err := <-probeErr:
		t.Fatalf("probe failed before update: %v", err)
	}
	var workers sync.WaitGroup
	for range 2 {
		workers.Go(func() { assert.NoError(t, runtime.check(t.Context())) })
	}
	workers.Wait()
	close(resume)
	require.NoError(t, <-probeErr)
	assert.EqualValues(t, 1, registryCalls.Load())
	current, err := runtime.load(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "1.0.1", current.Version)
	assert.NotEqual(t, old.Directory, current.Directory)
	assert.FileExists(t, filepath.Join(old.Directory, "node_modules", "lmfpd", "bin", "fpd.js"))
	assert.WithinDuration(t, time.Now().Add(24*time.Hour), runtime.nextCheck, time.Minute)
	// A fresh process must recover the persisted, validated version offline.
	restarted := runtimeFixture(t)
	restarted.root = runtime.root
	restarted.autoUpdate = false
	restarted.bundled = filepath.Join(t.TempDir(), "missing")
	recovered, err := restarted.ready(t.Context())
	require.NoError(t, err)
	assert.Equal(t, current, recovered)
	// Replacing an existing pointer must also work on Windows.
	oldTime := time.Now().Add(-7 * 24 * time.Hour)
	require.NoError(t, os.Chtimes(current.Directory, oldTime, oldTime))
	runtime.nextCheck = time.Time{}
	runtime.client.Transport = registryFixture(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"name":"lmfpd","version":"1.0.2"}`))}, nil
	})
	require.NoError(t, runtime.check(t.Context()))
	previousInfo, err := os.Stat(current.Directory)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), previousInfo.ModTime(), time.Minute, "retention starts when a package is retired")
	restarted.active = nil
	recovered, err = restarted.ready(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "1.0.2", recovered.Version)
}

func TestPackageRetentionProtectsActiveAndRecentVersions(t *testing.T) {
	runtime := runtimeFixture(t)
	for i, name := range []string{"lmfpd-expired", "lmfpd-active", "lmfpd-previous", "lmfpd-recent", "unrelated"} {
		dir := filepath.Join(runtime.root, name)
		require.NoError(t, os.Mkdir(dir, 0700))
		modified := time.Now().Add(-time.Duration(10-i) * 24 * time.Hour)
		if name == "lmfpd-recent" {
			modified = time.Now()
		}
		require.NoError(t, os.Chtimes(dir, modified, modified))
	}
	runtime.prunePackages(filepath.Join(runtime.root, "lmfpd-active"))
	assert.NoDirExists(t, filepath.Join(runtime.root, "lmfpd-expired"))
	for _, name := range []string{"lmfpd-active", "lmfpd-previous", "lmfpd-recent", "unrelated"} {
		assert.DirExists(t, filepath.Join(runtime.root, name))
	}
}

func TestPackageUpdateFailuresRetainWorkingVersion(t *testing.T) {
	for _, mode := range []string{"network", "install-error", "invalid"} {
		t.Run(mode, func(t *testing.T) {
			runtime := runtimeFixture(t)
			old, err := runtime.load(t.Context())
			require.NoError(t, err)
			if mode == "network" {
				runtime.client.Transport = registryFixture(func(*http.Request) (*http.Response, error) { return nil, errors.New("offline") })
			} else {
				data, err := common.Marshal(cliFixture{Mode: mode, Result: validDetectionFixture()})
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(runtime.root, "install-fixture.json"), data, 0600))
			}
			require.Error(t, runtime.check(t.Context()))
			current, err := runtime.ready(t.Context())
			require.NoError(t, err)
			assert.Equal(t, old, current)
			retry := time.Hour
			if mode == "invalid" {
				retry = 24 * time.Hour
			}
			assert.WithinDuration(t, time.Now().Add(retry), runtime.nextCheck, time.Minute)
			assert.NoFileExists(t, filepath.Join(runtime.root, "active.json"))
		})
	}
	runtime := runtimeFixture(t)
	runtime.autoUpdate = false
	runtime.bundled = filepath.Join(t.TempDir(), "missing")
	_, err := runtime.ready(t.Context())
	require.ErrorContains(t, err, "no usable lmfpd")
	runtime.bun = filepath.Join(t.TempDir(), "no-bun")
	_, err = runtime.ready(t.Context())
	require.ErrorContains(t, err, "Bun >= 1.4.2")
}

func TestOfficialCLI(t *testing.T) {
	bun, dir := os.Getenv("LMFPD_TEST_BUN"), os.Getenv("LMFPD_TEST_PACKAGE_DIR")
	if bun == "" || dir == "" {
		t.Skip("set LMFPD_TEST_BUN and LMFPD_TEST_PACKAGE_DIR for the real-package integration test")
	}
	runtime := &cliRuntime{bun: bun, root: t.TempDir(), bundled: dir, updating: make(chan struct{}, 1)}
	installed, err := runtime.ready(t.Context())
	require.NoError(t, err)
	numbers := make([]string, 300)
	for i := range numbers {
		numbers[i] = strconv.Itoa((i*73)%355 + 1)
	}
	calls := 0
	analysis, err := runtime.detect(t.Context(), installed, "example", func(_ context.Context, prompt string) (Completion, error) {
		calls++
		assert.NotEmpty(t, prompt)
		return Completion{Text: strings.Join(numbers, ", "), FinishReason: "stop"}, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 3, calls)
	assert.NotEmpty(t, analysis.Prediction)
	assert.NotEmpty(t, analysis.Decision)
	if os.Getenv("LMFPD_TEST_INSTALL") == "true" {
		bootstrap := &cliRuntime{
			bun: bun, root: t.TempDir(), bundled: filepath.Join(t.TempDir(), "missing"),
			autoUpdate: true, updating: make(chan struct{}, 1), client: &http.Client{Timeout: 30 * time.Second},
		}
		downloaded, err := bootstrap.ready(t.Context())
		require.NoError(t, err)
		assert.NotEmpty(t, downloaded.Version)
		assert.NotEmpty(t, downloaded.ReferenceSHA256)
		assert.FileExists(t, filepath.Join(bootstrap.root, "active.json"))
		// A real update must survive restart with networking explicitly disabled.
		bootstrap.active = nil
		bootstrap.autoUpdate = false
		recovered, err := bootstrap.ready(t.Context())
		require.NoError(t, err)
		assert.Equal(t, downloaded, recovered)
	}
}

func TestMultipleExpectedNamesMatchExactly(t *testing.T) {
	for _, tc := range []struct {
		name, expected, prediction, display, decision string
		passed, scored                                bool
	}{
		{"legacy", "model-v1", "model-v1", "Model", "ranked", true, true},
		{"second version", "model-v1\nmodel-v2", "model-v2", "Model 2", "ranked", true, true},
		{"display name", "model-v1\n Model 2 ", "other", "Model 2", "ranked", true, true},
		{"no substring matching", "model-v1\nmodel-v2", "model-v20", "Model 20", "ranked", false, true},
		{"unscorable", "model-v1\nmodel-v2", "model-v2", "Model 2", "unscorable", false, false},
		{"empty defaults to request", " ", "request-model", "Model", "ranked", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			passed, _, scored := Match(ExpectedName("request-model", tc.expected), tc.prediction, tc.display, tc.decision)
			assert.Equal(t, tc.passed, passed)
			assert.Equal(t, tc.scored, scored)
		})
	}
	normalized, err := NormalizeExpectedNames(" model-v1 \r\n\nmodel-v2\nmodel-v1 ")
	require.NoError(t, err)
	assert.Equal(t, "model-v1\nmodel-v2", normalized)
	_, err = NormalizeExpectedNames(strings.Repeat("a", 257))
	require.Error(t, err)
	var names []string
	for i := range 33 {
		names = append(names, fmt.Sprint("model-", i))
	}
	_, err = NormalizeExpectedNames(strings.Join(names, "\n"))
	require.Error(t, err)
}
