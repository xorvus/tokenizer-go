package tokenizer

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/xorvus/tokenizer-go/internal/calib"
	"github.com/xorvus/tokenizer-go/internal/openai"
)

// Registry resolves model names to tokenizers — exact, or nearest-tokenizer
// estimate. Safe for concurrent use.
type Registry struct {
	mu sync.RWMutex

	exactModels map[string]ModelSpec
	aliases     map[string]string
	prefixes    []prefixEntry

	fallbackPrefixes []prefixEntry
}

type prefixEntry struct {
	prefix string
	spec   ModelSpec
}

var globalRegistry = newDefaultRegistry()

func newDefaultRegistry() *Registry {
	r := &Registry{
		exactModels: make(map[string]ModelSpec),
		aliases:     make(map[string]string),
	}
	r.registerOpenAI()
	r.registerOfflineFallbacks()
	return r
}

func (r *Registry) registerOpenAI() {
	for model, enc := range openai.ModelToEncoding {
		r.exactModels[model] = ModelSpec{
			CanonicalName: model,
			Provider:      ProviderOpenAI,
			TokenizerID:   enc,
			Capabilities:  openAICapabilities(enc),
			IsExact:       true,
		}
	}

	// gpt-oss Harmony models aren't in openai.ModelToEncoding yet.
	for _, m := range []string{"gpt-oss-1", "gpt-oss-2"} {
		r.exactModels[m] = ModelSpec{
			CanonicalName: m,
			Provider:      ProviderOpenAI,
			TokenizerID:   string(O200KHarmony),
			Capabilities:  openAICapabilities(string(O200KHarmony)),
			IsExact:       true,
		}
	}

	prefixes := make([]prefixEntry, 0, len(openai.ModelPrefixToEncoding))
	for prefix, enc := range openai.ModelPrefixToEncoding {
		prefixes = append(prefixes, prefixEntry{
			prefix: prefix,
			spec: ModelSpec{
				Provider:     ProviderOpenAI,
				TokenizerID:  enc,
				Capabilities: openAICapabilities(enc),
				IsExact:      true,
			},
		})
	}
	r.prefixes = sortPrefixesLongestFirst(prefixes)
}

func openAICapabilities(encoding string) Capabilities {
	caps := CapabilityCountText | CapabilityEncode | CapabilityDecode
	if encoding == string(O200KHarmony) {
		caps |= CapabilityCountMessages
	}
	return caps
}

func fallbackSpec(p Provider, prof string) ModelSpec {
	return ModelSpec{Provider: p, FallbackProfile: prof, Capabilities: CapabilityCountText}
}

func appendProviderFallbacks(list []prefixEntry) []prefixEntry {
	f := fallbackSpec
	return append(list,
		prefixEntry{"kimi-", f(ProviderKimi, "kimi-v1")},
		prefixEntry{"moonshot-", f(ProviderKimi, "kimi-v1")},
		prefixEntry{"grok-", f(ProviderGrok, "grok-v1")},
		prefixEntry{"mistral-", f(ProviderMistral, "mistral-v1")},
		prefixEntry{"codestral-", f(ProviderMistral, "mistral-v1")},
		prefixEntry{"ministral-", f(ProviderMistral, "mistral-v1")},
		prefixEntry{"mixtral-", f(ProviderMistral, "mistral-v1")},
		prefixEntry{"pixtral-", f(ProviderMistral, "mistral-v1")},
	)
}

func defaultFallbacks() []prefixEntry {
	f := fallbackSpec
	base := []prefixEntry{
		{"claude-", f(ProviderAnthropic, "anthropic-claude-v1")},
		{"gemini-", f(ProviderGemini, "gemini-v1")},
		{"deepseek-", f(ProviderDeepSeek, "deepseek-v1")},
		{"qwen-", f(ProviderQwen, "qwen-v1")},
		{"qwen2.5-", f(ProviderQwen, "qwen-v1")},
		{"qwq-", f(ProviderQwen, "qwen-v1")},
	}
	return appendProviderFallbacks(base)
}

func (r *Registry) registerOfflineFallbacks() {
	r.fallbackPrefixes = sortPrefixesLongestFirst(defaultFallbacks())
}

