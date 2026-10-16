package session

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ErrNoSession is returned by resume helpers when the workspace has no recorded
// previous session in index.json.
var ErrNoSession = errors.New("no previous session for workspace")

// sessionFileExt is the extension used for JSONL session logs.
const sessionFileExt = ".jsonl"

// hashShortBytes is the number of SHA-256 bytes kept for the workspace hash.
// 6 bytes = 12 hex characters, enough for a short but low-collision directory
// name while keeping the layout human-readable.
const hashShortBytes = 6

// crockfordAlphabet is the Crockford base32 alphabet used by ULID.
const crockfordAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// ulidLength is the number of Crockford base32 characters in a ULID.
const ulidLength = 26

// timestampLayout is the compact UTC timestamp prefix for session filenames.
// It matches the chronologically-sortable example in plan §7.
const timestampLayout = "20060102T150405"

// DefaultStateRoot returns the plan §7 kirsch data root path:
// ~/.local/share/kirsch.
//
// It follows the XDG-adjacent location shown in plan §7 line 367, resolving
// the user's home directory with os.UserHomeDir. Session files live under
// <root>/sessions; index.json and usage.json live directly at <root>.
func DefaultStateRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", "kirsch"), nil
}

// Canonicalize returns an absolute, cleaned workspace path.
//
// The canonical path is the key stored in index.json and hashed for the
// session directory name (plan §7 line 376).
func Canonicalize(workspace string) (string, error) {
	if workspace == "" {
		return "", fmt.Errorf("workspace path is empty")
	}
	abs, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("resolve workspace path: %w", err)
	}
	return filepath.Clean(abs), nil
}

// WorkspaceHash returns a short, deterministic hash of the canonical workspace
// path. The result is used as the <workspace-hash> directory name in the plan
// §7 layout.
func WorkspaceHash(workspace string) string {
	sum := sha256.Sum256([]byte(workspace))
	return fmt.Sprintf("%x", sum[:hashShortBytes])
}

// SessionDir returns the per-workspace directory that holds session files.
//
// The returned path is <root>/sessions/<workspace-hash>, matching plan §7
// line 369.
func SessionDir(root, workspace string) string {
	return filepath.Join(root, "sessions", WorkspaceHash(workspace))
}

// SessionPath returns the full path for a session file with the given id.
//
// The returned path is root/sessions/<workspace-hash>/<id>.jsonl, matching
// plan §7 line 370.
func SessionPath(root, workspace, id string) string {
	return filepath.Join(SessionDir(root, workspace), id+sessionFileExt)
}

// SessionID returns a unique, lexicographically-sortable session identifier in
// the form <timestamp>-<ulid> required by plan §7 line 370.
//
// The timestamp prefix sorts to one-second resolution; the ULID suffix is a
// stdlib-only implementation using 48-bit millisecond timestamp plus 80 bits
// of crypto/rand entropy, Crockford base32-encoded to 26 characters.
func SessionID(now time.Time) (string, error) {
	ulid, err := newULID(now, rand.Reader)
	if err != nil {
		return "", fmt.Errorf("generate ulid: %w", err)
	}
	return now.UTC().Format(timestampLayout) + "-" + ulid, nil
}

// newULID generates a 26-character Crockford base32 ULID for the given time.
//
// It encodes a 48-bit millisecond timestamp in the high bits and 80 bits of
// cryptographically secure randomness in the low bits. The result is globally
// unique (within clock/rand guarantees) and lexicographically sortable by time.
func newULID(now time.Time, rnd io.Reader) (string, error) {
	ms := uint64(now.UnixNano()) / 1e6

	var b [16]byte
	// Place the 48-bit timestamp in the most significant 48 bits.
	binary.BigEndian.PutUint64(b[:], ms<<16)
	if _, err := io.ReadFull(rnd, b[6:]); err != nil {
		return "", err
	}

	return encodeCrockford(b), nil
}

