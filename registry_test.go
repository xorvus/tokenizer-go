package tokenizer_test

import (
	"errors"
	"testing"

	"github.com/xorvus/tokenizer-go"
)

func TestRegistryResolution(t *testing.T) {
	res, err := tokenizer.ResolveModel("gpt-4o")
	if err != nil {
		t.Fatalf("ResolveModel('gpt-4o') error: %v", err)
	}
	if res.CanonicalModel != "gpt-4o" || res.TokenizerID != "o200k_base" {
		t.Errorf("unexpected resolution: %+v", res)
	}

	_, err = tokenizer.ResolveModel("nonexistent-model-xyz")
	if !errors.Is(err, tokenizer.ErrUnknownModel) {
		t.Errorf("expected ErrUnknownModel, got %v", err)
	}
}
