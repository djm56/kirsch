package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// The tests in this file hold where the tui gets a tool card's target and the rest
// of its head. A provider's start event can carry the call's name and no input, so a
// card made from ToolStartedMsg may have no target; ToolTargetMsg gives it one once the
// complete call is known, and ToolCompletedMsg carries it too. The read_file modal takes
// its title and its highlighting path from the card's target, so a card without one
// opens a modal with no path and no lexer. Every field the head draws can hold text the
// model wrote, and each is sanitised where the tui stores it. The live message sequence
// itself is held to internal/app's real producers by internal/app/live_tool_test.go.

// toolTargetModel is a model at 100x30 with a started read_file card, id 1, that has no
// target, as the Anthropic start event leaves it.
func toolTargetModel(t *testing.T) Model {
	t.Helper()
	m := drive(New(capOptions(t, false, true)), tea.WindowSizeMsg{Width: 100, Height: 30})
	return drive(m, ToolStartedMsg{ID: 1, Name: "read_file"})
}

// toolCardOf is the tool card the model's tool ID names, or nil.
func toolCardOf(m Model, id int64) *ToolCard {
	cardID, ok := m.toolCards[id]
	if !ok {
		return nil
	}
	it, _, found := m.tr.Find(cardID)
	if !found || it.Kind != KindTool {
		return nil
	}
	return it.Tool
}

// openToolModal presses Shift+↑ and d on a model whose last item is a finished card.
func openToolModal(t *testing.T, m Model) *ModalState {
	t.Helper()
	m = drive(m, keyType(tea.KeyShiftUp), key('d'))
	if m.modal == nil {
		t.Fatal("d opened no modal on the finished card")
	}
	return m.modal
}

// TestToolTargetMsgGivesARunningCardItsTarget: entry point Update with ToolStartedMsg
// that has no target, then ToolTargetMsg, then ToolCompletedMsg with no target of its
// own. The card's target is empty after the start and the path after the ToolTargetMsg,
// and the head the frame draws names the path before the tool has finished. The modal
// opened on the finished card has the path and a lexer for it, and a path with no
// lexer stays plain. Controls: before the ToolTargetMsg the frame's head does not name
// the path, and after the completion of a card that never got a target the modal has
// no path and no lexer, so the assertions can see an absence.
func TestToolTargetMsgGivesARunningCardItsTarget(t *testing.T) {
	const path = "calc/usage.go"
	m := toolTargetModel(t)
	if got := toolCardOf(m, 1).Target; got != "" {
		t.Fatalf("control: the card made from a start with no target has target %q", got)
	}
	if strings.Contains(m.View(), path) {
		t.Fatalf("control: the frame names %q before any message carried it", path)
	}

	m = drive(m, ToolTargetMsg{ID: 1, Target: path})
	if got := toolCardOf(m, 1).Target; got != path {
		t.Errorf("after the ToolTargetMsg the card's target is %q, want %q", got, path)
	}
	if !strings.Contains(m.View(), "read_file "+path) {
		t.Errorf("the head of the running card does not name the path:\n%s", m.View())
	}

	m = drive(m, ToolCompletedMsg{ID: 1, Name: "read_file", OK: true, Summary: "lines 1-1", Content: "1\tpackage calc\n"})
	if got := toolTargetOf(m); got != path {
		t.Errorf("a completion with no target of its own changed the card's target to %q, want %q", got, path)
	}
	md := openToolModal(t, m)
	if md.Path != path || !md.ReadFile {
		t.Errorf("the modal has Path %q ReadFile %v, want %q and true", md.Path, md.ReadFile, path)
	}
	if md.highlight("").lex == nil {
		t.Error("the modal for a .go path has no lexer")
	}

	plain := drive(toolTargetModel(t), ToolTargetMsg{ID: 1, Target: "notes.nosuchlanguage"},
		ToolCompletedMsg{ID: 1, Name: "read_file", OK: true, Summary: "lines 1-1", Content: "1\ta\n"})
	if md := openToolModal(t, plain); md.highlight("").lex != nil {
		t.Error("a path with no lexer was given one")
	}

	never := drive(toolTargetModel(t),
		ToolCompletedMsg{ID: 1, Name: "read_file", OK: true, Summary: "lines 1-1", Content: "1\ta\n"})
	if md := openToolModal(t, never); md.Path != "" || md.highlight("").lex != nil {
		t.Errorf("control: a card that never got a target opened a modal with Path %q and lexer %v", md.Path, md.highlight("").lex)
	}
}

