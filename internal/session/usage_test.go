package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// absFloat returns the absolute value of a float64.
func absFloat(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

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
	u.Add(10, 5, 2, 1, 0.0005, true, now)
	if u.PromptTokens != 10 {
		t.Fatalf("PromptTokens = %d, want 10", u.PromptTokens)
	}
	if u.CompletionTokens != 5 {
		t.Fatalf("CompletionTokens = %d, want 5", u.CompletionTokens)
	}
	if u.CacheReadTokens != 2 {
		t.Fatalf("CacheReadTokens = %d, want 2", u.CacheReadTokens)
	}
	if u.CacheWriteTokens != 1 {
		t.Fatalf("CacheWriteTokens = %d, want 1", u.CacheWriteTokens)
	}
	if u.TotalTokens != 18 {
		t.Fatalf("TotalTokens = %d, want 18", u.TotalTokens)
	}
	if u.CostUSD != 0.0005 {
		t.Fatalf("CostUSD = %v, want 0.0005", u.CostUSD)
	}
	if !u.UpdatedAt.Equal(now) {
		t.Fatalf("UpdatedAt not set")
	}

	u.Add(2, 3, 1, 0, 0.0003, true, now.Add(time.Hour))
	if u.TotalTokens != 24 {
		t.Fatalf("TotalTokens = %d, want 24", u.TotalTokens)
	}
	if absFloat(u.CostUSD-0.0008) > 1e-10 {
		t.Fatalf("CostUSD = %v, want ≈ 0.0008", u.CostUSD)
	}
}

