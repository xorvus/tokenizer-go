# tokenizer-go (v0.1.0)

Fast, thread-safe Byte Pair Encoding (BPE) tokenizer for OpenAI models in Go.

## Features
- **Official Encodings**: Full support for `cl100k_base` and `o200k_base`.
- **Built-in Model Mappings**: Mappings for OpenAI model families (`o1`, `gpt-4o`, `gpt-4o-mini`, `gpt-4-turbo`, `text-embedding-3`, etc.).
- **Offline & Embedded**: Embedded BPE vocabularies via `//go:embed` for zero-network execution.
- **Thread-Safe**: Safe for concurrent use across multiple goroutines (`go test -race` verified).
- **High Performance**: Zero-copy special-token scanning and local small-buffer BPE optimizations.

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
