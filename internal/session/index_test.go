package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestReadIndexMissing verifies that a missing index is treated as empty.
func TestReadIndexMissing(t *testing.T) {
	root := t.TempDir()
	idx, err := ReadIndex(root)
	if err != nil {
		t.Fatalf("ReadIndex: %v", err)
	}
	if idx == nil {
		t.Fatal("expected non-nil index")
	}
	if len(idx.Sessions) != 0 {
		t.Fatalf("expected empty sessions, got %d", len(idx.Sessions))
	}
}

// TestWriteAndReadIndex verifies a round-trip through index.json.
func TestWriteAndReadIndex(t *testing.T) {
	root := t.TempDir()
	idx := &Index{Sessions: make(map[string]IndexEntry)}
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	idx.Set("/home/alice/kirsch", "20261014T120000-01ARZ3NDEKTSV4RRFFQ69G5FAV", now)

	if err := WriteIndex(root, idx); err != nil {
		t.Fatalf("WriteIndex: %v", err)
	}

	got, err := ReadIndex(root)
	if err != nil {
		t.Fatalf("ReadIndex: %v", err)
	}
	entry, ok := got.Sessions["/home/alice/kirsch"]
	if !ok {
		t.Fatal("workspace missing from round-tripped index")
	}
	if entry.LastSessionID != "20261014T120000-01ARZ3NDEKTSV4RRFFQ69G5FAV" {
		t.Fatalf("LastSessionID = %q", entry.LastSessionID)
	}
	if !entry.UpdatedAt.Equal(now) {
		t.Fatalf("UpdatedAt = %v, want %v", entry.UpdatedAt, now)
	}

	if _, err := os.Stat(filepath.Join(root, "index.json")); err != nil {
		t.Fatalf("index.json not created: %v", err)
	}
}

// TestIndexSetUpdatesEntry verifies that Set replaces an existing entry.
func TestIndexSetUpdatesEntry(t *testing.T) {
	idx := &Index{Sessions: make(map[string]IndexEntry)}
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	idx.Set("/ws", "id-1", now)
	idx.Set("/ws", "id-2", now.Add(time.Hour))

	entry := idx.Sessions["/ws"]
	if entry.LastSessionID != "id-2" {
		t.Fatalf("LastSessionID = %q, want id-2", entry.LastSessionID)
	}
	if !entry.UpdatedAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("UpdatedAt not updated")
	}
}

// TestIndexMultipleWorkspaces verifies independent entries per workspace.
func TestIndexMultipleWorkspaces(t *testing.T) {
	root := t.TempDir()
	idx := &Index{Sessions: make(map[string]IndexEntry)}
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	idx.Set("/ws/a", "id-a", now)
	idx.Set("/ws/b", "id-b", now)

	if err := WriteIndex(root, idx); err != nil {
		t.Fatalf("WriteIndex: %v", err)
	}
	got, err := ReadIndex(root)
	if err != nil {
		t.Fatalf("ReadIndex: %v", err)
	}
	if got.Sessions["/ws/a"].LastSessionID != "id-a" {
		t.Fatal("workspace a entry corrupted")
	}
	if got.Sessions["/ws/b"].LastSessionID != "id-b" {
		t.Fatal("workspace b entry corrupted")
	}
}
