package tokenizer

import (
	_ "embed"
	"strings"

	"github.com/pkoukk/tokenizer-go/internal/openai"
)

//go:embed testdata/cl100k_base.tiktoken
var cl100kVocab string

//go:embed testdata/o200k_base.tiktoken
var o200kVocab string

func getEmbeddedCL100K() (*Tokenizer, error) {
	return NewFromVocabulary(strings.NewReader(cl100kVocab), openai.PatternCL100K, openai.SpecialTokensCL100K())
}

func getEmbeddedO200K() (*Tokenizer, error) {
	return NewFromVocabulary(strings.NewReader(o200kVocab), openai.PatternO200K, openai.SpecialTokensO200K())
}
