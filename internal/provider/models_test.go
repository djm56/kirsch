package provider

import (
	"testing"
)

// absFloat returns the absolute value of a float64.
func absFloat(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// TestUnknownModelFallback verifies unknown models return sensible defaults.
func TestUnknownModelFallback(t *testing.T) {
	m := LookupModel("unknown-future-model-xyz")

	if m.Known {
		t.Error("Unknown model should have Known = false")
	}
	// The fallback is derived from the table, so the expectation is derived
	// from it too: a literal here would pin whatever the table holds today.
	wantWindow, wantOut := tableMinimum()
	if m.ContextWindow != wantWindow {
		t.Errorf("Unknown model ContextWindow = %d, want table minimum %d", m.ContextWindow, wantWindow)
	}
	if m.MaxOutput != wantOut {
		t.Errorf("Unknown model MaxOutput = %d, want %d (smallest MaxOutput on the minimum-window rows)", m.MaxOutput, wantOut)
	}
	if m.Pricing != PricingUnknown {
		t.Errorf("Unknown model Pricing = %v, want PricingUnknown", m.Pricing)
	}
	if m.ID != "unknown-future-model-xyz" {
		t.Errorf("Unknown model ID = %q, want %q", m.ID, "unknown-future-model-xyz")
	}
}

// TestMinimaxM3 verifies that minimax-m3 (the opencode default model) is a
// known model with PricingFlat and the configured fallback dimensions of
// 128000 context and 4096 output.
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

// TestMinimaxM2_7 verifies that minimax-m2.7 is a known model with
// PricingFlat and the configured fallback dimensions of 128000 context and
// 4096 output.
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

// TestCostPerTokenModel verifies cost computation for per-token pricing models
// with distinct input, output, and cache prices. Each count is non-zero and distinct
// so a swapped price fails.
func TestCostPerTokenModel(t *testing.T) {
	m := LookupModel("claude-sonnet-5-5")
	// Sonnet: InputPerMTok=2.0, OutputPerMTok=10.0, CacheReadPerMTok=0.1, CacheWritePerMTok=2.5
	// Input: 1000 * 2.0 / 1e6 = 0.002
	// Output: 500 * 10.0 / 1e6 = 0.005
	// CacheRead: 200 * 0.1 / 1e6 = 0.00002
	// CacheWrite: 100 * 2.5 / 1e6 = 0.00025
	// Total: 0.00727
	costUSD, ok := m.Cost(1000, 500, 200, 100)
	if !ok {
		t.Error("Cost should return ok=true for PricingPerToken model")
	}
	expected := 0.00727
	if absFloat(costUSD-expected) > 1e-10 {
		t.Errorf("Cost = %v, want %v", costUSD, expected)
	}
}

// TestCostFlatModel verifies cost computation returns ok=false for flat-rate models.
func TestCostFlatModel(t *testing.T) {
	m := LookupModel("minimax-m3")
	costUSD, ok := m.Cost(1000, 500, 200, 100)
	if ok {
		t.Error("Cost should return ok=false for PricingFlat model")
	}
	if costUSD != 0 {
		t.Errorf("Cost should return 0 for PricingFlat model, got %v", costUSD)
	}
}

// TestCostUnknownModel verifies cost computation returns ok=false for unknown models.
func TestCostUnknownModel(t *testing.T) {
	m := LookupModel("unknown-model-xyz")
	costUSD, ok := m.Cost(1000, 500, 200, 100)
	if ok {
		t.Error("Cost should return ok=false for unknown model")
	}
	if costUSD != 0 {
		t.Errorf("Cost should return 0 for unknown model, got %v", costUSD)
	}
}

// TestCostAllPerTokenModels verifies all four per-token models produce the correct
// cost for the same token counts with their respective prices.
func TestCostAllPerTokenModels(t *testing.T) {
	tests := []struct {
		modelID string
		in      int64
		out     int64
		read    int64
		write   int64
		// cost = (in*inputPrice + out*outputPrice + read*readPrice + write*writePrice) / 1e6
		wantCost float64
	}{
		// Sonnet: input=2.0, output=10.0, read=0.1, write=2.5
		// cost = (100*2.0 + 50*10.0 + 20*0.1 + 10*2.5) / 1e6 = (200 + 500 + 2 + 25) / 1e6 = 727 / 1e6
		{"claude-sonnet-5-5", 100, 50, 20, 10, 0.000727},

		// Opus: input=4.0, output=20.0, read=0.2, write=5.0
		// cost = (100*4.0 + 50*20.0 + 20*0.2 + 10*5.0) / 1e6 = (400 + 1000 + 4 + 50) / 1e6 = 1454 / 1e6
		{"claude-opus-5-5", 100, 50, 20, 10, 0.001454},

		// Fable: input=10.0, output=50.0, read=0.25, write=12.5
		// cost = (100*10.0 + 50*50.0 + 20*0.25 + 10*12.5) / 1e6 = (1000 + 2500 + 5 + 125) / 1e6 = 3630 / 1e6
		{"claude-fable-5-1", 100, 50, 20, 10, 0.00363},

		// Haiku: input=1.0, output=5.0, read=0.1, write=1.25
		// cost = (100*1.0 + 50*5.0 + 20*0.1 + 10*1.25) / 1e6 = (100 + 250 + 2 + 12.5) / 1e6 = 364.5 / 1e6
		{"claude-haiku-4-5-20251001", 100, 50, 20, 10, 0.0003645},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			m := LookupModel(tt.modelID)
			cost, ok := m.Cost(tt.in, tt.out, tt.read, tt.write)
			if !ok {
				t.Fatal("Cost should return ok=true for known per-token model")
			}
			if absFloat(cost-tt.wantCost) > 1e-10 {
				t.Errorf("Cost = %v, want %v", cost, tt.wantCost)
			}
		})
	}
}

