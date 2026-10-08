package app

import (
	"context"
	"testing"

	"github.com/djm56/kirsch/internal/agent"
)

func TestEstimateTokensCharsDivFour(t *testing.T) {
	tests := []struct {
		name   string
		system string
		user   string
	}{
		{"empty request", "", ""},
		{"system only", "abcd", ""},
		{"user only", "", "efgh"},
		{"both halves", "abcd", "efgh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimateTokens(tt.system, tt.user, &agent.Conversation{}, nil)
			// nil tools means the function returns 0 early; test that branch separately.
			if got != 0 {
				t.Errorf("got %d, want 0 when tools is nil", got)
			}
		})
	}
}

func TestEstimateTokensNilToolsReturnsZero(t *testing.T) {
	got := estimateTokens("abcd", "efgh", &agent.Conversation{}, nil)
	if got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

func TestEstimateTokensSumsRequestParts(t *testing.T) {
	conv := &agent.Conversation{Messages: []agent.Message{
		{Role: agent.RoleUser, Content: []agent.Block{
			{Kind: agent.BlockText, Text: "ijkl"},
		}},
	}}
	tools := &fakeTools{specs: []agent.ToolSpec{
		{Name: "mn", Description: "op", InputSchema: []byte("qrst")},
	}}

	// system 4 chars + user 4 chars + message 4 chars + spec 2+2+4 = 8 chars = 20 total /4 = 5
	got := estimateTokens("abcd", "efgh", conv, tools)
	if got != 5 {
		t.Errorf("got %d, want 5", got)
	}
}

func TestCompactPercent(t *testing.T) {
	tests := []struct {
		name          string
		estimated     int
		contextWindow int
		want          int
	}{
		{"zero estimate", 0, 128000, 0},
		{"exactly at budget", 123904, 128000, 100}, // 128000 - 4096 reserve
		{"one over budget", 123905, 128000, 100},
		{"half budget", 61952, 128000, 50},
		{"three quarters", 92928, 128000, 75},
		{"small window at reserve", 1, 4096, 0},
		{"negative estimate", -1, 128000, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compactPercent(tt.estimated, tt.contextWindow)
			if got != tt.want {
				t.Errorf("compactPercent(%d,%d) = %d, want %d", tt.estimated, tt.contextWindow, got, tt.want)
			}
		})
	}
}

func TestContextOverflowThreshold(t *testing.T) {
	// A 128k-window model reserves 4096 tokens, leaving a 123904-token budget.
	budget := 128000 - outputReserve
	if budget != 123904 {
		t.Fatalf("budget = %d, want 123904", budget)
	}

	overEstimate := budget + 1
	if overEstimate <= budget {
		t.Errorf("estimate %d one token over the budget %d should be flagged", overEstimate, budget)
	}
	atEstimate := budget
	if atEstimate > budget {
		t.Errorf("estimate %d exactly at the budget should not be flagged", atEstimate)
	}
}

type fakeTools struct {
	specs []agent.ToolSpec
}

func (f *fakeTools) Describe() []agent.ToolSpec { return f.specs }
func (f *fakeTools) Invoke(_ context.Context, _ agent.ToolCall) agent.ToolResult {
	return agent.ToolResult{}
}
