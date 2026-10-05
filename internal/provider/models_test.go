package provider

import (
	"testing"
)

// TestUnknownModelFallback verifies unknown models return sensible defaults.
func TestUnknownModelFallback(t *testing.T) {
	m := LookupModel("unknown-future-model-xyz")

	if m.Known {
		t.Error("Unknown model should have Known = false")
	}
	if m.ContextWindow != 128000 {
		t.Errorf("Unknown model ContextWindow = %d, want 128000", m.ContextWindow)
	}
	if m.MaxOutput != 4096 {
		t.Errorf("Unknown model MaxOutput = %d, want 4096", m.MaxOutput)
	}
	if m.Pricing != PricingUnknown {
		t.Errorf("Unknown model Pricing = %v, want PricingUnknown", m.Pricing)
	}
	if m.ID != "unknown-future-model-xyz" {
		t.Errorf("Unknown model ID = %q, want %q", m.ID, "unknown-future-model-xyz")
	}
}

// TestMinimaxM3 verifies that minimax-m3 is a known model with PricingFlat
// and the configured fallback dimensions of 128000 context and 4096 output.
func TestMinimaxM3(t *testing.T) {
	m := LookupModel("minimax-m3")

	if !m.Known {
		t.Error("minimax-m3 should be Known = true")
	}
	if m.Pricing != PricingFlat {
		t.Errorf("minimax-m3: Pricing = %v, want PricingFlat", m.Pricing)
	}
	if m.ContextWindow != 128000 {
		t.Errorf("minimax-m3: ContextWindow = %d, want 128000", m.ContextWindow)
	}
	if m.MaxOutput != 4096 {
		t.Errorf("minimax-m3: MaxOutput = %d, want 4096", m.MaxOutput)
	}
}

// TestMinimaxM2_7 verifies that minimax-m2.7 (the opencode default model) is
// a known model with PricingFlat and the configured fallback dimensions of
// 128000 context and 4096 output.
func TestMinimaxM2_7(t *testing.T) {
	m := LookupModel("minimax-m2.7")

	if !m.Known {
		t.Error("minimax-m2.7 should be Known = true")
	}
	if m.Pricing != PricingFlat {
		t.Errorf("minimax-m2.7: Pricing = %v, want PricingFlat", m.Pricing)
	}
	if m.ContextWindow != 128000 {
		t.Errorf("minimax-m2.7: ContextWindow = %d, want 128000", m.ContextWindow)
	}
	if m.MaxOutput != 4096 {
		t.Errorf("minimax-m2.7: MaxOutput = %d, want 4096", m.MaxOutput)
	}
}

// TestPricingUnknownIsZeroValue verifies PricingUnknown is the zero value for Pricing.
func TestPricingUnknownIsZeroValue(t *testing.T) {
	var zero Pricing
	if zero != PricingUnknown {
		t.Errorf("zero Pricing should be PricingUnknown, got %v", zero)
	}

	// Verify a zero-initialized ModelInfo has PricingUnknown.
	m := ModelInfo{}
	if m.Pricing != PricingUnknown {
		t.Errorf("zero ModelInfo.Pricing should be PricingUnknown, got %v", m.Pricing)
	}
}
