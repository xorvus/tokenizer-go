package tokenizer_test

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/pkoukk/tiktoken-go/tokenizer-go"
	"github.com/pkoukk/tiktoken-go/tokenizer-go/internal/openai"
)

type TestCase struct {
	Text   string `json:"text"`
	Tokens []int  `json:"tokens"`
}

func loadGoldenCases(t *testing.T) []TestCase {
	data, err := os.ReadFile("testdata/cl100k_base.json")
	if err != nil {
		t.Fatalf("failed reading testdata: %v", err)
	}
	var cases []TestCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("failed parsing testdata: %v", err)
	}
	return cases
}

func TestRealCL100KCompatibility(t *testing.T) {
	tok, err := tokenizer.NewFromFile("testdata/cl100k_base.tiktoken", openai.PatternCL100K, openai.SpecialTokensCL100K())
	if err != nil {
		t.Fatalf("failed loading cl100k_base.tiktoken: %v", err)
	}
	cases := loadGoldenCases(t)
	for _, tc := range cases {
		tokens, err := tok.EncodeOrdinary(tc.Text)
		if err != nil {
			t.Errorf("EncodeOrdinary(%q) error: %v", tc.Text, err)
			continue
		}
		if !reflect.DeepEqual(tokens, tc.Tokens) {
			t.Errorf("EncodeOrdinary(%q) = %v, want %v", tc.Text, tokens, tc.Tokens)
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
