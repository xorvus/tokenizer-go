package tokenizer

type Resolution struct {
	RequestedModel string   `json:"requested_model"`
	CanonicalModel string   `json:"canonical_model"`
	Provider       Provider `json:"provider"`
	TokenizerID    string   `json:"tokenizer_id"`
	Accuracy       Accuracy `json:"accuracy"`
	UsedFallback   bool     `json:"used_fallback"`
	Reason         string   `json:"reason,omitempty"`
}
