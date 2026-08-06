package main

import (
	"os"
	"strings"
	"testing"

	"github.com/xorvus/tokenizer-go"
)

// BenchmarkAblationSuite runs normalized isolation benchmarks across feature layers.
func BenchmarkAblationSuite(b *testing.B) {
	tok, err := tokenizer.GetEncoding(tokenizer.CL100KBase)
	if err != nil {
		b.Fatalf("failed loading CL100KBase: %v", err)
	}

	shortText := "Hello, world! This is a test prompt for tokenization benchmark."

	content, err := os.ReadFile("/tmp/udhr.txt")
	var udhrLines []string
	if err == nil {
		udhrLines = strings.Split(string(content), "\n")
	} else {
		udhrLines = []string{shortText}
	}

	b.Run("1_SingleLine_ShortASCII", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = tok.EncodeOrdinary(shortText)
		}
	})

	b.Run("2_SingleLine_CountOnly_ShortASCII", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = tok.Count(shortText)
		}
	})

	b.Run("3_Batch_SequentialLoop", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, line := range udhrLines {
				_, _ = tok.EncodeOrdinary(line)
			}
		}
	})

	b.Run("4_Batch_StaticPartitioning", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = tok.EncodeOrdinaryBatch(udhrLines)
		}
	})

	b.Run("5_Batch_CountOnlyPath", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = tok.CountOrdinaryBatch(udhrLines)
		}
	})
}
