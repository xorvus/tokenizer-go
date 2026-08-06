package tokenizer

import (
	_ "embed"
	"strings"

	"github.com/pkoukk/tiktoken-go/tokenizer-go/internal/openai"
)

//go:embed testdata/cl100k_base.tiktoken
var cl100kVocab string

func GetEmbeddedCL100K() (*Tokenizer, error) {
	return NewFromVocabulary(strings.NewReader(cl100kVocab), openai.PatternCL100K, openai.SpecialTokensCL100K())
}
