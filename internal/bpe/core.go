package bpe

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/dlclark/regexp2/v2"
)

type Config struct {
	Pattern        string
	MergeableRanks map[string]int
	DecoderSlice   []string
	SpecialTokens  map[string]int
}

type CoreBPE struct {
	Encoder              map[string]int
	RankIndex            RankIndex
	DecoderSlice         []string
	SpecialTokensEncoder map[string]int
	SpecialTokensDecoder map[int]string
	Regex                *regexp2.Regexp
	SpecialRegex         *regexp2.Regexp
}

type SpecialMatch struct {
	Token     string
	RuneEnd   int
	ByteStart int
	ByteEnd   int
}

func ValidateTokenIDs(encoder map[string]int, special map[string]int) error {
	ids := make(map[int]string, len(encoder)+len(special))
	validate := func(kind string, tokens map[string]int) error {
		for token, id := range tokens {
			if token == "" {
				return fmt.Errorf("%s token cannot be empty", kind)
			}
			if id < 0 {
				return fmt.Errorf("%s token %q has negative ID %d", kind, token, id)
			}
			if previous, exists := ids[id]; exists {
				return fmt.Errorf("token ID %d is used by %q and %q", id, previous, token)
			}
			ids[id] = token
		}
		return nil
	}
	if err := validate("regular", encoder); err != nil {
		return err
	}
	return validate("special", special)
}

func BuildSpecialRegex(specialTokensEncoder map[string]int) (*regexp2.Regexp, error) {
	if len(specialTokensEncoder) == 0 {
		return nil, nil
	}
	tokens := make([]string, 0, len(specialTokensEncoder))
	for token := range specialTokensEncoder {
		if token == "" {
			return nil, errors.New("special token cannot be empty")
		}
		tokens = append(tokens, token)
	}
	sort.Slice(tokens, func(i, j int) bool {
		if len(tokens[i]) != len(tokens[j]) {
			return len(tokens[i]) > len(tokens[j])
		}
		return tokens[i] < tokens[j]
	})
	for i := range tokens {
		tokens[i] = regexp.QuoteMeta(tokens[i])
	}
	return regexp2.Compile(strings.Join(tokens, "|"), regexp2.None)
}

func New(cfg Config) (*CoreBPE, error) {
	return NewCoreBPE(cfg.MergeableRanks, cfg.DecoderSlice, cfg.SpecialTokens, cfg.Pattern)
}

func NewCoreBPE(encoder map[string]int, decoder []string, special map[string]int, pattern string) (*CoreBPE, error) {
	if err := ValidateTokenIDs(encoder, special); err != nil {
		return nil, err
	}
	regex, err := regexp2.Compile(pattern, regexp2.None)
	if err != nil {
		return nil, fmt.Errorf("error compiling regex: %w", err)
	}
	specialRegex, err := BuildSpecialRegex(special)
	if err != nil {
		return nil, fmt.Errorf("error compiling special regex: %w", err)
	}
	specialDecoder := make(map[int]string, len(special))
	for k, v := range special {
		specialDecoder[v] = k
	}
	return &CoreBPE{
		Encoder:              encoder,
		RankIndex:            NewRankIndex(encoder),
		DecoderSlice:         decoder,
		SpecialTokensEncoder: special,
		SpecialTokensDecoder: specialDecoder,
		Regex:                regex,
		SpecialRegex:         specialRegex,
	}, nil
}
