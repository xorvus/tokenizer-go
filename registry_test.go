package tokenizer

import (
	"errors"
	"testing"
)

func TestRegistryResolution(t *testing.T) {
	res, err := ResolveModel("gpt-4o")
	if err != nil || res.CanonicalModel != "gpt-4o" || res.TokenizerID != "o200k_base" {
		t.Fatalf("ResolveModel('gpt-4o') unexpected result: %+v, err: %v", res, err)
	}
	if _, err := ResolveModel("nonexistent-model-xyz"); !errors.Is(err, ErrUnknownModel) {
		t.Errorf("expected ErrUnknownModel, got %v", err)
	}
}

func TestRegistryPrefixMatching(t *testing.T) {
	res, err := ResolveModel("gpt-4o-2024-05-13")
	if err != nil || res.CanonicalModel != "gpt-4o-2024-05-13" || res.TokenizerID != "o200k_base" {
		t.Fatalf("unexpected prefix resolution: %+v, err: %v", res, err)
	}
}

func TestRegistryAliasMatching(t *testing.T) {
	r := newDefaultRegistry()
	r.aliases["my-alias"] = "gpt-4"
	res, err := r.Resolve("my-alias")
	if err != nil || res.CanonicalModel != "gpt-4" || res.TokenizerID != "cl100k_base" {
		t.Fatalf("unexpected alias resolution: %+v, err: %v", res, err)
	}
}

func TestRegistryExactPrecedesAlias(t *testing.T) {
	r := newDefaultRegistry()
	r.aliases["gpt-4o"] = "gpt-3.5-turbo"
	res, err := r.Resolve("gpt-4o")
	if err != nil || res.CanonicalModel != "gpt-4o" || res.TokenizerID != "o200k_base" {
		t.Fatalf("exact match should take precedence over alias: %+v, err: %v", res, err)
	}
}
