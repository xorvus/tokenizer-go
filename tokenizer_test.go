package tokenizer_test

import (
	"errors"
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
