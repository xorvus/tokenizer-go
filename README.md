# tokenizer-go (v0.4.0)

Fast, thread-safe Byte Pair Encoding (BPE) tokenizer for OpenAI models in Pure Go.

## Features
- **Official Encodings**: Full support for `cl100k_base` and `o200k_base`.
- **Built-in Model Mappings**: Mappings for OpenAI model families (`o1`, `gpt-4o`, `gpt-4o-mini`, `gpt-4-turbo`, `text-embedding-3`, etc.).
- **Offline & Embedded**: Embedded BPE vocabularies via `//go:embed` for zero-network execution.
- **Thread-Safe**: Safe for concurrent use across multiple goroutines (`go test -race` verified).
- **High Performance**: Index-only pre-tokenization matcher, 2-byte lookup table, ASCII offset bypass, and ordered piece-boundary parallel BPE.

## Benchmarks (v0.4.0)

Environment: Apple M4 Pro (macOS arm64), 3.277 MiB UDHR corpus.

| Implementation | Short ASCII Latency | Ops / sec | Full UDHR Execution Time | Throughput | Total Allocated Bytes | Allocation Count |
| :--- | ---: | ---: | ---: | ---: | ---: | ---: |
| **`tiktoken-go` (Original)** | 3,720 ns/op | 268,817 ops/s | 774.37 ms/op | 4.23 MiB/s | 441.99 MB/op | 4,385,692/op |
| **Python OpenAI `tiktoken` API** | 1,393 ns/op | 717,794 ops/s | 310.57 ms/op | 10.55 MiB/s | *Not measured setara* | *Not measured setara* |
| **`tokenizer-go` (v0.4.0)** | **994.7 ns/op** | **1,005,328 ops/s** | **187.14 ms/op** | **17.51 MiB/s** | **201.18 MB/op** | **408/op** |

> `tokenizer-go v0.4.0` adds ordered piece-boundary parallel BPE encoding for documents of at least 1 MiB with 10,000 or more pre-tokenized pieces. On the normalized 3.277 MiB UDHR corpus, using four workers on an Apple M4 Pro, it reached 17.51 MiB/s at 187.14 ms/op—approximately 66.0% higher throughput, or 39.7% lower wall-clock latency, than the Python OpenAI tiktoken API in the same benchmark environment. On short ASCII inputs, `tokenizer-go` delivered approximately 40.0% higher operation throughput, or 28.6% lower latency, than the Python API. Large-document results use bounded four-worker parallelism; short-input results remain sequential.
>
> In multi-caller testing, aggregate throughput increased from 5.11 requests/s (16.74 MiB/s) with one caller to 19.87 requests/s (65.11 MiB/s) with eight callers. Four concurrent callers provided the strongest throughput-to-latency trade-off in this test, reaching 55.28 MiB/s with a measured p95 latency of 237.13 ms. At eight callers, aggregate throughput continued to increase, while p95 latency rose to 402.48 ms as CPU resources became saturated. Parallel output remained identical to sequential output across the parity, race, and fuzz tests performed.

## Installation
```bash
go get github.com/pkoukk/tokenizer-go
```

## Quick Start
```go
package main

import (
	"fmt"

	"github.com/pkoukk/tokenizer-go"
)

func main() {
	tok, err := tokenizer.GetEncoding(tokenizer.CL100KBase)
	if err != nil {
		panic(err)
	}

	tokens, err := tok.EncodeOrdinary("Hello, world!")
	if err != nil {
		panic(err)
	}
	fmt.Println("Tokens:", tokens)

	text, err := tok.Decode(tokens)
	if err != nil {
		panic(err)
	}
	fmt.Println("Decoded:", text)
}
```

## License
MIT License. See [LICENSE](LICENSE) and [NOTICE](NOTICE) for details.
