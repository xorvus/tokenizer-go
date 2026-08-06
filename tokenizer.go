package tokenizer

import (
	"fmt"
	"io"
	"os"

	"github.com/pkoukk/tiktoken-go/tokenizer-go/internal/bpe"
	"github.com/pkoukk/tiktoken-go/tokenizer-go/internal/openai"
)

type Tokenizer struct {
	core *bpe.CoreBPE
}

func NewFromVocabulary(r io.Reader, pattern string, special map[string]int) (*Tokenizer, error) {
	vocab, err := openai.ParseVocabulary(r)
	if err != nil {
		return nil, err
	}
	core, err := bpe.NewCoreBPE(vocab.Encoder, vocab.Decoder, special, pattern)
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

func GetEncoding(name Encoding) (*Tokenizer, error) {
	if name != CL100KBase {
		return nil, fmt.Errorf("%w: %s", ErrUnknownEncoding, name)
	}
	return nil, fmt.Errorf("use NewFromFile or NewFromVocabulary with cl100k BPE file")
}
