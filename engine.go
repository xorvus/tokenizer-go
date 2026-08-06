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

type Engine interface {
	Encode(text string) ([]int, error)
	Decode(tokens []int) (string, error)
	Count(text string) (int, error)
}
