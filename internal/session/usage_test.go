package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestReadUsageMissing verifies that a missing usage file is treated as zero.
func TestReadUsageMissing(t *testing.T) {
	root := t.TempDir()
	u, err := ReadUsage(root)
	if err != nil {
		t.Fatalf("ReadUsage: %v", err)
	}
	if u == nil {
		t.Fatal("expected non-nil usage")
	}
	if u.PromptTokens != 0 || u.CompletionTokens != 0 || u.TotalTokens != 0 {
		t.Fatalf("expected zero usage, got %+v", u)
	}
}

// TestWriteAndReadUsage verifies a round-trip through usage.json.
func TestWriteAndReadUsage(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	u := &Usage{
		PromptTokens:     10,
		CompletionTokens: 20,
		TotalTokens:      30,
		UpdatedAt:        now,
	}

	if err := WriteUsage(root, u); err != nil {
		t.Fatalf("WriteUsage: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "usage.json")); err != nil {
		t.Fatalf("usage.json not created: %v", err)
	}

	got, err := ReadUsage(root)
	if err != nil {
		t.Fatalf("ReadUsage: %v", err)
	}
	if got.PromptTokens != 10 {
		t.Fatalf("PromptTokens = %d, want 10", got.PromptTokens)
	}
	if got.CompletionTokens != 20 {
		t.Fatalf("CompletionTokens = %d, want 20", got.CompletionTokens)
	}
	if got.TotalTokens != 30 {
		t.Fatalf("TotalTokens = %d, want 30", got.TotalTokens)
	}
	if !got.UpdatedAt.Equal(now) {
		t.Fatalf("UpdatedAt = %v, want %v", got.UpdatedAt, now)
	}
}

// TestUsageAddUpdatesTotals verifies cumulative token accounting.
func TestUsageAddUpdatesTotals(t *testing.T) {
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	u := &Usage{}
	u.Add(10, 5, now)
	if u.PromptTokens != 10 {
		t.Fatalf("PromptTokens = %d, want 10", u.PromptTokens)
	}
	if u.CompletionTokens != 5 {
		t.Fatalf("CompletionTokens = %d, want 5", u.CompletionTokens)
	}
	if u.TotalTokens != 15 {
		t.Fatalf("TotalTokens = %d, want 15", u.TotalTokens)
	}
	if !u.UpdatedAt.Equal(now) {
		t.Fatalf("UpdatedAt not set")
	}

	u.Add(2, 3, now.Add(time.Hour))
	if u.TotalTokens != 20 {
		t.Fatalf("TotalTokens = %d, want 20", u.TotalTokens)
	}
}

// TestAddUsageRootUpdatesFile verifies the full update path from a synthetic
// usage event.
func TestAddUsageRootUpdatesFile(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	if err := AddUsage(root, 100, 50, now); err != nil {
		t.Fatalf("AddUsage: %v", err)
	}
	if err := AddUsage(root, 25, 10, now.Add(time.Hour)); err != nil {
		t.Fatalf("AddUsage: %v", err)
	}

	got, err := ReadUsage(root)
	if err != nil {
		t.Fatalf("ReadUsage: %v", err)
	}
	if got.PromptTokens != 125 {
		t.Fatalf("PromptTokens = %d, want 125", got.PromptTokens)
	}
	if got.CompletionTokens != 60 {
		t.Fatalf("CompletionTokens = %d, want 60", got.CompletionTokens)
	}
	if got.TotalTokens != 185 {
		t.Fatalf("TotalTokens = %d, want 185", got.TotalTokens)
	}
}
