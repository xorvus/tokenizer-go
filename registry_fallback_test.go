package tokenizer_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/xorvus/tokenizer-go"
)

func TestClaudeResolvesToNearestTokenizerFallback(t *testing.T) {
	res, err := tokenizer.ResolveModel("claude-3-5-sonnet-20241022")
	if err != nil {
		t.Fatalf("ResolveModel error: %v", err)
	}
	if !res.UsedFallback {
		t.Error("expected UsedFallback == true for a Claude model")
	}
	if res.Provider != tokenizer.ProviderAnthropic {
		t.Errorf("Provider = %v, want %v", res.Provider, tokenizer.ProviderAnthropic)
	}
	if res.TokenizerID != "o200k_base" {
		t.Errorf("TokenizerID = %q, want %q (nearest embedded tokenizer)", res.TokenizerID, "o200k_base")
	}
	// The shipped anthropic-claude-v1 profile has sample_count 0, so this
	// must be labeled Heuristic, not Calibrated, even though it goes
	// through the calibration code path.
	if res.Accuracy != tokenizer.AccuracyEstimatedHeuristic {
		t.Errorf("Accuracy = %v, want %v (profile is an uncalibrated placeholder)", res.Accuracy, tokenizer.AccuracyEstimatedHeuristic)
	}
	if res.Reason == "" {
		t.Error("expected a non-empty Reason explaining the fallback")
	}
}

func TestGeminiResolvesToNearestTokenizerFallback(t *testing.T) {
	res, err := tokenizer.ResolveModel("gemini-1.5-pro")
	if err != nil {
		t.Fatalf("ResolveModel error: %v", err)
	}
	if !res.UsedFallback || res.Provider != tokenizer.ProviderGemini || res.TokenizerID != "o200k_base" {
		t.Fatalf("unexpected resolution: %+v", res)
	}
	// The shipped gemini-v1 profile is calibrated (sample_count > 0), so the
	// estimate must be labeled Calibrated, not Heuristic.
	if res.Accuracy != tokenizer.AccuracyEstimatedCalibrated {
		t.Errorf("Accuracy = %v, want %v (profile is calibrated)", res.Accuracy, tokenizer.AccuracyEstimatedCalibrated)
	}
}

func TestForModelRejectsFallbackOnlyModels(t *testing.T) {
	_, err := tokenizer.ForModel("claude-3-5-sonnet-20241022")
	if !errors.Is(err, tokenizer.ErrExactTokenizerUnavailable) {
		t.Errorf("ForModel(claude-*) error = %v, want ErrExactTokenizerUnavailable", err)
	}

	_, err = tokenizer.EncodingForModel("gemini-1.5-pro")
	if !errors.Is(err, tokenizer.ErrExactTokenizerUnavailable) {
		t.Errorf("EncodingForModel(gemini-*) error = %v, want ErrExactTokenizerUnavailable", err)
	}
}

func TestCountForModelDefaultRejectsUncalibratedFallback(t *testing.T) {
	_, err := tokenizer.CountForModel("claude-3-5-sonnet-20241022", "hello world")
	if !errors.Is(err, tokenizer.ErrExactTokenizerUnavailable) {
		t.Errorf("expected default config to reject an uncalibrated heuristic estimate, got %v", err)
	}
}

func TestCountForModelGeminiCalibratedByDefault(t *testing.T) {
	// A calibrated fallback (gemini) must be allowed with no opt-in and must
	// report the calibration profile.
	res, err := tokenizer.CountForModel("gemini-1.5-pro", "hello world")
	if err != nil {
		t.Fatalf("CountForModel(gemini-*) with calibrated fallback error: %v", err)
	}
	if res.Accuracy != tokenizer.AccuracyEstimatedCalibrated {
		t.Errorf("Accuracy = %v, want %v", res.Accuracy, tokenizer.AccuracyEstimatedCalibrated)
	}
	if res.Calibration == nil || res.Calibration.SampleCount <= 0 {
		t.Fatalf("expected calibration metadata with samples, got %+v", res.Calibration)
	}
	if res.Tokens <= 0 {
		t.Errorf("Tokens = %d, want > 0", res.Tokens)
	}
}