// toolTargetOf is the target of the only tool card in the transcript.
func toolTargetOf(m Model) string {
	for _, it := range m.tr.Items() {
		if it.Kind == KindTool {
			return it.Tool.Target
		}
	}
	return "(no tool card)"
}

// TestToolCompletedMsgAdoptsItsTarget: entry point Update with ToolStartedMsg that has no
// target and a ToolCompletedMsg that has one, with no ToolTargetMsg between, which is
// the sequence when the call's end event was not seen. The card takes the completion's
// target, and the modal opened on it has that path and a lexer. Controls: a completion
// whose target is empty leaves the card's own target alone (the ToolTargetMsg's), and
// a completion for a card with the same target changes nothing.
func TestToolCompletedMsgAdoptsItsTarget(t *testing.T) {
	const path = "calc/usage.go"
	done := ToolCompletedMsg{ID: 1, Name: "read_file", Target: path, OK: true, Summary: "lines 1-1", Content: "1\tpackage calc\n"}

	m := drive(toolTargetModel(t), done)
	if got := toolTargetOf(m); got != path {
		t.Errorf("the finished card's target is %q, want the completion's %q", got, path)
	}
	if md := openToolModal(t, m); md.Path != path || md.highlight("").lex == nil {
		t.Errorf("the modal has Path %q and lexer %v, want %q and a lexer", md.Path, md.highlight("").lex, path)
	}

	empty := done
	empty.Target = ""
	kept := drive(toolTargetModel(t), ToolTargetMsg{ID: 1, Target: path}, empty)
	if got := toolTargetOf(kept); got != path {
		t.Errorf("control: a completion with no target changed the card's target to %q, want %q", got, path)
	}
	same := drive(toolTargetModel(t), ToolTargetMsg{ID: 1, Target: path}, done)
	if got := toolTargetOf(same); got != path {
		t.Errorf("control: a completion with the card's own target changed it to %q", got)
	}
}

// TestToolTargetMsgChangesOnlyTheRunningCardItNames: entry point Update with
// ToolTargetMsg for an ID no card has, with an empty target, and for a card that has
// finished. Each is stopped by applyToolTarget's own guard: the unknown ID adds
// no card and changes no card; the empty target does not erase the running card's; and
// the finished card keeps its target because applyToolResult removed its ID from
// toolCards, so the late message is an unknown ID too and never reaches MutateTool's
// terminal-state guard (TestToolTargetMsgLeavesACancelledCardAlone holds that guard).
// Control: the same message for the running card's own ID changes its target.
func TestToolTargetMsgChangesOnlyTheRunningCardItNames(t *testing.T) {
	m := toolTargetModel(t)
	before := m.tr.Len()
	m = drive(m, ToolTargetMsg{ID: 99, Target: "other.go"})
	if m.tr.Len() != before || toolTargetOf(m) != "" {
		t.Errorf("a ToolTargetMsg for an unknown ID left %d items (was %d) and target %q", m.tr.Len(), before, toolTargetOf(m))
	}

	m = drive(m, ToolTargetMsg{ID: 1, Target: "first.go"}, ToolTargetMsg{ID: 1, Target: ""})
	if got := toolTargetOf(m); got != "first.go" {
		t.Errorf("an empty ToolTargetMsg changed the target to %q, want first.go", got)
	}

	done := drive(m, ToolCompletedMsg{ID: 1, Name: "read_file", OK: true, Summary: "lines 1-1", Content: "1\ta\n"})
	card := toolCardOf(done, 1)
	if card != nil {
		t.Fatal("setup: a finished card is still in toolCards")
	}
	for _, it := range done.tr.Items() {
		if it.Kind == KindTool && it.Tool.State != StateOK {
			t.Fatalf("setup: the card is %v, want StateOK", it.Tool.State)
		}
	}
	late := drive(done, ToolTargetMsg{ID: 1, Target: "late.go"})
	if got := toolTargetOf(late); got != "first.go" {
		t.Errorf("a ToolTargetMsg after the completion changed the target to %q, want first.go", got)
	}
}

// escapeOnly is a target a model could write that holds nothing drawable: a CSI colour
// sequence, an OSC 52 clipboard write ending in BEL and a CSI reset. It is not empty,
// and SanitizeSingleLine makes it empty.
const escapeOnly = "\x1b[31m\x1b]52;c;ZXZpbA==\x07\x1b[0m"

