package tokenizer_test

import (
	"reflect"
	"testing"

	"github.com/xorvus/tokenizer-go"
	"github.com/xorvus/tokenizer-go/internal/openai"
)

func TestO200KBaseEncodeParity(t *testing.T) {
	tok, err := tokenizer.NewFromFile("testdata/o200k_base.tiktoken", openai.PatternO200K, openai.SpecialTokensO200K())
	if err != nil {
		t.Fatalf("failed loading o200k_base.tiktoken: %v", err)
	}
	cases := loadGoldenCases(t, "testdata/o200k_base.json")
	for _, tc := range cases {
		tokens, err := tok.EncodeOrdinary(tc.Text)
		if err != nil {
			t.Errorf("EncodeOrdinary(%q) error: %v", tc.Text, err)
			continue
		}
		if !reflect.DeepEqual(tokens, tc.Tokens) {
			t.Errorf("EncodeOrdinary(%q) = %v, want %v", tc.Text, tokens, tc.Tokens)
		}
	}
}

func TestO200KBaseDecodeRoundTrip(t *testing.T) {
	tok, err := tokenizer.NewFromFile("testdata/o200k_base.tiktoken", openai.PatternO200K, openai.SpecialTokensO200K())
	if err != nil {
		t.Fatalf("failed loading o200k_base.tiktoken: %v", err)
	}
	cases := loadGoldenCases(t, "testdata/o200k_base.json")
	for _, tc := range cases {
		tokens, err := tok.EncodeOrdinary(tc.Text)
		if err != nil {
			continue
		}
		decoded, err := tok.Decode(tokens)
		if err != nil {
			t.Errorf("Decode(%v) error: %v", tokens, err)
			continue
		}
		if decoded != tc.Text {
			t.Errorf("Decode(%v) = %q, want %q", tokens, decoded, tc.Text)
		}
	}
}

func TestO200KBaseSpecialTokens(t *testing.T) {
	tok, err := tokenizer.NewFromFile("testdata/o200k_base.tiktoken", openai.PatternO200K, openai.SpecialTokensO200K())
	if err != nil {
		t.Fatalf("failed loading o200k_base.tiktoken: %v", err)
	}
	tokens, err := tok.EncodeOrdinary("<|endoftext|>")
	if err != nil {
		t.Fatalf("EncodeOrdinary error: %v", err)
	}
	if len(tokens) == 0 {
		t.Errorf("expected tokens")
	}
}
