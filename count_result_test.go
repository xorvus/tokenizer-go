package tokenizer_test

import (
	"testing"

	"github.com/xorvus/tokenizer-go"
)

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
