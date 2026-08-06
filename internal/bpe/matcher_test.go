package bpe

import (
	"os"
	"testing"

	"github.com/dlclark/regexp2/v2"
	"github.com/xorvus/tokenizer-go/internal/openai"
)

func collectRegexp2Matches(text string, re *regexp2.Regexp) [][2]int {
	var matches [][2]int
	m, _ := re.FindStringMatch(text)
	ascii := isASCII(text)
	for m != nil {
		var start, end int
		if ascii {
			start = m.RuneIndex
			end = start + m.RuneLength
		} else {
			byteStart, byteLength := m.ByteRange()
			start = byteStart
			end = byteStart + byteLength
		}
		matches = append(matches, [2]int{start, end})
		m, _ = re.FindNextMatch(m)
	}
	return matches
}

func collectFindAllStringIndexMatches(text string, re *regexp2.Regexp) [][2]int {
	indices, err := re.FindAllStringIndex(text, -1)
	if err != nil || len(indices) == 0 {
		return nil
	}
	matches := make([][2]int, len(indices))
	for i, pair := range indices {
		matches[i] = [2]int{pair[0], pair[1]}
	}
	return matches
}

func TestMatcherParityASCII(t *testing.T) {
	re := regexp2.MustCompile(openai.PatternCL100K, regexp2.None)
	text := "Hello, world! This is a test for tiktoken pre-tokenization."
	want := collectRegexp2Matches(text, re)
	got := collectFindAllStringIndexMatches(text, re)
	if len(want) != len(got) {
		t.Fatalf("length mismatch: want %d, got %d", len(want), len(got))
	}
	for i := range want {
		if want[i] != got[i] {
			t.Fatalf("match %d mismatch: want %v, got %v", i, want[i], got[i])
		}
	}
}

func TestMatcherParityUnicode(t *testing.T) {
	re := regexp2.MustCompile(openai.PatternCL100K, regexp2.None)
	text := "你好世界 こんにちは 👨👩👧👦 e\u0301"
	want := collectRegexp2Matches(text, re)
	got := collectFindAllStringIndexMatches(text, re)
	if len(want) != len(got) {
		t.Fatalf("length mismatch: want %d, got %d", len(want), len(got))
	}
	for i := range want {
		if want[i] != got[i] {
			t.Fatalf("match %d mismatch: want %v, got %v", i, want[i], got[i])
		}
	}
}

func TestMatcherParityUDHR(t *testing.T) {
	re := regexp2.MustCompile(openai.PatternCL100K, regexp2.None)
	data, err := os.ReadFile("/tmp/udhr.txt")
	if err != nil {
		t.Skip("/tmp/udhr.txt not found")
	}
	text := string(data)
	want := collectRegexp2Matches(text, re)
	got := collectFindAllStringIndexMatches(text, re)
	if len(want) != len(got) {
		t.Fatalf("length mismatch: want %d, got %d", len(want), len(got))
	}
	for i := range want {
		if want[i] != got[i] {
			t.Fatalf("match %d mismatch: want %v, got %v", i, want[i], got[i])
		}
	}
}

func BenchmarkRegexp2MatchIndexesUDHR(b *testing.B) {
	re := regexp2.MustCompile(openai.PatternCL100K, regexp2.None)
	data, err := os.ReadFile("/tmp/udhr.txt")
	if err != nil {
		b.Skip("/tmp/udhr.txt not found")
	}
	text := string(data)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = collectRegexp2Matches(text, re)
	}
}

func collectStreamingMatches(text string, re *regexp2.Regexp) [][2]int {
	it := NewMatchIterator(text, re)
	var matches [][2]int
	for {
		start, end, ok, err := it.Next(text)
		if err != nil || !ok {
			break
		}
		matches = append(matches, [2]int{start, end})
	}
	return matches
}



func TestStreamingParityUnicode(t *testing.T) {
	re := regexp2.MustCompile(openai.PatternCL100K, regexp2.None)
	text := "你好世界 こんにちは 👨👩👧👦 e\u0301"
	want := collectRegexp2Matches(text, re)
	got := collectStreamingMatches(text, re)
	if len(want) != len(got) {
		t.Fatalf("length mismatch: want %d, got %d", len(want), len(got))
	}
	for i := range want {
		if want[i] != got[i] {
			t.Fatalf("match %d mismatch: want %v, got %v", i, want[i], got[i])
		}
	}
}

func TestStreamingParityUDHR(t *testing.T) {
	re := regexp2.MustCompile(openai.PatternCL100K, regexp2.None)
	data, err := os.ReadFile("/tmp/udhr.txt")
	if err != nil {
		t.Skip("/tmp/udhr.txt not found")
	}
	text := string(data)
	want := collectRegexp2Matches(text, re)
	got := collectStreamingMatches(text, re)
	if len(want) != len(got) {
		t.Fatalf("length mismatch: want %d, got %d", len(want), len(got))
	}
	for i := range want {
		if want[i] != got[i] {
			t.Fatalf("match %d mismatch: want %v, got %v", i, want[i], got[i])
		}
	}
}

func BenchmarkStreamingMatcherUDHR(b *testing.B) {
	re := regexp2.MustCompile(openai.PatternCL100K, regexp2.None)
	data, err := os.ReadFile("/tmp/udhr.txt")
	if err != nil {
		b.Skip("/tmp/udhr.txt not found")
	}
	text := string(data)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = collectStreamingMatches(text, re)
	}
}
