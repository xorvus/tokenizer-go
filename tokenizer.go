package tokenizer

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/xorvus/tokenizer-go/internal/bpe"
	"github.com/xorvus/tokenizer-go/internal/calib"
	"github.com/xorvus/tokenizer-go/internal/openai"
)

type Config struct {
	Name           string
	Pattern        string
	MergeableRanks map[string]int
	SpecialTokens  map[string]int
}

type Tokenizer struct {
	core    *bpe.CoreBPE
	options Options
}

// var _ Engine = (*Tokenizer)(nil): compile-time assertion that *Tokenizer
// implements Engine.
var _ Engine = (*Tokenizer)(nil)

func validateAndCopyConfig(cfg Config) (map[string]int, []string, map[string]int, error) {
	if cfg.Pattern == "" {
		return nil, nil, nil, ErrEmptyPattern
	}
	if len(cfg.MergeableRanks) == 0 {
		return nil, nil, nil, ErrEmptyVocabulary
	}

	// Copy MergeableRanks for immutability and validate.
	ranksCopy, usedRanks, maxRank, byteSeen, err := collectMergeableRanks(cfg.MergeableRanks)
	if err != nil {
		return nil, nil, nil, err
	}

	// Verify all 0-255 single-byte tokens exist.
	for b := 0; b <= 255; b++ {
		if !byteSeen[b] {
			return nil, nil, nil, fmt.Errorf("%w: missing byte 0x%02X (%d)", ErrMissingByteToken, b, b)
		}
	}

	// Reconstruct decoder slice.
	decoderSlice := make([]string, maxRank+1)
	for token, rank := range ranksCopy {
		decoderSlice[rank] = token
	}

	// Copy SpecialTokens for immutability and validate.
	specialCopy, err := collectSpecialTokens(cfg.SpecialTokens, ranksCopy, usedRanks)
	if err != nil {
		return nil, nil, nil, err
	}

	return ranksCopy, decoderSlice, specialCopy, nil
}

// collectMergeableRanks copies and validates ranks, tracking used ranks,
// max rank, and present single bytes for the byte-completeness check.
func collectMergeableRanks(mergeable map[string]int) (ranksCopy map[string]int, usedRanks map[int]string, maxRank int, byteSeen []bool, err error) {
	byteSeen = make([]bool, 256)
	maxRank = -1
	usedRanks = make(map[int]string, len(mergeable))
	ranksCopy = make(map[string]int, len(mergeable))
	for token, rank := range mergeable {
		if rank < 0 {
			return nil, nil, 0, nil, fmt.Errorf("%w: token %q has rank %d", ErrNegativeRank, token, rank)
		}
		if existingToken, exists := usedRanks[rank]; exists {
			return nil, nil, 0, nil, fmt.Errorf("%w: rank %d used by %q and %q", ErrDuplicateRank, rank, existingToken, token)
		}
		usedRanks[rank] = token
		ranksCopy[token] = rank

		if len(token) == 1 {
			byteSeen[token[0]] = true
		}
		if rank > maxRank {
			maxRank = rank
		}
	}
	return ranksCopy, usedRanks, maxRank, byteSeen, nil
}

// collectSpecialTokens copies and validates SpecialTokens.
func collectSpecialTokens(special map[string]int, ranksCopy map[string]int, usedRanks map[int]string) (map[string]int, error) {
	if len(special) == 0 {
		return map[string]int{}, nil
	}
	specialCopy := make(map[string]int, len(special))
	for token, rank := range special {
		if rank < 0 {
			return nil, fmt.Errorf("%w: special token %q has rank %d", ErrNegativeRank, token, rank)
		}
		if _, exists := ranksCopy[token]; exists {
			return nil, fmt.Errorf("%w: token %q is both regular and special", ErrTokenConflict, token)
		}
		if existingToken, exists := usedRanks[rank]; exists {
			return nil, fmt.Errorf("%w: special rank %d used by %q and %q", ErrDuplicateRank, rank, existingToken, token)
		}
		usedRanks[rank] = token
		specialCopy[token] = rank
	}
	return specialCopy, nil
}

func newTokenizer(core *bpe.CoreBPE) *Tokenizer {
	return &Tokenizer{
		core:    core,
		options: DefaultOptions(),
	}
}

func New(config Config) (*Tokenizer, error) {
	ranks, decoder, special, err := validateAndCopyConfig(config)
	if err != nil {
		return nil, err
	}
	core, err := bpe.NewCoreBPE(ranks, decoder, special, config.Pattern)
	if err != nil {
		return nil, err
	}
	return newTokenizer(core), nil
}

