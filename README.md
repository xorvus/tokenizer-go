# tokenizer-go (v1.0.0 Stable)

High-performance, pure Go implementation of OpenAI's `tiktoken` BPE tokenizer with zero external CGO dependencies.

## Features

- **Blazing Fast**: Single-thread `37.4 µs/op` on `o200k_base` and `36.5 µs/op` on `cl100k_base` via PGO inlining.
- **100% Pure Go**: Zero CGO dependencies with embedded vocabulary files (`//go:embed`).
- **Thread-Safe & Immutable**: All `Tokenizer` instances are fully thread-safe and read-only immutable after construction.
- **OpenAI Parity & Harmony**: Supports `cl100k_base`, `o200k_base`, and `o200k_harmony` with model mappings for `gpt-4o`, `gpt-4.5`, `gpt-oss-*`, `o1`, `o3`, etc.
- **Production Control**: Context cancellation APIs (`EncodeContext`, `CountContext`, `EncodeOrdinaryBatchContext`) for AI Gateways with zero hot-path CPU overhead.
- **Resource Options**: Configurable worker pools and byte thresholds via `tok.WithOptions(Options{...})`.
- **Custom Encodings**: Create custom BPE tokenizers safely using `tokenizer.New(tokenizer.Config{...})` with strict validation.
- **API Parity**: Includes `EncodeSingleToken`, `DecodeSingleTokenBytes`, `TokenByteValues`, `DecodeWithOffsets`, `DecodeBatch`, and `DecodeBytesBatch`.
- **Profile-Guided Optimization (PGO)**: Includes built-in `default.pgo` profiles for automatic compiler inlining.

## Performance Benchmarks (Apple M4 Pro, macOS 15.0)

| Scenario / Task | `tokenizer-go` (Pure Go v1.0.0 + PGO) |
| :--- | :---: |
| **`o200k_base` (GPT-4o)** | `37,448 ns/op` |
| **`cl100k_base` (GPT-4)** | `36,515 ns/op` |
| **Short ASCII** ("Hello, world!") | `954.7 ns/op` |
| **Full Document UDHR** (3.4 MB) | `208.88 ms/op` (`16.46 MB/s`) |
| **Multi-Line Batch** (8,348 texts) | `86.58 ms` |
| **Count-Only Batch** (8,348 texts) | `83.19 ms` (`81.95 MB/op`) |

## Installation

```bash
go get github.com/xorvus/tokenizer-go@v1.0.0
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/xorvus/tokenizer-go"
)

func main() {
	// 1. Get tokenizer for standard OpenAI model
	tok, err := tokenizer.ForModel("gpt-4o")
	if err != nil {
		log.Fatalf("failed loading model: %v", err)
	}

	// 2. Encode text
	tokens, _ := tok.EncodeOrdinary("Hello, world!")
	fmt.Printf("Tokens: %v (Count: %d)\n", tokens, len(tokens))

	// 3. Decode with byte offsets for visualizer/editor UI
	text, offsets, _ := tok.DecodeWithOffsets(tokens)
	fmt.Printf("Decoded: %s, Offsets: %v\n", text, offsets)

	// 4. Production Context Cancellation API
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	batchTokens, err := tok.EncodeOrdinaryBatchContext(ctx, []string{"Line 1", "Line 2"})
	if err != nil {
		log.Fatalf("batch tokenization canceled: %v", err)
	}
	fmt.Printf("Batch Tokens: %v\n", batchTokens)
}
```

## Multi-Provider Token Counting (Offline Only)

`CountForModel` counts text the way a given model's provider would count it, for providers beyond OpenAI, **without ever making a network call**:

```go
res, err := tokenizer.CountForModel("gpt-4o", "Hello, world!")
// res.Accuracy == tokenizer.AccuracyExactLocal — OpenAI's real embedded tokenizer.

res, err = tokenizer.CountForModel("claude-3-5-sonnet-20241022", "Hello, world!",
	tokenizer.WithHeuristicFallback())
// res.Accuracy == tokenizer.AccuracyEstimatedHeuristic — Anthropic and Google
// do not publish a tokenizer, so this counts with the nearest embedded
// tokenizer (o200k_base) instead. Use res.UpperBound(), not res.Tokens,
// when enforcing a hard budget against an estimate.
```

Every model resolves through one `Registry` (`ResolveModel`/`LookupModel`), so `ForModel`, `EncodingForModel`, and `CountForModel` can never disagree about the same model name:

| Tier | Providers | Mechanism | `Accuracy` |
| :--- | :--- | :--- | :--- |
| Exact | OpenAI | embedded tiktoken BPE | `AccuracyExactLocal` |
| Estimated | Anthropic, Google Gemini | nearest embedded tokenizer × a committed calibration profile | `AccuracyEstimatedCalibrated` (real measurements) or `AccuracyEstimatedHeuristic` (no measurements yet) |

`ForModel` stays **exact-only**: it returns `ErrExactTokenizerUnavailable` for `claude-*`/`gemini-*` rather than silently handing back an approximate tokenizer under an exact-sounding name. `CountForModel` is where estimates live, and they are opt-in by default (`DefaultCountConfig` rejects uncalibrated heuristic estimates unless you pass `WithHeuristicFallback()`; pass `WithExactOnly()` to reject every kind of estimate).

The two shipped profiles (`internal/calib/profiles/anthropic-claude-v1.json`, `gemini-v1.json`) are currently **uncalibrated identity placeholders**: `sample_count: 0`, meaning nobody has yet run real measurements against Anthropic's or Google's counting APIs, so the "estimate" is simply the nearest tokenizer's raw count with no correction applied — which is exactly why it reports `AccuracyEstimatedHeuristic` rather than `AccuracyEstimatedCalibrated`. Maintainers can upgrade a profile by running `scripts/calibrate.py` (maintainer-only; the *only* place in this project that calls a provider's network API, and never at library runtime) against a real corpus and committing the result — the accuracy label upgrades automatically once `sample_count > 0`, with no code change required.

Extending this to another provider without forking the library:

```go
// after registering (or embedding) a calib.Profile under this ID:
tokenizer.RegisterModelFallbackPrefix("mistral-", tokenizer.ProviderMistral, "mistral-v1")
```

## Thread Safety

All `Tokenizer` instances are immutable and **safe for concurrent use by multiple goroutines**.

## Compatibility & Minimum Go Version

- **Go Version**: Requires Go 1.21+ (for automatic Profile-Guided Optimization support).
- **Supported OS & Architectures**:
  - Linux (`amd64`, `arm64`)
  - macOS (`arm64`, `amd64`)
  - Windows (`amd64`)

## Semantic Versioning

`tokenizer-go` strictly adheres to [Semantic Versioning 2.0.0](https://semver.org/). Version `1.0.0` freezes all public API signatures. Breaking changes will only occur in major version bumps.

## License

MIT License. See [LICENSE](LICENSE) and [NOTICE](NOTICE) for details.
