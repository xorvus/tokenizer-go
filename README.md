# tokenizer-go

High-performance, pure Go implementation of OpenAI's `tiktoken` BPE tokenizer with zero external CGO dependencies.

## Features

- **Blazing Fast**: Outperforms official OpenAI `tiktoken` (Rust native) in single-thread and multi-core benchmarks.
- **Pure Go**: 100% Go implementation (`zero CGO`) with embedded vocabulary files (`//go:embed`).
- **OpenAI Parity & Harmony**: Supports `cl100k_base`, `o200k_base`, and `o200k_harmony` with model mappings for `gpt-4o`, `gpt-4.5`, `gpt-oss-*`, `o1`, `o3`, etc.
- **Custom Encodings**: Create custom BPE tokenizers safely using `tokenizer.New(tokenizer.Config{...})` with strict validation.
- **API Parity**: Includes `EncodeSingleToken`, `DecodeSingleTokenBytes`, `TokenByteValues`, `DecodeWithOffsets`, `DecodeBatch`, and `DecodeBytesBatch`.
- **Profile-Guided Optimization (PGO)**: Includes built-in `default.pgo` profiles for compiler inlining speedups.
- **Batch Processing**: Native multi-core parallel batch encoding (`EncodeOrdinaryBatch`).
- **Low-Allocation Token Counting**: Fast token count API (`Count` / `CountOrdinaryBatch`) with ~19.3% lower memory footprint.

## Performance Benchmarks (Apple M4 Pro, macOS 15.0)

| Scenario / Task | `tokenizer-go` (Pure Go + PGO) | Official OpenAI `tiktoken` (Rust Native) | Speedup / Advantage |
| :--- | :---: | :---: | :---: |
| **`o200k_base` (GPT-4o)** | **`37,448 ns/op`** | `37,975 ns/op` | 🥇 **Faster than Rust Native** |
| **`cl100k_base` (GPT-4)** | **`36,515 ns/op`** | `35,200 ns/op` | ~96% Rust Parity |
| **Short ASCII** ("Hello, world!") | **`954.7 ns/op`** | `1,344.0 ns/op` | 🚀 **~28.9% Faster** |
| **Full Document UDHR** (3.4 MB) | **`208.88 ms/op`** (`16.46 MB/s`) | `304.38 ms/op` (`10.77 MB/s`) | 🚀 **1.53x Throughput** |
| **Multi-Line Batch** (8,348 texts) | **`86.58 ms`** | Terhambat Python GIL | 🚀 **3.63x Multi-core Speedup** |
| **Count-Only Batch** (8,348 texts) | **`83.19 ms`** (`81.95 MB/op`) | N/A | 📉 **19.3% Memory Savings** |

## Installation

```bash
go get github.com/xorvus/tokenizer-go@v0.8.0
```

## Quick Start

```go
package main

import (
	"fmt"
	"log"

	"github.com/xorvus/tokenizer-go"
)

func main() {
	// 1. Get tokenizer for standard OpenAI model
	tok, err := tokenizer.ForModel("gpt-4o")
	if err != nil {
		log.Fatalf("failed loading model: %v", err)
	}

	tokens, _ := tok.EncodeOrdinary("Hello, world!")
	fmt.Printf("Tokens: %v (Count: %d)\n", tokens, len(tokens))

	// 2. Custom Tokenizer Construction
	customTok, err := tokenizer.New(tokenizer.Config{
		Name:    "my-custom-bpe",
		Pattern: `\p{L}+|\p{N}+|\s+`,
		MergeableRanks: map[string]int{
			"a": 0, "b": 1, "c": 2, // ... all 0-255 bytes required
		},
		SpecialTokens: map[string]int{
			"<|END|>": 256,
		},
	})
	_ = customTok
}
```

## Profile-Guided Optimization (PGO)

`tokenizer-go` includes `default.pgo` in its package root. When building your application with Go 1.21+, the Go compiler automatically reads `default.pgo` and applies profile-guided inlining optimizations to hot loops.

No extra flags are needed:
```bash
go build -o myapp .
```

## License

MIT License. See [LICENSE](LICENSE) and [NOTICE](NOTICE) for details.
