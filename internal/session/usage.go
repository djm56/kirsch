package session

import (
	"fmt"
	"os"
	"time"
)

// Usage holds cumulative token totals at the state root.
//
// The shape matches the usage.json schema described in plan §7 line 373.
type Usage struct {
	// PromptTokens is the cumulative number of prompt tokens consumed.
	PromptTokens int64 `json:"prompt_tokens"`
	// CompletionTokens is the cumulative number of completion tokens consumed.
	CompletionTokens int64 `json:"completion_tokens"`
	// TotalTokens is the cumulative total of prompt and completion tokens.
	TotalTokens int64 `json:"total_tokens"`
	// UpdatedAt is the time the totals were last changed.
	UpdatedAt time.Time `json:"updated_at"`
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

// Add increments cumulative totals by the given token counts and updates the
// timestamp.
func (u *Usage) Add(promptTokens, completionTokens int64, now time.Time) {
	u.PromptTokens += promptTokens
	u.CompletionTokens += completionTokens
	u.TotalTokens = u.PromptTokens + u.CompletionTokens
	u.UpdatedAt = now.UTC()
}

// AddUsage updates usage.json with a token delta.
//
// It loads the current usage file, applies the delta, and writes it back. This
// is the update path that m4-d3 will call when usage events arrive.
func AddUsage(root string, promptTokens, completionTokens int64, now time.Time) error {
	u, err := ReadUsage(root)
	if err != nil {
		return err
	}
	u.Add(promptTokens, completionTokens, now)
	return WriteUsage(root, u)
}
