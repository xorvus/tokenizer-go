package tokenizer

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Registry struct {
	mu          sync.RWMutex
	exactModels map[string]ModelSpec
	aliases     map[string]string
	prefixes    []prefixEntry
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
	return r
}

func (r *Registry) registerOpenAI() {
	r.registerOpenAIExact()
	r.registerOpenAIPrefixes()
}

func (r *Registry) registerOpenAIExact() {
	o200k := []string{"gpt-4o", "gpt-4o-mini", "gpt-4.5-preview", "o1", "o1-mini", "o1-preview", "o3-mini", "gpt-5"}
	for _, m := range o200k {
		r.exactModels[m] = ModelSpec{CanonicalName: m, Provider: ProviderOpenAI, TokenizerID: string(O200KBase), Capabilities: CapabilityCountText | CapabilityEncode | CapabilityDecode, IsExact: true}
	}
	harmony := []string{"gpt-oss-1", "gpt-oss-2"}
	for _, m := range harmony {
		r.exactModels[m] = ModelSpec{CanonicalName: m, Provider: ProviderOpenAI, TokenizerID: string(O200KHarmony), Capabilities: CapabilityCountText | CapabilityEncode | CapabilityDecode | CapabilityCountMessages, IsExact: true}
	}
	cl100k := []string{"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo", "text-embedding-ada-002", "text-embedding-3-small", "text-embedding-3-large"}
	for _, m := range cl100k {
		r.exactModels[m] = ModelSpec{CanonicalName: m, Provider: ProviderOpenAI, TokenizerID: string(CL100KBase), Capabilities: CapabilityCountText | CapabilityEncode | CapabilityDecode, IsExact: true}
	}
}

func (r *Registry) registerOpenAIPrefixes() {
	p := []prefixEntry{
		{"gpt-oss-", ModelSpec{Provider: ProviderOpenAI, TokenizerID: string(O200KHarmony), IsExact: true}},
		{"o1-", ModelSpec{Provider: ProviderOpenAI, TokenizerID: string(O200KBase), IsExact: true}},
		{"o3-", ModelSpec{Provider: ProviderOpenAI, TokenizerID: string(O200KBase), IsExact: true}},
		{"gpt-5-", ModelSpec{Provider: ProviderOpenAI, TokenizerID: string(O200KBase), IsExact: true}},
		{"gpt-4.5-", ModelSpec{Provider: ProviderOpenAI, TokenizerID: string(O200KBase), IsExact: true}},
		{"gpt-4o-", ModelSpec{Provider: ProviderOpenAI, TokenizerID: string(O200KBase), IsExact: true}},
		{"gpt-4-", ModelSpec{Provider: ProviderOpenAI, TokenizerID: string(CL100KBase), IsExact: true}},
		{"gpt-3.5-turbo-", ModelSpec{Provider: ProviderOpenAI, TokenizerID: string(CL100KBase), IsExact: true}},
		{"ft:gpt-4o", ModelSpec{Provider: ProviderOpenAI, TokenizerID: string(O200KBase), IsExact: true}},
		{"ft:gpt-4", ModelSpec{Provider: ProviderOpenAI, TokenizerID: string(CL100KBase), IsExact: true}},
	}
	sort.Slice(p, func(i, j int) bool { return len(p[i].prefix) > len(p[j].prefix) })
	r.prefixes = p
}

func ResolveModel(model string) (Resolution, error) {
	return globalRegistry.Resolve(model)
}

func (r *Registry) Resolve(model string) (Resolution, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if res, ok := r.resolveExactOrAlias(model); ok {
		return res, nil
	}
	if res, ok := r.resolvePrefix(model); ok {
		return res, nil
	}
	return Resolution{}, fmt.Errorf("%w: %s", ErrUnknownModel, model)
}

func (r *Registry) resolveExactOrAlias(model string) (Resolution, bool) {
	target := model
	if canonical, ok := r.aliases[model]; ok {
		target = canonical
	}
	spec, ok := r.exactModels[target]
	if !ok {
		return Resolution{}, false
	}
	res := Resolution{RequestedModel: model, CanonicalModel: spec.CanonicalName, Provider: spec.Provider, TokenizerID: spec.TokenizerID, Accuracy: AccuracyExactLocal}
	return res, true
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
