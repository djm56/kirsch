package session

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/agent"
)

// TestRepairTruncatesTornFinalLine verifies that a real session file with a
// torn final line is repaired when opened.
//
// The damage is inflicted on a file written by the Store, mid-line, as required
// by the m4-d2 ground rule.
func TestRepairTruncatesTornFinalLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")

	s, err := New(path, WithCadence(time.Hour))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	s.Record(agent.RecordAppend, userText("first"))
	s.Record(agent.RecordAppend, userText("second"))
	s.Record(agent.RecordAppend, userText("third"))
	if err := s.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Confirm the store wrote three complete lines.
	before := readLines(t, path)
	if len(before) != 3 {
		t.Fatalf("expected 3 lines before damage, got %d", len(before))
	}

	// Truncate mid-way through the final line. Strip the trailing newline first
	// so the last newline in the buffer is the end of the second-to-last line.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read session file: %v", err)
	}
	trimmed := bytes.TrimRight(data, "\n")
	lastNewline := bytes.LastIndexByte(trimmed, '\n')
	if lastNewline < 0 {
		t.Fatal("expected at least one newline in the session file")
	}
	tornOffset := int64(lastNewline + 5)
	if tornOffset >= int64(len(data)) {
		t.Fatal("test setup produced an offset past the end of the file")
	}
	if err := os.Truncate(path, tornOffset); err != nil {
		t.Fatalf("truncate session file: %v", err)
	}

	// Re-open the store: it must recover silently.
	s, err = New(path, WithCadence(time.Hour))
	if err != nil {
		t.Fatalf("New on torn session: %v", err)
	}
	if !s.Recovered() {
		t.Fatal("expected store to report recovered")
	}

	// The recovered file must contain exactly the first two good lines.
	after := readLines(t, path)
	if len(after) != 2 {
		t.Fatalf("expected 2 lines after repair, got %d", len(after))
	}
	if after[1]["content"] != "second" {
		t.Fatalf("last good line = %v, want second", after[1]["content"])
	}

	// The store must be able to append after recovery.
	s.Record(agent.RecordAppend, userText("fourth"))
	if err := s.Flush(); err != nil {
		t.Fatalf("Flush after recovery: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close after recovery: %v", err)
	}

	final := readLines(t, path)
	if len(final) != 3 {
		t.Fatalf("expected 3 lines after append, got %d", len(final))
	}
	if final[2]["content"] != "fourth" {
		t.Fatalf("appended line = %v, want fourth", final[2]["content"])
	}
}

// TestRepairNoCorruption verifies that a valid session file is not altered.
func TestRepairNoCorruption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	s, err := New(path, WithCadence(time.Hour))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	s.Record(agent.RecordAppend, userText("intact"))
	if err := s.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	s, err = New(path, WithCadence(time.Hour))
	if err != nil {
		t.Fatalf("New on intact session: %v", err)
	}
	if s.Recovered() {
		t.Fatal("expected no recovery for an intact file")
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	lines := readLines(t, path)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
}

// TestRepairEmptyFile verifies that an empty session file opens cleanly.
func TestRepairEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(path, []byte{}, 0o600); err != nil {
		t.Fatalf("create empty file: %v", err)
	}

	s, err := New(path, WithCadence(time.Hour))
	if err != nil {
		t.Fatalf("New on empty session: %v", err)
	}
	if s.Recovered() {
		t.Fatal("expected no recovery for an empty file")
	}

	s.Record(agent.RecordAppend, userText("after empty"))
	if err := s.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	lines := readLines(t, path)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
}