// encodeCrockford encodes 16 bytes (128 bits) as a 26-character Crockford
// base32 string. The 128-bit value is zero-padded to 130 bits so it divides
// evenly into 26 five-bit groups.
func encodeCrockford(b [16]byte) string {
	out := make([]byte, ulidLength)
	for i := 0; i < ulidLength; i++ {
		var v byte
		for j := 0; j < 5; j++ {
			bit := i*5 + j // 0..129; bits 128 and 129 are zero padding.
			v = (v << 1) | bitAt(b[:], bit)
		}
		out[i] = crockfordAlphabet[v]
	}
	return string(out)
}

// bitAt returns bit i of data when the most significant bit of data[0] is bit
// zero. Bits beyond the slice length are zero.
func bitAt(data []byte, i int) byte {
	if i < 0 || i >= len(data)*8 {
		return 0
	}
	return (data[i/8] >> (7 - (i % 8))) & 1
}

// Layout knows how to place session files on disk for one workspace.
//
// It is a pure path/layout helper; it does not open files or run goroutines.
// Callers create a Store with the path returned by CreateSession.
type Layout struct {
	// Root is the plan §7 kirsch data root directory.
	Root string
	// Workspace is the canonical workspace path used as the index key and for
	// the workspace-hash directory.
	Workspace string
}

// NewLayout returns a Layout for workspace. The workspace path is canonicalised
// before it is stored.
func NewLayout(root, workspace string) (*Layout, error) {
	if root == "" {
		return nil, fmt.Errorf("state root is empty")
	}
	canonical, err := Canonicalize(workspace)
	if err != nil {
		return nil, err
	}
	return &Layout{Root: root, Workspace: canonical}, nil
}

// CreateSession creates a new session directory and id, writes index.json with
// the new session as the last session, and returns the full file path and id.
//
// The returned path can be passed directly to New to open the session store.
func (l *Layout) CreateSession(now time.Time) (path, id string, err error) {
	id, err = SessionID(now)
	if err != nil {
		return "", "", err
	}

	sessionDir := SessionDir(l.Root, l.Workspace)
	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		return "", "", fmt.Errorf("create session directory: %w", err)
	}

	path = SessionPath(l.Root, l.Workspace, id)

	idx, err := ReadIndex(l.Root)
	if err != nil {
		return "", "", fmt.Errorf("read index: %w", err)
	}
	idx.Set(l.Workspace, id, now)
	if err := WriteIndex(l.Root, idx); err != nil {
		return "", "", fmt.Errorf("write index: %w", err)
	}

	return path, id, nil
}

// LastSession reads index.json and returns the last session id for this
// workspace, if any.
func (l *Layout) LastSession() (id string, updatedAt time.Time, ok bool, err error) {
	idx, err := ReadIndex(l.Root)
	if err != nil {
		return "", time.Time{}, false, err
	}
	entry, ok := idx.Sessions[l.Workspace]
	if !ok {
		return "", time.Time{}, false, nil
	}
	return entry.LastSessionID, entry.UpdatedAt, true, nil
}

// RecordSession updates index.json so that id is recorded as the last session
// for this workspace. It is used when a session is loaded (plan §7 index.json
// is updated on create/load) as well as when it is created.
func (l *Layout) RecordSession(id string, now time.Time) error {
	idx, err := ReadIndex(l.Root)
	if err != nil {
		return fmt.Errorf("read index: %w", err)
	}
	idx.Set(l.Workspace, id, now)
	if err := WriteIndex(l.Root, idx); err != nil {
		return fmt.Errorf("write index: %w", err)
	}
	return nil
}

// IndexPath returns the path to index.json at the kirsch data root.
func IndexPath(root string) string {
	return filepath.Join(root, "index.json")
}

// UsagePath returns the path to usage.json at the kirsch data root.
func UsagePath(root string) string {
	return filepath.Join(root, "usage.json")
}