// TestEscapeOnlyTargetKeepsTheCardsTarget: entry point Update with ToolStartedMsg that
// has no target and a ToolTargetMsg that gives the card a path, then, per subtest, one
// message whose target is escapeOnly: a second ToolTargetMsg (stopped by
// applyToolTarget's empty check on the sanitised target) or a ToolCompletedMsg (stopped
// by applyToolResult's check on the sanitised target). The card keeps the path, and the
// frame's head still names it. The emptiness each guard tests is the sanitised value's,
// not the raw message's, which an empty raw Target cannot tell apart; so the setup
// asserts escapeOnly is non-empty and sanitises to empty. Control: before the
// escape-only message the card's target is the path and the head names it, so the
// assertions after it can see the path go.
func TestEscapeOnlyTargetKeepsTheCardsTarget(t *testing.T) {
	const path = "calc/usage.go"
	if escapeOnly == "" || SanitizeSingleLine(escapeOnly) != "" {
		t.Fatalf("setup: escapeOnly sanitises to %q, want it non-empty raw and empty sanitised", SanitizeSingleLine(escapeOnly))
	}
	cases := []struct {
		name string
		msg  tea.Msg
	}{
		{"ToolTargetMsg", ToolTargetMsg{ID: 1, Target: escapeOnly}},
		{"ToolCompletedMsg", ToolCompletedMsg{
			ID: 1, Name: "read_file", Target: escapeOnly, OK: true, Summary: "lines 1-1", Content: "1\tpackage calc\n",
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := drive(toolTargetModel(t), ToolTargetMsg{ID: 1, Target: path})
			if got := toolTargetOf(m); got != path {
				t.Fatalf("control: the card's target is %q before the escape-only message, want %q", got, path)
			}
			if !strings.Contains(m.View(), "read_file "+path) {
				t.Fatalf("control: the head does not name %q before the escape-only message:\n%s", path, m.View())
			}
			m = drive(m, tc.msg)
			if got := toolTargetOf(m); got != path {
				t.Errorf("an escape-only %s target changed the card's target to %q, want %q", tc.name, got, path)
			}
			if !strings.Contains(m.View(), "read_file "+path) {
				t.Errorf("after an escape-only %s target the head does not name %q:\n%s", tc.name, path, m.View())
			}
		})
	}
}

// TestToolTargetMsgLeavesACancelledCardAlone: entry point Update with ToolStartedMsg and
// a ToolTargetMsg, then Model.cancelTurn called directly (what Esc on a busy turn runs),
// then a late ToolTargetMsg. cancelTurn marks the running card cancelled and, unlike a
// completion, leaves its ID in toolCards, so the late message passes applyToolTarget's
// own guard and is stopped only by MutateTool's terminal-state guard: the cancelled card
// keeps its target (ui-spec §3.8, a card is never rewritten once terminal). The setup
// asserts the ID is still in toolCards and the card is cancelled, so the test is known
// to reach that guard. Control: the first ToolTargetMsg, sent while the card runs,
// changes the target.
func TestToolTargetMsgLeavesACancelledCardAlone(t *testing.T) {
	m := drive(toolTargetModel(t), ToolTargetMsg{ID: 1, Target: "first.go"})
	if got := toolTargetOf(m); got != "first.go" {
		t.Fatalf("control: the ToolTargetMsg for the running card left target %q, want first.go", got)
	}
	m = m.cancelTurn()
	if _, ok := m.toolCards[1]; !ok {
		t.Fatal("setup: cancelTurn removed the card's ID from toolCards, so the late message would stop at the unknown-ID guard")
	}
	for _, it := range m.tr.Items() {
		if it.Kind == KindTool && it.Tool.State != StateCancelled {
			t.Fatalf("setup: the card is %v after cancelTurn, want StateCancelled", it.Tool.State)
		}
	}
	m = drive(m, ToolTargetMsg{ID: 1, Target: "late.go"})
	if got := toolTargetOf(m); got != "first.go" {
		t.Errorf("a ToolTargetMsg after the card was cancelled changed the target to %q, want first.go", got)
	}
}

