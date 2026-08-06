package tokenizer

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pkoukk/tokenizer-go/internal/bpe"
	"github.com/pkoukk/tokenizer-go/internal/openai"
)

type Tokenizer struct {
	core *bpe.CoreBPE
}

func NewFromVocabulary(r io.Reader, pattern string, special map[string]int) (*Tokenizer, error) {
	vocab, err := openai.ParseVocabulary(r)
	if err != nil {
		return nil, err
	}
	core, err := bpe.New(bpe.Config{
		Pattern:        pattern,
		MergeableRanks: vocab.Encoder,
		DecoderSlice:   vocab.Decoder,
		SpecialTokens:  special,
	})
	if err != nil {
		return nil, err
	}
	return &Tokenizer{core: core}, nil
}

func NewFromFile(path string, pattern string, special map[string]int) (*Tokenizer, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return NewFromVocabulary(f, pattern, special)
}

func GetEncoding(name Encoding) (*Tokenizer, error) {
	switch name {
	case CL100KBase:
		return getEmbeddedCL100K()
	case O200KBase:
		return getEmbeddedO200K()
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownEncoding, name)
	}
}

func ForModel(model string) (*Tokenizer, error) {
	encoding, err := EncodingForModel(model)
	if err != nil {
		return nil, err
	}
	return GetEncoding(encoding)
}

func EncodingForModel(model string) (Encoding, error) {
	if enc, ok := openai.ModelToEncoding[model]; ok {
		return Encoding(enc), nil
	}
	for prefix, enc := range openai.ModelPrefixToEncoding {
		if strings.HasPrefix(model, prefix) {
			return Encoding(enc), nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrUnknownModel, model)
}

func (t *Tokenizer) Encode(text string) ([]int, error) {
	tokens, _, err := t.core.EncodeNative(text, nil)
	return tokens, err
}

func (t *Tokenizer) EncodeOrdinary(text string) ([]int, error) {
	return t.core.EncodeOrdinaryNative(text)
}

func (t *Tokenizer) Count(text string) (int, error) {
	tokens, err := t.EncodeOrdinary(text)
	if err != nil {
		return 0, err
	}
	return len(tokens), nil
}

func (t *Tokenizer) Decode(tokens []int) (string, error) {
	bytes, err := t.core.DecodeNative(tokens)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (t *Tokenizer) DecodeBytes(tokens []int) ([]byte, error) {
	return t.core.DecodeNative(tokens)
}
