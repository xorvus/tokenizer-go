package calib

import "unicode"

// Bucket classifies text by script so a Profile can apply a per-bucket
// ratio — the token-count ratio varies far more between scripts than
// across an error budget, so one global multiplier would be badly wrong.
type Bucket string

const (
	// BucketLatin is Latin-script prose (the common case).
	BucketLatin Bucket = "latin"
	// BucketZh is Chinese (Han-script only) text.
	BucketZh Bucket = "zh"
	// BucketJa is Japanese (Han plus kana) text.
	BucketJa Bucket = "ja"
	// BucketKo is Korean (Hangul) text.
	BucketKo Bucket = "ko"
	// BucketCode is source code, JSON, markup, or other punctuation-dense text.
	BucketCode Bucket = "code"
	// BucketOther is everything unclassified. Every Profile must define a
	// ratio for it so lookups never fail.
	BucketOther Bucket = "other"
)

// maxClassifySample bounds how many bytes Classify inspects, keeping
// classification O(1) while still sampling enough of a real prompt.
const maxClassifySample = 4096

// Classify returns the script bucket for text, inspecting at most
// maxClassifySample bytes. Thresholds are heuristics to tune against a
// real calibration corpus (scripts/calibrate.py), not fixed constants.
func Classify(text string) Bucket {
	sample := text
	if len(sample) > maxClassifySample {
		sample = sample[:maxClassifySample]
		// A truncated rune at the boundary decodes as replacement runes;
		// negligible noise at this sample size.
	}
	return countScripts(sample).bucket()
}

// scriptCounts tallies runes Classify needs; kept together so the two
// helpers stay readable.
type scriptCounts struct {
	cjk, kana, hangul, letters, latinLetters, punct, total int
}

func countScripts(sample string) scriptCounts {
	var c scriptCounts
	for _, r := range sample {
		c.total++
		switch {
		case isCJK(r):
			c.cjk++
			c.letters++
			if isKana(r) {
				c.kana++
			} else if isHangul(r) {
				c.hangul++
			}
		case unicode.IsLetter(r):
			c.letters++
			if isLatin(r) {
				c.latinLetters++
			}
		case r < 0x80 && (unicode.IsPunct(r) || unicode.IsSymbol(r)):
			c.punct++
		}
	}
	return c
}

func (c scriptCounts) bucket() Bucket {
	if c.total == 0 {
		return BucketLatin
	}
	switch {
	case pct(c.cjk, c.total) >= 20:
		switch {
		case c.hangul > 0:
			// Hangul presence marks Korean even mixed with Han.
			return BucketKo
		case c.kana > 0:
			// Japanese mixes Han kanji with kana; kana presence marks it.
			return BucketJa
		default:
			return BucketZh
		}
	case pct(c.punct, c.total) >= 15:
		// Dense braces/semicolons/quotes/operators signal code or markup.
		return BucketCode
	case pct(c.letters, c.total) >= 40 && c.latinLetters == c.letters:
		return BucketLatin
	default:
		return BucketOther
	}
}

func isKana(r rune) bool {
	return unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r)
}

func isHangul(r rune) bool {
	return unicode.Is(unicode.Hangul, r)
}

func pct(n, total int) int {
	if total == 0 {
		return 0
	}
	return n * 100 / total
}

func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) ||
		unicode.Is(unicode.Hangul, r)
}

func isLatin(r rune) bool {
	return unicode.Is(unicode.Latin, r)
}
