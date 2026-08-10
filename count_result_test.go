package tokenizer_test

import (
	"encoding/json"
	"testing"

	"github.com/xorvus/tokenizer-go"
)

func TestAccuracyZeroValueIsUnknownNotExact(t *testing.T) {
	var zero tokenizer.Accuracy
	if zero == tokenizer.AccuracyExactLocal {
		t.Fatal("the zero value of Accuracy must not equal AccuracyExactLocal — a CountResult{} returned alongside an error must not read as exact")
	}
	if zero != tokenizer.AccuracyUnknown {
		t.Fatalf("zero value of Accuracy = %v, want AccuracyUnknown", zero)
	}
	if got := zero.String(); got != "Unknown" {
		t.Errorf("zero value Accuracy.String() = %q, want %q", got, "Unknown")
	}
}

func TestCountResultZeroValueOnErrorIsHonest(t *testing.T) {
	// A CountResult returned alongside a non-nil error (e.g. from an
	// unknown model) must not claim AccuracyExactLocal by virtue of being
	// a zero-valued struct.
	_, err := tokenizer.CountForModel("no-such-model-xyz", "hello")
	if err == nil {
		t.Fatal("expected an error for an unknown model")
	}
	var zeroResult tokenizer.CountResult
	if zeroResult.Accuracy == tokenizer.AccuracyExactLocal {
		t.Fatal("zero-valued CountResult must not report AccuracyExactLocal")
	}
}

func TestAccuracyJSONRoundTrip(t *testing.T) {
	for _, acc := range []tokenizer.Accuracy{
		tokenizer.AccuracyUnknown,
		tokenizer.AccuracyExactLocal,
		tokenizer.AccuracyEstimatedCalibrated,
		tokenizer.AccuracyEstimatedHeuristic,
	} {
		b, err := json.Marshal(acc)
		if err != nil {
			t.Fatalf("Marshal(%v) error: %v", acc, err)
		}
		want := `"` + acc.String() + `"`
		if string(b) != want {
			t.Errorf("Marshal(%v) = %s, want %s", acc, b, want)
		}
		var got tokenizer.Accuracy
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("Unmarshal(%s) error: %v", b, err)
		}
		if got != acc {
			t.Errorf("round trip: got %v, want %v", got, acc)
		}
	}
}

func TestAccuracyUnmarshalRejectsUnknownString(t *testing.T) {
	var a tokenizer.Accuracy
	if err := json.Unmarshal([]byte(`"NotARealValue"`), &a); err == nil {
		t.Error("expected an error unmarshaling an unrecognized Accuracy string")
	}
}

func TestUpperBoundExactVsEstimated(t *testing.T) {
	exact := tokenizer.CountResult{Tokens: 100, Accuracy: tokenizer.AccuracyExactLocal}
	if got := exact.UpperBound(); got != 100 {
		t.Errorf("UpperBound() for exact result = %d, want 100 (unchanged)", got)
	}

	calibratedNoData := tokenizer.CountResult{Tokens: 100, Accuracy: tokenizer.AccuracyEstimatedCalibrated}
	if got := calibratedNoData.UpperBound(); got <= 100 {
		t.Errorf("UpperBound() with no Calibration attached = %d, want > 100 (must not silently trust an estimate with no error data)", got)
	}

	calibrated := tokenizer.CountResult{
		Tokens:      100,
		Accuracy:    tokenizer.AccuracyEstimatedCalibrated,
		Calibration: &tokenizer.Calibration{SampleCount: 500, P95AbsErrorPct: 8},
	}
	if got := calibrated.UpperBound(); got != 108 {
		t.Errorf("UpperBound() with 8%% P95 error = %d, want 108", got)
	}
}

func TestCountResultSafetyMargin(t *testing.T) {
	res := tokenizer.CountResult{
		Tokens:   100,
		Accuracy: tokenizer.AccuracyEstimatedCalibrated,
	}
	safe := res.WithSafetyMargin(10)
	if safe != 110 {
		t.Errorf("got %d, want 110", safe)
	}

	if res.WithSafetyMargin(0) != 100 {
		t.Errorf("got %d, want 100 for 0%% margin", res.WithSafetyMargin(0))
	}

	if res.WithSafetyMargin(-5) != 100 {
		t.Errorf("got %d, want 100 for negative margin", res.WithSafetyMargin(-5))
	}

	resFraction := tokenizer.CountResult{Tokens: 15}
	if safeFrac := resFraction.WithSafetyMargin(10); safeFrac != 17 {
		t.Errorf("got %d, want 17 for 15 with 10%% margin (ceil of 1.5)", safeFrac)
	}
}

func TestAccuracyString(t *testing.T) {
	tests := []struct {
		acc  tokenizer.Accuracy
		want string
	}{
		{tokenizer.AccuracyUnknown, "Unknown"},
		{tokenizer.AccuracyExactLocal, "ExactLocal"},
		{tokenizer.AccuracyEstimatedCalibrated, "EstimatedCalibrated"},
		{tokenizer.AccuracyEstimatedHeuristic, "EstimatedHeuristic"},
		{tokenizer.Accuracy(99), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.acc.String(); got != tt.want {
			t.Errorf("Accuracy(%d).String() = %q, want %q", tt.acc, got, tt.want)
		}
	}
}

func TestResolutionStruct(t *testing.T) {
	r := tokenizer.Resolution{
		RequestedModel: "gpt-4o",
		CanonicalModel: "gpt-4o-2024-05-13",
		Provider:       tokenizer.ProviderOpenAI,
		TokenizerID:    "o200k_base",
		Accuracy:       tokenizer.AccuracyExactLocal,
		UsedFallback:   false,
	}
	if r.RequestedModel != "gpt-4o" || r.Provider != tokenizer.ProviderOpenAI {
		t.Errorf("unexpected resolution struct values: %+v", r)
	}
}
