package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// usageLock represents an advisory lock protecting usage.json from concurrent writes.
// On Unix platforms, it holds an exclusive flock on usage.json.lock. On non-Unix
// platforms, it is a stub that always succeeds.
//
// The lock file (usage.json.lock) is never removed. Unlinking an flock file races
// with concurrent waiters: a waiter may open the file after it is unlinked but before
// it relinquishes the lock, creating a separate inode that the opener holds exclusively
// while other writers lock a new file. Keeping the lock file persists the inode,
// ensuring all processes wait on the same lock.
type usageLock struct {
	fileLock *Lock
}

// acquireUsageLock acquires an advisory lock on usage.json.lock at the state root.
// On Unix platforms, the lock is exclusive, non-blocking, and retried up to 20 times
// at 50ms intervals (approximately 1s total). After all retries are exhausted, it returns
// ErrLockHeld and the caller drops the increment; this package logs nothing. On non-Unix
// platforms, the lock is always acquired (platforms are not supported in v0.1).
func acquireUsageLock(root string) (*usageLock, error) {
	// Create the state root directory if needed
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("create state root: %w", err)
	}

	// The lock file path
	lockPath := filepath.Join(root, "usage.json.lock")

	// Open or create the lock file
	file, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE, 0o600) // #nosec G304 -- fixed basename under the operator-configured state root, created 0700
	if err != nil {
		return nil, fmt.Errorf("open usage lock file: %w", err)
	}

	// On Unix, retry briefly if the lock is held. Each retry waits ~50ms.
	// On non-Unix, tryLock always succeeds immediately.
	const maxRetries = 20
	const retryDelay = time.Millisecond * 50

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := tryLock(file)
		if err == nil {
			return &usageLock{fileLock: &Lock{file: file}}, nil
		}
		if !errors.Is(err, ErrLockHeld) {
			// Not contention (for example permission denied): do not retry.
			_ = file.Close()
			return nil, err
		}
		// Held by another writer: wait and try again.
		time.Sleep(retryDelay)
	}

	// Still held after every retry.
	_ = file.Close()
	return nil, ErrLockHeld
}

// Usage holds cumulative token and cost totals persisted to usage.json at the state root.
type Usage struct {
	// PromptTokens is the cumulative number of prompt tokens consumed.
	PromptTokens int64 `json:"prompt_tokens"`
	// CompletionTokens is the cumulative number of completion tokens consumed.
	CompletionTokens int64 `json:"completion_tokens"`
	// CacheReadTokens is the cumulative number of cache read tokens consumed.
	CacheReadTokens int64 `json:"cache_read_tokens"`
	// CacheWriteTokens is the cumulative number of cache write tokens consumed.
	CacheWriteTokens int64 `json:"cache_write_tokens"`
	// TotalTokens is the cumulative total of all token types: prompt + completion + cache_read + cache_write.
	TotalTokens int64 `json:"total_tokens"`
	// CostUSD is the cumulative cost in USD for tokens with known pricing.
	CostUSD float64 `json:"cost_usd"`
	// CostUnknownTokens counts the total tokens from usage events whose cost could not be computed
	// (unknown model or pricing model), so a partial total is visible.
	CostUnknownTokens int64 `json:"cost_unknown_tokens"`
	// UpdatedAt is the time the totals were last changed.
	UpdatedAt time.Time `json:"updated_at"`
}

// Release unlocks and closes the usage lock. It is safe to call multiple times.
func (ul *usageLock) Release() error {
	if ul.fileLock == nil {
		return nil
	}
	return ul.fileLock.Release()
}

// ReadUsage loads usage.json from the state root.
//
// If the file does not exist, it returns a zero Usage so callers can treat a
// missing usage file the same as an empty one.
func ReadUsage(root string) (*Usage, error) {
	u := &Usage{}
	path := UsagePath(root)
	if err := readJSONFile(path, u); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read usage: %w", err)
	}
	return u, nil
}

