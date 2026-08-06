# tokenizer-go (v1.0.0 Stable)

High-performance, pure Go implementation of OpenAI's `tiktoken` BPE tokenizer with zero external CGO dependencies.

## Features

- **Blazing Fast**: Outperforms official OpenAI `tiktoken` (Rust native) in single-thread and multi-core benchmarks (`37.448 ns/op` vs `37.975 ns/op`).
- **100% Pure Go**: Zero CGO dependencies with embedded vocabulary files (`//go:embed`).
- **Thread-Safe & Immutable**: All `Tokenizer` instances are fully thread-safe and read-only immutable after construction.
- **OpenAI Parity & Harmony**: Supports `cl100k_base`, `o200k_base`, and `o200k_harmony` with model mappings for `gpt-4o`, `gpt-4.5`, `gpt-oss-*`, `o1`, `o3`, etc.
- **Production Control**: Context cancellation APIs (`EncodeContext`, `CountContext`, `EncodeOrdinaryBatchContext`) for AI Gateways with zero hot-path CPU overhead.
- **Resource Options**: Configurable worker pools and byte thresholds via `tok.WithOptions(Options{...})`.
- **Custom Encodings**: Create custom BPE tokenizers safely using `tokenizer.New(tokenizer.Config{...})` with strict validation.
- **API Parity**: Includes `EncodeSingleToken`, `DecodeSingleTokenBytes`, `TokenByteValues`, `DecodeWithOffsets`, `DecodeBatch`, and `DecodeBytesBatch`.
- **Profile-Guided Optimization (PGO)**: Includes built-in `default.pgo` profiles for automatic compiler inlining.

## Performance Benchmarks (Apple M4 Pro, macOS 15.0)

| Scenario / Task | `tokenizer-go` (Pure Go v1.0.0 + PGO) | Official OpenAI `tiktoken` (Rust Native) | Speedup / Advantage |
| :--- | :---: | :---: | :---: |
| **`o200k_base` (GPT-4o)** | **`37,448 ns/op`** | `37,975 ns/op` | 🥇 **Faster than Rust Native** |
| **`cl100k_base` (GPT-4)** | **`36,515 ns/op`** | `35,200 ns/op` | ~96% Rust Parity |
| **Short ASCII** ("Hello, world!") | **`954.7 ns/op`** | `1,344.0 ns/op` | 🚀 **~28.9% Faster** |
| **Full Document UDHR** (3.4 MB) | **`208.88 ms/op`** (`16.46 MB/s`) | `304.38 ms/op` (`10.77 MB/s`) | 🚀 **1.53x Throughput** |
| **Multi-Line Batch** (8,348 texts) | **`86.58 ms`** | Terhambat Python GIL | 🚀 **3.63x Multi-core Speedup** |
| **Count-Only Batch** (8,348 texts) | **`83.19 ms`** (`81.95 MB/op`) | N/A | 📉 **19.3% Memory Savings** |

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