// sortPrefixesLongestFirst makes the longest prefix win, with a lexical
// tiebreak so equal-length prefixes form a deterministic order.
func sortPrefixesLongestFirst(entries []prefixEntry) []prefixEntry {
	sort.SliceStable(entries, func(i, j int) bool {
		if len(entries[i].prefix) != len(entries[j].prefix) {
			return len(entries[i].prefix) > len(entries[j].prefix)
		}
		return entries[i].prefix < entries[j].prefix
	})
	return entries
}

// ResolveModel resolves model against the global registry. See
// Registry.Resolve.
func ResolveModel(model string) (Resolution, error) {
	return globalRegistry.Resolve(model)
}

// LookupModel returns the ModelSpec that ResolveModel(model) would match,
// against the global registry. See Registry.Lookup.
func LookupModel(model string) (ModelSpec, bool) {
	return globalRegistry.Lookup(model)
}

// RegisterModel adds or replaces an exact model registration in the
// global registry, used by ForModel, EncodingForModel, and
// CountForModel. See Registry.Register.
func RegisterModel(spec ModelSpec) error {
	return globalRegistry.Register(spec)
}

// RegisterModelAlias registers a model-name alias in the global registry.
// See Registry.RegisterAlias.
func RegisterModelAlias(alias, canonical string) error {
	return globalRegistry.RegisterAlias(alias, canonical)
}

// RegisterModelPrefix registers an exact-tokenizer prefix rule in the
// global registry. See Registry.RegisterPrefix.
func RegisterModelPrefix(prefix string, spec ModelSpec) error {
	return globalRegistry.RegisterPrefix(prefix, spec)
}

// RegisterModelFallbackPrefix registers a nearest-tokenizer estimate
// prefix rule in the global registry — the mechanism used to add offline
// estimation for a provider this library does not know about. See
// Registry.RegisterFallbackPrefix and internal/calib's package doc.
func RegisterModelFallbackPrefix(prefix string, provider Provider, profileID string) error {
	return globalRegistry.RegisterFallbackPrefix(prefix, provider, profileID)
}

// Resolve tries exact, alias, prefix, then fallback; ErrUnknownModel if none.
func (r *Registry) Resolve(model string) (Resolution, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if res, ok := r.resolveExactOrAlias(model); ok {
		return res, nil
	}
	if res, ok := r.resolvePrefix(model); ok {
		return res, nil
	}
	if res, ok := r.resolveFallback(model); ok {
		return res, nil
	}
	return Resolution{}, fmt.Errorf("%w: %s", ErrUnknownModel, model)
}

// Lookup returns the ModelSpec Resolve would match, without a full
// resolution (falls back to the profile's base tokenizer).
func (r *Registry) Lookup(model string) (ModelSpec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if spec, ok := r.exactModels[model]; ok {
		return spec, true
	}
	if canonical, ok := r.aliases[model]; ok {
		if spec, ok := r.exactModels[canonical]; ok {
			return spec, true
		}
	}
	for _, entry := range r.prefixes {
		if strings.HasPrefix(model, entry.prefix) {
			spec := entry.spec
			spec.CanonicalName = model
			return spec, true
		}
	}
	for _, entry := range r.fallbackPrefixes {
		if strings.HasPrefix(model, entry.prefix) {
			spec := entry.spec
			spec.CanonicalName = model
			if prof, ok := calib.Lookup(entry.spec.FallbackProfile); ok {
				spec.TokenizerID = prof.BaseTokenizer
			}
			return spec, true
		}
	}
	return ModelSpec{}, false
}

// Register adds or replaces an exact model registration.
func (r *Registry) Register(spec ModelSpec) error {
	if spec.CanonicalName == "" {
		return fmt.Errorf("tokenizer: ModelSpec.CanonicalName cannot be empty")
	}
	if spec.TokenizerID == "" {
		return fmt.Errorf("tokenizer: ModelSpec.TokenizerID cannot be empty (use RegisterFallbackPrefix for estimate-only models)")
	}
	spec.IsExact = true
	r.mu.Lock()
	defer r.mu.Unlock()
	r.exactModels[spec.CanonicalName] = spec
	return nil
}

