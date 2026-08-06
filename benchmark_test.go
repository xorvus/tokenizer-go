package tokenizer_test

import (
	"os"
	"strings"
	"testing"

	"github.com/xorvus/tokenizer-go"
)

func BenchmarkEmbeddedCL100KShortASCII(b *testing.B) {
	tok, err := tokenizer.GetEncoding(tokenizer.CL100KBase)
	if err != nil {
		b.Fatalf("failed loading tokenizer: %v", err)
	}
	text := "Hello, world!"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tok.EncodeOrdinary(text)
	}
}

func BenchmarkEmbeddedCL100KUnicode(b *testing.B) {
	tok, err := tokenizer.GetEncoding(tokenizer.CL100KBase)
	if err != nil {
		b.Fatalf("failed loading tokenizer: %v", err)
	}
	text := "你好世界こんにちは😀"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tok.EncodeOrdinary(text)
	}
}

func BenchmarkEmbeddedCL100KFullUDHR(b *testing.B) {
	tok, err := tokenizer.GetEncoding(tokenizer.CL100KBase)
	if err != nil {
		b.Fatalf("failed loading tokenizer: %v", err)
	}
	text, err := os.ReadFile("/tmp/udhr.txt")
	if err != nil {
		b.Skip("udhr.txt not found in /tmp")
	}
	input := string(text)
	b.ResetTimer()
	b.SetBytes(int64(len(input)))
	for i := 0; i < b.N; i++ {
		_, _ = tok.EncodeOrdinary(input)
	}
}

func BenchmarkEncodingInFullLanguage(b *testing.B) {
	data, err := os.ReadFile("/tmp/udhr.txt")
	if err != nil {
		b.Skip("udhr.txt not found in /tmp")
	}

	lines := strings.Split(string(data), "\n")
	tok, err := tokenizer.ForModel("gpt-4o")
	if err != nil {
		b.Fatalf("failed loading tokenizer: %v", err)
	}
	lineCount := len(lines)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, _ = tok.EncodeOrdinary(lines[n%lineCount])
	}
}
