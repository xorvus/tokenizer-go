package main

import (
	"log"
	"os"
	"strings"
	"testing"

	"github.com/xorvus/tokenizer-go"
)

// go test -benchmem -run=^$ -bench ^BenchmarkEncodingInFullLanguage$ -benchtime=100000x github.com/xorvus/tokenizer-go/test

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
