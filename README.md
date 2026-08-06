# tokenizer-go (v0.4.0)

Fast, thread-safe Byte Pair Encoding (BPE) tokenizer for OpenAI models in Pure Go.

## Features
- **Official Encodings**: Full support for `cl100k_base` and `o200k_base`.
- **Built-in Model Mappings**: Mappings for OpenAI model families (`o1`, `gpt-4o`, `gpt-4o-mini`, `gpt-4-turbo`, `text-embedding-3`, etc.).
- **Offline & Embedded**: Embedded BPE vocabularies via `//go:embed` for zero-network execution.
- **Thread-Safe**: Safe for concurrent use across multiple goroutines (`go test -race` verified).
- **High Performance**: Index-only pre-tokenization matcher, 2-byte lookup table, ASCII offset bypass, and ordered piece-boundary parallel BPE.

## Installation
```bash
go get github.com/xorvus/tokenizer-go
```

## Quick Start
```go
package main

import (
	"fmt"

	"github.com/xorvus/tokenizer-go"
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

## Benchmark
You can run benchmark in test folder.

### Benchmark result
| name | time/op | os | cpu | text | times |
| :--- | :--- | :--- | :--- | :--- | :--- |
| tokenizer-go | 977 ns | macOS arm64 | Apple M4 Pro | Short ASCII | 1000000 |
| tiktoken | 1393 ns | macOS arm64 | Apple M4 Pro | Short ASCII | 1000000 |

It looks like the performance is faster on short inputs.

Maybe the difference is due to the difference in the performance of the machine.

## License
MIT License. See [LICENSE](LICENSE) and [NOTICE](NOTICE) for details.
