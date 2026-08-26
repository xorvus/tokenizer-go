package openai

import (
	"fmt"
	"sort"
	"strings"
)

var ModelPrefixToEncoding = map[string]string{
	"o1-":              "o200k_base",
	"o3-":              "o200k_base",
	"o4-mini-":         "o200k_base",
	"gpt-5-":           "o200k_base",
	"gpt-4.5-":         "o200k_base",
	"gpt-4.1-":         "o200k_base",
	"chatgpt-4o-":      "o200k_base",
	"gpt-4o-":          "o200k_base",
	"gpt-4-":           "cl100k_base",
	"gpt-3.5-turbo-":   "cl100k_base",
	"gpt-35-turbo-":    "cl100k_base",
	"gpt-oss-":         "o200k_harmony",
	"ft:gpt-4o":        "o200k_base",
	"ft:gpt-4":         "cl100k_base",
	"ft:gpt-3.5-turbo": "cl100k_base",
	"ft:davinci-002":   "cl100k_base",
	"ft:babbage-002":   "cl100k_base",
}

var ModelToEncoding = map[string]string{
	"o1":                           "o200k_base",
	"o3":                           "o200k_base",
	"o4-mini":                      "o200k_base",
	"gpt-5":                        "o200k_base",
	"gpt-4.1":                      "o200k_base",
	"gpt-4o":                       "o200k_base",
	"gpt-4":                        "cl100k_base",
	"gpt-3.5-turbo":                "cl100k_base",
	"gpt-3.5":                      "cl100k_base",
	"gpt-35-turbo":                 "cl100k_base",
	"davinci-002":                  "cl100k_base",
	"babbage-002":                  "cl100k_base",
	"text-embedding-ada-002":       "cl100k_base",
	"text-embedding-3-small":       "cl100k_base",
	"text-embedding-3-large":       "cl100k_base",
	"text-davinci-003":             "p50k_base",
	"text-davinci-002":             "p50k_base",
	"text-davinci-001":             "r50k_base",
	"text-curie-001":               "r50k_base",
	"text-babbage-001":             "r50k_base",
	"text-ada-001":                 "r50k_base",
	"davinci":                      "r50k_base",
	"curie":                        "r50k_base",
	"babbage":                      "r50k_base",
	"ada":                          "r50k_base",
	"code-davinci-002":             "p50k_base",
	"code-davinci-001":             "p50k_base",
	"code-cushman-002":             "p50k_base",
	"code-cushman-001":             "p50k_base",
	"davinci-codex":                "p50k_base",
	"cushman-codex":                "p50k_base",
	"text-davinci-edit-001":        "p50k_edit",
	"code-davinci-edit-001":        "p50k_edit",
	"text-similarity-davinci-001": "r50k_base",
	"text-similarity-curie-001":   "r50k_base",
	"text-similarity-babbage-001": "r50k_base",
	"text-similarity-ada-001":     "r50k_base",
	"text-search-davinci-doc-001": "r50k_base",
	"text-search-curie-doc-001":   "r50k_base",
	"text-search-babbage-doc-001": "r50k_base",
	"text-search-ada-doc-001":     "r50k_base",
	"code-search-babbage-code-001": "r50k_base",
	"code-search-ada-code-001":     "r50k_base",
	"gpt2":                         "gpt2",
	"gpt-2":                        "gpt2",
}

type prefixMapping struct {
	prefix   string
	encoding string
}

var sortedModelPrefixes = buildSortedPrefixes()

func buildSortedPrefixes() []prefixMapping {
	res := make([]prefixMapping, 0, len(ModelPrefixToEncoding))
	for p, enc := range ModelPrefixToEncoding {
		res = append(res, prefixMapping{prefix: p, encoding: enc})
	}
	sort.SliceStable(res, func(i, j int) bool {
		if len(res[i].prefix) != len(res[j].prefix) {
			return len(res[i].prefix) > len(res[j].prefix)
		}
		return res[i].prefix < res[j].prefix
	})
	return res
}

func EncodingNameForModel(modelName string) (string, error) {
	if enc, ok := ModelToEncoding[modelName]; ok {
		return enc, nil
	}
	for _, entry := range sortedModelPrefixes {
		if strings.HasPrefix(modelName, entry.prefix) {
			return entry.encoding, nil
		}
	}
	return "", fmt.Errorf("could not automatically map %s to a tokenizer", modelName)
}
