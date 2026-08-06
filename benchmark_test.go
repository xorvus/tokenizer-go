package tokenizer_test

import (
	"strings"
	"testing"
)

func BenchmarkEncodeShortASCII(b *testing.B) {
	text := "Hello world"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = len(text)
	}
}

func BenchmarkEncodeLongASCII(b *testing.B) {
	text := strings.Repeat("Hello world, this is a test. ", 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = len(text)
	}
}

func BenchmarkEncodeUnicode(b *testing.B) {
	text := "你好世界こんにちは😀"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = len(text)
	}
}
