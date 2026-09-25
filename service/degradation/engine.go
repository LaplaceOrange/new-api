package degradation

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"fmt"
	"io"
	"sync"

	"github.com/QuantumNous/new-api/common"

	_ "embed"
	"github.com/grafana/sobek"
)

//go:embed data/unified_bank.json.gz
var bankGzip []byte

//go:embed data/shared_detector.json.gz
var detectorGzip []byte

//go:embed detector.bundle.js
var detectorBundle string

type Challenge struct {
	ID            string `json:"id"`
	ExpectedCount int    `json:"expected_count"`
	Prompt        string `json:"prompt"`
}

type Sample struct {
	Text          string `json:"text"`
	ExpectedCount int    `json:"expected_count"`
}

type Analysis struct {
	Prediction     string `json:"prediction"`
	PredictionName string `json:"prediction_name"`
	Decision       string `json:"decision"`
}

var (
	engineOnce sync.Once
	engineErr  error
	engineVM   *sobek.Runtime
	engineMu   sync.Mutex
)

func gunzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func randomUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "00000000-0000-4000-8000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func initEngine() {
	bank, err := gunzip(bankGzip)
	if err != nil {
		engineErr = fmt.Errorf("decode fingerprint bank: %w", err)
		return
	}
	detector, err := gunzip(detectorGzip)
	if err != nil {
		engineErr = fmt.Errorf("decode fingerprint detector: %w", err)
		return
	}
	vm := sobek.New()
	if err = vm.Set("goRandomBytes", func(n int) []int {
		if n < 0 || n > 1<<20 {
			return []int{}
		}
		buf := make([]byte, n)
		if _, readErr := rand.Read(buf); readErr != nil {
			return []int{}
		}
		out := make([]int, n)
		for i, value := range buf {
			out[i] = int(value)
		}
		return out
	}); err != nil {
		engineErr = err
		return
	}
	if err = vm.Set("goRandomUUID", randomUUID); err != nil {
		engineErr = err
		return
	}
	if _, err = vm.RunString(`globalThis.crypto = {
  getRandomValues(buffer) {
    const bytes = goRandomBytes(buffer.byteLength);
    const view = new Uint8Array(buffer.buffer, buffer.byteOffset, buffer.byteLength);
    for (let i = 0; i < bytes.length; i++) view[i] = bytes[i];
    return buffer;
  },
  randomUUID() { return goRandomUUID(); }
};`); err != nil {
		engineErr = fmt.Errorf("install crypto: %w", err)
		return
	}
	if _, err = vm.RunString(detectorBundle); err != nil {
		engineErr = fmt.Errorf("load detector: %w", err)
		return
	}
	if err = vm.Set("bankJSON", string(bank)); err != nil {
		engineErr = err
		return
	}
	if err = vm.Set("detectorJSON", string(detector)); err != nil {
		engineErr = err
		return
	}
	if _, err = vm.RunString(`globalThis.lmBank = JSON.parse(bankJSON);
globalThis.lmDetector = JSON.parse(detectorJSON);
bankJSON = "";
detectorJSON = "";`); err != nil {
		engineErr = fmt.Errorf("parse fingerprint data: %w", err)
		return
	}
	engineVM = vm
}

func ensureEngine() error {
	engineOnce.Do(initEngine)
	return engineErr
}

func callJSON(script string) ([]byte, error) {
	if err := ensureEngine(); err != nil {
		return nil, err
	}
	engineMu.Lock()
	defer engineMu.Unlock()
	value, err := engineVM.RunString(script)
	if err != nil {
		return nil, err
	}
	exported, ok := value.Export().(string)
	if !ok {
		return nil, fmt.Errorf("detector returned %T", value.Export())
	}
	return []byte(exported), nil
}

func GenerateChallenges() ([]Challenge, error) {
	raw, err := callJSON(`JSON.stringify(lmGenerateChallenges())`)
	if err != nil {
		return nil, err
	}
	var challenges []Challenge
	if err := common.Unmarshal(raw, &challenges); err != nil {
		return nil, err
	}
	if len(challenges) != 3 {
		return nil, fmt.Errorf("detector generated %d challenges", len(challenges))
	}
	return challenges, nil
}

func Analyze(samples []Sample) (Analysis, error) {
	payload, err := common.Marshal(samples)
	if err != nil {
		return Analysis{}, err
	}
	if err = ensureEngine(); err != nil {
		return Analysis{}, err
	}
	engineMu.Lock()
	defer engineMu.Unlock()
	if err = engineVM.Set("outputsJSON", string(payload)); err != nil {
		return Analysis{}, err
	}
	value, err := engineVM.RunString("JSON.stringify(lmAnalyze(JSON.parse(outputsJSON), lmBank, lmDetector))")
	if err != nil {
		return Analysis{}, err
	}
	exported, ok := value.Export().(string)
	if !ok {
		return Analysis{}, fmt.Errorf("detector returned %T", value.Export())
	}
	var analysis Analysis
	if err = common.Unmarshal([]byte(exported), &analysis); err != nil {
		return Analysis{}, err
	}
	return analysis, nil
}
