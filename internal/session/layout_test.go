package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// TestDefaultStateRoot verifies the plan §7 kirsch data root derivation.
//
// The data root is ~/.local/share/kirsch (plan §7 line 367); session files
// live under <root>/sessions and metadata files live directly at <root>.
func TestDefaultStateRoot(t *testing.T) {
	root, err := DefaultStateRoot()
	if err != nil {
		t.Fatalf("DefaultStateRoot: %v", err)
	}
	if !filepath.IsAbs(root) {
		t.Fatalf("state root must be absolute, got %q", root)
	}
	wantSuffix := filepath.Join(".local", "share", "kirsch")
	if !strings.HasSuffix(root, wantSuffix) {
		t.Fatalf("state root %q does not end with %q", root, wantSuffix)
	}
}

// TestCanonicalize verifies workspace path canonicalisation.
func TestCanonicalize(t *testing.T) {
	t.Run("empty errors", func(t *testing.T) {
		if _, err := Canonicalize(""); err == nil {
			t.Fatal("expected error for empty workspace")
		}
	})

	t.Run("relative becomes absolute", func(t *testing.T) {
		got, err := Canonicalize(".")
		if err != nil {
			t.Fatalf("Canonicalize: %v", err)
		}
		if !filepath.IsAbs(got) {
			t.Fatalf("expected absolute path, got %q", got)
		}
	})

	t.Run("trailing separator cleaned", func(t *testing.T) {
		base := t.TempDir()
		withSlash := base + string(filepath.Separator)
		got, err := Canonicalize(withSlash)
		if err != nil {
			t.Fatalf("Canonicalize: %v", err)
		}
		if strings.HasSuffix(got, string(filepath.Separator)) {
			t.Fatalf("expected cleaned path, got %q", got)
		}
	})
}

// TestWorkspaceHash verifies the short hash derivation.
//
// The workspace-hash is a short hash of the canonical workspace path (plan §7
// line 376).
func TestWorkspaceHash(t *testing.T) {
	ws := "/home/alice/projects/kirsch"
	first := WorkspaceHash(ws)
	second := WorkspaceHash(ws)
	if first != second {
		t.Fatalf("workspace hash not deterministic: %q vs %q", first, second)
	}
	if len(first) != hashShortBytes*2 {
		t.Fatalf("expected %d hex characters, got %d (%q)", hashShortBytes*2, len(first), first)
	}
	if strings.ToLower(first) != first {
		t.Fatalf("workspace hash must be lower-case hex, got %q", first)
	}
}

// TestSessionID verifies the id format and chronological sortability.
//
// The id is <timestamp>-<ulid> (plan §7 line 370) and filenames must sort
// chronologically.
func TestSessionID(t *testing.T) {
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	id, err := SessionID(now)
	if err != nil {
		t.Fatalf("SessionID: %v", err)
	}
	parts := strings.Split(id, "-")
	if len(parts) != 2 {
		t.Fatalf("expected two dash-separated parts, got %q", id)
	}
	if len(parts[0]) != len(timestampLayout) {
		t.Fatalf("timestamp length = %d, want %d", len(parts[0]), len(timestampLayout))
	}
	if len(parts[1]) != ulidLength {
		t.Fatalf("ulid length = %d, want %d", len(parts[1]), ulidLength)
	}

	earlier := now.Add(-time.Second)
	later := now.Add(time.Second)
	idEarlier, err := SessionID(earlier)
	if err != nil {
		t.Fatalf("SessionID earlier: %v", err)
	}
	idLater, err := SessionID(later)
	if err != nil {
		t.Fatalf("SessionID later: %v", err)
	}
	if idEarlier >= idLater {
		t.Fatalf("ids do not sort chronologically: %q >= %q", idEarlier, idLater)
	}
}

// TestSessionPath verifies the full session file path layout.
//
// The layout is root/sessions/<workspace-hash>/<id>.jsonl (plan §7 line 370).
func TestSessionPath(t *testing.T) {
	root := t.TempDir()
	workspace := "/home/alice/projects/kirsch"
	id := "20261014T120000-01ARZ3NDEKTSV4RRFFQ69G5FAV"
	got := SessionPath(root, workspace, id)

	wantDir := filepath.Join(root, "sessions", WorkspaceHash(workspace))
	want := filepath.Join(wantDir, id+sessionFileExt)
	if got != want {
		t.Fatalf("SessionPath = %q, want %q", got, want)
	}

	rel, err := filepath.Rel(root, got)
	if err != nil {
		t.Fatalf("Rel: %v", err)
	}
	if strings.HasPrefix(rel, "..") {
		t.Fatalf("session path escapes root: %q", rel)
	}
}

// TestNewLayout verifies layout creation and workspace canonicalisation.
func TestNewLayout(t *testing.T) {
	t.Run("empty root errors", func(t *testing.T) {
		if _, err := NewLayout("", "/x"); err == nil {
			t.Fatal("expected error for empty root")
		}
	})

	t.Run("canonicalises workspace", func(t *testing.T) {
		root := t.TempDir()
		rel := "./foo"
		layout, err := NewLayout(root, rel)
		if err != nil {
			t.Fatalf("NewLayout: %v", err)
		}
		if layout.Root != root {
			t.Fatalf("Root = %q, want %q", layout.Root, root)
		}
		if !filepath.IsAbs(layout.Workspace) {
			t.Fatalf("workspace not canonicalised: %q", layout.Workspace)
		}
	})
}

