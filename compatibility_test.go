package tokenizer_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/pkoukk/tiktoken-go/tokenizer-go"
	"github.com/pkoukk/tiktoken-go/tokenizer-go/internal/openai"
)

type TestCase struct {
	Text   string `json:"text"`
	Tokens []int  `json:"tokens"`
}

func TestCompatibility(t *testing.T) {
	data, err := os.ReadFile("testdata/cl100k_base.json")
	if err != nil {
		t.Fatalf("failed reading testdata: %v", err)
	}
	var cases []TestCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("failed parsing testdata: %v", err)
	}

	vocabData := "aGVsbG8= 15339\nIHdvcmxk 1917\n"
	tok, err := tokenizer.NewFromVocabulary(strings.NewReader(vocabData), openai.PatternCL100K, nil)
	if err != nil {
		t.Fatalf("failed creating tokenizer: %v", err)
	}

	tokens, err := tok.Encode("hello world")
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}
	if len(tokens) != 2 {
		t.Errorf("got len %d, want 2", len(tokens))
	}
}
