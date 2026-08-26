# Multi-Provider Tokenizer & Offline Token Counter Implementation Plan (Phase 1: Multi-Engine Foundation)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Transform `tokenizer-go` into a multi-provider offline tokenizer runtime with deterministic local execution, model-aware counting (`CountForModel`), resolution (`ResolveModel`), and transparent fallback tracking.

**Architecture:** Define core `Engine`, `Provider`, `Capabilities`, `Accuracy`, `CountResult`, and `Resolution` abstractions. Introduce an immutable model registry that maps canonical names, aliases, and prefixes to exact tokenizers or fallbacks. Wrap existing OpenAI BPE core into the new `Engine` interface while keeping existing `ForModel` exact-only and 100% backward compatible.

**Tech Stack:** Go 1.21+, pure Go, zero external CGO, table-driven tests, fuzzing, race detector.

## Global Constraints

- No external API requests (OpenAI, Anthropic, Google, xAI, etc.). No network calls.
- Pure Go, thread-safe, zero CGO.
- Backward compatibility for existing OpenAI APIs (`GetEncoding`, `ForModel`, `EncodingForModel`, `Encode`, `Decode`, `Count`).
- `ForModel` MUST remain exact-only and return `ErrExactTokenizerUnavailable` for fallback-only models.
- `CountForModel` provides model-aware token counting with explicit accuracy labels (`AccuracyExactLocal`, `AccuracyEstimatedCalibrated`, `AccuracyEstimatedHeuristic`).
- Zero heap allocations for model resolution lookups.
- OpenAI benchmark regression MUST NOT exceed 3%.

---

### Task 1: Core Engine, Capabilities, and Provider Contracts

**Files:**
- Create: `engine.go`
- Create: `capabilities.go`
- Create: `provider.go`
- Test: `engine_test.go`

**Interfaces:**
- Consumes: Go standard library
- Produces: `Engine`, `EngineMetadata`, `Algorithm`, `Capabilities`, `Provider`

- [ ] **Step 1: Write failing unit test for Engine interface and metadata**

```go
package tokenizer_test

import (
	"testing"

	"github.com/xorvus/tokenizer-go"
)

type dummyEngine struct{}

func (d *dummyEngine) Encode(text string) ([]int, error) { return []int{1, 2}, nil }
func (d *dummyEngine) Decode(tokens []int) (string, error) { return "hello", nil }
func (d *dummyEngine) Count(text string) (int, error) { return 2, nil }

func TestEngineContract(t *testing.T) {
	var eng tokenizer.Engine = &dummyEngine{}
	tokens, err := eng.Encode("hello")
	if err != nil || len(tokens) != 2 {
		t.Fatalf("unexpected encode result: %v, %v", tokens, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v -run=TestEngineContract`
Expected: FAIL with "undefined: tokenizer.Engine"

- [ ] **Step 3: Implement Engine, Capabilities, Provider types**

`engine.go`:
```go
package tokenizer

type Algorithm string

const (
	AlgorithmTiktokenBPE   Algorithm = "tiktoken_bpe"
	AlgorithmHFByteBPE     Algorithm = "hf_byte_bpe"
	AlgorithmSentencePiece Algorithm = "sentencepiece"
	AlgorithmTekken        Algorithm = "tekken"
)

type EngineMetadata struct {
	ID           string
	Algorithm    Algorithm
	VocabSize    int
	Version      string
	AssetSHA256  string
	Capabilities Capabilities
}

type Engine interface {
	Encode(text string) ([]int, error)
	Decode(tokens []int) (string, error)
	Count(text string) (int, error)
}
```

`capabilities.go`:
```go
package tokenizer

type Capabilities uint32

const (
	CapabilityCountText Capabilities = 1 << iota
	CapabilityEncode
	CapabilityDecode
	CapabilityCountMessages
	CapabilityTools
	CapabilityMultimodalEstimate
)
```

