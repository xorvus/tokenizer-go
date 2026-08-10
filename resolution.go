package tokenizer

type Resolution struct {
	RequestedModel string   `json:"requested_model"`
	CanonicalModel string   `json:"canonical_model"`
	Provider       Provider `json:"provider"`
	TokenizerID    string   `json:"tokenizer_id"`
	Accuracy       Accuracy `json:"accuracy"`
	UsedFallback   bool     `json:"used_fallback"`
	// ProfileID names the calibration profile used to scale the nearest
	// embedded tokenizer's count. Only set when UsedFallback is true.
	ProfileID string `json:"profile_id,omitempty"`
	Reason    string `json:"reason,omitempty"`
}
