package bpe_test

import (
	"testing"

	"github.com/pkoukk/tiktoken-go/tokenizer-go/internal/bpe"
	"github.com/pkoukk/tiktoken-go/tokenizer-go/internal/openai"
)

func TestRegexSplit(t *testing.T) {
	encoder := map[string]int{"hello": 1, " ": 2, "world": 3}
	decoder := []string{"", "hello", " ", "world"}
	core, err := bpe.NewCoreBPE(encoder, decoder, nil, openai.PatternCL100K)
	if err != nil {
		t.Fatalf("failed creating core BPE: %v", err)
	}
	tokens, err := core.EncodeOrdinaryNative("hello world")
	if err != nil {
		t.Fatalf("encoding error: %v", err)
	}
	if len(tokens) == 0 {
		t.Errorf("expected non-empty tokens")
	}
}