// TestPublishedPerTokenFigures pins every per-token figure of the Claude entries
// to the figures read from the vendor pricing page (https://platform.claude.com/docs/en/about-claude/pricing.md)
// on 2026-11-05: input, 5-minute cache write, cache hit, output, in USD per MTok.
// It reads the table through LookupModel, so a drifted row fails by field name.
func TestPublishedPerTokenFigures(t *testing.T) {
	tests := []struct {
		modelID                    string
		input, write, read, output float64
	}{
		// The rows quoted below omit the page's footnote markers.
		// | Claude Sonnet 5.5 | $2 / MTok | $2.50 / MTok | $4 / MTok | $0.10 / MTok | $10 / MTok |
		{"claude-sonnet-5-5", 2, 2.5, 0.1, 10},
		// | Claude Opus 5.5 | $4 / MTok | $5 / MTok | $8 / MTok | $0.20 / MTok | $20 / MTok |
		{"claude-opus-5-5", 4, 5, 0.2, 20},
		// | Claude Fable 5.1 | $10 / MTok | $12.50 / MTok | $20 / MTok | $0.25 / MTok | $50 / MTok |
		{"claude-fable-5-1", 10, 12.5, 0.25, 50},
		// | Claude Haiku 4.5 | $1 / MTok | $1.25 / MTok | $2 / MTok | $0.10 / MTok | $5 / MTok |
		{"claude-haiku-4-5-20251001", 1, 1.25, 0.1, 5},
	}
	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			m := LookupModel(tt.modelID)
			if m.Pricing != PricingPerToken {
				t.Fatalf("Pricing = %v, want PricingPerToken", m.Pricing)
			}
			for _, f := range []struct {
				name      string
				got, want float64
			}{
				{"InputPerMTok", m.InputPerMTok, tt.input},
				{"CacheWritePerMTok", m.CacheWritePerMTok, tt.write},
				{"CacheReadPerMTok", m.CacheReadPerMTok, tt.read},
				{"OutputPerMTok", m.OutputPerMTok, tt.output},
			} {
				if absFloat(f.got-f.want) > 1e-12 {
					t.Errorf("%s = %v, want %v (published)", f.name, f.got, f.want)
				}
			}
		})
	}
}

// TestCostZeroTokens verifies cost is zero when all token counts are zero.
func TestCostZeroTokens(t *testing.T) {
	m := LookupModel("claude-sonnet-5-5")
	costUSD, ok := m.Cost(0, 0, 0, 0)
	if !ok {
		t.Error("Cost should return ok=true")
	}
	if costUSD != 0 {
		t.Errorf("Cost for zero tokens should be 0, got %v", costUSD)
	}
}

// tableMinimum scans the model table for the smallest ContextWindow and, among
// the rows holding it, the smallest MaxOutput. It is written independently of
// conservativeLimits so the two can disagree.
func tableMinimum() (window, maxOut int) {
	window, maxOut = -1, -1
	for _, m := range models {
		if window == -1 || m.ContextWindow < window {
			window, maxOut = m.ContextWindow, m.MaxOutput
			continue
		}
		if m.ContextWindow == window && m.MaxOutput < maxOut {
			maxOut = m.MaxOutput
		}
	}
	return window, maxOut
}

// withModels replaces the model table for one test and restores it afterwards.
func withModels(t *testing.T, replacement map[string]ModelInfo) {
	t.Helper()
	saved := models
	models = replacement
	t.Cleanup(func() { models = saved })
}