// TestLayoutCreateSession verifies that CreateSession writes the directory,
// returns a usable path, and updates index.json.
func TestLayoutCreateSession(t *testing.T) {
	root := t.TempDir()
	layout, err := NewLayout(root, "/home/alice/kirsch")
	if err != nil {
		t.Fatalf("NewLayout: %v", err)
	}

	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	path, id, err := layout.CreateSession(now)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if path != SessionPath(root, layout.Workspace, id) {
		t.Fatalf("path = %q, want %q", path, SessionPath(root, layout.Workspace, id))
	}

	info, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat session directory: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("expected directory at %q", filepath.Dir(path))
	}

	idx, err := ReadIndex(root)
	if err != nil {
		t.Fatalf("ReadIndex: %v", err)
	}
	entry, ok := idx.Sessions[layout.Workspace]
	if !ok {
		t.Fatalf("workspace missing from index")
	}
	if entry.LastSessionID != id {
		t.Fatalf("LastSessionID = %q, want %q", entry.LastSessionID, id)
	}
	if !entry.UpdatedAt.Equal(now) {
		t.Fatalf("UpdatedAt = %v, want %v", entry.UpdatedAt, now)
	}
}

// TestLayoutCreateSessionSortable verifies that ids from successive seconds sort
// chronologically.
func TestLayoutCreateSessionSortable(t *testing.T) {
	root := t.TempDir()
	layout, err := NewLayout(root, "/home/alice/kirsch")
	if err != nil {
		t.Fatalf("NewLayout: %v", err)
	}

	var ids []string
	base := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		_, id, err := layout.CreateSession(base.Add(time.Duration(i) * time.Second))
		if err != nil {
			t.Fatalf("CreateSession: %v", err)
		}
		ids = append(ids, id)
	}

	sorted := append([]string(nil), ids...)
	sort.Strings(sorted)
	for i := range ids {
		if ids[i] != sorted[i] {
			t.Fatalf("ids not chronologically sorted: %v", ids)
		}
	}
}

// TestLayoutLastSession verifies LastSession and RecordSession.
func TestLayoutLastSession(t *testing.T) {
	root := t.TempDir()
	layout, err := NewLayout(root, "/home/alice/kirsch")
	if err != nil {
		t.Fatalf("NewLayout: %v", err)
	}

	if _, _, ok, _ := layout.LastSession(); ok {
		t.Fatal("expected no last session for fresh workspace")
	}

	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	_, firstID, err := layout.CreateSession(now)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	got, _, ok, err := layout.LastSession()
	if err != nil {
		t.Fatalf("LastSession: %v", err)
	}
	if !ok || got != firstID {
		t.Fatalf("LastSession = %q, ok=%v, want %q", got, ok, firstID)
	}

	otherID := "20261014T130000-01ARZ3NDEKTSV4RRFFQ69G5FAV"
	if err := layout.RecordSession(otherID, now.Add(time.Hour)); err != nil {
		t.Fatalf("RecordSession: %v", err)
	}
	got, _, ok, err = layout.LastSession()
	if err != nil {
		t.Fatalf("LastSession: %v", err)
	}
	if !ok || got != otherID {
		t.Fatalf("LastSession after RecordSession = %q, want %q", got, otherID)
	}
}

// TestIndexPathAndUsagePath verify the metadata file locations.
func TestIndexPathAndUsagePath(t *testing.T) {
	root := t.TempDir()
	if got := IndexPath(root); got != filepath.Join(root, "index.json") {
		t.Fatalf("IndexPath = %q, want %q", got, filepath.Join(root, "index.json"))
	}
	if got := UsagePath(root); got != filepath.Join(root, "usage.json") {
		t.Fatalf("UsagePath = %q, want %q", got, filepath.Join(root, "usage.json"))
	}
}

// TestLayoutIndexJSONSchema verifies the on-disk shape of index.json.
func TestLayoutIndexJSONSchema(t *testing.T) {
	root := t.TempDir()
	layout, err := NewLayout(root, "/home/alice/kirsch")
	if err != nil {
		t.Fatalf("NewLayout: %v", err)
	}

	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	_, id, err := layout.CreateSession(now)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	data, err := os.ReadFile(IndexPath(root))
	if err != nil {
		t.Fatalf("read index.json: %v", err)
	}
	var raw map[string]map[string]struct {
		LastSessionID string    `json:"last_session_id"`
		UpdatedAt     time.Time `json:"updated_at"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal index.json: %v", err)
	}
	entry, ok := raw["sessions"][layout.Workspace]
	if !ok {
		t.Fatalf("workspace entry missing from index.json")
	}
	if entry.LastSessionID != id {
		t.Fatalf("last_session_id = %q, want %q", entry.LastSessionID, id)
	}
}
