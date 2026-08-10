package tokenizer

import (
	"encoding/json"
	"strings"
)

type Capabilities uint32

const (
	CapabilityCountText Capabilities = 1 << iota
	CapabilityEncode
	CapabilityDecode
	CapabilityCountMessages
	CapabilityTools
	CapabilityMultimodalEstimate
)

var capabilityNames = []struct {
	bit  Capabilities
	name string
}{
	{CapabilityCountText, "count_text"},
	{CapabilityEncode, "encode"},
	{CapabilityDecode, "decode"},
	{CapabilityCountMessages, "count_messages"},
	{CapabilityTools, "tools"},
	{CapabilityMultimodalEstimate, "multimodal_estimate"},
}

// Has reports whether all bits in want are set.
func (c Capabilities) Has(want Capabilities) bool {
	return c&want == want
}

func (c Capabilities) String() string {
	if c == 0 {
		return "none"
	}
	var names []string
	for _, cn := range capabilityNames {
		if c&cn.bit != 0 {
			names = append(names, cn.name)
		}
	}
	return strings.Join(names, "|")
}

// MarshalJSON encodes Capabilities as a list of flag names rather than the
// raw bitmask, so wire consumers do not need this package's bit layout.
func (c Capabilities) MarshalJSON() ([]byte, error) {
	names := make([]string, 0, len(capabilityNames))
	for _, cn := range capabilityNames {
		if c&cn.bit != 0 {
			names = append(names, cn.name)
		}
	}
	return json.Marshal(names)
}

// UnmarshalJSON accepts the flag-name list produced by MarshalJSON.
func (c *Capabilities) UnmarshalJSON(data []byte) error {
	var names []string
	if err := json.Unmarshal(data, &names); err != nil {
		return err
	}
	var out Capabilities
	for _, n := range names {
		for _, cn := range capabilityNames {
			if cn.name == n {
				out |= cn.bit
				break
			}
		}
	}
	*c = out
	return nil
}
