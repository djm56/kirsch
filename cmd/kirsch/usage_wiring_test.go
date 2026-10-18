package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/djm56/kirsch/internal/app"
	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

// usageSSE is a complete Anthropic stream whose usage is 1000 input, 300 output,
// 8000 cache read and 2000 cache write tokens: 0.0116 USD on claude-sonnet-5-5.
const usageSSE = "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":" +
	"{\"input_tokens\":1000,\"cache_read_input_tokens\":8000,\"cache_creation_input_tokens\":2000,\"output_tokens\":0}}}\n\n" +
	"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n" +
	"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"hello back\"}}\n\n" +
	"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n" +
	"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":300}}\n\n" +
	"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"

// sink collects what the app sends, in arrival order.
type sink struct {
	mu   sync.Mutex
	msgs []any
}

func (s *sink) send(m any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgs = append(s.msgs, m)
}

func (s *sink) snapshot() []any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]any(nil), s.msgs...)
}

// wiredUsageFrame submits a prompt through a model wired by wireCallbacks over
// a real App whose provider is the real Anthropic client talking to a local
// server that serves usageSSE, feeds every message the app sends into the
// model, and returns the rendered frame. model is the endpoint's model id.
func wiredUsageFrame(t *testing.T, model string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, usageSSE)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("KIRSCH_USAGE_WIRING_KEY", "test-key")

	ws, err := workspace.Detect(t.TempDir())
	if err != nil {
		t.Fatalf("workspace.Detect: %v", err)
	}
	cfg := config.Defaults()
	cfg.Provider.Default = "wiring"
	cfg.Provider.Endpoints["wiring"] = config.EndpointConfig{
		BaseURL:   srv.URL,
		Auth:      "x-api-key",
		APIKeyEnv: "KIRSCH_USAGE_WIRING_KEY",
		Model:     model,
	}
	cfg.Session.StorageDir = filepath.Join(t.TempDir(), "sessions")
	log := telemetry.Disabled()
	a := app.New(ws, cfg, log)
	t.Cleanup(func() { a.Close() })
	s := &sink{}
	a.SetVersion("0.1.0-test") // the provider client requires one for its User-Agent
	a.SetSendFn(s.send)

	m := tui.New(tui.Options{Version: "0.1.0-test", Caps: tui.Caps{Colour: false, Unicode: true}})
	wireCallbacks(&m, a, log)
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 140, Height: 40})

	for _, r := range "hello" {
		tm, _ = tm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	tm, _ = tm.Update(tea.KeyMsg{Type: tea.KeyEnter})

	deadline := time.Now().Add(10 * time.Second)
	for {
		var complete, usage bool
		for _, msg := range s.snapshot() {
			switch msg.(type) {
			case tui.TurnCompleteMsg:
				complete = true
			case tui.SessionUsageMsg:
				usage = true
			case tui.TurnErrorMsg:
				t.Fatalf("turn failed: %#v", msg)
			}
		}
		if complete && usage {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("no usage and completion after 10s; got %#v", s.snapshot())
		}
		time.Sleep(5 * time.Millisecond)
	}
	for _, msg := range s.snapshot() {
		tm, _ = tm.Update(msg)
	}
	return tm.View()
}

// TestUsageEventThroughRealWiringReachesTheStatusBar runs one turn from the
// composer keypress to the rendered frame: wireCallbacks' Submit, the real app
// and Anthropic client, the usage event in the stream, recordUsage,
// SessionUsageMsg, Update and the status row. Each subtest names its model; the
// per-token model is the control that the same stream does render totals and a
// cost, so the flat and unknown results are not an empty frame.
func TestUsageEventThroughRealWiringReachesTheStatusBar(t *testing.T) {
	const totals = "in 1.0k out 300 cache 8.0k/2.0k"
	const notice = "cost display unavailable, conservative budget in use"

	t.Run("per-token model shows totals and cost, no notice", func(t *testing.T) {
		view := wiredUsageFrame(t, "claude-sonnet-5-5")
		if !strings.Contains(view, totals+" · $0.01") {
			t.Errorf("status bar lacks %q:\n%s", totals+" · $0.01", view)
		}
		if strings.Contains(view, notice) {
			t.Errorf("notice shown for a known model:\n%s", view)
		}
	})

	t.Run("flat model shows totals and no money", func(t *testing.T) {
		view := wiredUsageFrame(t, "minimax-m2.7")
		if !strings.Contains(view, totals) {
			t.Errorf("status bar lacks %q:\n%s", totals, view)
		}
		if strings.Contains(view, "$") {
			t.Errorf("flat model frame shows a dollar sign:\n%s", view)
		}
		if strings.Contains(view, notice) {
			t.Errorf("notice shown for a known model:\n%s", view)
		}
	})

	t.Run("unlisted model shows $? and the notice once", func(t *testing.T) {
		view := wiredUsageFrame(t, "mystery-model-1")
		if !strings.Contains(view, totals+" · $?") {
			t.Errorf("status bar lacks %q:\n%s", totals+" · $?", view)
		}
		if n := strings.Count(view, notice); n != 1 {
			t.Errorf("notice shown %d times, want 1:\n%s", n, view)
		}
	})
}
