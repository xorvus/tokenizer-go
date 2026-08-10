package tokenizer

import (
	"encoding/json"
	"fmt"
	"math"
)

type Accuracy uint8

// AccuracyUnknown is the zero value; must stay at position 0 so a
// zero-valued Accuracy is never mistaken for AccuracyExactLocal.
const (
	AccuracyUnknown Accuracy = iota
	AccuracyExactLocal
	AccuracyEstimatedCalibrated
	AccuracyEstimatedHeuristic
)

func (a Accuracy) String() string {
	switch a {
	case AccuracyUnknown:
		return "Unknown"
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

// MarshalJSON encodes Accuracy as its String() form so wire output matches
// the documented enum names instead of a bare integer.
func (a Accuracy) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

// UnmarshalJSON accepts the String() form produced by MarshalJSON.
func (a *Accuracy) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch s {
	case "Unknown":
		*a = AccuracyUnknown
	case "ExactLocal":
		*a = AccuracyExactLocal
	case "EstimatedCalibrated":
		*a = AccuracyEstimatedCalibrated
	case "EstimatedHeuristic":
		*a = AccuracyEstimatedHeuristic
	default:
		return fmt.Errorf("tokenizer: unknown Accuracy value %q", s)
	}
	return nil
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

// UpperBound returns a conservative token count for budget enforcement:
// Tokens unchanged for exact results; inflated by the calibration P95
// margin for estimates; a fixed 25% margin when unmeasured.
func (r CountResult) UpperBound() int {
	if r.Accuracy == AccuracyExactLocal {
		return r.Tokens
	}
	if r.Calibration != nil && r.Calibration.SampleCount > 0 {
		return r.WithSafetyMargin(r.Calibration.P95AbsErrorPct)
	}
	const uncalibratedMarginPct = 25.0
	return r.WithSafetyMargin(uncalibratedMarginPct)
}