`provider.go`:
```go
package tokenizer

type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderQwen      Provider = "qwen"
	ProviderKimi      Provider = "kimi"
	ProviderDeepSeek  Provider = "deepseek"
	ProviderGrok      Provider = "grok"
	ProviderMistral   Provider = "mistral"
	ProviderGemini    Provider = "gemini"
)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v -run=TestEngineContract`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add engine.go capabilities.go provider.go engine_test.go
git commit -m "feat: add core Engine, Capabilities, and Provider types"
```

---

### Task 2: Accuracy, Calibration, CountResult, and Resolution Types

**Files:**
- Create: `count_result.go`
- Create: `resolution.go`
- Test: `count_result_test.go`

**Interfaces:**
- Consumes: `Provider`
- Produces: `Accuracy`, `Calibration`, `CountResult`, `Resolution`, `WithSafetyMargin`

- [ ] **Step 1: Write failing unit test for CountResult, SafetyMargin, and Resolution**

```go
package tokenizer_test

import (
	"testing"

	"github.com/xorvus/tokenizer-go"
)

func TestCountResultSafetyMargin(t *testing.T) {
	res := tokenizer.CountResult{
		Tokens:   100,
		Accuracy: tokenizer.AccuracyEstimatedCalibrated,
	}
	safe := res.WithSafetyMargin(10)
	if safe != 110 {
		t.Errorf("got %d, want 110", safe)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v -run=TestCountResultSafetyMargin`
Expected: FAIL with "undefined: tokenizer.AccuracyEstimatedCalibrated"

- [ ] **Step 3: Implement Accuracy, Calibration, CountResult, Resolution**

`count_result.go`:
```go
package tokenizer

import "math"

type Accuracy uint8

const (
	AccuracyExactLocal Accuracy = iota
	AccuracyEstimatedCalibrated
	AccuracyEstimatedHeuristic
)

func (a Accuracy) String() string {
	switch a {
	case AccuracyExactLocal:
		return "ExactLocal"
	case AccuracyEstimatedCalibrated:
		return "EstimatedCalibrated"
	case AccuracyEstimatedHeuristic:
		return "EstimatedHeuristic"
	default:
		return "Unknown"
	}
}

type Calibration struct {
	ProfileID       string  `json:"profile_id"`
	SampleCount     int     `json:"sample_count"`
	MeanAbsErrorPct float64 `json:"mean_abs_error_pct"`
	P95AbsErrorPct  float64 `json:"p95_abs_error_pct"`
	MaxAbsErrorPct  float64 `json:"max_abs_error_pct"`
	CorpusSHA256    string  `json:"corpus_sha256"`
}

type CountResult struct {
	Tokens         int          `json:"tokens"`
	RequestedModel string       `json:"requested_model"`
	CanonicalModel string       `json:"canonical_model"`
	TokenizerID    string       `json:"tokenizer_id"`
	Provider       Provider     `json:"provider"`
	Accuracy       Accuracy     `json:"accuracy"`
	UsedFallback   bool         `json:"used_fallback"`
	FallbackReason string       `json:"fallback_reason,omitempty"`
	Calibration    *Calibration `json:"calibration,omitempty"`
}

func (r CountResult) WithSafetyMargin(percent float64) int {
	if percent <= 0 {
		return r.Tokens
	}
	margin := math.Ceil(float64(r.Tokens) * (percent / 100.0))
	return r.Tokens + int(margin)
}
```

`resolution.go`:
```go
package tokenizer

type Resolution struct {
	RequestedModel string   `json:"requested_model"`
	CanonicalModel string   `json:"canonical_model"`
	Provider       Provider `json:"provider"`
	TokenizerID    string   `json:"tokenizer_id"`
	Accuracy       Accuracy `json:"accuracy"`
	UsedFallback   bool     `json:"used_fallback"`
	Reason         string   `json:"reason,omitempty"`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v -run=TestCountResultSafetyMargin`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add count_result.go resolution.go count_result_test.go
git commit -m "feat: add Accuracy, Calibration, CountResult, and Resolution types"
```

---

### Task 3: Multi-Provider Model Registry

**Files:**
- Create: `model.go`
- Create: `registry.go`
- Test: `registry_test.go`

**Interfaces:**
- Consumes: `Provider`, `Accuracy`, `Capabilities`
- Produces: `ModelSpec`, `Registry`, `GlobalRegistry`, `ResolveModel`

- [ ] **Step 1: Write failing unit test for Registry resolution order and prefix matching**

```go
package tokenizer_test

import (
	"errors"
	"testing"

	"github.com/xorvus/tokenizer-go"
)

func TestRegistryResolution(t *testing.T) {
	res, err := tokenizer.ResolveModel("gpt-4o")
	if err != nil {
		t.Fatalf("ResolveModel('gpt-4o') error: %v", err)
	}
	if res.CanonicalModel != "gpt-4o" || res.TokenizerID != "o200k_base" {
		t.Errorf("unexpected resolution: %+v", res)
	}

	_, err = tokenizer.ResolveModel("nonexistent-model-xyz")
	if !errors.Is(err, tokenizer.ErrUnknownModel) {
		t.Errorf("expected ErrUnknownModel, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v -run=TestRegistryResolution`
Expected: FAIL with "undefined: tokenizer.ResolveModel"

- [ ] **Step 3: Implement ModelSpec and Registry**

`model.go`:
```go
package tokenizer

type ModelSpec struct {
	CanonicalName   string
	Provider        Provider
	Aliases         []string
	Prefixes        []string
	TokenizerID     string
	FallbackProfile string
	ChatTemplateID  string
	Capabilities    Capabilities
	IsExact         bool
}
```

`registry.go`:
```go
package tokenizer

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Registry struct {
	mu           sync.RWMutex
	exactModels  map[string]ModelSpec
	aliases      map[string]string
	prefixes     []prefixEntry
}

type prefixEntry struct {
	prefix string
	spec   ModelSpec
}

var globalRegistry = newDefaultRegistry()

func newDefaultRegistry() *Registry {
	r := &Registry{
		exactModels: make(map[string]ModelSpec),
		aliases:     make(map[string]string),
	}

	// Register default OpenAI models
	r.registerOpenAI()
	return r
}

func (r *Registry) registerOpenAI() {
	// Exact OpenAI models
	o200kModels := []string{
		"gpt-4o", "gpt-4o-mini", "gpt-4.5-preview", "o1", "o1-mini", "o1-preview", "o3-mini", "gpt-5",
	}
	for _, m := range o200kModels {
		r.exactModels[m] = ModelSpec{
			CanonicalName: m,
			Provider:      ProviderOpenAI,
			TokenizerID:   string(O200KBase),
			Capabilities:  CapabilityCountText | CapabilityEncode | CapabilityDecode,
			IsExact:       true,
		}
	}

	harmonyModels := []string{
		"gpt-oss-1", "gpt-oss-2",
	}
	for _, m := range harmonyModels {
		r.exactModels[m] = ModelSpec{
			CanonicalName: m,
			Provider:      ProviderOpenAI,
			TokenizerID:   string(O200KHarmony),
			Capabilities:  CapabilityCountText | CapabilityEncode | CapabilityDecode | CapabilityCountMessages,
			IsExact:       true,
		}
	}

	cl100kModels := []string{
		"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo", "text-embedding-ada-002", "text-embedding-3-small", "text-embedding-3-large",
	}
	for _, m := range cl100kModels {
		r.exactModels[m] = ModelSpec{
			CanonicalName: m,
			Provider:      ProviderOpenAI,
			TokenizerID:   string(CL100KBase),
			Capabilities:  CapabilityCountText | CapabilityEncode | CapabilityDecode,
			IsExact:       true,
		}
	}

	// Prefix mappings sorted longest-first
	prefixes := []prefixEntry{
		{prefix: "gpt-oss-", spec: ModelSpec{Provider: ProviderOpenAI, TokenizerID: string(O200KHarmony), IsExact: true}},
		{prefix: "o1-", spec: ModelSpec{Provider: ProviderOpenAI, TokenizerID: string(O200KBase), IsExact: true}},
		{prefix: "o3-", spec: ModelSpec{Provider: ProviderOpenAI, TokenizerID: string(O200KBase), IsExact: true}},
		{prefix: "gpt-5-", spec: ModelSpec{Provider: ProviderOpenAI, TokenizerID: string(O200KBase), IsExact: true}},
		{prefix: "gpt-4.5-", spec: ModelSpec{Provider: ProviderOpenAI, TokenizerID: string(O200KBase), IsExact: true}},
		{prefix: "gpt-4o-", spec: ModelSpec{Provider: ProviderOpenAI, TokenizerID: string(O200KBase), IsExact: true}},
		{prefix: "gpt-4-", spec: ModelSpec{Provider: ProviderOpenAI, TokenizerID: string(CL100KBase), IsExact: true}},
		{prefix: "gpt-3.5-turbo-", spec: ModelSpec{Provider: ProviderOpenAI, TokenizerID: string(CL100KBase), IsExact: true}},
		{prefix: "ft:gpt-4o", spec: ModelSpec{Provider: ProviderOpenAI, TokenizerID: string(O200KBase), IsExact: true}},
		{prefix: "ft:gpt-4", spec: ModelSpec{Provider: ProviderOpenAI, TokenizerID: string(CL100KBase), IsExact: true}},
	}
	sort.Slice(prefixes, func(i, j int) bool {
		return len(prefixes[i].prefix) > len(prefixes[j].prefix)
	})
	r.prefixes = prefixes
}

func ResolveModel(model string) (Resolution, error) {
	return globalRegistry.Resolve(model)
}

func (r *Registry) Resolve(model string) (Resolution, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 1. Exact match
	if spec, ok := r.exactModels[model]; ok {
		return Resolution{
			RequestedModel: model,
			CanonicalModel: spec.CanonicalName,
			Provider:       spec.Provider,
			TokenizerID:    spec.TokenizerID,
			Accuracy:       AccuracyExactLocal,
			UsedFallback:   false,
		}, nil
	}

	// 2. Alias match
	if canonical, ok := r.aliases[model]; ok {
		if spec, ok := r.exactModels[canonical]; ok {
			return Resolution{
				RequestedModel: model,
				CanonicalModel: spec.CanonicalName,
				Provider:       spec.Provider,
				TokenizerID:    spec.TokenizerID,
				Accuracy:       AccuracyExactLocal,
				UsedFallback:   false,
			}, nil
		}
	}

	// 3. Documented prefix match
	for _, entry := range r.prefixes {
		if strings.HasPrefix(model, entry.prefix) {
			return Resolution{
				RequestedModel: model,
				CanonicalModel: model,
				Provider:       entry.spec.Provider,
				TokenizerID:    entry.spec.TokenizerID,
				Accuracy:       AccuracyExactLocal,
				UsedFallback:   false,
			}, nil
		}
	}

	return Resolution{}, fmt.Errorf("%w: %s", ErrUnknownModel, model)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v -run=TestRegistryResolution`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add model.go registry.go registry_test.go
git commit -m "feat: add ModelSpec and Registry with zero-alloc resolution"
```

---

### Task 4: Model-Aware `CountForModel` and Count Options

**Files:**
- Modify: `tokenizer.go`
- Create: `count_options.go`
- Test: `count_for_model_test.go`

**Interfaces:**
- Consumes: `CountResult`, `Resolution`, `CountConfig`, `CountOption`
- Produces: `CountForModel`, `WithExactOnly`, `WithCalibratedFallback`, `WithHeuristicFallback`

- [ ] **Step 1: Write failing unit test for `CountForModel`**

```go
package tokenizer_test

import (
	"testing"

	"github.com/xorvus/tokenizer-go"
)

func TestCountForModelExact(t *testing.T) {
	res, err := tokenizer.CountForModel("gpt-4o", "Hello world")
	if err != nil {
		t.Fatalf("CountForModel error: %v", err)
	}
	if res.Tokens != 2 {
		t.Errorf("got %d tokens, want 2", res.Tokens)
	}
	if res.Accuracy != tokenizer.AccuracyExactLocal {
		t.Errorf("got accuracy %v, want ExactLocal", res.Accuracy)
	}
	if res.UsedFallback {
		t.Errorf("expected UsedFallback == false")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v -run=TestCountForModelExact`
Expected: FAIL with "undefined: tokenizer.CountForModel"

- [ ] **Step 3: Implement CountConfig options and CountForModel**

`count_options.go`:
```go
package tokenizer

type CountConfig struct {
	AllowCalibratedFallback bool
	AllowHeuristicFallback  bool
	ExactOnly               bool
}

type CountOption func(*CountConfig)

func DefaultCountConfig() CountConfig {
	return CountConfig{
		AllowCalibratedFallback: true,
		AllowHeuristicFallback:  false,
		ExactOnly:               false,
	}
}

func WithExactOnly() CountOption {
	return func(cfg *CountConfig) {
		cfg.ExactOnly = true
		cfg.AllowCalibratedFallback = false
		cfg.AllowHeuristicFallback = false
	}
}

func WithCalibratedFallback() CountOption {
	return func(cfg *CountConfig) {
		cfg.AllowCalibratedFallback = true
	}
}

func WithHeuristicFallback() CountOption {
	return func(cfg *CountConfig) {
		cfg.AllowHeuristicFallback = true
	}
}
```

In `tokenizer.go`, ensure `Tokenizer` implements `Engine` and add `CountForModel`:
```go
func CountForModel(model string, text string, opts ...CountOption) (CountResult, error) {
	cfg := DefaultCountConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	res, err := ResolveModel(model)
	if err != nil {
		return CountResult{}, err
	}

	if cfg.ExactOnly && res.UsedFallback {
		return CountResult{}, fmt.Errorf("%w: model %s requires fallback but ExactOnly requested", ErrExactTokenizerUnavailable, model)
	}

	tok, err := GetEncoding(Encoding(res.TokenizerID))
	if err != nil {
		return CountResult{}, err
	}

	count, err := tok.Count(text)
	if err != nil {
		return CountResult{}, err
	}

	return CountResult{
		Tokens:         count,
		RequestedModel: res.RequestedModel,
		CanonicalModel: res.CanonicalModel,
		TokenizerID:    res.TokenizerID,
		Provider:       res.Provider,
		Accuracy:       res.Accuracy,
		UsedFallback:   res.UsedFallback,
		FallbackReason: res.Reason,
	}, nil
}
```

Add error in `errors.go`:
```go
ErrExactTokenizerUnavailable = errors.New("exact tokenizer unavailable for model")
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v -run=TestCountForModelExact`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add tokenizer.go count_options.go count_for_model_test.go errors.go
git commit -m "feat: implement CountForModel and CountOption helpers"
```

---

### Task 5: Verification, Benchmarks, and Zero-Allocation Verification

**Files:**
- Test: `test/ablation_test.go`
- Test: `registry_bench_test.go`

**Interfaces:**
- Consumes: All `v0.7.0` multi-engine foundation components
- Produces: Benchmark verification suite & zero-alloc guarantee tests

- [ ] **Step 1: Write zero-allocation test for `ResolveModel` and benchmark**

`registry_bench_test.go`:
```go
package tokenizer_test

import (
	"testing"

	"github.com/xorvus/tokenizer-go"
)

func TestResolveModelZeroAllocation(t *testing.T) {
	allocs := testing.AllocsPerRun(100, func() {
		_, _ = tokenizer.ResolveModel("gpt-4o")
	})
	if allocs > 0 {
		t.Errorf("ResolveModel allocated %f heap objects, want 0", allocs)
	}
}

func BenchmarkResolveModel(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = tokenizer.ResolveModel("gpt-4o")
	}
}
```

- [ ] **Step 2: Run zero-allocation test and benchmark**

Run: `go test -v -run=TestResolveModelZeroAllocation`
Expected: PASS with 0 allocations.

Run: `go test -bench=BenchmarkResolveModel -benchmem`
Expected: `<200 ns/op` and `0 B/op`.

- [ ] **Step 3: Run entire repository test suite with race detector**

Run: `go test -race ./...`
Expected: PASS with zero race warnings.

- [ ] **Step 4: Commit**

```bash
git add registry_bench_test.go
git commit -m "test: add zero-allocation tests and benchmark for ResolveModel"
```

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-08-06-multi-provider-tokenizer.md`. Two execution options:

**1. Subagent-Driven (recommended)** - Dispatch a fresh subagent per task, review between tasks, fast iteration.

**2. Inline Execution** - Execute tasks in this session using `executing-plans`, batch execution with checkpoints.

Which approach would you like to proceed with?
