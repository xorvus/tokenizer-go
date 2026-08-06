package tokenizer_test

import (
	"reflect"
	"testing"

	"github.com/xorvus/tokenizer-go"
)

func TestO200KHarmonyAndModelMappings(t *testing.T) {
	tok, err := tokenizer.GetEncoding(tokenizer.O200KHarmony)
	if err != nil {
		t.Fatalf("GetEncoding(O200KHarmony) error: %v", err)
	}

	tokens, err := tok.EncodeOrdinary("hello world")
	if err != nil {
		t.Fatalf("EncodeOrdinary error: %v", err)
	}
	if len(tokens) == 0 {
		t.Errorf("expected non-empty tokens for O200KHarmony")
	}

	modelTok, err := tokenizer.ForModel("gpt-oss-1")
	if err != nil {
		t.Fatalf("ForModel(gpt-oss-1) error: %v", err)
	}
	modelTokens, err := modelTok.EncodeOrdinary("hello world")
	if err != nil {
		t.Fatalf("EncodeOrdinary error: %v", err)
	}
	if !reflect.DeepEqual(tokens, modelTokens) {
		t.Errorf("gpt-oss-1 encoding mismatch: %v vs %v", modelTokens, tokens)
	}
}

func TestAPIParityMethods(t *testing.T) {
	tok, err := tokenizer.GetEncoding(tokenizer.O200KBase)
	if err != nil {
		t.Fatalf("failed loading o200k_base: %v", err)
	}

	// 1. EncodeSingleToken
	singleToken, err := tok.EncodeSingleToken("hello")
	if err != nil {
		t.Fatalf("EncodeSingleToken('hello') error: %v", err)
	}

	// 2. DecodeSingleTokenBytes
	b, err := tok.DecodeSingleTokenBytes(singleToken)
	if err != nil {
		t.Fatalf("DecodeSingleTokenBytes(%d) error: %v", singleToken, err)
	}
	if string(b) != "hello" {
		t.Errorf("got %q, want 'hello'", string(b))
	}

	// 3. TokenByteValues
	allBytes := tok.TokenByteValues()
	if len(allBytes) == 0 {
		t.Fatalf("TokenByteValues returned empty slice")
	}

	// 4. DecodeWithOffsets
	prompt := "hello world from go"
	tokens, err := tok.EncodeOrdinary(prompt)
	if err != nil {
		t.Fatalf("EncodeOrdinary error: %v", err)
	}
	decodedText, offsets, err := tok.DecodeWithOffsets(tokens)
	if err != nil {
		t.Fatalf("DecodeWithOffsets error: %v", err)
	}
	if decodedText != prompt {
		t.Errorf("decodedText = %q, want %q", decodedText, prompt)
	}
	if len(offsets) != len(tokens) {
		t.Errorf("offsets len = %d, want %d", len(offsets), len(tokens))
	}

	// 5. DecodeBatch
	batchInput := [][]int{tokens, tokens}
	decodedBatch, err := tok.DecodeBatch(batchInput)
	if err != nil {
		t.Fatalf("DecodeBatch error: %v", err)
	}
	if len(decodedBatch) != 2 || decodedBatch[0] != prompt || decodedBatch[1] != prompt {
		t.Errorf("DecodeBatch result mismatch: %v", decodedBatch)
	}

	// 6. DecodeBytesBatch
	decodedBytesBatch, err := tok.DecodeBytesBatch(batchInput)
	if err != nil {
		t.Fatalf("DecodeBytesBatch error: %v", err)
	}
	if len(decodedBytesBatch) != 2 || string(decodedBytesBatch[0]) != prompt {
		t.Errorf("DecodeBytesBatch result mismatch: %v", decodedBytesBatch)
	}
}
