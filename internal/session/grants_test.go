package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLoadGrantsReadsScopeGrantedEvents verifies that LoadGrants returns the
// action/pattern pairs recorded by Store.RecordScopeGranted.
func TestLoadGrantsReadsScopeGrantedEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	s, err := New(path, WithCadence(time.Hour))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	s.RecordScopeGranted("run_command", "go test ./...")
	s.RecordScopeGranted("run_command", "git status")
	if err := s.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	grants, err := LoadGrants(path)
	if err != nil {
		t.Fatalf("LoadGrants: %v", err)
	}
	if len(grants) != 2 {
		t.Fatalf("grants = %d, want 2", len(grants))
	}
	if grants[0].Action != "run_command" || grants[0].Pattern != "go test ./..." {
		t.Errorf("grant[0] = %+v, want run_command go test ./...", grants[0])
	}
	if grants[1].Action != "run_command" || grants[1].Pattern != "git status" {
		t.Errorf("grant[1] = %+v, want run_command git status", grants[1])
	}
}

// TestLoadGrantsStopsAtTornLine verifies that a corrupt/truncated line ends
// grant loading, matching the repair behaviour of Store.New and Resume.
func TestLoadGrantsStopsAtTornLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	s, err := New(path, WithCadence(time.Hour))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	s.RecordScopeGranted("run_command", "go test ./...")
	if err := s.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Append a torn line that cannot be parsed as JSON.
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open for append: %v", err)
	}
	if _, err := f.WriteString("{\"type\":\"approval.scope_granted\"\n"); err != nil {
		_ = f.Close()
		t.Fatalf("append torn line: %v", err)
	}
	_ = f.Close()

	grants, err := LoadGrants(path)
	if err != nil {
		t.Fatalf("LoadGrants: %v", err)
	}
	if len(grants) != 1 {
		t.Fatalf("grants = %d, want 1", len(grants))
	}
	if grants[0].Pattern != "go test ./..." {
		t.Errorf("grant[0] = %+v", grants[0])
	}
}

// TestLoadGrantsEmptyFile verifies that an empty session file yields an empty
// grant list without error.
func TestLoadGrantsEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	grants, err := LoadGrants(path)
	if err != nil {
		t.Fatalf("LoadGrants: %v", err)
	}
	if len(grants) != 0 {
		t.Errorf("grants = %d, want 0", len(grants))
	}
}

// TestLoadGrantsMissingFile verifies that a missing session file surfaces the
// filesystem error to the caller.
func TestLoadGrantsMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.jsonl")
	if _, err := LoadGrants(path); err == nil {
		t.Fatal("LoadGrants missing file: expected error")
	}
}

// writeGrantLog records the given steps into a fresh session log and returns
// its path. A step is either "grant:<pattern>" or "clear".
func writeGrantLog(t *testing.T, steps ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "session.jsonl")
	s, err := New(path, WithCadence(time.Hour))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for _, step := range steps {
		if step == "clear" {
			s.RecordScopesCleared()
		} else {
			s.RecordScopeGranted("run_command", strings.TrimPrefix(step, "grant:"))
		}
	}
	if err := s.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return path
}

// TestLoadGrantsClearThenGrantKeepsOnlyLater verifies that a clear event
// discards earlier grants while a grant recorded after it still counts.
func TestLoadGrantsClearThenGrantKeepsOnlyLater(t *testing.T) {
	path := writeGrantLog(t, "grant:go test", "clear", "grant:git status")
	grants, err := LoadGrants(path)
	if err != nil {
		t.Fatalf("LoadGrants: %v", err)
	}
	if len(grants) != 1 || grants[0].Pattern != "git status" {
		t.Fatalf("grants = %+v, want only [git status]", grants)
	}
}

// TestLoadGrantsClearLeavesNone verifies that grant followed by clear yields
// no grants.
func TestLoadGrantsClearLeavesNone(t *testing.T) {
	path := writeGrantLog(t, "grant:go test", "grant:git status", "clear")
	grants, err := LoadGrants(path)
	if err != nil {
		t.Fatalf("LoadGrants: %v", err)
	}
	if len(grants) != 0 {
		t.Fatalf("grants = %+v, want none", grants)
	}
}

// TestLoadGrantsWithoutClearEventUnchanged is the backward-compatibility case:
// a log written before the clear event existed restores every grant.
func TestLoadGrantsWithoutClearEventUnchanged(t *testing.T) {
	path := writeGrantLog(t, "grant:go test", "grant:git status")
	grants, err := LoadGrants(path)
	if err != nil {
		t.Fatalf("LoadGrants: %v", err)
	}
	if len(grants) != 2 || grants[0].Pattern != "go test" || grants[1].Pattern != "git status" {
		t.Fatalf("grants = %+v, want [go test, git status]", grants)
	}
}

// TestRecordScopesClearedWritesEvent verifies the line written to the log has
// the approval.scopes_cleared type.
func TestRecordScopesClearedWritesEvent(t *testing.T) {
	path := writeGrantLog(t, "clear")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"type":"approval.scopes_cleared"`) {
		t.Fatalf("log lacks approval.scopes_cleared event:\n%s", data)
	}
}