// TestAddUsageRootUpdatesFile verifies the full update path from a synthetic
// usage event.
func TestAddUsageRootUpdatesFile(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	if _, err := AddUsage(root, 100, 50, 10, 5, 0.001, true, now); err != nil {
		t.Fatalf("AddUsage: %v", err)
	}
	if _, err := AddUsage(root, 25, 10, 2, 1, 0.0002, true, now.Add(time.Hour)); err != nil {
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
	if got.CacheReadTokens != 12 {
		t.Fatalf("CacheReadTokens = %d, want 12", got.CacheReadTokens)
	}
	if got.CacheWriteTokens != 6 {
		t.Fatalf("CacheWriteTokens = %d, want 6", got.CacheWriteTokens)
	}
	if got.TotalTokens != 203 {
		t.Fatalf("TotalTokens = %d, want 203", got.TotalTokens)
	}
	if absFloat(got.CostUSD-0.0012) > 1e-10 {
		t.Fatalf("CostUSD = %v, want ≈ 0.0012", got.CostUSD)
	}
}

// TestUsageAccumulationWithUnknownCost verifies that tokens from unknown-cost events
// are accumulated in CostUnknownTokens.
func TestUsageAccumulationWithUnknownCost(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	// Add tokens with known cost
	if _, err := AddUsage(root, 100, 50, 10, 5, 0.001, true, now); err != nil {
		t.Fatalf("AddUsage: %v", err)
	}
	// Add tokens with unknown cost
	if _, err := AddUsage(root, 25, 10, 2, 1, 0, false, now.Add(time.Hour)); err != nil {
		t.Fatalf("AddUsage: %v", err)
	}

	// A second unknown-cost event: the counter must accumulate, not be replaced.
	if _, err := AddUsage(root, 7, 3, 1, 1, 0, false, now.Add(2*time.Hour)); err != nil {
		t.Fatalf("AddUsage: %v", err)
	}

	got, err := ReadUsage(root)
	if err != nil {
		t.Fatalf("ReadUsage: %v", err)
	}
	if got.CostUSD != 0.001 {
		t.Fatalf("CostUSD = %v, want 0.001 (only known costs)", got.CostUSD)
	}
	// CostUnknownTokens should be (25+10+2+1) + (7+3+1+1) = 38 + 12 = 50
	if got.CostUnknownTokens != 50 {
		t.Fatalf("CostUnknownTokens = %d, want 50", got.CostUnknownTokens)
	}
}

// TestUsageBackwardCompatibility verifies that old-format usage.json (without the new fields)
// still loads correctly with missing fields as zero.
func TestUsageBackwardCompatibility(t *testing.T) {
	root := t.TempDir()
	usagePath := filepath.Join(root, "usage.json")

	// Write an old-format usage.json with only the original three fields
	oldFormat := `{"prompt_tokens":100,"completion_tokens":50,"total_tokens":150,"updated_at":"2026-10-14T12:00:00Z"}`
	if err := os.WriteFile(usagePath, []byte(oldFormat), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Read should succeed and treat missing fields as zero
	got, err := ReadUsage(root)
	if err != nil {
		t.Fatalf("ReadUsage: %v", err)
	}
	if got.PromptTokens != 100 {
		t.Fatalf("PromptTokens = %d, want 100", got.PromptTokens)
	}
	if got.CompletionTokens != 50 {
		t.Fatalf("CompletionTokens = %d, want 50", got.CompletionTokens)
	}
	if got.CacheReadTokens != 0 {
		t.Fatalf("CacheReadTokens = %d, want 0", got.CacheReadTokens)
	}
	if got.CacheWriteTokens != 0 {
		t.Fatalf("CacheWriteTokens = %d, want 0", got.CacheWriteTokens)
	}
	if got.CostUSD != 0 {
		t.Fatalf("CostUSD = %v, want 0", got.CostUSD)
	}
	if got.CostUnknownTokens != 0 {
		t.Fatalf("CostUnknownTokens = %d, want 0", got.CostUnknownTokens)
	}
}

// TestAddUsageConcurrentGoroutines verifies that concurrent calls to AddUsage
// from multiple goroutines preserve all updates and accumulate correctly.
// The flock on usage.json.lock ensures read-modify-write atomicity.
func TestAddUsageConcurrentGoroutines(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

	// Number of goroutines and calls per goroutine
	const numGoroutines = 2
	const callsPerGoroutine = 50

	// Launch goroutines to add usage concurrently
	done := make(chan error, numGoroutines)
	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			for i := 0; i < callsPerGoroutine; i++ {
				// Each call adds 100 prompt, 50 completion, 10 cache read, 5 cache write
				_, err := AddUsage(root, 100, 50, 10, 5, 0.0005, true, now.Add(time.Duration(i)*time.Millisecond))
				if err != nil {
					done <- err
					return
				}
			}
			done <- nil
		}(g)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		if err := <-done; err != nil {
			t.Fatalf("goroutine error: %v", err)
		}
	}

	// Read the final totals
	final, err := ReadUsage(root)
	if err != nil {
		t.Fatalf("ReadUsage: %v", err)
	}

	// Expected totals: each goroutine calls 50 times with (100, 50, 10, 5)
	// 2 goroutines × 50 calls = 100 calls total
	expectedPrompt := int64(numGoroutines * callsPerGoroutine * 100)
	expectedCompletion := int64(numGoroutines * callsPerGoroutine * 50)
	expectedCacheRead := int64(numGoroutines * callsPerGoroutine * 10)
	expectedCacheWrite := int64(numGoroutines * callsPerGoroutine * 5)
	expectedTotal := expectedPrompt + expectedCompletion + expectedCacheRead + expectedCacheWrite
	expectedCost := float64(numGoroutines*callsPerGoroutine) * 0.0005

	if final.PromptTokens != expectedPrompt {
		t.Errorf("PromptTokens = %d, want %d", final.PromptTokens, expectedPrompt)
	}
	if final.CompletionTokens != expectedCompletion {
		t.Errorf("CompletionTokens = %d, want %d", final.CompletionTokens, expectedCompletion)
	}
	if final.CacheReadTokens != expectedCacheRead {
		t.Errorf("CacheReadTokens = %d, want %d", final.CacheReadTokens, expectedCacheRead)
	}
	if final.CacheWriteTokens != expectedCacheWrite {
		t.Errorf("CacheWriteTokens = %d, want %d", final.CacheWriteTokens, expectedCacheWrite)
	}
	if final.TotalTokens != expectedTotal {
		t.Errorf("TotalTokens = %d, want %d", final.TotalTokens, expectedTotal)
	}
	if absFloat(final.CostUSD-expectedCost) > 1e-10 {
		t.Errorf("CostUSD = %v, want ≈ %v", final.CostUSD, expectedCost)
	}
}

// globTemp returns the names matching .tmp* directly under dir.
func globTemp(t *testing.T, dir string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, ".tmp*"))
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	return matches
}

