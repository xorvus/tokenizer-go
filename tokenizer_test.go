package tokenizer_test

import (
	"testing"

	"github.com/pkoukk/tokenizer-go"
)

func TestEmbeddedCL100K(t *testing.T) {
	tok, err := tokenizer.GetEmbeddedCL100K()
	if err != nil {
		t.Fatalf("failed loading embedded cl100k: %v", err)
	}
	tokens, err := tok.EncodeOrdinary("hello world")
	if err != nil {
		t.Fatalf("EncodeOrdinary error: %v", err)
	}
	if len(tokens) != 2 || tokens[0] != 15339 || tokens[1] != 1917 {
		t.Errorf("got %v, want [15339, 1917]", tokens)
	}
}