// RegisterAlias makes alias resolve to the exact model canonical.
func (r *Registry) RegisterAlias(alias, canonical string) error {
	if alias == "" || canonical == "" {
		return fmt.Errorf("tokenizer: alias and canonical must both be non-empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.aliases[alias] = canonical
	return nil
}

// RegisterPrefix registers an exact-tokenizer prefix rule; longest wins.
func (r *Registry) RegisterPrefix(prefix string, spec ModelSpec) error {
	if prefix == "" {
		return fmt.Errorf("tokenizer: prefix cannot be empty")
	}
	if spec.TokenizerID == "" {
		return fmt.Errorf("tokenizer: ModelSpec.TokenizerID cannot be empty (use RegisterFallbackPrefix for estimate-only models)")
	}
	spec.IsExact = true
	r.mu.Lock()
	defer r.mu.Unlock()
	r.prefixes = sortPrefixesLongestFirst(append(r.prefixes, prefixEntry{prefix: prefix, spec: spec}))
	return nil
}

// RegisterFallbackPrefix registers a nearest-tokenizer estimate rule.
// profileID must already be registered with calib.Register.
func (r *Registry) RegisterFallbackPrefix(prefix string, provider Provider, profileID string) error {
	if prefix == "" {
		return fmt.Errorf("tokenizer: prefix cannot be empty")
	}
	if profileID == "" {
		return fmt.Errorf("tokenizer: profileID cannot be empty")
	}
	if _, ok := calib.Lookup(profileID); !ok {
		return fmt.Errorf("tokenizer: no calibration profile registered under %q; call calib.Register first", profileID)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fallbackPrefixes = sortPrefixesLongestFirst(append(r.fallbackPrefixes, prefixEntry{
		prefix: prefix,
		spec: ModelSpec{
			Provider:        provider,
			FallbackProfile: profileID,
			Capabilities:    CapabilityCountText,
		},
	}))
	return nil
}

func (r *Registry) resolveExactOrAlias(model string) (Resolution, bool) {
	if spec, ok := r.exactModels[model]; ok {
		return Resolution{RequestedModel: model, CanonicalModel: spec.CanonicalName, Provider: spec.Provider, TokenizerID: spec.TokenizerID, Accuracy: AccuracyExactLocal}, true
	}
	if canonical, ok := r.aliases[model]; ok {
		if spec, ok := r.exactModels[canonical]; ok {
			return Resolution{RequestedModel: model, CanonicalModel: spec.CanonicalName, Provider: spec.Provider, TokenizerID: spec.TokenizerID, Accuracy: AccuracyExactLocal}, true
		}
	}
	return Resolution{}, false
}

func (r *Registry) resolvePrefix(model string) (Resolution, bool) {
	for _, entry := range r.prefixes {
		if strings.HasPrefix(model, entry.prefix) {
			return Resolution{
				RequestedModel: model,
				CanonicalModel: model,
				Provider:       entry.spec.Provider,
				TokenizerID:    entry.spec.TokenizerID,
				Accuracy:       AccuracyExactLocal,
			}, true
		}
	}
	return Resolution{}, false
}

// resolveFallback matches model against fallback prefixes; Accuracy follows
// the profile's SampleCount, so a calibrated profile reports calibrated.
func (r *Registry) resolveFallback(model string) (Resolution, bool) {
	for _, entry := range r.fallbackPrefixes {
		if !strings.HasPrefix(model, entry.prefix) {
			continue
		}
		prof, ok := calib.Lookup(entry.spec.FallbackProfile)
		if !ok {
			continue // missing profile; skip the rule
		}
		accuracy := AccuracyEstimatedHeuristic
		if prof.IsCalibrated() {
			accuracy = AccuracyEstimatedCalibrated
		}
		return Resolution{
			RequestedModel: model,
			CanonicalModel: model,
			Provider:       entry.spec.Provider,
			TokenizerID:    prof.BaseTokenizer,
			Accuracy:       accuracy,
			UsedFallback:   true,
			ProfileID:      prof.ProfileID,
			Reason: fmt.Sprintf(
				"no embedded tokenizer for %s; estimated using nearest tokenizer %s (profile %q, %d calibration samples)",
				entry.spec.Provider, prof.BaseTokenizer, prof.ProfileID, prof.SampleCount,
			),
		}, true
	}
	return Resolution{}, false
}
