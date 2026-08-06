package bpe_test

import (
	"testing"

	"github.com/pkoukk/tiktoken-go/tokenizer-go/internal/bpe"
	"github.com/pkoukk/tiktoken-go/tokenizer-go/internal/openai"
)

func TestSpecialTokenHandling(t *testing.T) {
	encoder := map[string]int{"hello": 1, " ": 2}
	decoder := []string{"", "hello", " "}
	special := map[string]int{"<|endoftext|>": 100}
	core, err := bpe.NewCoreBPE(encoder, decoder, special, openai.PatternCL100K)
	if err != nil {
		t.Fatalf("failed creating core BPE: %v", err)
	}
	allowed := map[string]any{"<|endoftext|>": nil}
	tokens, _, err := core.EncodeNative("hello <|endoftext|>", allowed)
	if err != nil {
		t.Fatalf("encoding error: %v", err)
	}
	if len(tokens) == 0 {
		t.Errorf("expected non-empty tokens with special token")
	}
}