func NewFromVocabulary(r io.Reader, pattern string, special map[string]int) (*Tokenizer, error) {
	vocab, err := openai.ParseVocabulary(r)
	if err != nil {
		return nil, err
	}
	core, err := bpe.New(bpe.Config{
		Pattern:        pattern,
		MergeableRanks: vocab.Encoder,
		DecoderSlice:   vocab.Decoder,
		SpecialTokens:  special,
	})
	if err != nil {
		return nil, err
	}
	return newTokenizer(core), nil
}

func NewFromFile(path string, pattern string, special map[string]int) (*Tokenizer, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return NewFromVocabulary(f, pattern, special)
}

func GetEncoding(name Encoding) (*Tokenizer, error) {
	switch name {
	case CL100KBase:
		return getEmbeddedCL100K()
	case O200KBase:
		return getEmbeddedO200K()
	case O200KHarmony:
		return getEmbeddedO200KHarmony()
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownEncoding, name)
	}
}

// ForModel returns the exact tokenizer for model. Exact-only: returns
// ErrExactTokenizerUnavailable for models that only resolve to an estimate;
// use CountForModel with WithCalibratedFallback to opt into those.
func ForModel(model string) (*Tokenizer, error) {
	encoding, err := EncodingForModel(model)
	if err != nil {
		return nil, err
	}
	return GetEncoding(encoding)
}

// EncodingForModel resolves model to its exact embedded Encoding. See ForModel.
func EncodingForModel(model string) (Encoding, error) {
	res, err := ResolveModel(model)
	if err != nil {
		return "", err
	}
	if res.UsedFallback {
		return "", fmt.Errorf("%w: %s has no embedded exact tokenizer (nearest-tokenizer estimate available via CountForModel)", ErrExactTokenizerUnavailable, model)
	}
	return Encoding(res.TokenizerID), nil
}

