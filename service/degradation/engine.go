package degradation

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	probeTimeout = 180 * time.Second
	maxCLIOutput = 4 << 20
)

type Analysis struct {
	Prediction     string `json:"prediction"`
	PredictionName string `json:"prediction_name"`
	Decision       string `json:"decision"`
}

// Completion preserves the host relay's completion status. A length-limited
// or filtered answer must not be presented to the CLI as complete.
type Completion struct {
	Text         string
	FinishReason string
}

type Probe func(context.Context, string) (Completion, error)

type detectionResult struct {
	Schema          string `json:"schema"`
	Cancelled       bool   `json:"cancelled"`
	RequestedRounds int    `json:"requested_rounds"`
	CompletedRounds int    `json:"completed_rounds"`
	ScoredRounds    int    `json:"scored_rounds"`
	Bank            struct {
		ReferenceSHA256 string `json:"reference_sha256"`
		Models          int    `json:"models"`
	} `json:"bank"`
	Rounds []struct {
		Error   string `json:"error"`
		Samples []struct {
			State string `json:"state"`
		} `json:"samples"`
	} `json:"rounds"`
	Analysis Analysis `json:"analysis"`
}

func parseDetection(raw []byte) (detectionResult, error) {
	var result detectionResult
	if err := common.Unmarshal(raw, &result); err != nil {
		return result, errors.New("lmfpd returned invalid JSON")
	}
	if result.Schema != "fpd-detection-v1" || result.Cancelled || result.RequestedRounds != 1 ||
		result.CompletedRounds != 1 || result.ScoredRounds != 1 || len(result.Rounds) != 1 ||
		result.Bank.Models <= 0 || len(result.Bank.ReferenceSHA256) != 64 {
		return result, errors.New("lmfpd returned an incompatible or unscorable result")
	}
	if _, err := hex.DecodeString(result.Bank.ReferenceSHA256); err != nil {
		return result, errors.New("lmfpd returned an invalid bank identity")
	}
	round := result.Rounds[0]
	if round.Error != "" || len(round.Samples) != 3 {
		return result, errors.New("lmfpd did not complete three samples")
	}
	for _, sample := range round.Samples {
		if sample.State != "complete" && sample.State != "truncated" {
			return result, errors.New("lmfpd sample failed")
		}
	}
	if strings.TrimSpace(result.Analysis.Prediction) == "" || result.Analysis.Decision == "" || result.Analysis.Decision == "unscorable" {
		return result, errors.New("lmfpd could not score the samples")
	}
	return result, nil
}

// boundedOutput prevents a faulty package from exhausting host memory. Stderr
// is never logged: it may contain echoed samples or credentials.
type boundedOutput struct{ buffer bytes.Buffer }

func (b *boundedOutput) Write(p []byte) (int, error) {
	if len(p) > maxCLIOutput-b.buffer.Len() {
		return 0, errors.New("lmfpd output limit exceeded")
	}
	return b.buffer.Write(p)
}

func runBun(ctx context.Context, bun, dir string, args, extraEnv []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, bun, args...)
	cmd.Dir = dir
	// Do not inherit DB credentials, channel secrets, NODE_OPTIONS, Bun preload
	// options, or proxy settings into the detector. Its only HTTP peer is loopback.
	cmd.Env = []string{"CI=1", "NO_COLOR=1", "FPD_NO_UPDATE_CHECK=1", "BUN_BE_BUN=1", "BUN_RUNTIME_TRANSPILER_CACHE_PATH=0", "NO_PROXY=*", "HOME=" + dir, "USERPROFILE=" + dir}
	for _, name := range []string{"PATH", "SystemRoot", "WINDIR", "TEMP", "TMP", "TMPDIR"} {
		if value, ok := os.LookupEnv(name); ok {
			cmd.Env = append(cmd.Env, name+"="+value)
		}
	}
	cmd.Env = append(cmd.Env, extraEnv...)
	var output boundedOutput
	cmd.Stdout = &output
	cmd.Stderr = io.Discard
	cmd.WaitDelay = 2 * time.Second
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("lmfpd process failed: %w", err)
	}
	return output.buffer.Bytes(), nil
}

type probeBridge struct {
	model, token string
	probe        Probe
	mu           sync.Mutex
	requests     int
	failed       bool
	seen         map[string]bool
	authFailure  sync.Once
}

