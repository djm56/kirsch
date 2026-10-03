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
}

// models holds the known model information.
// Figures sourced from published model documentation, read 2026-10-02.
// Cache read pricing: 10% of input price (5% on claude-opus-5-5, 2.5% on claude-fable-5-1).
var models = map[string]ModelInfo{
	"claude-sonnet-5-5": {
		ID:               "claude-sonnet-5-5",
		Known:            true,
		ContextWindow:    1000000,
		MaxOutput:        128000,
		Pricing:          PricingPerToken,
		InputPerMTok:     2.0,
		OutputPerMTok:    10.0,
		CacheReadPerMTok: 0.2,
	},
	"claude-opus-5-5": {
		ID:               "claude-opus-5-5",
		Known:            true,
		ContextWindow:    1000000,
		MaxOutput:        128000,
		Pricing:          PricingPerToken,
		InputPerMTok:     4.0,
		OutputPerMTok:    20.0,
		CacheReadPerMTok: 0.2,
	},
	"claude-fable-5-1": {
		ID:               "claude-fable-5-1",
		Known:            true,
		ContextWindow:    1000000,
		MaxOutput:        128000,
		Pricing:          PricingPerToken,
		InputPerMTok:     10.0,
		OutputPerMTok:    50.0,
		CacheReadPerMTok: 0.25,
	},
	"claude-haiku-4-5-20251001": {
		ID:               "claude-haiku-4-5-20251001",
		Known:            true,
		ContextWindow:    200000,
		MaxOutput:        64000,
		Pricing:          PricingPerToken,
		InputPerMTok:     1.0,
		OutputPerMTok:    5.0,
		CacheReadPerMTok: 0.1,
	},
	"minimax-m3": {
		ID:            "minimax-m3",
		Known:         true,
		ContextWindow: 128000, // fallback
		MaxOutput:     4096,   // fallback
		Pricing:       PricingFlat,
	},
}

// LookupModel returns information about a model ID.
// If the model is not known, it returns the fallback with Known: false.
func LookupModel(id string) ModelInfo {
	if m, ok := models[id]; ok {
		return m
	}
	// Fallback for unknown ID
	return ModelInfo{
		ID:            id,
		Known:         false,
		ContextWindow: 128000,
		MaxOutput:     4096,
		Pricing:       PricingUnknown,
	}
}
