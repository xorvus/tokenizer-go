package tokenizer_test

import (
	"os"
	"testing"

	"github.com/pkoukk/tokenizer-go"
)

func BenchmarkEmbeddedCL100KShortASCII(b *testing.B) {
	tok, err := tokenizer.GetEmbeddedCL100K()
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
	tok, err := tokenizer.GetEmbeddedCL100K()
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
	tok, err := tokenizer.GetEmbeddedCL100K()
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