// Following OWASP ASVS 5.0.0 and the Authentication/Session Management cheat
// sheets, this capability is random, short-lived, scoped and never logged.
// It is local process IPC, independent of browser sessions and API tokens.
func (b *probeBridge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+b.token)) != 1 || r.Header.Get("Origin") != "" {
		b.authFailure.Do(func() {
			common.SysError(fmt.Sprintf("degradation probe authorization rejected model=%q", b.model))
		})
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" || r.URL.RawQuery != "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var request struct {
		Model    string `json:"model"`
		Stream   bool   `json:"stream"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
	if err != nil || common.Unmarshal(raw, &request) != nil ||
		request.Model != b.model || request.Stream || len(request.Messages) != 1 ||
		request.Messages[0].Role != "user" || strings.TrimSpace(request.Messages[0].Content) == "" {
		http.Error(w, "invalid probe", http.StatusBadRequest)
		return
	}
	// Serialize even if a future CLI ignores --parallel 1. Failed requests and
	// replayed prompts cannot dispatch another upstream call in this attempt.
	b.mu.Lock()
	defer b.mu.Unlock()
	prompt := request.Messages[0].Content
	if b.failed || b.requests >= 3 || b.seen[prompt] || r.Context().Err() != nil {
		http.Error(w, "probe exhausted", http.StatusTooManyRequests)
		return
	}
	b.requests++
	b.seen[prompt] = true
	ctx, cancel := context.WithTimeout(r.Context(), probeTimeout)
	defer cancel()
	completion, err := b.probe(ctx, prompt)
	if err != nil || ctx.Err() != nil || len(completion.Text) > 1<<20 {
		b.failed = true
		http.Error(w, "upstream probe failed", http.StatusBadGateway)
		return
	}
	payload, err := common.Marshal(map[string]any{
		"model": b.model,
		"choices": []any{map[string]any{
			"index": 0, "finish_reason": completion.FinishReason,
			"message": map[string]any{"role": "assistant", "content": completion.Text},
		}},
	})
	if err != nil {
		b.failed = true
		http.Error(w, "invalid completion", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(payload)
}

// Detect pins one immutable installation for the entire challenge/score cycle.
func Detect(ctx context.Context, model string, probe Probe) (Analysis, error) {
	runtime := detectorRuntime()
	installed, err := runtime.ready(ctx)
	if err != nil {
		return Analysis{}, err
	}
	return runtime.detect(ctx, installed, model, probe)
}

func (r *cliRuntime) detect(ctx context.Context, installed installation, model string, probe Probe) (Analysis, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*probeTimeout+30*time.Second)
	defer cancel()
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return Analysis{}, err
	}
	bridge := &probeBridge{model: model, token: hex.EncodeToString(secret[:]), probe: probe, seen: map[string]bool{}}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return Analysis{}, err
	}
	server := &http.Server{
		Handler: bridge, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second,
		WriteTimeout: probeTimeout + 5*time.Second, IdleTimeout: 5 * time.Second, MaxHeaderBytes: 8 << 10,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}
	defer server.Close()
	go func() { _ = server.Serve(listener) }()
	args := []string{filepath.Join(installed.Directory, "node_modules", "lmfpd", "bin", "fpd.js"),
		"--api", "chatcompletion", "--count", "3", "--parallel", "1", "--repeat", "1",
		"--no-stream", "--timeout", "180", "--json", "--no-update-check"}
	output, err := runBun(ctx, r.bun, installed.Directory, args, []string{
		"API_KEY=" + bridge.token, "MODEL=" + model, "BASE_URL=http://" + listener.Addr().String() + "/v1",
	})
	if err != nil {
		return Analysis{}, err
	}
	bridge.mu.Lock()
	complete := bridge.requests == 3 && !bridge.failed
	bridge.mu.Unlock()
	if !complete {
		return Analysis{}, errors.New("lmfpd did not execute all three probes")
	}
	result, err := parseDetection(output)
	if err != nil {
		return Analysis{}, err
	}
	if result.Bank.ReferenceSHA256 != installed.ReferenceSHA256 {
		return Analysis{}, errors.New("lmfpd bank changed during detection")
	}
	common.SysLog(fmt.Sprintf("degradation detection version=%s bank=%s", installed.Version, installed.ReferenceSHA256))
	return result.Analysis, nil
}
