package tokenizer_test

import (
	"encoding/json"
	"testing"

	"github.com/xorvus/tokenizer-go"
)

func TestCapabilitiesHas(t *testing.T) {
	c := tokenizer.CapabilityCountText | tokenizer.CapabilityEncode
	if !c.Has(tokenizer.CapabilityCountText) {
		t.Error("Has(CountText) = false, want true")
	}
	if c.Has(tokenizer.CapabilityDecode) {
		t.Error("Has(Decode) = true, want false")
	}
	if !c.Has(tokenizer.CapabilityCountText | tokenizer.CapabilityEncode) {
		t.Error("Has should report true when all requested bits are set")
	}
	if c.Has(tokenizer.CapabilityCountText | tokenizer.CapabilityDecode) {
		t.Error("Has should report false when only some requested bits are set")
	}
}

func TestCapabilitiesJSONRoundTrip(t *testing.T) {
	c := tokenizer.CapabilityCountText | tokenizer.CapabilityEncode | tokenizer.CapabilityDecode | tokenizer.CapabilityCountMessages
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var got tokenizer.Capabilities
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal(%s) error: %v", b, err)
	}
	if got != c {
		t.Errorf("round trip: got %v, want %v", got, c)
	}
}

func TestCapabilitiesJSONIsReadableNames(t *testing.T) {
	c := tokenizer.CapabilityCountText
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	// Must serialize as a list of names, not a bare integer bitmask.
	var names []string
	if err := json.Unmarshal(b, &names); err != nil {
		t.Fatalf("Capabilities did not marshal as a JSON string array: %s (%v)", b, err)
	}
	if len(names) != 1 || names[0] != "count_text" {
		t.Errorf("Marshal(CapabilityCountText) = %s, want [\"count_text\"]", b)
	}
}

func TestModelSpecCapabilitiesAreReadableViaLookup(t *testing.T) {
	// Regression guard for the audit finding that Capabilities was
	// written into ModelSpec but had no read path (Registry had no
	// Lookup method). LookupModel closes that gap.
	spec, ok := tokenizer.LookupModel("gpt-oss-1")
	if !ok {
		t.Fatal("LookupModel(gpt-oss-1) not found")
	}
	if !spec.Capabilities.Has(tokenizer.CapabilityCountMessages) {
		t.Errorf("gpt-oss-1 (Harmony) ModelSpec.Capabilities = %v, want CapabilityCountMessages set", spec.Capabilities)
	}

	spec, ok = tokenizer.LookupModel("gpt-4o")
	if !ok {
		t.Fatal("LookupModel(gpt-4o) not found")
	}
	if !spec.Capabilities.Has(tokenizer.CapabilityCountText | tokenizer.CapabilityEncode | tokenizer.CapabilityDecode) {
		t.Errorf("gpt-4o ModelSpec.Capabilities = %v, missing expected base capabilities", spec.Capabilities)
	}
}