// TestConservativeBudgetEqualsTableMinimum checks the unknown-model budget
// against a minimum computed inside the test, and that a known model keeps its
// own window (the mirror: the conservative figure must not leak onto it).
func TestConservativeBudgetEqualsTableMinimum(t *testing.T) {
	wantWindow, wantOut := tableMinimum()

	t.Run("unknown model gets the table minimum", func(t *testing.T) {
		got := LookupModel("an-id-that-is-not-in-the-table")
		if got.Known {
			t.Fatal("control: the id is in the table")
		}
		if got.ContextWindow != wantWindow || got.MaxOutput != wantOut {
			t.Errorf("unknown model window/maxOutput = %d/%d, want table minimum %d/%d",
				got.ContextWindow, got.MaxOutput, wantWindow, wantOut)
		}
		if ConservativeBudget() != wantWindow {
			t.Errorf("ConservativeBudget() = %d, want %d", ConservativeBudget(), wantWindow)
		}
		if ConservativeMaxOutput() != wantOut {
			t.Errorf("ConservativeMaxOutput() = %d, want %d", ConservativeMaxOutput(), wantOut)
		}
	})

	t.Run("known model keeps its own window", func(t *testing.T) {
		for id, row := range models {
			got := LookupModel(id)
			if got.ContextWindow != row.ContextWindow || got.MaxOutput != row.MaxOutput {
				t.Errorf("%s window/maxOutput = %d/%d, want its row %d/%d",
					id, got.ContextWindow, got.MaxOutput, row.ContextWindow, row.MaxOutput)
			}
		}
		if LookupModel("claude-sonnet-5-5").ContextWindow == wantWindow {
			t.Error("control: claude-sonnet-5-5 has the minimum window, so this mirror cannot tell its own window from the fallback")
		}
	})
}

// TestConservativeBudgetFollowsTheTable swaps the table for synthetic ones, so
// the figure cannot be a constant that merely equals today's rows.
func TestConservativeBudgetFollowsTheTable(t *testing.T) {
	t.Run("a smaller row lowers the budget", func(t *testing.T) {
		withModels(t, map[string]ModelInfo{
			"big":   {ID: "big", Known: true, ContextWindow: 500000, MaxOutput: 9000},
			"small": {ID: "small", Known: true, ContextWindow: 50000, MaxOutput: 1234},
		})
		got := LookupModel("neither")
		if got.ContextWindow != 50000 || got.MaxOutput != 1234 {
			t.Errorf("window/maxOutput = %d/%d, want 50000/1234", got.ContextWindow, got.MaxOutput)
		}
	})

	t.Run("every row above one million is not capped", func(t *testing.T) {
		withModels(t, map[string]ModelInfo{
			"a": {ID: "a", Known: true, ContextWindow: 2000000, MaxOutput: 70000},
			"b": {ID: "b", Known: true, ContextWindow: 3000000, MaxOutput: 80000},
		})
		got := LookupModel("neither")
		if got.ContextWindow != 2000000 || got.MaxOutput != 70000 {
			t.Errorf("window/maxOutput = %d/%d, want 2000000/70000", got.ContextWindow, got.MaxOutput)
		}
	})

	t.Run("a minimum row of exactly one million keeps its MaxOutput", func(t *testing.T) {
		withModels(t, map[string]ModelInfo{
			"only": {ID: "only", Known: true, ContextWindow: 1000000, MaxOutput: 128000},
		})
		got := LookupModel("neither")
		if got.ContextWindow != 1000000 || got.MaxOutput != 128000 {
			t.Errorf("window/maxOutput = %d/%d, want 1000000/128000", got.ContextWindow, got.MaxOutput)
		}
	})

	t.Run("a tie on the window takes the smaller MaxOutput whatever the map order", func(t *testing.T) {
		withModels(t, map[string]ModelInfo{
			"x": {ID: "x", Known: true, ContextWindow: 100000, MaxOutput: 8000},
			"y": {ID: "y", Known: true, ContextWindow: 100000, MaxOutput: 2000},
			"z": {ID: "z", Known: true, ContextWindow: 100000, MaxOutput: 5000},
		})
		for i := 0; i < 200; i++ {
			got := LookupModel("neither")
			if got.MaxOutput != 2000 {
				t.Fatalf("run %d: MaxOutput = %d, want 2000", i, got.MaxOutput)
			}
		}
	})
}

// TestCostStateStrings pins the three strings the TUI receives.
func TestCostStateStrings(t *testing.T) {
	for _, tc := range []struct {
		p    Pricing
		want string
	}{
		{PricingPerToken, "per_token"},
		{PricingFlat, "flat"},
		{PricingUnknown, "unknown"},
	} {
		if got := CostState(tc.p); got != tc.want {
			t.Errorf("CostState(%v) = %q, want %q", tc.p, got, tc.want)
		}
	}
}
