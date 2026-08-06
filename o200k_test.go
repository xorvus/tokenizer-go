package tokenizer_test

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/pkoukk/tiktoken-go/tokenizer-go"
	"github.com/pkoukk/tiktoken-go/tokenizer-go/internal/openai"
)

func loadO200KCases(t *testing.T) []TestCase {
	data, err := os.ReadFile("testdata/o200k_base.json")
	if err != nil {
		t.Fatalf("failed reading o200k testdata: %v", err)
	}
	var cases []TestCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("failed parsing o200k testdata: %v", err)
	}
	return cases
}

func TestO200KBaseEncodeParity(t *testing.T) {
	tok, err := tokenizer.NewFromFile("testdata/o200k_base.tiktoken", openai.PatternO200K, openai.SpecialTokensO200K())
	if err != nil {
		t.Fatalf("failed loading o200k_base.tiktoken: %v", err)
	}
	cases := loadO200KCases(t)
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
	cases := loadO200KCases(t)
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
