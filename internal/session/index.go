package session

import (
	"fmt"
	"os"
	"time"
)

// IndexEntry records the most recent session for one workspace.
type IndexEntry struct {
	// LastSessionID is the filename stem (timestamp-ulid) of the most recent
	// session file for the workspace.
	LastSessionID string `json:"last_session_id"`
	// UpdatedAt is the time the entry was last changed.
	UpdatedAt time.Time `json:"updated_at"`
}

// Index is the on-disk shape of index.json.
type Index struct {
	// Sessions maps a canonical workspace path to its most recent session.
	Sessions map[string]IndexEntry `json:"sessions"`
}

// ReadIndex loads index.json from the state root.
//
// If the file does not exist, it returns an empty Index so that callers can
// treat a missing index the same as an empty one.
func ReadIndex(root string) (*Index, error) {
	idx := &Index{Sessions: make(map[string]IndexEntry)}
	path := IndexPath(root)
	if err := readJSONFile(path, idx); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read index: %w", err)
	}
	if idx.Sessions == nil {
		idx.Sessions = make(map[string]IndexEntry)
	}
	return idx, nil
}

// WriteIndex writes idx to index.json atomically.
//
// It writes to a temporary file and renames it into place so readers never see
// a partially-written index.
func WriteIndex(root string, idx *Index) error {
	path := IndexPath(root)
	if err := os.MkdirAll(root, 0o700); err != nil {
		return fmt.Errorf("create state root: %w", err)
	}
	if err := writeJSONFile(path, idx); err != nil {
		return fmt.Errorf("write index: %w", err)
	}
	return nil
}

// Set updates the index entry for workspace to record sessionID at now.
func (idx *Index) Set(workspace, sessionID string, now time.Time) {
	idx.Sessions[workspace] = IndexEntry{
		LastSessionID: sessionID,
		UpdatedAt:     now.UTC(),
	}
}
