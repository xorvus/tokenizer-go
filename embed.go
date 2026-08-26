package tokenizer

import (
	_ "embed"
	"strings"
	"sync"

	"github.com/xorvus/tokenizer-go/internal/openai"
)

//go:embed internal/openai/cl100k_base.tiktoken
var cl100kVocab string

//go:embed internal/openai/o200k_base.tiktoken
var o200kVocab string

var (
	loadCL100K = sync.OnceValues(func() (*Tokenizer, error) {
		return NewFromVocabulary(strings.NewReader(cl100kVocab), openai.PatternCL100K, openai.SpecialTokensCL100K())
	})
	loadO200K = sync.OnceValues(func() (*Tokenizer, error) {
		return NewFromVocabulary(strings.NewReader(o200kVocab), openai.PatternO200K, openai.SpecialTokensO200K())
	})
	loadO200KHarmony = sync.OnceValues(func() (*Tokenizer, error) {
		return NewFromVocabulary(strings.NewReader(o200kVocab), openai.PatternO200K, openai.SpecialTokensO200KHarmony())
	})
)

func getEmbeddedCL100K() (*Tokenizer, error) {
	return loadCL100K()
}

func getEmbeddedO200K() (*Tokenizer, error) {
	return loadO200K()
}

func getEmbeddedO200KHarmony() (*Tokenizer, error) {
	return loadO200KHarmony()
}
