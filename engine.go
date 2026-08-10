package tokenizer

type Algorithm string

const (
	AlgorithmTiktokenBPE   Algorithm = "tiktoken_bpe"
	AlgorithmHFByteBPE     Algorithm = "hf_byte_bpe"
	AlgorithmSentencePiece Algorithm = "sentencepiece"
	AlgorithmTekken        Algorithm = "tekken"
)

type EngineMetadata struct {
	ID           string
	Algorithm    Algorithm
	VocabSize    int
	Version      string
	AssetSHA256  string
	Capabilities Capabilities
}

// Counter counts tokens for text. Implemented by every provider tier,
// including nearest-tokenizer estimates.
type Counter interface {
	Count(text string) (int, error)
}

// Codec produces and consumes exact token IDs; only providers with an
// embedded exact tokenizer implement it (estimates can't fake a token ID
// sequence, hence Counter/Codec stay separate).
type Codec interface {
	Encode(text string) ([]int, error)
	Decode(tokens []int) (string, error)
}

// Engine is Counter + Codec: a full exact tokenizer. Estimates implement
// only Counter, not Engine.
type Engine interface {
	Counter
	Codec
}
