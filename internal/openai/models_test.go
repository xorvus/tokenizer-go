package openai_test

import (
	"testing"

	"github.com/xorvus/tokenizer-go/internal/openai"
)

func TestEncodingNameForModel(t *testing.T) {
	tests := map[string]string{
		"gpt-4o":                  "o200k_base",
		"o1-preview":              "o200k_base",
		"gpt-4-0314":              "cl100k_base",
		"gpt-3.5-turbo-0613":      "cl100k_base",
		"text-embedding-3-small":  "cl100k_base",
		"text-davinci-003":        "p50k_base",
		"davinci":                 "r50k_base",
	}

	for model, expected := range tests {
		got, err := openai.EncodingNameForModel(model)
		if err != nil {
			t.Errorf("EncodingNameForModel(%q) unexpected error: %v", model, err)
			continue
		}
		if got != expected {
			t.Errorf("EncodingNameForModel(%q) = %q, want %q", model, got, expected)
		}
	}

	_, err := openai.EncodingNameForModel("non-existent-model")
	if err == nil {
		t.Errorf("expected error for unknown model")
	}
}
