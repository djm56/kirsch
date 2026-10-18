package provider

// Pricing identifies the pricing model for a model.
type Pricing int

const (
	// PricingUnknown: no figures, so cost displays as ?.
	PricingUnknown Pricing = iota
	// PricingPerToken: USD per million tokens, from the table.
	PricingPerToken
	// PricingFlat: a subscription, so the UI shows usage, not money.
	PricingFlat
)

// ModelInfo holds information about a known model.
type ModelInfo struct {
	// ID is the model identifier.
	ID string
	// Known is true if this model's details are known; false for unknown models.
	Known bool
	// ContextWindow is the maximum context window size.
	ContextWindow int
	// MaxOutput is the maximum output tokens.
	MaxOutput int
	// Pricing is the pricing model for this model.
	Pricing Pricing
	// InputPerMTok is the input cost in USD per million tokens (PricingPerToken only).
	InputPerMTok float64
	// OutputPerMTok is the output cost in USD per million tokens (PricingPerToken only).
	OutputPerMTok float64
	// CacheReadPerMTok is the cache read cost in USD per million tokens (PricingPerToken only).
	CacheReadPerMTok float64
	// CacheWritePerMTok is the cache write cost in USD per million tokens (PricingPerToken only).
	// DERIVED from the published 5-minute cache-write multiplier (1.25× input), not read from
	// a price sheet. Unverified: confirm against current pricing before relying on cost_usd.
	CacheWritePerMTok float64
}

// models holds the known model information.
// Figures sourced from published model documentation, read 2026-10-02.
// Cache read pricing: 10% of input price (5% on claude-opus-5-5, 2.5% on claude-fable-5-1).
var models = map[string]ModelInfo{
	"claude-sonnet-5-5": {
		ID:                "claude-sonnet-5-5",
		Known:             true,
		ContextWindow:     1000000,
		MaxOutput:         128000,
		Pricing:           PricingPerToken,
		InputPerMTok:      2.0,
		OutputPerMTok:     10.0,
		CacheReadPerMTok:  0.2,
		CacheWritePerMTok: 2.5,
	},
	"claude-opus-5-5": {
		ID:                "claude-opus-5-5",
		Known:             true,
		ContextWindow:     1000000,
		MaxOutput:         128000,
		Pricing:           PricingPerToken,
		InputPerMTok:      4.0,
		OutputPerMTok:     20.0,
		CacheReadPerMTok:  0.2,
		CacheWritePerMTok: 5.0,
	},
	"claude-fable-5-1": {
		ID:                "claude-fable-5-1",
		Known:             true,
		ContextWindow:     1000000,
		MaxOutput:         128000,
		Pricing:           PricingPerToken,
		InputPerMTok:      10.0,
		OutputPerMTok:     50.0,
		CacheReadPerMTok:  0.25,
		CacheWritePerMTok: 12.5,
	},
	"claude-haiku-4-5-20251001": {
		ID:                "claude-haiku-4-5-20251001",
		Known:             true,
		ContextWindow:     200000,
		MaxOutput:         64000,
		Pricing:           PricingPerToken,
		InputPerMTok:      1.0,
		OutputPerMTok:     5.0,
		CacheReadPerMTok:  0.1,
		CacheWritePerMTok: 1.25,
	},
	"minimax-m3": {
		ID:            "minimax-m3",
		Known:         true,
		ContextWindow: 128000, // fallback
		MaxOutput:     4096,   // fallback
		Pricing:       PricingFlat,
	},
	"minimax-m2.7": {
		ID:            "minimax-m2.7",
		Known:         true,
		ContextWindow: 128000, // fallback
		MaxOutput:     4096,   // fallback
		Pricing:       PricingFlat,
	},
}

// LookupModel returns information about a model ID.
// If the model is not known, it returns the fallback with Known: false.
// The fallback uses the conservative budget (smallest known context window).
func LookupModel(id string) ModelInfo {
	if m, ok := models[id]; ok {
		return m
	}
	// Fallback for unknown ID: compute conservative budget from the table (m4-d9).
	window, maxOut := conservativeLimits()
	return ModelInfo{
		ID:            id,
		Known:         false,
		ContextWindow: window,
		MaxOutput:     maxOut,
		Pricing:       PricingUnknown,
	}
}

// Cost computes the USD cost for the given token counts.
// For PricingPerToken models, it returns the sum of each token count divided by 1e6
// and multiplied by the respective price. For PricingFlat and PricingUnknown models,
// it returns 0 and false.
func (m ModelInfo) Cost(in, out, cacheRead, cacheWrite int64) (usd float64, ok bool) {
	if m.Pricing != PricingPerToken {
		return 0, false
	}
	return (float64(in)*m.InputPerMTok + float64(out)*m.OutputPerMTok +
		float64(cacheRead)*m.CacheReadPerMTok + float64(cacheWrite)*m.CacheWritePerMTok) / 1e6, true
}

// conservativeLimits returns the context window and maximum output of the most
// constrained row in the model table, from one scan. The seed is the first row
// visited rather than a sentinel, so no row value can sit above it. Rows tie on
// the window by taking the smaller MaxOutput, which makes the result independent
// of map iteration order. An empty table returns 0, 0.
func conservativeLimits() (window, maxOut int) {
	first := true
	for _, m := range models {
		switch {
		case first,
			m.ContextWindow < window,
			m.ContextWindow == window && m.MaxOutput < maxOut:
			window, maxOut = m.ContextWindow, m.MaxOutput
			first = false
		}
	}
	return window, maxOut
}

// ConservativeBudget returns the smallest context window among all rows of the
// model table. LookupModel uses it for a model the table does not know (m4-d9).
func ConservativeBudget() int {
	w, _ := conservativeLimits()
	return w
}

// ConservativeMaxOutput returns the smallest MaxOutput among the table rows that
// share the smallest context window.
func ConservativeMaxOutput() int {
	_, o := conservativeLimits()
	return o
}

// CostState returns the string form of a Pricing value that the TUI receives in
// SessionUsageMsg.CostState: "per_token", "flat", or "unknown" for any other value.
func CostState(p Pricing) string {
	switch p {
	case PricingPerToken:
		return "per_token"
	case PricingFlat:
		return "flat"
	default:
		return "unknown"
	}
}