func (t *Tokenizer) WithOptions(opts Options) *Tokenizer {
	return &Tokenizer{
		core:    t.core,
		options: sanitizeOptions(opts),
	}
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func (t *Tokenizer) Encode(text string) ([]int, error) {
	tokens, _, err := t.core.EncodeNative(text, nil)
	return tokens, err
}

func (t *Tokenizer) EncodeOrdinary(text string) ([]int, error) {
	return t.core.EncodeOrdinaryNative(text)
}

func (t *Tokenizer) EncodeOrdinaryBatch(texts []string) ([][]int, error) {
	return t.core.EncodeOrdinaryBatchNative(texts)
}

func (t *Tokenizer) EncodeContext(ctx context.Context, text string) ([]int, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	return t.EncodeOrdinary(text)
}

func (t *Tokenizer) CountContext(ctx context.Context, text string) (int, error) {
	if err := checkContext(ctx); err != nil {
		return 0, err
	}
	return t.Count(text)
}

func (t *Tokenizer) EncodeOrdinaryBatchContext(ctx context.Context, texts []string) ([][]int, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	res := make([][]int, len(texts))
	for i, txt := range texts {
		if i%10 == 0 {
			if err := checkContext(ctx); err != nil {
				return nil, err
			}
		}
		tokens, err := t.EncodeOrdinary(txt)
		if err != nil {
			return nil, err
		}
		res[i] = tokens
	}
	return res, nil
}

func (t *Tokenizer) CountOrdinaryBatchContext(ctx context.Context, texts []string) ([]int, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	res := make([]int, len(texts))
	for i, txt := range texts {
		if i%10 == 0 {
			if err := checkContext(ctx); err != nil {
				return nil, err
			}
		}
		count, err := t.Count(txt)
		if err != nil {
			return nil, err
		}
		res[i] = count
	}
	return res, nil
}

func (t *Tokenizer) EncodeSingleToken(text string) (int, error) {
	if id, ok := t.core.Encoder[text]; ok {
		return id, nil
	}
	if id, ok := t.core.SpecialTokensEncoder[text]; ok {
		return id, nil
	}
	tokens, err := t.EncodeOrdinary(text)
	if err != nil {
		return 0, err
	}
	if len(tokens) != 1 {
		return 0, fmt.Errorf("%w: %q encodes into %d tokens", ErrInvalidSingleToken, text, len(tokens))
	}
	return tokens[0], nil
}

func (t *Tokenizer) Count(text string) (int, error) {
	return t.core.CountOrdinarySequential(text)
}

func (t *Tokenizer) CountOrdinaryBatch(texts []string) ([]int, error) {
	return t.core.CountOrdinaryBatchNative(texts)
}

func (t *Tokenizer) Decode(tokens []int) (string, error) {
	bytes, err := t.core.DecodeNative(tokens)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (t *Tokenizer) DecodeBytes(tokens []int) ([]byte, error) {
	return t.core.DecodeNative(tokens)
}

func (t *Tokenizer) DecodeSingleTokenBytes(token int) ([]byte, error) {
	if token >= 0 && token < len(t.core.DecoderSlice) && t.core.DecoderSlice[token] != "" {
		return []byte(t.core.DecoderSlice[token]), nil
	}
	if val, ok := t.core.SpecialTokensDecoder[token]; ok {
		return []byte(val), nil
	}
	return nil, fmt.Errorf("%w: %d", ErrUnknownTokenID, token)
}

func (t *Tokenizer) TokenByteValues() [][]byte {
	res := make([][]byte, len(t.core.DecoderSlice))
	for i, s := range t.core.DecoderSlice {
		res[i] = []byte(s)
	}
	return res
}

func (t *Tokenizer) DecodeWithOffsets(tokens []int) (string, []int, error) {
	var sb strings.Builder
	offsets := make([]int, 0, len(tokens))
	currentOffset := 0

	for _, tok := range tokens {
		b, err := t.DecodeSingleTokenBytes(tok)
		if err != nil {
			return "", nil, err
		}
		offsets = append(offsets, currentOffset)
		sb.Write(b)
		currentOffset += len(b)
	}

	return sb.String(), offsets, nil
}

func (t *Tokenizer) DecodeBatch(batch [][]int) ([]string, error) {
	res := make([]string, len(batch))
	for i, tokens := range batch {
		str, err := t.Decode(tokens)
		if err != nil {
			return nil, err
		}
		res[i] = str
	}
	return res, nil
}

func (t *Tokenizer) DecodeBytesBatch(batch [][]int) ([][]byte, error) {
	res := make([][]byte, len(batch))
	for i, tokens := range batch {
		b, err := t.DecodeBytes(tokens)
		if err != nil {
			return nil, err
		}
		res[i] = b
	}
	return res, nil
}

// CountForModel counts text as model would be counted by its provider.
// Exact for embedded tokenizers; an offline nearest-tokenizer estimate
// scaled by a calibration profile otherwise. Estimates are allowed only
// when calibrated; use CountResult.UpperBound() for a hard budget, or
// WithHeuristicFallback()/WithExactOnly() to change fallback policy.
func CountForModel(model string, text string, opts ...CountOption) (CountResult, error) {
	cfg := DefaultCountConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	res, err := ResolveModel(model)
	if err != nil {
		return CountResult{}, err
	}

	if err := checkFallbackAllowed(res, cfg); err != nil {
		return CountResult{}, err
	}

	tok, err := GetEncoding(Encoding(res.TokenizerID))
	if err != nil {
		return CountResult{}, err
	}

	base, err := tok.Count(text)
	if err != nil {
		return CountResult{}, err
	}

	out := CountResult{
		Tokens:         base,
		RequestedModel: res.RequestedModel,
		CanonicalModel: res.CanonicalModel,
		TokenizerID:    res.TokenizerID,
		Provider:       res.Provider,
		Accuracy:       res.Accuracy,
		UsedFallback:   res.UsedFallback,
		FallbackReason: res.Reason,
	}
	if !res.UsedFallback {
		return out, nil
	}

	prof, ok := calib.Lookup(res.ProfileID)
	if !ok {
		// Resolve only sets UsedFallback when it found the profile, so
		// this should be unreachable barring a profile being unregistered
		// between Resolve and here. Report the unscaled base count rather
		// than fail outright.
		return out, nil
	}
	ratio := prof.RatioFor(calib.Classify(text))
	if ratio.Mean > 0 {
		out.Tokens = int(math.Ceil(float64(base) * ratio.Mean))
	}
	out.Calibration = &Calibration{
		ProfileID:       prof.ProfileID,
		SampleCount:     prof.SampleCount,
		MeanAbsErrorPct: ratio.MeanAbsErrorPct,
		P95AbsErrorPct:  ratio.P95AbsErrorPct,
		MaxAbsErrorPct:  ratio.MaxAbsErrorPct,
		CorpusSHA256:    prof.CorpusSHA256,
	}
	return out, nil
}

// checkFallbackAllowed reports whether an estimate for res is allowed.
func checkFallbackAllowed(res Resolution, cfg CountConfig) error {
	if !res.UsedFallback {
		return nil
	}
	switch res.Accuracy {
	case AccuracyEstimatedCalibrated:
		if !cfg.AllowCalibratedFallback {
			return fmt.Errorf("%w: %s only resolves to a calibrated estimate, and calibrated fallback is disabled", ErrExactTokenizerUnavailable, res.RequestedModel)
		}
	case AccuracyEstimatedHeuristic:
		if !cfg.AllowHeuristicFallback {
			return fmt.Errorf("%w: %s has no calibrated profile yet (only an uncalibrated heuristic estimate); pass WithHeuristicFallback() to allow it", ErrExactTokenizerUnavailable, res.RequestedModel)
		}
	}
	return nil
}