func TestCountForModelWithHeuristicFallbackSucceeds(t *testing.T) {
	res, err := tokenizer.CountForModel("claude-3-5-sonnet-20241022", "hello world", tokenizer.WithHeuristicFallback())
	if err != nil {
		t.Fatalf("CountForModel with WithHeuristicFallback error: %v", err)
	}
	if res.Tokens <= 0 {
		t.Errorf("Tokens = %d, want > 0", res.Tokens)
	}
	if res.Accuracy != tokenizer.AccuracyEstimatedHeuristic {
		t.Errorf("Accuracy = %v, want %v", res.Accuracy, tokenizer.AccuracyEstimatedHeuristic)
	}
	if res.Calibration == nil {
		t.Fatal("expected Calibration to be populated for a fallback result")
	}
	if res.Calibration.SampleCount != 0 {
		t.Errorf("Calibration.SampleCount = %d, want 0 for the shipped placeholder profile", res.Calibration.SampleCount)
	}

	// Cross-check against the raw nearest-tokenizer count: with an
	// uncalibrated identity profile (ratio 1.0), CountForModel's result
	// must equal counting the same text directly with o200k_base.
	base, err := tokenizer.GetEncoding(tokenizer.O200KBase)
	if err != nil {
		t.Fatalf("GetEncoding error: %v", err)
	}
	baseCount, err := base.Count("hello world")
	if err != nil {
		t.Fatalf("base.Count error: %v", err)
	}
	if res.Tokens != baseCount {
		t.Errorf("fallback Tokens = %d, want %d (unscaled nearest-tokenizer count for an identity profile)", res.Tokens, baseCount)
	}
}

func TestCountForModelExactOnlyRejectsFallback(t *testing.T) {
	_, err := tokenizer.CountForModel("claude-3-5-sonnet-20241022", "hello world", tokenizer.WithExactOnly())
	if !errors.Is(err, tokenizer.ErrExactTokenizerUnavailable) {
		t.Errorf("WithExactOnly() should reject a fallback model, got %v", err)
	}

	// WithExactOnly must not affect models that already resolve exactly.
	res, err := tokenizer.CountForModel("gpt-4o", "hello world", tokenizer.WithExactOnly())
	if err != nil {
		t.Fatalf("WithExactOnly() unexpectedly rejected an exact model: %v", err)
	}
	if res.Accuracy != tokenizer.AccuracyExactLocal {
		t.Errorf("Accuracy = %v, want ExactLocal", res.Accuracy)
	}
}

func TestUnknownModelStillErrors(t *testing.T) {
	_, err := tokenizer.ResolveModel("totally-unregistered-model-xyz")
	if !errors.Is(err, tokenizer.ErrUnknownModel) {
		t.Errorf("expected ErrUnknownModel, got %v", err)
	}
}

func TestUpperBoundInflatesUncalibratedEstimates(t *testing.T) {
	res, err := tokenizer.CountForModel("claude-3-5-sonnet-20241022", "hello world", tokenizer.WithHeuristicFallback())
	if err != nil {
		t.Fatalf("CountForModel error: %v", err)
	}
	if ub := res.UpperBound(); ub <= res.Tokens {
		t.Errorf("UpperBound() = %d, want > Tokens (%d) for an uncalibrated estimate", ub, res.Tokens)
	}

	exact, err := tokenizer.CountForModel("gpt-4o", "hello world")
	if err != nil {
		t.Fatalf("CountForModel error: %v", err)
	}
	if ub := exact.UpperBound(); ub != exact.Tokens {
		t.Errorf("UpperBound() = %d, want exactly Tokens (%d) for an exact result", ub, exact.Tokens)
	}
}

func TestPublicRegistryExtensionAPI(t *testing.T) {
	if err := tokenizer.RegisterModel(tokenizer.ModelSpec{
		CanonicalName: "my-custom-exact-model",
		Provider:      tokenizer.ProviderOpenAI,
		TokenizerID:   "o200k_base",
	}); err != nil {
		t.Fatalf("RegisterModel error: %v", err)
	}
	spec, ok := tokenizer.LookupModel("my-custom-exact-model")
	if !ok || spec.TokenizerID != "o200k_base" || !spec.IsExact {
		t.Fatalf("LookupModel after RegisterModel: %+v, ok=%v", spec, ok)
	}
	res, err := tokenizer.ResolveModel("my-custom-exact-model")
	if err != nil || res.Accuracy != tokenizer.AccuracyExactLocal {
		t.Fatalf("ResolveModel after RegisterModel: %+v, err=%v", res, err)
	}

	if err := tokenizer.RegisterModelAlias("my-alias-xyz", "gpt-4o"); err != nil {
		t.Fatalf("RegisterModelAlias error: %v", err)
	}
	res, err = tokenizer.ResolveModel("my-alias-xyz")
	if err != nil || res.CanonicalModel != "gpt-4o" {
		t.Fatalf("ResolveModel after RegisterModelAlias: %+v, err=%v", res, err)
	}

	if err := tokenizer.RegisterModelPrefix("my-custom-prefix-", tokenizer.ModelSpec{
		Provider:    tokenizer.ProviderOpenAI,
		TokenizerID: "cl100k_base",
	}); err != nil {
		t.Fatalf("RegisterModelPrefix error: %v", err)
	}
	res, err = tokenizer.ResolveModel("my-custom-prefix-2026")
	if err != nil || res.TokenizerID != "cl100k_base" || res.UsedFallback {
		t.Fatalf("ResolveModel after RegisterModelPrefix: %+v, err=%v", res, err)
	}

	if err := tokenizer.RegisterCalibrationProfile(tokenizer.CalibrationProfile{
		ProfileID:     "test-mistral-v1",
		BaseTokenizer: "cl100k_base",
		Ratios: map[tokenizer.Bucket]tokenizer.CalibrationRatio{
			tokenizer.BucketOther: {Mean: 1.2},
		},
	}); err != nil {
		t.Fatalf("RegisterCalibrationProfile error: %v", err)
	}
	if err := tokenizer.RegisterModelFallbackPrefix("test-mistral-", tokenizer.ProviderMistral, "test-mistral-v1"); err != nil {
		t.Fatalf("RegisterModelFallbackPrefix error: %v", err)
	}
	res, err = tokenizer.ResolveModel("test-mistral-large-latest")
	if err != nil {
		t.Fatalf("ResolveModel error: %v", err)
	}
	if !res.UsedFallback || res.TokenizerID != "cl100k_base" || res.Provider != tokenizer.ProviderMistral {
		t.Fatalf("unexpected resolution for registered fallback prefix: %+v", res)
	}
}

