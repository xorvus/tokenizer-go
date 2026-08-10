package tokenizer

import "github.com/xorvus/tokenizer-go/internal/calib"

// Bucket is a coarse script/content classification used to select a
// per-bucket correction ratio within a CalibrationProfile.
type Bucket = calib.Bucket

const (
	BucketLatin Bucket = calib.BucketLatin
	BucketZh    Bucket = calib.BucketZh
	BucketJa    Bucket = calib.BucketJa
	BucketKo    Bucket = calib.BucketKo
	BucketCode  Bucket = calib.BucketCode
	BucketOther Bucket = calib.BucketOther
)

// ClassifyBucket returns the Bucket CountForModel would use for text.
func ClassifyBucket(text string) Bucket {
	return calib.Classify(text)
}

// CalibrationRatio is the per-bucket target/base count ratio and error stats.
type CalibrationRatio = calib.Ratio

// CalibrationProfile records how to estimate one target model's count from
// a nearest embedded tokenizer. SampleCount == 0 is an uncalibrated
// placeholder (resolves to AccuracyEstimatedHeuristic).
type CalibrationProfile = calib.Profile

// RegisterCalibrationProfile registers a profile at runtime, e.g. one
// produced by scripts/calibrate.py and loaded from disk. Pair it with
// RegisterModelFallbackPrefix to route model names through it.
func RegisterCalibrationProfile(p CalibrationProfile) error {
	return calib.Register(p)
}

// LookupCalibrationProfile returns the profile registered under id.
func LookupCalibrationProfile(id string) (CalibrationProfile, bool) {
	return calib.Lookup(id)
}
