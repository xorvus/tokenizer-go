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

## Benchmark
You can run benchmark in test folder.

### Benchmark result
| name | time/op | os | cpu | text | times |
| :--- | :--- | :--- | :--- | :--- | :--- |
| tiktoken-go | 8795ns | macOS 13.2 | Apple M1 | UDHR | 100000 |
| tiktoken | 8838ns | macOS 13.2 | Apple M1 | UDHR | 100000 |

It looks like the performance is almost the same.

Maybe the difference is due to the difference in the performance of the machine.

Or maybe my benchmark method is not appropriate.

If you have better benchmark method or if you want add your benchmark result, please feel free to submit a PR.

For new o200k_base encoding, it seems slower than cl100k_base. tiktoken-go is slightly slower than tiktoken on the following benchmark.

| name | encoding | time/op | os | cpu | text | times |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| tiktoken-go | o200k_base | 108522 ns | Ubuntu 22.04 | AMD Ryzen 9 5900HS | UDHR | 100000 |
| tiktoken | o200k_base | 70198 ns | Ubuntu 22.04 | AMD Ryzen 9 5900HS | UDHR | 100000 |
| tiktoken-go | cl100k_base | 94502 ns | Ubuntu 22.04 | AMD Ryzen 9 5900HS | UDHR | 100000 |
| tiktoken | cl100k_base | 54642 ns | Ubuntu 22.04 | AMD Ryzen 9 5900HS | UDHR | 100000 |

## License
MIT License. See [LICENSE](LICENSE) and [NOTICE](NOTICE) for details.
