package tokenizer_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/xorvus/tokenizer-go"
)

// TestAllSingleByteTokensExist verifies all 256 byte values (0-255) can be encoded without panic or error.
func TestAllSingleByteTokensExist(t *testing.T) {
	encodings := []tokenizer.Encoding{
		tokenizer.CL100KBase,
		tokenizer.O200KBase,
	}

	for _, encName := range encodings {
		t.Run(string(encName), func(t *testing.T) {
			tok, err := tokenizer.GetEncoding(encName)
			if err != nil {
				t.Fatalf("failed loading %s: %v", encName, err)
			}

			// Test every single byte 0..255 individually
			for b := 0; b <= 255; b++ {
				singleByte := []byte{byte(b)}
				tokens, err := tok.EncodeOrdinary(string(singleByte))
				if err != nil {
					t.Fatalf("EncodeOrdinary failed for byte 0x%02X (%d): %v", b, b, err)
				}
				if len(tokens) == 0 {
					t.Errorf("EncodeOrdinary returned 0 tokens for byte 0x%02X (%d)", b, b)
				}
			}

			// Test sequence of all 256 bytes in one string
			allBytes := make([]byte, 256)
			for b := 0; b <= 255; b++ {
				allBytes[b] = byte(b)
			}

			tokens, err := tok.EncodeOrdinary(string(allBytes))
			if err != nil {
				t.Fatalf("EncodeOrdinary failed for full 0-255 byte sequence: %v", err)
			}
			if len(tokens) == 0 {
				t.Errorf("EncodeOrdinary returned 0 tokens for full byte sequence")
			}

			count, err := tok.Count(string(allBytes))
			if err != nil {
				t.Fatalf("Count failed for full 0-255 byte sequence: %v", err)
			}
			if count != len(tokens) {
				t.Errorf("Count (%d) != len(EncodeOrdinary) (%d) for 0-255 byte sequence", count, len(tokens))
			}
		})
	}
}

// TestAdversarialAndMalformedInputs verifies resilience against unexpected inputs.
func TestAdversarialAndMalformedInputs(t *testing.T) {
	tok, err := tokenizer.GetEncoding(tokenizer.O200KBase)
	if err != nil {
		t.Fatalf("failed loading o200k_base: %v", err)
	}

	adversarialCases := []struct {
		name string
		data []byte
	}{
		{"NullBytes", []byte("\x00\x00\x00\x00")},
		{"InvalidUTF8_Truncated2Byte", []byte("\xc2")},
		{"InvalidUTF8_Truncated3Byte", []byte("\xe2\x82")},
		{"InvalidUTF8_Truncated4Byte", []byte("\xf0\x90\x80")},
		{"InvalidUTF8_OverlongEncoding", []byte("\xc0\xaf")},
		{"InvalidUTF8_SurrogateHalf", []byte("\xed\xa0\x80")},
		{"InvalidUTF8_FFByte", []byte("\xff\xff\xff\xff")},
		{"AdversarialSpaces", []byte("        \t\t\n\r\n        ")},
		{"GiantRepeatedString", bytes.Repeat([]byte("a"), 100000)},
		{"GiantUnstructuredBase64", bytes.Repeat([]byte("aB3+/="), 20000)},
	}

	for _, tc := range adversarialCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("PANIC detected on case %s: %v", tc.name, r)
				}
			}()

			tokens, err := tok.EncodeOrdinary(string(tc.data))
			if err != nil {
				t.Fatalf("unexpected error on %s: %v", tc.name, err)
			}
			count, err := tok.Count(string(tc.data))
			if err != nil {
				t.Fatalf("unexpected count error on %s: %v", tc.name, err)
			}
			if count != len(tokens) {
				t.Errorf("%s: Count (%d) != len(EncodeOrdinary) (%d)", tc.name, count, len(tokens))
			}
		})
	}
}

// FuzzEncodeOrdinary provides fuzz testing for EncodeOrdinary.
func FuzzEncodeOrdinary(f *testing.F) {
	seeds := [][]byte{
		[]byte("hello world"),
		[]byte("https://example.com/api/v1?test=true"),
		[]byte("{\x22key\x22: \x22value\x22}"),
		[]byte("🚀👨‍👩‍👧‍👦✨🎉"),
		[]byte("\xff\xfe\xfd\x00\x01\x02"),
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	tok, err := tokenizer.GetEncoding(tokenizer.O200KBase)
	if err != nil {
		f.Fatalf("failed loading o200k_base: %v", err)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		text := string(data)
		tokens, err := tok.EncodeOrdinary(text)
		if err != nil {
			t.Fatalf("EncodeOrdinary error: %v", err)
		}
		count, err := tok.Count(text)
		if err != nil {
			t.Fatalf("Count error: %v", err)
		}
		if count != len(tokens) {
			t.Fatalf("Fuzz Count mismatch: count=%d, len=%d for text len=%d", count, len(tokens), len(text))
		}
		fmt.Sprint(tokens)
	})
}
