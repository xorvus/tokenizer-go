package tokenizer

type ModelSpec struct {
	CanonicalName   string
	Provider        Provider
	Aliases         []string
	Prefixes        []string
	TokenizerID     string
	FallbackProfile string
	ChatTemplateID  string
	Capabilities    Capabilities
	IsExact         bool
}
