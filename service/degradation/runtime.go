package degradation

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
)

//go:embed runtime/samples.json
var validationSamples []byte

var packageVersion = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?$`)

type installation struct {
	Directory       string    `json:"directory"`
	Version         string    `json:"version"`
	ReferenceSHA256 string    `json:"reference_sha256"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type cliRuntime struct {
	bun, root, bundled string
	autoUpdate         bool
	client             *http.Client
	mu                 sync.Mutex
	active             *installation
	updating           chan struct{}
	nextCheck          time.Time // guarded by updating
	lastUpdateErr      error
}

var detectorRuntime = sync.OnceValue(func() *cliRuntime {
	bun := os.Getenv("DEGRADATION_BUN_PATH")
	if bun == "" {
		bun = "bun"
	} else if strings.ContainsAny(bun, `/\`) {
		bun, _ = filepath.Abs(bun)
	}
	root := os.Getenv("DEGRADATION_CACHE_DIR")
	if root == "" {
		root = filepath.Join("data", "degradation")
	}
	// Different workers must not publish competing pointers into a shared /data.
	// NODE_NAME is already the host's stable instance identity.
	node := sha256.Sum256([]byte(common.NodeName))
	root, _ = filepath.Abs(filepath.Join(root, fmt.Sprintf("%x", node[:8])))
	bundled := os.Getenv("DEGRADATION_BUNDLED_DIR")
	if bundled == "" {
		bundled = "/opt/new-api/lmfpd"
	}
	bundled, _ = filepath.Abs(bundled)
	return &cliRuntime{
		bun: bun, root: root, bundled: bundled, autoUpdate: os.Getenv("DEGRADATION_AUTO_UPDATE") != "false",
		updating: make(chan struct{}, 1),
		client: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("npm registry redirect refused")
		}},
	}
})

func (r *cliRuntime) load(ctx context.Context) (installation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active != nil {
		return *r.active, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	if err := r.verifyBun(ctx); err != nil {
		return installation{}, err
	}
	var saved installation
	if data, readErr := os.ReadFile(filepath.Join(r.root, "active.json")); readErr == nil {
		if common.Unmarshal(data, &saved) == nil && filepath.IsLocal(saved.Directory) && filepath.Base(saved.Directory) == saved.Directory && saved.Directory != "." {
			// Persist only a basename; an active pointer cannot select arbitrary code.
			installed, validateErr := r.validate(ctx, filepath.Join(r.root, saved.Directory), saved.Version)
			if validateErr == nil && installed.ReferenceSHA256 == saved.ReferenceSHA256 {
				installed.UpdatedAt = saved.UpdatedAt
				r.active = &installed
				return installed, nil
			}
		}
		common.SysError("degradation cached package invalid; trying bundled package")
	}
	installed, err := r.validate(ctx, r.bundled, "")
	if err != nil {
		return installation{}, fmt.Errorf("no usable lmfpd package: %w", err)
	}
	r.active = &installed
	return installed, nil
}

func (r *cliRuntime) verifyBun(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	version, err := runBun(ctx, r.bun, os.TempDir(), []string{"--version"}, nil)
	if err != nil {
		return fmt.Errorf("degradation requires Bun >= 1.4.2 (DEGRADATION_BUN_PATH): %w", err)
	}
	parts := strings.Split(strings.TrimSpace(string(version)), ".")
	if len(parts) != 3 {
		return errors.New("unrecognized Bun version; Bun >= 1.4.2 is required")
	}
	var numbers [3]int
	for i, part := range parts {
		numbers[i], err = strconv.Atoi(part)
		if err != nil || numbers[i] < 0 {
			return errors.New("unrecognized Bun version; Bun >= 1.4.2 is required")
		}
	}
	if numbers[0] < 1 || (numbers[0] == 1 && (numbers[1] < 4 || (numbers[1] == 4 && numbers[2] < 2))) {
		return errors.New("degradation requires Bun >= 1.4.2; upgrade DEGRADATION_BUN_PATH")
	}
	return nil
}

func (r *cliRuntime) ready(ctx context.Context) (installation, error) {
	installed, err := r.load(ctx)
	if err == nil || !r.autoUpdate {
		return installed, err
	}
	if err := r.check(ctx); err != nil {
		return installation{}, err
	}
	return r.load(ctx)
}

func (r *cliRuntime) validate(ctx context.Context, dir, expectedVersion string) (installation, error) {
	manifest, err := os.ReadFile(filepath.Join(dir, "node_modules", "lmfpd", "package.json"))
	if err != nil {
		return installation{}, errors.New("lmfpd package is not installed")
	}
	var identity struct{ Name, Version string }
	if common.Unmarshal(manifest, &identity) != nil || identity.Name != "lmfpd" || len(identity.Version) > 128 ||
		!packageVersion.MatchString(identity.Version) || (expectedVersion != "" && identity.Version != expectedVersion) {
		return installation{}, errors.New("lmfpd package identity mismatch")
	}
	input, err := os.CreateTemp("", "lmfpd-validation-*.json")
	if err != nil {
		return installation{}, err
	}
	defer os.Remove(input.Name())
	_, writeErr := input.Write(validationSamples)
	closeErr := input.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return installation{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	raw, err := runBun(ctx, r.bun, dir, []string{
		filepath.Join(dir, "node_modules", "lmfpd", "bin", "fpd.js"),
		"--input", input.Name(), "--json", "--no-update-check",
	}, nil)
	if err != nil {
		return installation{}, err
	}
	result, err := parseDetection(raw)
	if err != nil {
		return installation{}, err
	}
	return installation{Directory: dir, Version: identity.Version, ReferenceSHA256: result.Bank.ReferenceSHA256}, nil
}

func (r *cliRuntime) check(ctx context.Context) (err error) {
	select {
	case r.updating <- struct{}{}:
		defer func() { <-r.updating }()
	case <-ctx.Done():
		return ctx.Err()
	}
	if time.Now().Before(r.nextCheck) {
		return r.lastUpdateErr
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	retry := time.Hour
	defer func() {
		r.nextCheck = time.Now().Add(retry)
		r.lastUpdateErr = err
		if err != nil {
			common.SysError(fmt.Sprintf("degradation package update failed; retaining current version, next_check=%s err=%v", r.nextCheck.UTC().Format(time.RFC3339), err))
		}
	}()
	if err := r.verifyBun(ctx); err != nil {
		return err
	}
	installed, _ := r.load(ctx)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://registry.npmjs.org/lmfpd/latest", nil)
	if err != nil {
		return err
	}
	response, err := r.client.Do(request)
	if err != nil {
		return fmt.Errorf("check npm registry: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("npm registry returned HTTP %d", response.StatusCode)
	}
	var metadata struct{ Name, Version string }
	if err := common.DecodeJson(io.LimitReader(response.Body, 1<<20), &metadata); err != nil ||
		metadata.Name != "lmfpd" || len(metadata.Version) > 128 || !packageVersion.MatchString(metadata.Version) {
		retry = 24 * time.Hour
		return errors.New("invalid lmfpd registry metadata")
	}
	if installed.Version == metadata.Version {
		retry = 24 * time.Hour
		r.prunePackages(installed.Directory)
		common.SysLog(fmt.Sprintf("degradation package current version=%s bank=%s checked_at=%s", installed.Version, installed.ReferenceSHA256, time.Now().UTC().Format(time.RFC3339)))
		return nil
	}
	if err := os.MkdirAll(r.root, 0700); err != nil {
		return err
	}
	dir, err := os.MkdirTemp(r.root, "lmfpd-"+metadata.Version+"-")
	if err != nil {
		return err
	}
	published := false
	defer func() {
		if !published {
			_ = os.RemoveAll(dir)
		}
	}()
	manifest, err := common.Marshal(map[string]any{"private": true, "dependencies": map[string]string{"lmfpd": metadata.Version}})
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), manifest, 0600); err != nil {
		return err
	}
	installEnv := []string{"BUN_INSTALL_CACHE_DIR=" + filepath.Join(r.root, "npm-cache"), "NO_PROXY="}
	for _, name := range []string{"HTTPS_PROXY", "HTTP_PROXY", "NO_PROXY"} {
		if value, ok := os.LookupEnv(name); ok {
			installEnv = append(installEnv, name+"="+value)
		}
	}
	// Bun verifies registry integrity hashes by default. Resolve an exact version
	// in an isolated project; never run lifecycle scripts or install during detect.
	if _, err := runBun(ctx, r.bun, dir, []string{"install", "--ignore-scripts", "--no-cache", "--registry", "https://registry.npmjs.org"}, installEnv); err != nil {
		return fmt.Errorf("install lmfpd %s: %w", metadata.Version, err)
	}
	candidate, err := r.validate(ctx, dir, metadata.Version)
	if err != nil {
		retry = 24 * time.Hour
		return fmt.Errorf("reject lmfpd %s: %w", metadata.Version, err)
	}
	candidate.UpdatedAt = time.Now().UTC()
	saved := candidate
	saved.Directory = filepath.Base(candidate.Directory)
	data, err := common.Marshal(saved)
	if err != nil {
		return err
	}
	// A unique temp file and rename leave the previous pointer intact on failure.
	state, err := os.CreateTemp(r.root, "active-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(state.Name())
	_, writeErr := state.Write(data)
	syncErr := state.Sync()
	closeErr := state.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return err
	}
	if filepath.Dir(installed.Directory) == r.root {
		// Measure retention from deactivation, not installation: a months-old
		// package can still be serving a probe when its successor is activated.
		if err := os.Chtimes(installed.Directory, candidate.UpdatedAt, candidate.UpdatedAt); err != nil {
			return err
		}
	}
	if err := os.Rename(state.Name(), filepath.Join(r.root, "active.json")); err != nil {
		return err
	}
	r.mu.Lock()
	r.active = &candidate
	r.mu.Unlock()
	published = true
	// A detection pins its installation and has a 570-second hard deadline.
	// Retain the last two packages and a much longer grace period for older ones.
	r.prunePackages(candidate.Directory)
	retry = 24 * time.Hour
	common.SysLog(fmt.Sprintf("degradation package updated version=%s bank=%s updated_at=%s", candidate.Version, candidate.ReferenceSHA256, candidate.UpdatedAt.Format(time.RFC3339)))
	return nil
}

func (r *cliRuntime) prunePackages(active string) {
	entries, err := os.ReadDir(r.root)
	if err != nil {
		return
	}
	type cachedPackage struct {
		name     string
		modified time.Time
	}
	var packages []cachedPackage
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "lmfpd-") {
			continue
		}
		info, err := entry.Info()
		if err == nil {
			packages = append(packages, cachedPackage{entry.Name(), info.ModTime()})
		}
	}
	slices.SortFunc(packages, func(a, b cachedPackage) int { return b.modified.Compare(a.modified) })
	for i, pkg := range packages {
		path := filepath.Join(r.root, pkg.name)
		if i < 2 || path == active || time.Since(pkg.modified) < 48*time.Hour {
			continue
		}
		if err := os.RemoveAll(path); err != nil {
			common.SysError("degradation could not remove an expired package directory")
		}
	}
}

// StartUpdater is local to each process, not a DB-leased system task: every
// worker needs its own runnable package even when another worker owns the probe.
func StartUpdater(ctx context.Context) {
	runtime := detectorRuntime()
	if !runtime.autoUpdate {
		return
	}
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			if ctx.Err() != nil {
				return
			}
			if LoadConfig().Enabled {
				_ = runtime.check(ctx)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}