// hostileTarget is a target a model could write: a CSI colour sequence, an OSC 52
// clipboard write ending in BEL, a newline and a carriage return. hostileName is a
// tool name with a screen-clearing CSI, an OSC 52 write and a newline.
const (
	hostileTarget = "evil\x1b[31mred\x1b[0m\x1b]52;c;ZXZpbA==\x07\nnext\rcr.go"
	hostileName   = "rogue\x1b[2J\x1b]52;c;ZXZpbA==\x07\ntool"
)

// rawLeak names the first thing in s that a sanitised one-row field must not hold: an
// escape, a BEL, a newline, a carriage return or the OSC 52 payload. It is "" when s
// holds none of them.
func rawLeak(s string) string {
	for _, bad := range []string{"\x1b", "\x07", "\n", "\r", "52;c;"} {
		if strings.Contains(s, bad) {
			return fmt.Sprintf("%q", bad)
		}
	}
	return ""
}

// onlyToolCard is the transcript's only tool card; the test fails when there is not
// exactly one.
func onlyToolCard(t *testing.T, m Model) *ToolCard {
	t.Helper()
	var cards []*ToolCard
	for _, it := range m.tr.Items() {
		if it.Kind == KindTool {
			cards = append(cards, it.Tool)
		}
	}
	if len(cards) != 1 {
		t.Fatalf("setup: the transcript has %d tool cards, want 1", len(cards))
	}
	return cards[0]
}

// headSite is one place the tui stores a tool card's head text. send builds the
// messages that put raw through that site alone, field reads what the site stored, and
// running says whether the card is still running afterwards (a running card is finished
// with a clean completion before its modal is opened).
type headSite struct {
	name    string
	send    func(raw string) []tea.Msg
	field   func(c *ToolCard) string
	running bool
	target  bool // the site stores the target, which the modal's title and Path take
	isName  bool // the site stores the name, which hostileName is sent through
}

// headSites lists every place Update stores a tool card's name, target or summary.
func headSites() []headSite {
	done := func(id int64) ToolCompletedMsg {
		return ToolCompletedMsg{ID: id, Name: "read_file", OK: true, Summary: "lines 1-1", Content: "1\tpackage calc\n"}
	}
	name := func(c *ToolCard) string { return c.Name }
	target := func(c *ToolCard) string { return c.Target }
	summary := func(c *ToolCard) string { return c.Summary }
	return []headSite{
		{"ToolStartedMsg target", func(raw string) []tea.Msg {
			return []tea.Msg{ToolStartedMsg{ID: 1, Name: "read_file", Target: raw}}
		}, target, true, true, false},
		{"ToolStartedMsg name", func(raw string) []tea.Msg {
			return []tea.Msg{ToolStartedMsg{ID: 1, Name: raw}}
		}, name, true, false, true},
		{"ToolTargetMsg", func(raw string) []tea.Msg {
			return []tea.Msg{ToolStartedMsg{ID: 1, Name: "read_file"}, ToolTargetMsg{ID: 1, Target: raw}}
		}, target, true, true, false},
		{"ToolCompletedMsg target on a running card", func(raw string) []tea.Msg {
			d := done(1)
			d.Target = raw
			return []tea.Msg{ToolStartedMsg{ID: 1, Name: "read_file"}, d}
		}, target, false, true, false},
		{"ToolCompletedMsg name for a card it makes", func(raw string) []tea.Msg {
			d := done(7)
			d.Name = raw
			return []tea.Msg{d}
		}, name, false, false, true},
		{"ToolCompletedMsg summary", func(raw string) []tea.Msg {
			d := done(1)
			d.Summary = raw
			return []tea.Msg{ToolStartedMsg{ID: 1, Name: "read_file"}, d}
		}, summary, false, false, false},
		{"ToolCompletedMsg error message", func(raw string) []tea.Msg {
			return []tea.Msg{ToolStartedMsg{ID: 1, Name: "read_file"}, ToolCompletedMsg{
				ID: 1, Name: "read_file", OK: false, ErrorKind: "file_not_found", ErrorMsg: raw,
			}}
		}, summary, false, false, false},
		{"ToolCompletedMsg cancelled message", func(raw string) []tea.Msg {
			return []tea.Msg{ToolStartedMsg{ID: 1, Name: "read_file"}, ToolCompletedMsg{
				ID: 1, Name: "read_file", OK: false, ErrorKind: "cancelled", ErrorMsg: raw,
			}}
		}, summary, false, false, false},
		{"TranscriptLoadedMsg target", func(raw string) []tea.Msg {
			return []tea.Msg{TranscriptLoadedMsg{Items: []Item{{Kind: KindTool, Tool: &ToolCard{
				Name: "read_file", Target: raw, State: StateOK, Out: []string{"1\tpackage calc"},
			}}}}}
		}, target, false, true, false},
		{"TranscriptLoadedMsg name", func(raw string) []tea.Msg {
			return []tea.Msg{TranscriptLoadedMsg{Items: []Item{{Kind: KindTool, Tool: &ToolCard{
				Name: raw, State: StateOK,
			}}}}}
		}, name, false, false, true},
		{"TranscriptLoadedMsg summary", func(raw string) []tea.Msg {
			return []tea.Msg{TranscriptLoadedMsg{Items: []Item{{Kind: KindTool, Tool: &ToolCard{
				Name: "read_file", Summary: raw, State: StateOK,
			}}}}}
		}, summary, false, false, false},
	}
}

