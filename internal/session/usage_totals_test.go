package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/agent"
)

// writeUsageEvents writes a session log through Store holding two grant events
// (non-usage events the sum must skip), one before and one after the given
// usage events, and returns its path.
func writeUsageEvents(t *testing.T, usages ...agent.Usage) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "session.jsonl")
	s, err := New(path, WithCadence(time.Hour))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	s.RecordScopeGranted("run_command", "go test ./...")
	for _, u := range usages {
		s.RecordUsage(u)
	}
	s.RecordScopeGranted("run_command", "git status")
	if err := s.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return path
}

// appendRaw appends s to the file at path.
func appendRaw(t *testing.T, path, s string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(s); err != nil {
		t.Fatal(err)
	}
}

// TestLoadUsageTotalsSumsUsageEvents writes three usage events through
// Store.RecordUsage and checks each of the four fields of the sum. Every field
// gets a distinct value, so a crossed field fails.
func TestLoadUsageTotalsSumsUsageEvents(t *testing.T) {
	path := writeUsageEvents(t,
		agent.Usage{InputTokens: 1000, OutputTokens: 200, CacheReadTokens: 30, CacheWriteTokens: 7},
		agent.Usage{InputTokens: 2000, OutputTokens: 300},
		agent.Usage{InputTokens: 500, OutputTokens: 50, CacheReadTokens: 10, CacheWriteTokens: 5},
	)

	got, err := LoadUsageTotals(path)
	if err != nil {
		t.Fatalf("LoadUsageTotals: %v", err)
	}
	want := UsageTotals{InputTokens: 3500, OutputTokens: 550, CacheReadTokens: 40, CacheWriteTokens: 12}
	if got != want {
		t.Errorf("totals = %+v, want %+v", got, want)
	}
}

// TestLoadUsageTotalsNoUsageEvents reads a log with other events and no usage
// event. The zero sum, with the log holding two grant events, is the control
// that the loader skips events that are not usage.
func TestLoadUsageTotalsNoUsageEvents(t *testing.T) {
	path := writeUsageEvents(t)

	got, err := LoadUsageTotals(path)
	if err != nil {
		t.Fatalf("LoadUsageTotals: %v", err)
	}
	if got != (UsageTotals{}) {
		t.Errorf("totals = %+v, want zero", got)
	}
}

// TestLoadUsageTotalsStopsAtTornLine appends a torn line and then a whole usage
// event. The event before the tear counts; the one after does not.
func TestLoadUsageTotalsStopsAtTornLine(t *testing.T) {
	path := writeUsageEvents(t, agent.Usage{InputTokens: 10, OutputTokens: 5})
	appendRaw(t, path, "{\"v\":1,\"type\":\"usage\",\"input_to\n"+
		`{"v":1,"type":"usage","input_tokens":900,"output_tokens":900,"cache_read":0,"cache_write":0}`+"\n")

	got, err := LoadUsageTotals(path)
	if err != nil {
		t.Fatalf("LoadUsageTotals: %v", err)
	}
	want := UsageTotals{InputTokens: 10, OutputTokens: 5}
	if got != want {
		t.Errorf("totals = %+v, want %+v", got, want)
	}
}

// TestLoadUsageTotalsStopsAtUndecodableUsageEvent appends a usage event whose
// input_tokens is a string, which is valid JSON but does not decode, then a
// whole usage event. The event before counts; the malformed one and the one
// after do not.
func TestLoadUsageTotalsStopsAtUndecodableUsageEvent(t *testing.T) {
	path := writeUsageEvents(t, agent.Usage{InputTokens: 10, OutputTokens: 5})
	appendRaw(t, path, `{"v":1,"type":"usage","input_tokens":"many","output_tokens":900}`+"\n"+
		`{"v":1,"type":"usage","input_tokens":900,"output_tokens":900,"cache_read":0,"cache_write":0}`+"\n")

	got, err := LoadUsageTotals(path)
	if err != nil {
		t.Fatalf("LoadUsageTotals: %v", err)
	}
	want := UsageTotals{InputTokens: 10, OutputTokens: 5}
	if got != want {
		t.Errorf("totals = %+v, want %+v", got, want)
	}
}

// TestLoadUsageTotalsMissingFile checks that a missing log is an error, not a
// silent zero.
func TestLoadUsageTotalsMissingFile(t *testing.T) {
	if _, err := LoadUsageTotals(filepath.Join(t.TempDir(), "nope.jsonl")); err == nil {
		t.Fatal("LoadUsageTotals on a missing file returned nil error")
	}
}
