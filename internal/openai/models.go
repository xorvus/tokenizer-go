package openai

var ModelToEncoding = map[string]string{
	"gpt-4o":        "o200k_base",
	"gpt-4":         "cl100k_base",
	"gpt-3.5-turbo": "cl100k_base",
}

func EncodingForModel(model string) (string, bool) {
	enc, ok := ModelToEncoding[model]
	return enc, ok
}
