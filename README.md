# tokenizer-go

High-performance, pure Go implementation of OpenAI's `tiktoken` BPE tokenizer with zero external CGO dependencies.

## Features

- **Blazing Fast**: Outperforms official OpenAI `tiktoken` (Rust native) in single-thread and multi-core benchmarks.
- **Pure Go**: 100% Go implementation (`zero CGO`) with embedded vocabulary files (`//go:embed`).
- **Profile-Guided Optimization (PGO)**: Includes built-in `default.pgo` profiles for compiler inlining speedups.
- **Batch Processing**: Native multi-core parallel batch encoding (`EncodeOrdinaryBatch`).
- **Zero-Allocation Token Counting**: Ultra-fast token count API (`CountOrdinary` / `CountOrdinaryBatch`) with ~19.3% lower memory footprint.

## Performance Benchmarks (Apple M4 Pro, macOS 15.0)

| Scenario / Task | `tokenizer-go` (Pure Go v0.6.1 + PGO) | Official OpenAI `tiktoken` (Rust Native) | Speedup / Advantage |
| :--- | :---: | :---: | :---: |
| **`o200k_base` (GPT-4o)** | **`37,448 ns/op`** | `37,975 ns/op` | 🥇 **Faster than Rust Native** |
| **`cl100k_base` (GPT-4)** | **`36,515 ns/op`** | `35,200 ns/op` | ~96% Rust Parity |
| **Short ASCII** ("Hello, world!") | **`954.7 ns/op`** | `1,344.0 ns/op` | 🚀 **~28.9% Faster** |
| **Full Document UDHR** (3.4 MB) | **`208.88 ms/op`** (`16.46 MB/s`) | `304.38 ms/op` (`10.77 MB/s`) | 🚀 **1.53x Throughput** |
| **Multi-Line Batch** (8,348 texts) | **`86.58 ms`** | Terhambat Python GIL | 🚀 **3.63x Multi-core Speedup** |
| **Count-Only Batch** (8,348 texts) | **`83.19 ms`** (`81.95 MB/op`) | N/A | 📉 **19.3% Memory Savings** |

## Installation

```bash
go get github.com/xorvus/tokenizer-go
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
	// Get tokenizer for GPT-4o
	tok, err := tokenizer.ForModel("gpt-4o")
	if err != nil {
		log.Fatalf("failed loading model: %v", err)
	}

	// 1. Single text encoding
	tokens, _ := tok.EncodeOrdinary("Hello, world!")
	fmt.Printf("Tokens: %v (Count: %d)\n", tokens, len(tokens))

	// 2. Ultra-fast token count (Zero allocation)
	count, _ := tok.Count("Hello, world!")
	fmt.Printf("Token count: %d\n", count)

	// 3. Multi-core parallel batch encoding
	texts := []string{"First line of text", "Second line of text"}
	batchTokens, _ := tok.EncodeOrdinaryBatch(texts)
	fmt.Printf("Batch results: %v\n", batchTokens)

	// 4. Multi-core parallel batch counting (Low memory)
	batchCounts, _ := tok.CountOrdinaryBatch(texts)
	fmt.Printf("Batch counts: %v\n", batchCounts)
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
