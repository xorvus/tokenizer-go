package tokenizer

import "math"

type Accuracy uint8

const (
	AccuracyExactLocal Accuracy = iota
	AccuracyEstimatedCalibrated
	AccuracyEstimatedHeuristic
)

func (a Accuracy) String() string {
	switch a {
	case AccuracyExactLocal:
		return "ExactLocal"
	case AccuracyEstimatedCalibrated:
		return "EstimatedCalibrated"
	case AccuracyEstimatedHeuristic:
		return "EstimatedHeuristic"
	default:
		return "Unknown"
	}
}

type Calibration struct {
	ProfileID       string  `json:"profile_id"`
	SampleCount     int     `json:"sample_count"`
	MeanAbsErrorPct float64 `json:"mean_abs_error_pct"`
	P95AbsErrorPct  float64 `json:"p95_abs_error_pct"`
	MaxAbsErrorPct  float64 `json:"max_abs_error_pct"`
	CorpusSHA256    string  `json:"corpus_sha256"`
}

type CountResult struct {
	Tokens         int          `json:"tokens"`
	RequestedModel string       `json:"requested_model"`
	CanonicalModel string       `json:"canonical_model"`
	TokenizerID    string       `json:"tokenizer_id"`
	Provider       Provider     `json:"provider"`
	Accuracy       Accuracy     `json:"accuracy"`
	UsedFallback   bool         `json:"used_fallback"`
	FallbackReason string       `json:"fallback_reason,omitempty"`
	Calibration    *Calibration `json:"calibration,omitempty"`
}

func (r CountResult) WithSafetyMargin(percent float64) int {
	if percent <= 0 {
		return r.Tokens
	}
	margin := math.Ceil(float64(r.Tokens) * (percent / 100.0))
	return r.Tokens + int(margin)
}
