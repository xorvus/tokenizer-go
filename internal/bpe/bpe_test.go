package bpe_test

import (
	"reflect"
	"testing"

	"github.com/xorvus/tokenizer-go/internal/bpe"
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

func TestPackedKeyCollisionCornerCases(t *testing.T) {
	testTokens := []string{
		"abc",
		"abcd",
		"abcde",
		"abcdef",
		"abcdefg",
		"\x00\x61\x62",         // leading zero (3 bytes)
		"\x61\x62\x00",         // trailing zero (3 bytes)
		"\x61\x62\x00\x00",     // trailing zeros (4 bytes)
		"\x00\x00\x00",         // all zeros (3 bytes)
		"\x00\x00\x00\x00",     // all zeros (4 bytes)
		"\xff\xff\xff",         // 0xFF (3 bytes)
		"\xff\xff\xff\xff",     // 0xFF (4 bytes)
		"\xff\xff\xff\xff\xff", // 0xFF (5 bytes)
	}

	ranks := make(map[string]int)
	for i, tok := range testTokens {
		ranks[tok] = i + 100
	}

	idx := bpe.NewRankIndex(ranks)

	for _, tok := range testTokens {
		rank, ok := idx.Lookup(tok)
		if !ok {
			t.Errorf("Lookup(%q) failed, expected found", tok)
		}
		if rank != ranks[tok] {
			t.Errorf("Lookup(%q) = %d, want %d", tok, rank, ranks[tok])
		}
	}
}
