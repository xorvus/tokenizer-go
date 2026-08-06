package tokenizer_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/xorvus/tokenizer-go"
)

func createValidTestRanks() map[string]int {
	ranks := make(map[string]int, 256+5)
	for b := 0; b <= 255; b++ {
		ranks[string([]byte{byte(b)})] = b
	}
	ranks["he"] = 256
	ranks["ll"] = 257
	ranks["o"] = 258
	ranks["hello"] = 259
	return ranks
}

func TestCustomEncodingSuccessAndImmutability(t *testing.T) {
	ranks := createValidTestRanks()
	special := map[string]int{
		"<|endoftext|>": 260,
	}

	cfg := tokenizer.Config{
		Name:           "custom-test",
		Pattern:        `(?i:'s|'t|'re|'ve|'m|'ll|'d)|[^\r\n\p{L}\p{N}]?\p{L}+|\p{N}{1,3}| ?[^\s\p{L}\p{N}]+[\r\n]*|\s*[\r\n]+|\s+(?!\S)|\s+`,
		MergeableRanks: ranks,
		SpecialTokens:  special,
	}

	tok, err := tokenizer.New(cfg)
	if err != nil {
		t.Fatalf("New(cfg) error: %v", err)
	}

	// Test immutability: modify original map
	ranks["hello"] = 9999
	delete(ranks, "he")
	special["<|endoftext|>"] = 8888

	tokens, err := tok.EncodeOrdinary("hello")
	if err != nil {
		t.Fatalf("EncodeOrdinary error: %v", err)
	}
	if len(tokens) != 1 || tokens[0] != 259 {
		t.Errorf("got %v, want [259]", tokens)
	}

	singleID, err := tok.EncodeSingleToken("<|endoftext|>")
	if err != nil {
		t.Fatalf("EncodeSingleToken error: %v", err)
	}
	if singleID != 260 {
		t.Errorf("got %d, want 260", singleID)
	}
}

func TestCustomEncodingValidationFailures(t *testing.T) {
	validRanks := createValidTestRanks()
	validPattern := `\p{L}+|\p{N}+|\s+`

	tests := []struct {
		name        string
		cfg         tokenizer.Config
		expectedErr error
	}{
		{
			name: "EmptyPattern",
			cfg: tokenizer.Config{
				Pattern:        "",
				MergeableRanks: validRanks,
			},
			expectedErr: tokenizer.ErrEmptyPattern,
		},
		{
			name: "EmptyVocabulary",
			cfg: tokenizer.Config{
				Pattern:        validPattern,
				MergeableRanks: nil,
			},
			expectedErr: tokenizer.ErrEmptyVocabulary,
		},
		{
			name: "MissingByteToken",
			cfg: tokenizer.Config{
				Pattern: validPattern,
				MergeableRanks: func() map[string]int {
					r := createValidTestRanks()
					delete(r, "\x05")
					return r
				}(),
			},
			expectedErr: tokenizer.ErrMissingByteToken,
		},
		{
			name: "NegativeRank",
			cfg: tokenizer.Config{
				Pattern: validPattern,
				MergeableRanks: func() map[string]int {
					r := createValidTestRanks()
					r["bad"] = -10
					return r
				}(),
			},
			expectedErr: tokenizer.ErrNegativeRank,
		},
		{
			name: "DuplicateRank",
			cfg: tokenizer.Config{
				Pattern: validPattern,
				MergeableRanks: func() map[string]int {
					r := createValidTestRanks()
					r["dupe1"] = 300
					r["dupe2"] = 300
					return r
				}(),
			},
			expectedErr: tokenizer.ErrDuplicateRank,
		},
		{
			name: "TokenConflict",
			cfg: tokenizer.Config{
				Pattern:        validPattern,
				MergeableRanks: validRanks,
				SpecialTokens: map[string]int{
					"he": 999, // "he" already exists in MergeableRanks
				},
			},
			expectedErr: tokenizer.ErrTokenConflict,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tokenizer.New(tc.cfg)
			if err == nil {
				t.Fatalf("expected error %v, got nil", tc.expectedErr)
			}
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("got error %v, want %v", err, tc.expectedErr)
			}
			fmt.Sprint(err)
		})
	}
}
