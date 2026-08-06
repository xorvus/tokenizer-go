package tokenizer_test

import (
	"reflect"
	"testing"

	"github.com/xorvus/tokenizer-go"
)

func TestBatchAPIAcceptanceGates(t *testing.T) {
	tok, err := tokenizer.GetEncoding(tokenizer.CL100KBase)
	if err != nil {
		t.Fatalf("failed loading cl100k_base: %v", err)
	}

	t.Run("EmptyBatch", func(t *testing.T) {
		encoded, err := tok.EncodeOrdinaryBatch([]string{})
		if err != nil || len(encoded) != 0 {
			t.Errorf("expected empty result, got %v, err: %v", encoded, err)
		}
		counted, err := tok.CountOrdinaryBatch([]string{})
		if err != nil || len(counted) != 0 {
			t.Errorf("expected empty count result, got %v, err: %v", counted, err)
		}
	})

	t.Run("OrderAndParityAndEmptyTexts", func(t *testing.T) {
		texts := []string{
			"Hello, world!",
			"",
			"你好世界こんにちは😀",
			"Another independent prompt line",
			"",
			"Final line for batch tokenization testing",
		}

		encoded, err := tok.EncodeOrdinaryBatch(texts)
		if err != nil {
			t.Fatalf("EncodeOrdinaryBatch failed: %v", err)
		}
		if len(encoded) != len(texts) {
			t.Fatalf("expected len %d, got %d", len(texts), len(encoded))
		}

		counts, err := tok.CountOrdinaryBatch(texts)
		if err != nil {
			t.Fatalf("CountOrdinaryBatch failed: %v", err)
		}
		if len(counts) != len(texts) {
			t.Fatalf("expected len %d, got %d", len(texts), len(counts))
		}

		for i, text := range texts {
			expectedTokens, err := tok.EncodeOrdinary(text)
			if err != nil {
				t.Fatalf("EncodeOrdinary(%q) error: %v", text, err)
			}
			if !reflect.DeepEqual(encoded[i], expectedTokens) {
				t.Errorf("item %d (%q): batch token %v != expected %v", i, text, encoded[i], expectedTokens)
			}
			if counts[i] != len(expectedTokens) {
				t.Errorf("item %d (%q): batch count %d != expected %d", i, text, counts[i], len(expectedTokens))
			}
		}
	})
}
