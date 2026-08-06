package openai_test

import (
	"strings"
	"testing"

	"github.com/xorvus/tokenizer-go/internal/openai"
)

func TestParseVocabulary(t *testing.T) {
	input := "aGVsbG8= 15339\nd29ybGQ= 1917\n"
	vocab, err := openai.ParseVocabulary(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vocab.Encoder["hello"] != 15339 {
		t.Errorf("expected hello -> 15339, got %d", vocab.Encoder["hello"])
	}
	if vocab.Encoder["world"] != 1917 {
		t.Errorf("expected world -> 1917, got %d", vocab.Encoder["world"])
	}
}
