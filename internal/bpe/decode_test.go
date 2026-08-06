package bpe_test

import (
	"testing"

	"github.com/pkoukk/tokenizer-go/internal/bpe"
	"github.com/pkoukk/tokenizer-go/internal/openai"
)

func TestDecodeNative(t *testing.T) {
	encoder := map[string]int{"hello": 1, " ": 2, "world": 3}
	decoder := []string{"", "hello", " ", "world"}
	core, err := bpe.NewCoreBPE(encoder, decoder, nil, openai.PatternCL100K)
	if err != nil {
		t.Fatalf("failed creating core BPE: %v", err)
	}
	bytes, err := core.DecodeNative([]int{1, 2, 3})
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if string(bytes) != "hello world" {
		t.Errorf("got %q, want %q", string(bytes), "hello world")
	}
	_, err = core.DecodeNative([]int{999})
	if err == nil {
		t.Errorf("expected error for unknown token ID")
	}
}
