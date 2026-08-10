package tokenizer

// ModelSpec describes how a model name resolves to a tokenizer: exact
// (TokenizerID set) or a nearest-tokenizer estimate (FallbackProfile set;
// TokenizerID filled from the profile at resolution time).
type ModelSpec struct {
	CanonicalName   string
	Provider        Provider
	TokenizerID     string
	FallbackProfile string
	Capabilities    Capabilities
	IsExact         bool
}
