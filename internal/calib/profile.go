// Package calib provides offline token-count estimation for providers
// without a tokenizer artifact (e.g. Anthropic, Gemini). It never calls a
// network at runtime: a Profile is a committed data file recording the
// ratio between a target provider's real count and the nearest embedded
// tokenizer's count, measured by scripts/calibrate.py.
//
// Profiles measured from zero real samples (SampleCount == 0) are identity
// placeholders — ratios of 1.0, "use the raw count uncorrected" — and
// report AccuracyEstimatedHeuristic, never calibrated. Every shipped
// profile must state its true SampleCount.
package calib

import (
	"embed"
	"encoding/json"
	"fmt"
	"sync"
)

// Ratio is the correction factor for one Bucket within a Profile.
type Ratio struct {
	// Mean is target_tokens / base_tokens averaged over the calibration
	// corpus; CountResult.Tokens is scaled by it. 1.0 = no correction.
	Mean float64 `json:"mean"`
	P95  float64 `json:"p95"`
	Max  float64 `json:"max"`

	// The AbsErrorPct fields describe expected error of the scaled
	// estimate, as a percentage. They are 0 for an uncalibrated identity
	// profile and must not be read as zero error; CountResult.UpperBound()
	// checks SampleCount for this reason.
	MeanAbsErrorPct float64 `json:"mean_abs_error_pct"`
	P95AbsErrorPct  float64 `json:"p95_abs_error_pct"`
	MaxAbsErrorPct  float64 `json:"max_abs_error_pct"`
}

// Profile estimates one target model's token count from a nearest
// embedded tokenizer.
type Profile struct {
	ProfileID     string           `json:"profile_id"`
	TargetModel   string           `json:"target_model"`
	Provider      string           `json:"provider"`
	BaseTokenizer string           `json:"base_tokenizer"`
	Ratios        map[Bucket]Ratio `json:"ratios"`
	SampleCount   int              `json:"sample_count"`
	CorpusSHA256  string           `json:"corpus_sha256,omitempty"`
	GeneratedAt   string           `json:"generated_at,omitempty"`
	ToolVersion   string           `json:"tool_version,omitempty"`
	// Notes is a free-text human-readable disclaimer; not read programmatically.
	Notes string `json:"notes,omitempty"`
}

//go:embed profiles/*.json
var embeddedProfilesFS embed.FS

var (
	mu       sync.RWMutex
	profiles map[string]Profile
)

func init() {
	profiles = make(map[string]Profile)
	entries, err := embeddedProfilesFS.ReadDir("profiles")
	if err != nil {
		// Embedded at build time; a missing directory is a packaging bug.
		panic(fmt.Sprintf("calib: reading embedded profiles: %v", err))
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := embeddedProfilesFS.ReadFile("profiles/" + e.Name())
		if err != nil {
			panic(fmt.Sprintf("calib: reading embedded profile %s: %v", e.Name(), err))
		}
		var p Profile
		if err := json.Unmarshal(data, &p); err != nil {
			panic(fmt.Sprintf("calib: parsing embedded profile %s: %v", e.Name(), err))
		}
		if p.ProfileID == "" {
			panic(fmt.Sprintf("calib: embedded profile %s has empty profile_id", e.Name()))
		}
		if _, ok := p.Ratios[BucketOther]; !ok {
			panic(fmt.Sprintf("calib: embedded profile %s is missing a %q ratio (required as the lookup fallback)", p.ProfileID, BucketOther))
		}
		profiles[p.ProfileID] = p
	}
}

// Lookup returns the profile registered under id (embedded or registered).
func Lookup(id string) (Profile, bool) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := profiles[id]
	return p, ok
}

// Register adds or replaces a profile at runtime, e.g. to load a freshly
// calibrated one without a rebuild. Requires a BucketOther ratio.
func Register(p Profile) error {
	if p.ProfileID == "" {
		return fmt.Errorf("calib: profile_id cannot be empty")
	}
	if p.BaseTokenizer == "" {
		return fmt.Errorf("calib: profile %q: base_tokenizer cannot be empty", p.ProfileID)
	}
	if _, ok := p.Ratios[BucketOther]; !ok {
		return fmt.Errorf("calib: profile %q: missing required %q ratio", p.ProfileID, BucketOther)
	}
	mu.Lock()
	defer mu.Unlock()
	profiles[p.ProfileID] = p
	return nil
}

// RatioFor returns the ratio for bucket b, falling back to BucketOther.
// Every valid Profile defines BucketOther, so this always succeeds.
func (p Profile) RatioFor(b Bucket) Ratio {
	if r, ok := p.Ratios[b]; ok {
		return r
	}
	return p.Ratios[BucketOther]
}

// IsCalibrated reports whether this profile was derived from real
// measurements (SampleCount > 0) as opposed to an identity placeholder.
func (p Profile) IsCalibrated() bool {
	return p.SampleCount > 0
}
