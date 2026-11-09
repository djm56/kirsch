package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

// UsageTotals is the sum of the usage events in one session log. It is
// per-session, unlike Usage, which is the figure in usage.json, which holds the totals of every
// session that uses the same data root.
type UsageTotals struct {
	// InputTokens is the sum of every usage event's input_tokens.
	InputTokens int64
	// OutputTokens is the sum of every usage event's output_tokens.
	OutputTokens int64
	// CacheReadTokens is the sum of every usage event's cache_read.
	CacheReadTokens int64
	// CacheWriteTokens is the sum of every usage event's cache_write.
	CacheWriteTokens int64
}

// LoadUsageTotals reads the JSONL session log at path and returns the sum of
// its usage events. A log with no usage event returns the zero value and a nil
// error.
//
// A usage event records the tokens of one model call, a turn or a compaction
// summary, and carries no model id or price, so the sum is tokens only; the
// caller prices it.
//
// A torn or invalid JSON line, or a usage event whose fields do not decode, is
// treated as the end of the log, matching LoadGrants and the repair-on-load
// behaviour of Store.New. Events before it still count; nothing after it does.
func LoadUsageTotals(path string) (UsageTotals, error) {
	// #nosec G304 -- path is caller-supplied; containment is the caller's
	// responsibility, matching LoadGrants, Store.New and Resume.
	f, err := os.Open(path)
	if err != nil {
		return UsageTotals{}, err
	}
	defer func() { _ = f.Close() }()

	var total UsageTotals
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var base struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(line, &base); err != nil {
			// Torn or invalid line marks the end of the durable log.
			break
		}
		if base.Type != eventTypeUsage {
			continue
		}

		var ev usageEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			break
		}
		total.InputTokens += int64(ev.InputTokens)
		total.OutputTokens += int64(ev.OutputTokens)
		total.CacheReadTokens += int64(ev.CacheRead)
		total.CacheWriteTokens += int64(ev.CacheWrite)
	}
	if err := scanner.Err(); err != nil {
		return UsageTotals{}, fmt.Errorf("read session log: %w", err)
	}
	return total, nil
}
