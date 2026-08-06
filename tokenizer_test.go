package tokenizer_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/xorvus/tokenizer-go"
)

func TestForModel(t *testing.T) {
	tok, err := tokenizer.ForModel("gpt-4o")
	if err != nil {
		t.Fatalf("ForModel(gpt-4o) error: %v", err)
	}
	tokens, err := tok.EncodeOrdinary("hello world")
	if err != nil {
		t.Fatalf("EncodeOrdinary error: %v", err)
	}
	if len(tokens) != 2 || tokens[0] != 24912 || tokens[1] != 2375 {
		t.Errorf("got %v, want [24912, 2375]", tokens)
	}

	_, err = tokenizer.ForModel("unknown-model-xyz")
	if !errors.Is(err, tokenizer.ErrUnknownModel) {
		t.Errorf("expected ErrUnknownModel, got %v", err)
	}
}

func TestGetEncoding(t *testing.T) {
	tok, err := tokenizer.GetEncoding(tokenizer.CL100KBase)
	if err != nil {
		t.Fatalf("GetEncoding(CL100KBase) error: %v", err)
	}
	tokens, err := tok.EncodeOrdinary("hello world")
	if err != nil {
		t.Fatalf("EncodeOrdinary error: %v", err)
	}
	if len(tokens) != 2 || tokens[0] != 15339 || tokens[1] != 1917 {
		t.Errorf("got %v, want [15339, 1917]", tokens)
	}
}

func TestCountOrdinaryEquivalence(t *testing.T) {
	encodings := []tokenizer.Encoding{tokenizer.CL100KBase, tokenizer.O200KBase}
	testCases := []string{
		"",
		"Hello, world!",
		"Go programming language 2026",
		"你好世界！中文测试",
		"こんにちは世界",
		"🚀👨‍👩‍👧‍👦✨🎉",
		"e\u0301 accent test", // combining mark
		"invalid \x80\x81 UTF-8 \xff byte sequence",
		"Longer paragraph with spaces, numbers 1234567890, and special symbols (!@#$%^&*()_+{}|:\"<>?).",
	}

	for _, encName := range encodings {
		tok, err := tokenizer.GetEncoding(encName)
		if err != nil {
			t.Fatalf("failed loading %s: %v", encName, err)
		}

		for _, text := range testCases {
			tokens, err := tok.EncodeOrdinary(text)
			if err != nil {
				t.Fatalf("EncodeOrdinary error on %s: %v", text, err)
			}
			count, err := tok.Count(text)
			if err != nil {
				t.Fatalf("Count error on %s: %v", text, err)
			}
			if count != len(tokens) {
				t.Errorf("[%s] Count(%q) = %d, want len(EncodeOrdinary) = %d", encName, text, count, len(tokens))
			}
		}

		// Test Batch Count Equivalence
		batchCounts, err := tok.CountOrdinaryBatch(testCases)
		if err != nil {
			t.Fatalf("CountOrdinaryBatch error: %v", err)
		}
		batchTokens, err := tok.EncodeOrdinaryBatch(testCases)
		if err != nil {
			t.Fatalf("EncodeOrdinaryBatch error: %v", err)
		}

		if len(batchCounts) != len(testCases) {
			t.Fatalf("batchCounts length mismatch: got %d, want %d", len(batchCounts), len(testCases))
		}
		for i, text := range testCases {
			expectedCount := len(batchTokens[i])
			if batchCounts[i] != expectedCount {
				t.Errorf("[%s] CountOrdinaryBatch[%d] (%q) = %d, want %d", encName, i, text, batchCounts[i], expectedCount)
			}
		}
	}
}

func TestBatchResultsOrderingAndCompleteness(t *testing.T) {
	tok, err := tokenizer.GetEncoding(tokenizer.CL100KBase)
	if err != nil {
		t.Fatalf("GetEncoding error: %v", err)
	}

	texts := []string{
		"first line",
		"second line with more tokens to process",
		"third",
		"fourth line is quite long and contains many words to test worker load balancing across chunks",
		"fifth line",
	}

	batchTokens, err := tok.EncodeOrdinaryBatch(texts)
	if err != nil {
		t.Fatalf("EncodeOrdinaryBatch error: %v", err)
	}

	if len(batchTokens) != len(texts) {
		t.Fatalf("got %d batch results, want %d", len(batchTokens), len(texts))
	}

	for i, text := range texts {
		seqTokens, err := tok.EncodeOrdinary(text)
		if err != nil {
			t.Fatalf("EncodeOrdinary error: %v", err)
		}
		if !reflect.DeepEqual(batchTokens[i], seqTokens) {
			t.Errorf("batch result [%d] = %v, want %v", i, batchTokens[i], seqTokens)
		}
	}
}
