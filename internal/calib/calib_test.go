package calib

import "testing"

func TestClassifyBuckets(t *testing.T) {
	tests := []struct {
		name string
		text string
		want Bucket
	}{
		{"empty", "", BucketLatin},
		{"english_prose", "The quick brown fox jumps over the lazy dog near the riverbank.", BucketLatin},
		{"chinese", "你好世界，这是一个中文测试句子，用来验证分类器的行为是否正确。", BucketZh},
		{"japanese", "こんにちは世界、これは日本語のテスト文です。分類器が正しく動作するか確認します。", BucketJa},
		{"korean", "안녕하세요 세계, 이것은 한국어 테스트 문장입니다. 분류기가 올바르게 작동하는지 확인합니다.", BucketKo},
		{"code", `func main() { fmt.Println("hi"); if x == 1 { y := []int{1,2,3}; } }`, BucketCode},
		{"json", `{"a": 1, "b": [1,2,3], "c": {"d": true, "e": null}}`, BucketCode},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Classify(tt.text); got != tt.want {
				t.Errorf("Classify(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestClassifyLargeInputStaysBounded(t *testing.T) {
	// Regression guard: Classify must not scale with input size. This
	// doesn't measure time (too flaky in CI), just confirms it terminates
	// and returns a valid bucket on an input far larger than the sample cap.
	big := make([]byte, 5*1024*1024)
	for i := range big {
		big[i] = 'a'
	}
	got := Classify(string(big))
	if got != BucketLatin {
		t.Errorf("Classify(5MB of 'a') = %v, want %v", got, BucketLatin)
	}
}

func TestProfileRatioForFallsBackToOther(t *testing.T) {
	p := Profile{
		ProfileID:     "test-profile",
		BaseTokenizer: "o200k_base",
		Ratios: map[Bucket]Ratio{
			BucketLatin: {Mean: 1.1},
			BucketOther: {Mean: 1.5},
		},
	}
	if r := p.RatioFor(BucketLatin); r.Mean != 1.1 {
		t.Errorf("RatioFor(Latin).Mean = %v, want 1.1", r.Mean)
	}
	if r := p.RatioFor(BucketJa); r.Mean != 1.5 {
		t.Errorf("RatioFor(JA) with no explicit entry should fall back to Other: got %v, want 1.5", r.Mean)
	}
}

func TestProfileIsCalibrated(t *testing.T) {
	uncalibrated := Profile{SampleCount: 0}
	if uncalibrated.IsCalibrated() {
		t.Error("Profile with SampleCount 0 must report IsCalibrated() == false")
	}
	calibrated := Profile{SampleCount: 500}
	if !calibrated.IsCalibrated() {
		t.Error("Profile with SampleCount > 0 must report IsCalibrated() == true")
	}
}

func TestRegisterValidation(t *testing.T) {
	if err := Register(Profile{}); err == nil {
		t.Error("Register with empty ProfileID should error")
	}
	if err := Register(Profile{ProfileID: "x"}); err == nil {
		t.Error("Register with empty BaseTokenizer should error")
	}
	if err := Register(Profile{ProfileID: "x", BaseTokenizer: "o200k_base"}); err == nil {
		t.Error("Register without a BucketOther ratio should error")
	}
	err := Register(Profile{
		ProfileID:     "test-register-ok",
		BaseTokenizer: "o200k_base",
		Ratios:        map[Bucket]Ratio{BucketOther: {Mean: 1.0}},
	})
	if err != nil {
		t.Fatalf("valid Register call failed: %v", err)
	}
	got, ok := Lookup("test-register-ok")
	if !ok || got.BaseTokenizer != "o200k_base" {
		t.Fatalf("Lookup after Register failed: %+v, ok=%v", got, ok)
	}
}

func TestEmbeddedProfilesArePresentAndHonest(t *testing.T) {
	for _, id := range []string{"anthropic-claude-v1", "gemini-v1"} {
		p, ok := Lookup(id)
		if !ok {
			t.Fatalf("embedded profile %q not found", id)
		}
		if p.BaseTokenizer == "" {
			t.Errorf("profile %q: BaseTokenizer must not be empty", id)
		}
		if _, ok := p.Ratios[BucketOther]; !ok {
			t.Errorf("profile %q: missing required BucketOther ratio", id)
		}
		// The shipped seed profiles are documented as uncalibrated
		// identity placeholders. If this ever fails, someone has run a
		// real calibration and this test's expectation should be updated
		// (and celebrated) alongside it — it exists to catch someone
		// hand-editing the JSON with fabricated numbers instead.
		if p.SampleCount != 0 {
			continue
		}
		for bucket, r := range p.Ratios {
			if r.Mean != 1.0 {
				t.Errorf("profile %q bucket %q: SampleCount is 0 (uncalibrated) but Mean ratio is %v, not 1.0 — an uncalibrated profile must not claim a non-identity correction", id, bucket, r.Mean)
			}
		}
	}
}
