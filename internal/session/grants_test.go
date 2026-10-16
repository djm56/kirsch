package session

import (
	"os"
	"path/filepath"
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