// TestAddUsageCorruptFileRecovery drives AddUsage over a usage.json that is
// corrupt in each way json.Unmarshal reports as a syntax or type error. Each
// must be moved aside byte-for-byte, reported through the returned name, and
// replaced by a fresh file holding only the new increment. The last subtest is
// the mirror: a valid file is neither moved nor reported.
func TestAddUsageCorruptFileRecovery(t *testing.T) {
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		name    string
		content string
	}{
		{"JSON syntax error", `{"prompt_tokens": invalid json`},
		{"JSON type error", `{"prompt_tokens": "not a number"}`},
		{"empty file", ``},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			usagePath := filepath.Join(root, "usage.json")
			if err := os.WriteFile(usagePath, []byte(tc.content), 0o600); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}

			recovered, err := AddUsage(root, 100, 50, 10, 5, 0.0005, true, now)
			if err != nil {
				t.Fatalf("%s: AddUsage: %v", tc.name, err)
			}
			if !strings.HasPrefix(recovered, "usage.json.corrupt-") {
				t.Fatalf("%s: recovered = %q, want a usage.json.corrupt-* name", tc.name, recovered)
			}
			moved, err := os.ReadFile(filepath.Join(root, recovered))
			if err != nil {
				t.Fatalf("%s: moved-aside file %q unreadable: %v", tc.name, recovered, err)
			}
			if string(moved) != tc.content {
				t.Errorf("%s: moved-aside content = %q, want the original %q", tc.name, moved, tc.content)
			}
			final, err := ReadUsage(root)
			if err != nil {
				t.Fatalf("%s: ReadUsage: %v", tc.name, err)
			}
			if final.PromptTokens != 100 || final.CompletionTokens != 50 {
				t.Errorf("%s: fresh usage.json = %+v, want only the new increment (100, 50)", tc.name, final)
			}
		})
	}

	t.Run("valid file is not moved aside", func(t *testing.T) {
		root := t.TempDir()
		if _, err := AddUsage(root, 1, 1, 0, 0, 0, true, now); err != nil {
			t.Fatalf("seed AddUsage: %v", err)
		}
		recovered, err := AddUsage(root, 2, 2, 0, 0, 0, true, now)
		if err != nil {
			t.Fatalf("AddUsage: %v", err)
		}
		if recovered != "" {
			t.Errorf("valid file: recovered = %q, want empty", recovered)
		}
		if got, _ := filepath.Glob(filepath.Join(root, "usage.json.corrupt-*")); len(got) != 0 {
			t.Errorf("valid file: moved aside to %v", got)
		}
		if final, _ := ReadUsage(root); final.PromptTokens != 3 {
			t.Errorf("valid file: PromptTokens = %d, want 3 (accumulated)", final.PromptTokens)
		}
	})
}

// TestAddUsageRecoveriesInOneSecondKeepBothFiles corrupts usage.json twice
// back to back. Both moved-aside files must survive under distinct names with
// their own content: a timestamp with one-second resolution would overwrite
// the first.
func TestAddUsageRecoveriesInOneSecondKeepBothFiles(t *testing.T) {
	root := t.TempDir()
	usagePath := filepath.Join(root, "usage.json")
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

	var names [2]string
	contents := [2]string{`{"first": corrupt`, `{"second": corrupt`}
	for i, content := range contents {
		if err := os.WriteFile(usagePath, []byte(content), 0o600); err != nil {
			t.Fatalf("WriteFile %d: %v", i, err)
		}
		recovered, err := AddUsage(root, 1, 1, 0, 0, 0, true, now)
		if err != nil {
			t.Fatalf("AddUsage %d: %v", i, err)
		}
		names[i] = recovered
	}

	if names[0] == "" || names[0] == names[1] {
		t.Fatalf("recovery names = %q and %q, want two distinct non-empty names", names[0], names[1])
	}
	for i, content := range contents {
		got, err := os.ReadFile(filepath.Join(root, names[i]))
		if err != nil {
			t.Fatalf("moved-aside file %d (%q): %v", i, names[i], err)
		}
		if string(got) != content {
			t.Errorf("moved-aside file %d holds %q, want %q (overwritten by a later recovery?)", i, got, content)
		}
	}
}

// TestAddUsageUnreadableFileLeftUntouched verifies that AddUsage does not try
// to recover from permission errors or unreadable files; it returns the error
// and leaves the file untouched. Skipped as root, which can read a mode-0
// file.
func TestAddUsageUnreadableFileLeftUntouched(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read a mode-0 file, so there is no read error to provoke")
	}
	root := t.TempDir()
	usagePath := filepath.Join(root, "usage.json")
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

	// Write a valid usage.json file, then make it unreadable
	validData := `{"prompt_tokens": 100, "completion_tokens": 50, "total_tokens": 150}`
	if err := os.WriteFile(usagePath, []byte(validData), 0o000); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(usagePath, 0o600) })

	// AddUsage should return an error and leave the file untouched
	recovered, err := AddUsage(root, 10, 5, 1, 1, 0.00005, true, now)
	if err == nil {
		t.Fatal("AddUsage should have failed on unreadable file")
	}
	if recovered != "" {
		t.Errorf("unreadable file: recovered = %q, want empty", recovered)
	}

	// Verify the file still exists and is untouched (still has the original content)
	stat, err := os.Stat(usagePath)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if stat.Mode().Perm() != 0o000 {
		t.Errorf("file permissions changed; want 0o000, got %o", stat.Mode().Perm())
	}

	// Verify no .corrupt* file was created
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "usage.json.corrupt") {
			t.Errorf("corrupt file should not be created for permission error; found %s", e.Name())
		}
	}
}

