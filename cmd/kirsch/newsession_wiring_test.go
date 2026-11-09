package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/djm56/kirsch/internal/app"
	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

// sessionIDPattern picks the id out of /status's "session <id>" segment. The
// notice joins its segments with the glyph bullet (update.go, the "session "+id
// part, followed by the "tokens in ..." part), so the pattern takes the id only
// when one non-space glyph and "tokens in" follow it. Free text on the same
// frame, such as the empty-state suggestion "Where is the session lock taken?",
// has no "tokens in" behind it and cannot match.
var sessionIDPattern = regexp.MustCompile(`session ([0-9A-Za-z_-]+) \S tokens in`)

// slash types line into the composer, presses Enter and returns the model and
// the command Update returned.
func slash(tm tea.Model, line string) (tea.Model, tea.Cmd) {
	for _, r := range line {
		tm, _ = tm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return tm.Update(tea.KeyMsg{Type: tea.KeyEnter})
}

// doneMessages runs cmd, flattening tea.Batch, and returns the
// NewSessionDoneMsg values it produced. The spinner's tick is run and
// ignored.
func doneMessages(cmd tea.Cmd) []tui.NewSessionDoneMsg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tui.NewSessionDoneMsg
		for _, c := range batch {
			out = append(out, doneMessages(c)...)
		}
		return out
	}
	if done, ok := msg.(tui.NewSessionDoneMsg); ok {
		return []tui.NewSessionDoneMsg{done}
	}
	return nil
}

// statusSessionID types /status and returns the session id its notice shows.
func statusSessionID(t *testing.T, tm tea.Model) (tea.Model, string) {
	t.Helper()
	tm, _ = slash(tm, "/status")
	m := sessionIDPattern.FindStringSubmatch(tm.View())
	if m == nil {
		t.Fatalf("/status frame has no \"session <id>\":\n%s", tm.View())
	}
	return tm, m[1]
}

// TestSlashNewThroughRealWiringStartsANewSession runs /new through the model
// wired by wireCallbacks over a real App and a real session store. The entry
// point is the keypress; the session id /status reports must change, and the
// old session file must still be there next to the new one.
func TestSlashNewThroughRealWiringStartsANewSession(t *testing.T) {
	dir := t.TempDir()
	ws, err := workspace.Detect(t.TempDir())
	if err != nil {
		t.Fatalf("workspace.Detect: %v", err)
	}
	cfg := config.Defaults()
	cfg.Session.StorageDir = filepath.Join(dir, "sessions")
	log := telemetry.Disabled()
	a := app.New(ws, cfg, log)
	t.Cleanup(func() { a.Close() })
	if err := a.StartNewSession(); err != nil {
		t.Fatalf("StartNewSession: %v", err)
	}

	m := tui.New(tui.Options{Version: "0.1.0-test", Caps: tui.Caps{Colour: false, Unicode: false}})
	wireCallbacks(&m, a, log)
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 200, Height: 30})

	// The session StartNewSession created is still empty, and an empty
	// session is deliberately removed when /new replaces it (m5-d17a). The
	// test is about a real session surviving /new, so give the old one an
	// event first. ClearGrants is the app's own path for that: it records
	// approval.scopes_cleared on the live store and flushes it. Package main
	// cannot reach the store itself, and a second session.New on the file
	// would be a second writer.
	a.ClearGrants()
	// The setup above relies on ClearGrants writing to the store, which is a
	// side effect of that call rather than its purpose. Check it here, so a
	// change to ClearGrants fails setup instead of the assertions after /new.
	oldFiles, _ := filepath.Glob(filepath.Join(dir, "sessions", "*", "*.jsonl"))
	if len(oldFiles) != 1 {
		t.Fatalf("setup: session files before /new = %v, want exactly 1", oldFiles)
	}
	if info, err := os.Stat(oldFiles[0]); err != nil || info.Size() == 0 {
		t.Fatalf("setup: the old session %s is empty or unreadable before /new (err %v); ClearGrants recorded nothing", oldFiles[0], err)
	}

	tm, before := statusSessionID(t, tm)

	tm, cmd := slash(tm, "/new")
	dones := doneMessages(cmd)
	if len(dones) != 1 {
		t.Fatalf("/new produced %d NewSessionDoneMsg, want 1", len(dones))
	}
	if dones[0].Err != nil {
		t.Fatalf("NewSession through the wiring failed: %v", dones[0].Err)
	}
	tm, _ = tm.Update(dones[0])

	tm, after := statusSessionID(t, tm)
	if after == before {
		t.Errorf("/status session id unchanged after /new: %s", before)
	}
	if strings.Contains(tm.View(), before) {
		t.Errorf("the old session id %s is still on screen after /new", before)
	}
	files, _ := filepath.Glob(filepath.Join(dir, "sessions", "*", "*.jsonl"))
	if len(files) != 2 {
		all, _ := filepath.Glob(filepath.Join(dir, "sessions", "*", "*"))
		t.Errorf("session files = %v (all entries %v), want the old and the new", files, all)
	}
	// Exactly one of the two files holds content: the old session's event. The
	// new session is empty, so a surviving old file that was itself empty would
	// show as zero here.
	var nonEmpty int
	for _, f := range files {
		if info, err := os.Stat(f); err == nil && info.Size() > 0 {
			nonEmpty++
		}
	}
	if nonEmpty != 1 {
		t.Errorf("%d non-empty session files after /new, want exactly 1 (the old session's event)", nonEmpty)
	}
}
