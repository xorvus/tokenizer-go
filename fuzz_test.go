package tokenizer_test

import (
	"testing"

	"github.com/xorvus/tokenizer-go"
)

func FuzzEncodeDecode(f *testing.F) {
	tok, err := tokenizer.GetEncoding(tokenizer.CL100KBase)
	if err != nil {
		f.Fatalf("failed loading cl100k: %v", err)
	}

	f.Add("")
	f.Add("hello world")
	f.Add("Bahasa Indonesia")
	f.Add("你好世界")
	f.Add("👨👩👧👦")
	f.Add("e\u0301")

	f.Fuzz(func(t *testing.T, input string) {
		tokens, err := tok.EncodeOrdinary(input)
		if err != nil {
			t.Fatalf("EncodeOrdinary error: %v", err)
		}
		decoded, err := tok.Decode(tokens)
		if err != nil {
			t.Fatalf("Decode error: %v", err)
		}
		if decoded != input {
			t.Fatalf("round-trip mismatch: got %q, want %q", decoded, input)
		}
	})
}