// TestWriteJSONFileRemovesTempWhenRenameFails calls writeJSONFile with a
// target path that is an existing non-empty directory. CreateTemp, the write
// and the fsync all succeed and os.Rename then fails, so this is the path
// where the deferred removal of the temporary file is the only cleanup. The
// glob is first shown to find a planted .tmp file, so an empty result below
// means the temporary file was removed and not that the glob matches nothing.
func TestWriteJSONFileRemovesTempWhenRenameFails(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.json")
	if err := os.MkdirAll(filepath.Join(target, "occupant"), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	control := filepath.Join(dir, ".tmp-control")
	if err := os.WriteFile(control, nil, 0o600); err != nil {
		t.Fatalf("WriteFile control: %v", err)
	}
	if got := globTemp(t, dir); len(got) != 1 || got[0] != control {
		t.Fatalf("positive control: glob found %v, want exactly [%s]", got, control)
	}
	if err := os.Remove(control); err != nil {
		t.Fatalf("Remove control: %v", err)
	}

	err := writeJSONFile(target, map[string]string{"k": "v"})
	if err == nil {
		t.Fatal("writeJSONFile onto a non-empty directory should have failed")
	}
	if !strings.Contains(err.Error(), "rename temporary file") {
		t.Fatalf("error = %v, want the rename failure (the path under test)", err)
	}
	if got := globTemp(t, dir); len(got) != 0 {
		t.Errorf("temporary file leaked after a failed rename: %v", got)
	}
	if _, statErr := os.Stat(filepath.Join(target, "occupant")); statErr != nil {
		t.Errorf("the target directory was disturbed: %v", statErr)
	}
}

// TestWriteJSONFileLeavesNoTempOnSuccess is the mirror of the rename-failure
// test: after a successful write the file holds the data and no temporary
// file remains.
func TestWriteJSONFileLeavesNoTempOnSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ok.json")

	if err := writeJSONFile(path, map[string]string{"k": "v"}); err != nil {
		t.Fatalf("writeJSONFile: %v", err)
	}
	var got map[string]string
	if err := readJSONFile(path, &got); err != nil || got["k"] != "v" {
		t.Fatalf("readJSONFile = %v, %v; want k=v", got, err)
	}
	if left := globTemp(t, dir); len(left) != 0 {
		t.Errorf("temporary file left after success: %v", left)
	}
}

// TestWriteJSONFileCreateTempFailureReturnsError makes the directory
// read-only so os.CreateTemp itself fails. Nothing is created, so this covers
// the error return only; the leak paths are the rename tests above. Skipped as
// root, which ignores directory modes.
func TestWriteJSONFileCreateTempFailureReturnsError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions, so CreateTemp cannot be made to fail")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	err := writeJSONFile(filepath.Join(dir, "x.json"), map[string]string{"k": "v"})
	if err == nil || !strings.Contains(err.Error(), "create temporary file") {
		t.Fatalf("error = %v, want a create temporary file failure", err)
	}
}

// TestWriteJSONFileConcurrentWritersNeverCollide has eight goroutines replace
// the same path 40 times each. Unique temporary names (os.CreateTemp) make
// every write succeed and leave one complete document; a fixed temporary name
// lets one writer truncate or rename another's file, which surfaces as a
// failed rename or a torn document.
func TestWriteJSONFileConcurrentWritersNeverCollide(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shared.json")

	const writers, writes = 8, 40
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < writes; i++ {
				if err := writeJSONFile(path, map[string]int{"writer": w, "seq": i}); err != nil {
					errs <- fmt.Errorf("writer %d, write %d: %w", w, i, err)
					return
				}
			}
		}(w)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent write failed: %v", err)
	}

	var got map[string]int
	if err := readJSONFile(path, &got); err != nil {
		t.Errorf("final document unreadable: %v", err)
	}
	if left := globTemp(t, dir); len(left) != 0 {
		t.Errorf("temporary files left behind: %v", left)
	}
}
