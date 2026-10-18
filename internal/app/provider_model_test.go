package app

import (
	"testing"

	"github.com/djm56/kirsch/internal/agent"
	"github.com/djm56/kirsch/internal/provider"
)

// TestProviderEventToAgentMapsUsage calls providerEventToAgent directly with
// provider events and checks how usage crosses into agent.Event. Every count
// differs from the others, so a dropped or swapped field fails. The subtests
// that expect no usage are the mirror of the first: usage is carried only by
// MessageDone, and only when the provider supplied it.
func TestProviderEventToAgentMapsUsage(t *testing.T) {
	u := provider.Usage{InputTokens: 11, OutputTokens: 22, CacheReadTokens: 33, CacheWriteTokens: 44}

	t.Run("MessageDone carries all four counts", func(t *testing.T) {
		ev := providerEventToAgent(provider.StreamEvent{Type: provider.EventMessageDone, Usage: &u})
		want := agent.Usage{InputTokens: 11, OutputTokens: 22, CacheReadTokens: 33, CacheWriteTokens: 44}
		if ev.Usage == nil || *ev.Usage != want {
			t.Errorf("MessageDone with usage: agent usage = %+v, want %+v", ev.Usage, want)
		}
	})

	t.Run("MessageDone without usage stays nil", func(t *testing.T) {
		ev := providerEventToAgent(provider.StreamEvent{Type: provider.EventMessageDone})
		if ev.Usage != nil {
			t.Errorf("MessageDone without usage: agent usage = %+v, want nil", ev.Usage)
		}
	})

	t.Run("a text delta never carries usage", func(t *testing.T) {
		ev := providerEventToAgent(provider.StreamEvent{Type: provider.EventTextDelta, Text: "x", Usage: &u})
		if ev.Usage != nil {
			t.Errorf("text delta: agent usage = %+v, want nil", ev.Usage)
		}
	})
}
