package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

// GrantRecord is one session-scoped grant that was recorded in a session log.
type GrantRecord struct {
	// Action is the operation that was granted, e.g. "run_command".
	Action string
	// Pattern is the argv prefix or pattern that was granted, stored as a single
	// space-separated string.
	Pattern string
}

// LoadGrants reads approval.scope_granted events from the JSONL session log at
// path and returns the recorded grants in order.
//
// A torn or invalid JSON line is treated as the end of the log, matching the
// repair-on-load behaviour of Store.New. This keeps grant restoration
// consistent with conversation reconstruction in Resume.
func LoadGrants(path string) ([]GrantRecord, error) {
	// #nosec G304 -- path is caller-supplied; containment is the caller's
	// responsibility, matching Store.New and Resume.
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var grants []GrantRecord
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
		if base.Type != eventTypeApprovalScopeGranted {
			continue
		}

		var ev approvalScopeGrantedEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			break
		}
		grants = append(grants, GrantRecord{
			Action:  ev.Action,
			Pattern: ev.Pattern,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read session log: %w", err)
	}
	return grants, nil
}