func TestRegisterFallbackPrefixRejectsUnknownProfile(t *testing.T) {
	err := tokenizer.RegisterModelFallbackPrefix("some-other-prefix-", tokenizer.ProviderGrok, "no-such-profile-id")
	if err == nil {
		t.Error("expected an error when registering a fallback prefix against a profile that was never registered")
	}
}

func TestPrefixResolutionIsDeterministicUnderOverlap(t *testing.T) {
	// Two equal-length, mutually non-prefixing custom prefixes exercise
	// the lexical tiebreak: without it, relative order between
	// equal-length entries would depend on map iteration order and could
	// vary between runs.
	if err := tokenizer.RegisterModelPrefix("zz-tiebreak-a", tokenizer.ModelSpec{
		Provider: tokenizer.ProviderOpenAI, TokenizerID: "cl100k_base",
	}); err != nil {
		t.Fatalf("RegisterModelPrefix error: %v", err)
	}
	if err := tokenizer.RegisterModelPrefix("zz-tiebreak-b", tokenizer.ModelSpec{
		Provider: tokenizer.ProviderOpenAI, TokenizerID: "o200k_base",
	}); err != nil {
		t.Fatalf("RegisterModelPrefix error: %v", err)
	}

	// A model matching only the "a" prefix must always resolve to it,
	// repeatedly, across many calls.
	for i := 0; i < 50; i++ {
		res, err := tokenizer.ResolveModel("zz-tiebreak-a-2026")
		if err != nil || res.TokenizerID != "cl100k_base" {
			t.Fatalf("run %d: unexpected resolution: %+v, err=%v", i, res, err)
		}
	}
}

// TestConcurrentRegisterAndResolve exercises the global registry's mutex
// under concurrent readers (Resolve, via ResolveModel/CountForModel) and
// writers (RegisterModel/RegisterModelPrefix) at the same time. It is
// meaningful primarily under `go test -race`.
func TestConcurrentRegisterAndResolve(t *testing.T) {
	var readers, writers sync.WaitGroup
	stop := make(chan struct{})

	// Readers: continuously resolve a mix of stable and newly-registered
	// models while writers are mutating the registry, until told to stop.
	for i := 0; i < 8; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				_, _ = tokenizer.ResolveModel("gpt-4o")
				_, _ = tokenizer.ResolveModel("claude-3-5-sonnet-20241022")
				_, _ = tokenizer.CountForModel("gpt-4o", "hello world")
				_, _ = tokenizer.LookupModel("gpt-4o")
			}
		}()
	}

	// Writers: register distinct model names concurrently, then finish.
	for i := 0; i < 8; i++ {
		writers.Add(1)
		go func(id int) {
			defer writers.Done()
			for j := 0; j < 20; j++ {
				name := fmt.Sprintf("zz-concurrent-model-%d-%d", id, j)
				if err := tokenizer.RegisterModel(tokenizer.ModelSpec{
					CanonicalName: name,
					Provider:      tokenizer.ProviderOpenAI,
					TokenizerID:   "o200k_base",
				}); err != nil {
					t.Errorf("RegisterModel(%s) error: %v", name, err)
				}
			}
		}(i)
	}

	writers.Wait()
	close(stop)
	readers.Wait()
}
