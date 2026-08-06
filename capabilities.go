package tokenizer

type Capabilities uint32

const (
	CapabilityCountText Capabilities = 1 << iota
	CapabilityEncode
	CapabilityDecode
	CapabilityCountMessages
	CapabilityTools
	CapabilityMultimodalEstimate
)
