package tokenizer_test

import (
	"reflect"
	"testing"

	"github.com/xorvus/tokenizer-go"
	"github.com/xorvus/tokenizer-go/internal/openai"
)

type TestCase struct {
	Text   string
	Tokens []int
}

var goldenCL100KCases = []TestCase{
	{Text: "", Tokens: []int{}},
	{Text: "hello", Tokens: []int{15339}},
	{Text: "hello world", Tokens: []int{15339, 1917}},
	{Text: "Hello, world!", Tokens: []int{9906, 11, 1917, 0}},
	{Text: "你好世界", Tokens: []int{57668, 53901, 3574, 244, 98220}},
	{Text: "こんにちは", Tokens: []int{90115}},
	{Text: "😀", Tokens: []int{76460, 222}},
	{Text: "hello\nworld", Tokens: []int{15339, 198, 14957}},
}

func TestRealCL100KCompatibility(t *testing.T) {
	tok, err := tokenizer.NewFromFile("test/cl100k_base.tiktoken", openai.PatternCL100K, openai.SpecialTokensCL100K())
	if err != nil {
		t.Fatalf("failed loading cl100k_base.tiktoken: %v", err)
	}
	for _, tc := range goldenCL100KCases {
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