// TestToolCardHeadIsSanitisedWhereItIsStored: entry point Update, once per place it
// stores a tool card's name, target or summary (headSites), with model-written text
// that carries a CSI sequence, an OSC 52 clipboard write, a newline and a carriage
// return. At each site the stored field holds none of them and equals
// SanitizeSingleLine of the input; the colour-off frame holds no escape at all, and
// draws the field's visible part ("cr.go" for a target or summary, "rogue tool" for a
// name), so the absence is of the raw bytes and not of the card. For a name stored
// from ToolStartedMsg the busy verb in the status bar is the sanitised name too. On
// the finished card, the modal's title and, for a target, its Path hold none of them.
// Positive control through the same sites: an ordinary path, "src/naïve file.go", is
// stored byte for byte, and where the site stores the target the modal's Path is that
// path, ReadFile is set and the modal has a lexer, so highlighting by extension still
// works.
func TestToolCardHeadIsSanitisedWhereItIsStored(t *testing.T) {
	const ordinary = "src/naïve file.go"
	newModel := func() Model {
		return drive(New(capOptions(t, false, true)), tea.WindowSizeMsg{Width: 100, Height: 30})
	}
	finish := func(m Model, s headSite) Model {
		if s.running {
			return drive(m, ToolCompletedMsg{ID: 1, Name: "read_file", OK: true, Summary: "lines 1-1", Content: "1\tpackage calc\n"})
		}
		return m
	}
	for _, s := range headSites() {
		t.Run(s.name, func(t *testing.T) {
			raw, visible := hostileTarget, "cr.go"
			if s.isName {
				raw, visible = hostileName, "rogue tool"
			}
			m := drive(newModel(), s.send(raw)...)
			got := s.field(onlyToolCard(t, m))
			if leak := rawLeak(got); leak != "" || got != SanitizeSingleLine(raw) {
				t.Errorf("the stored field is %q (holds %s), want %q", got, leak, SanitizeSingleLine(raw))
			}
			frame := m.View()
			if strings.Contains(frame, "\x1b") || strings.Contains(frame, "\x07") {
				t.Errorf("the colour-off frame holds an escape or a BEL:\n%q", frame)
			}
			if !strings.Contains(frame, visible) {
				t.Errorf("control: the frame does not draw the field's visible part %q, so its absence of raw bytes proves nothing:\n%s", visible, frame)
			}
			if s.isName && s.running && m.busy.Verb != SanitizeSingleLine(raw) {
				t.Errorf("the busy verb is %q, want the sanitised name %q", m.busy.Verb, SanitizeSingleLine(raw))
			}
			md := openToolModal(t, finish(m, s))
			if leak := rawLeak(md.Title); leak != "" {
				t.Errorf("the modal title %q holds %s", md.Title, leak)
			}
			if leak := rawLeak(md.Path); leak != "" {
				t.Errorf("the modal Path %q holds %s", md.Path, leak)
			}

			if !s.target {
				return
			}
			ok := finish(drive(newModel(), s.send(ordinary)...), s)
			if got := onlyToolCard(t, ok).Target; got != ordinary {
				t.Errorf("control: the ordinary path was stored as %q, want %q byte for byte", got, ordinary)
			}
			md = openToolModal(t, ok)
			if md.Path != ordinary || !md.ReadFile || md.highlight("").lex == nil {
				t.Errorf("control: the modal for %q has Path %q ReadFile %v lexer %v, want the path, true and a lexer",
					ordinary, md.Path, md.ReadFile, md.highlight("").lex)
			}
		})
	}
}
