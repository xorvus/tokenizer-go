package tokenizer_test

import (
	"testing"

	"github.com/xorvus/tokenizer-go"
)

func TestCountForModelExact(t *testing.T) {
	res, err := tokenizer.CountForModel("gpt-4o", "Hello world")
	if err != nil {
		t.Fatalf("CountForModel error: %v", err)
	}
	if res.Tokens != 2 {
		t.Errorf("got %d tokens, want 2", res.Tokens)
	}
	if res.Accuracy != tokenizer.AccuracyExactLocal {
		t.Errorf("got accuracy %v, want ExactLocal", res.Accuracy)
	}
	if res.UsedFallback {
		t.Errorf("expected UsedFallback == false")
	}
}
