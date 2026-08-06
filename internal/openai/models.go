package openai

import "strings"

var ModelToEncoding = map[string]string{
	"o1-preview":             "o200k_base",
	"o1-mini":                "o200k_base",
	"gpt-4o":                 "o200k_base",
	"gpt-4o-mini":            "o200k_base",
	"gpt-4":                  "cl100k_base",
	"gpt-4-turbo":            "cl100k_base",
	"gpt-3.5-turbo":          "cl100k_base",
	"text-embedding-3-small": "cl100k_base",
	"text-embedding-3-large": "cl100k_base",
	"text-embedding-ada-002": "cl100k_base",
}

var PrefixToEncoding = map[string]string{
	"o1-":            "o200k_base",
	"gpt-4o-":        "o200k_base",
	"gpt-4-":         "cl100k_base",
	"gpt-3.5-turbo-": "cl100k_base",
}

func EncodingForModel(model string) (string, bool) {
	if enc, ok := ModelToEncoding[model]; ok {
		return enc, true
	}
	for prefix, enc := range PrefixToEncoding {
		if strings.HasPrefix(model, prefix) {
			return enc, true
		}
	}
	return "", false
}
