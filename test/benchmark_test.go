package main

import (
	"log"
	"os"
	"strings"
	"testing"

	"github.com/xorvus/tokenizer-go"
)

var tokenSink [][]int

func BenchmarkEncodingInFullLanguage(b *testing.B) {
	data, err := os.ReadFile("/tmp/udhr.txt")
	if err != nil {
		log.Fatal(err)
	}

	lines := strings.Split(string(data), "\n")
	tok, err := tokenizer.ForModel("gpt-4o")
	lineCount := len(lines)
	if err != nil {
		log.Fatal(err)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, _ = tok.EncodeOrdinary(lines[n%lineCount])
	}
}

func BenchmarkSequentialLoopUDHR(b *testing.B) {
	data, err := os.ReadFile("/tmp/udhr.txt")
	if err != nil {
		b.Skip("udhr.txt not found in /tmp")
	}

	lines := strings.Split(string(data), "\n")
	tok, err := tokenizer.ForModel("gpt-4o")
	if err != nil {
		b.Fatalf("failed loading tokenizer: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results := make([][]int, len(lines))
		for idx, line := range lines {
			results[idx], _ = tok.EncodeOrdinary(line)
		}
		tokenSink = results
	}
}

func BenchmarkEncodeOrdinaryBatchUDHR(b *testing.B) {
	data, err := os.ReadFile("/tmp/udhr.txt")
	if err != nil {
		b.Skip("udhr.txt not found in /tmp")
	}

	lines := strings.Split(string(data), "\n")
	tok, err := tokenizer.ForModel("gpt-4o")
	if err != nil {
		b.Fatalf("failed loading tokenizer: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tokenSink, _ = tok.EncodeOrdinaryBatch(lines)
	}
}

func BenchmarkCountOrdinaryBatchUDHR(b *testing.B) {
	data, err := os.ReadFile("/tmp/udhr.txt")
	if err != nil {
		b.Skip("udhr.txt not found in /tmp")
	}

	lines := strings.Split(string(data), "\n")
	tok, err := tokenizer.ForModel("gpt-4o")
	if err != nil {
		b.Fatalf("failed loading tokenizer: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tok.CountOrdinaryBatch(lines)
	}
}