// WriteUsage writes u to usage.json atomically.
//
// It writes to a temporary file and renames it into place so readers never see
// a partially-written usage file.
func WriteUsage(root string, u *Usage) error {
	path := UsagePath(root)
	if err := os.MkdirAll(root, 0o700); err != nil {
		return fmt.Errorf("create state root: %w", err)
	}
	if err := writeJSONFile(path, u); err != nil {
		return fmt.Errorf("write usage: %w", err)
	}
	return nil
}

// Add increments cumulative totals by the given token counts, cost, and updates the
// timestamp.
func (u *Usage) Add(in, out, cacheRead, cacheWrite int64, costUSD float64, costOK bool, now time.Time) {
	u.PromptTokens += in
	u.CompletionTokens += out
	u.CacheReadTokens += cacheRead
	u.CacheWriteTokens += cacheWrite
	u.TotalTokens = u.PromptTokens + u.CompletionTokens + u.CacheReadTokens + u.CacheWriteTokens
	if costOK {
		u.CostUSD += costUSD
	} else {
		// When cost is unknown, count the total tokens as unknown
		u.CostUnknownTokens += in + out + cacheRead + cacheWrite
	}
	u.UpdatedAt = now.UTC()
}

// AddUsage updates usage.json with a token delta.
//
// It loads the current usage file, applies the delta (in, out, cacheRead, cacheWrite tokens
// plus cost), and writes it back. costOK indicates whether the cost was successfully computed
// (true for known pricing models, false for flat-rate or unknown models).
//
// The read-modify-write sequence is guarded by an advisory flock on usage.json.lock.
// On Unix platforms, the lock is exclusive and non-blocking with a retry loop of up to 20
// attempts at 50ms intervals (about 1s total); if the lock cannot be acquired within that
// time, AddUsage returns ErrLockHeld and the increment is dropped. AddUsage does not log;
// the caller decides how to report the drop.
// The rename of the temporary file to usage.json is atomic at the OS level.
// On non-Unix platforms (where file locking is not enforced), two concurrent instances may
// lose updates.
//
// If usage.json is corrupt (a JSON syntax or type error), it is renamed aside to
// usage.json.corrupt-<UTC timestamp with nanoseconds> and a fresh file is started. The
// returned recoveredTo is the base name of that moved-aside file, and is empty when no
// recovery happened. It is set even when AddUsage then returns an error, so the caller can
// always report the move. Nothing is written to stdout or stderr: the TUI owns the terminal.
func AddUsage(root string, in, out, cacheRead, cacheWrite int64, costUSD float64, costOK bool, now time.Time) (recoveredTo string, err error) {
	lock, err := acquireUsageLock(root)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = lock.Release()
	}()

	u, err := ReadUsage(root)
	if err != nil {
		// Recover only from a JSON syntax or type error. Anything else
		// (permission denied, an I/O error) is returned as-is and the file is
		// left where it is.
		var syntaxErr *json.SyntaxError
		var unmarshalErr *json.UnmarshalTypeError
		if !errors.As(err, &syntaxErr) && !errors.As(err, &unmarshalErr) {
			return "", err
		}
		usagePath := UsagePath(root)
		if _, statErr := os.Stat(usagePath); statErr == nil {
			// Nanosecond resolution keeps two recoveries within one second
			// from overwriting the earlier moved-aside file.
			stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
			corruptPath := usagePath + ".corrupt-" + stamp
			if moveErr := os.Rename(usagePath, corruptPath); moveErr != nil {
				return "", err
			}
			recoveredTo = filepath.Base(corruptPath)
		} else if !os.IsNotExist(statErr) {
			return "", err
		}
		// A file that vanished between the read and the stat (a writer that
		// ignores the lock) leaves nothing to move aside; start fresh either way.
		u = &Usage{}
	}
	u.Add(in, out, cacheRead, cacheWrite, costUSD, costOK, now)
	return recoveredTo, WriteUsage(root, u)
}
