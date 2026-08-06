package bpe_test

import (
	"reflect"
	"testing"

	"github.com/pkoukk/tiktoken-go/tokenizer-go/internal/bpe"
)

func TestBytePairEncode(t *testing.T) {
	ranks := map[string]int{
		"h": 1, "e": 2, "l": 3, "o": 4,
		"he": 5, "ll": 6, "hell": 7, "hello": 8,
	}
	tokens := bpe.BytePairEncode("hello", ranks)
	expected := []int{8}
	if !reflect.DeepEqual(tokens, expected) {
		t.Errorf("got %v, want %v", tokens, expected)
	}

	emptyTokens := bpe.BytePairEncode("", ranks)
	if len(emptyTokens) != 0 {
		t.Errorf("expected empty tokens for empty input")
	}
}

