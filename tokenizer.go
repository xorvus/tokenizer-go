package tokenizer

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xorvus/tokenizer-go/internal/bpe"
	"github.com/xorvus/tokenizer-go/internal/openai"
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
	case O200KHarmony:
		return getEmbeddedO200KHarmony()
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

func (t *Tokenizer) EncodeOrdinaryBatch(texts []string) ([][]int, error) {
	return t.core.EncodeOrdinaryBatchNative(texts)
}

func (t *Tokenizer) EncodeSingleToken(text string) (int, error) {
	if id, ok := t.core.Encoder[text]; ok {
		return id, nil
	}
	if id, ok := t.core.SpecialTokensEncoder[text]; ok {
		return id, nil
	}
	tokens, err := t.EncodeOrdinary(text)
	if err != nil {
		return 0, err
	}
	if len(tokens) != 1 {
		return 0, fmt.Errorf("%w: %q encodes into %d tokens", ErrInvalidSingleToken, text, len(tokens))
	}
	return tokens[0], nil
}

func (t *Tokenizer) Count(text string) (int, error) {
	tokens, err := t.EncodeOrdinary(text)
	if err != nil {
		return 0, err
	}
	return len(tokens), nil
}

func (t *Tokenizer) CountOrdinaryBatch(texts []string) ([]int, error) {
	return t.core.CountOrdinaryBatchNative(texts)
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

func (t *Tokenizer) DecodeSingleTokenBytes(token int) ([]byte, error) {
	if token >= 0 && token < len(t.core.DecoderSlice) && t.core.DecoderSlice[token] != "" {
		return []byte(t.core.DecoderSlice[token]), nil
	}
	if val, ok := t.core.SpecialTokensDecoder[token]; ok {
		return []byte(val), nil
	}
	return nil, fmt.Errorf("%w: %d", ErrUnknownTokenID, token)
}

func (t *Tokenizer) TokenByteValues() [][]byte {
	res := make([][]byte, len(t.core.DecoderSlice))
	for i, s := range t.core.DecoderSlice {
		res[i] = []byte(s)
	}
	return res
}

func (t *Tokenizer) DecodeWithOffsets(tokens []int) (string, []int, error) {
	var sb strings.Builder
	offsets := make([]int, 0, len(tokens))
	currentOffset := 0

	for _, tok := range tokens {
		b, err := t.DecodeSingleTokenBytes(tok)
		if err != nil {
			return "", nil, err
		}
		offsets = append(offsets, currentOffset)
		sb.Write(b)
		currentOffset += len(b)
	}

	return sb.String(), offsets, nil
}

func (t *Tokenizer) DecodeBatch(batch [][]int) ([]string, error) {
	res := make([]string, len(batch))
	for i, tokens := range batch {
		str, err := t.Decode(tokens)
		if err != nil {
			return nil, err
		}
		res[i] = str
	}
	return res, nil
}

func (t *Tokenizer) DecodeBytesBatch(batch [][]int) ([][]byte, error) {
	res := make([][]byte, len(batch))
	for i, tokens := range batch {
		b, err := t.DecodeBytes(tokens)
		if err != nil {
			return nil, err
		}
		res[i] = b
	}
	return res, nil
}
