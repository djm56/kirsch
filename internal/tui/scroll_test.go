package tui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
	"github.com/muesli/cancelreader"
)

// Coverage for ui-spec §2.4's "↓ n new" indicator.
//
// The renderer for it was golden-verified from the start — screen 09 draws it —
// but the golden reaches that state by assigning Scroll{NewSince: 3} directly,
// so it proved only that the overlay draws. Nothing incremented the field at
// runtime, which made the indicator unreachable from any sequence of keys and
// messages while every test still passed. These tests drive it the way the
// program does.

// scrolledUp returns a driven model parked above the bottom of the transcript.
//
// `y` resolves the fixture's pending approval, which otherwise captures every
// key exclusively; ↑ moves focus to the transcript; PgUp unpins. The viewport is
// deliberately short, so the fixture overflows it and there is somewhere to
// scroll to.
func scrolledUp(t *testing.T) Model {
	t.Helper()
	m := drive(newDrivenSize(t, 80, 12), key('y'), keyType(tea.KeyUp), keyType(tea.KeyPgUp))
	if m.scroll.Pinned {
		t.Fatalf("setup: PgUp did not unpin (%d lines, viewport %d)",
			len(m.lines), m.layout().TranscriptH)
	}
	if m.scroll.NewSince != 0 {
		t.Fatalf("setup: NewSince = %d straight after unpinning, want 0", m.scroll.NewSince)
	}
	return m
}

// indicator is the exact string transcriptRows overlays on the bottom row.
func indicator(m Model, n int) string { return m.gly.New + " " + itoa(n) + " new" }

// hasIndicator reports whether any form of the indicator is on the band. It
// looks for the glyph plus its separating space rather than for a count, so it
// can assert absence without knowing what the count would have been. Nothing
// else in a transcript band draws that glyph.
func hasIndicator(m Model, band string) bool {
	return strings.Contains(band, m.gly.New+" ")
}

// TestNewSinceCountsArrivalsWhileUnpinned is the half of §2.4 that was missing
// entirely: scrolling up unpins, and blocks arriving after that are counted and
// announced bottom-right.
func TestNewSinceCountsArrivalsWhileUnpinned(t *testing.T) {
	m := drive(scrolledUp(t), NoticeMsg{Text: "first"})
	if m.scroll.NewSince != 1 {
		t.Fatalf("one block arrived while unpinned: NewSince = %d, want 1", m.scroll.NewSince)
	}

	m = drive(m, NoticeMsg{Text: "second"}, NoticeMsg{Text: "third"})
	if m.scroll.NewSince != 3 {
		t.Fatalf("three blocks arrived while unpinned: NewSince = %d, want 3", m.scroll.NewSince)
	}
	if m.scroll.Pinned {
		t.Fatal("arriving content re-pinned the transcript; §2.4 keeps an unpinned view where it is")
	}

	band := transcriptBand(t, m)
	if want := indicator(m, 3); !strings.Contains(band, want) {
		t.Errorf("the transcript band does not carry %q:\n%s", want, band)
	}
}

// TestEndClearsTheNewCount is §2.4's re-pin trigger seen from the indicator's
// side: End returns to the bottom, so by definition there is nothing new below
// the fold left to point at.
func TestEndClearsTheNewCount(t *testing.T) {
	m := drive(scrolledUp(t), NoticeMsg{Text: "first"}, NoticeMsg{Text: "second"})
	if m.scroll.NewSince != 2 {
		t.Fatalf("precondition: NewSince = %d before End, want 2 — End has nothing to clear",
			m.scroll.NewSince)
	}
	if band := transcriptBand(t, m); !hasIndicator(m, band) {
		t.Fatalf("precondition: the indicator is not on screen before End:\n%s", band)
	}

	m = drive(m, keyType(tea.KeyEnd))
	if m.scroll.NewSince != 0 {
		t.Errorf("End left NewSince = %d, want 0", m.scroll.NewSince)
	}
	if !m.scroll.Pinned {
		t.Error("End did not re-pin the transcript")
	}
	if band := transcriptBand(t, m); hasIndicator(m, band) {
		t.Errorf("End left the indicator drawn on the band:\n%s", band)
	}
}

// TestArrivalsWhilePinnedDoNotCount is the other direction, and the one a
// bare `NewSince++` on every append would break: a pinned viewport follows the
// content down, so nothing is ever unread and the indicator must stay away.
func TestArrivalsWhilePinnedDoNotCount(t *testing.T) {
	m := drive(newDrivenSize(t, 80, 12), key('y'))
	if !m.scroll.Pinned {
		t.Fatal("precondition: the fixture did not start pinned")
	}

	m = drive(m, NoticeMsg{Text: "first"}, NoticeMsg{Text: "second"})
	if m.scroll.NewSince != 0 {
		t.Errorf("blocks arriving while pinned raised NewSince to %d, want 0", m.scroll.NewSince)
	}
	if !m.scroll.Pinned {
		t.Error("blocks arriving while pinned unpinned the transcript")
	}
	if band := transcriptBand(t, m); hasIndicator(m, band) {
		t.Errorf("the indicator was drawn over a pinned transcript:\n%s", band)
	}
}

// TestToolCardTransitionIsNotANewBlock pins the judgement in appendBlock's
// docblock: §2.4 counts blocks that *arrive*, so a running card becoming a
// finished one is not a second block. Counting it would have the indicator
// promise more unread blocks below the fold than are there, and the user would
// scroll down to find fewer things than they were told about.
func TestToolCardTransitionIsNotANewBlock(t *testing.T) {
	m := drive(scrolledUp(t), ToolStartedMsg{ID: 7, Name: "run_command", Target: "go test ./..."})
	if m.scroll.NewSince != 1 {
		t.Fatalf("a tool card arriving while unpinned: NewSince = %d, want 1", m.scroll.NewSince)
	}

	m = drive(m, ToolCompletedMsg{
		ID: 7, Name: "run_command", Target: "go test ./...",
		OK: true, Summary: "ok",
	})
	if m.scroll.NewSince != 1 {
		t.Errorf("the same card reaching a terminal state counted as a second block: NewSince = %d, want 1",
			m.scroll.NewSince)
	}

	// A result whose start was never seen has no card to fold into, so
	// applyToolResult appends a fresh one — and that really is an arrival.
	m = drive(m, ToolCompletedMsg{ID: 99, Name: "read_file", Target: "calc/divide.go", OK: true})
	if m.scroll.NewSince != 2 {
		t.Errorf("an unmatched tool result appended a card without counting it: NewSince = %d, want 2",
			m.scroll.NewSince)
	}
}

// TestAppendBlockIsTheOnlyAppendPath makes appendBlock's claim on the count
// mechanical instead of conventional.
//
// A handler that calls t.Append directly compiles, renders, and passes every
// golden — it just never raises the count, so the indicator stays invisible for
// that one kind of block. There is no symptom a behavioural test would catch
// unless somebody thought to write one for the new path, and that is exactly
// the thought that gets skipped. Hence a structural check rather than more
// cases: this one covers the paths nobody has written yet.
//
// Matching rule: every non-test .go file in this directory is parsed, and any
// call whose function is a selector ending in `.Append(` is attributed to the
// top-level declaration that encloses it — the function for a func declaration,
// and the declared name for anything else. Everything except appendBlock is an
// error. The receiver is deliberately not inspected — an earlier version
// required it to be a selector named `tr`, which pinned the check to the single
// spelling `m.tr.Append(...)` and let a local alias (`tr := &m.tr;
// tr.Append(...)`), a helper method on Transcript, and a `Transcript.AppendRaw`
// wrapper all through untouched. The last of those reintroduced a known defect
// on the same untested path under a different name.
//
// Every top-level declaration is walked, not only the func declarations. An
// earlier version walked `*ast.FuncDecl` alone, so an append inside a
// package-level `var` initialised by a function literal was invisible to it —
// and that idiom is already live in this package, in fake.go. A gap a reader can
// find by looking at the file next to the test is the kind that gets used.
//
// What it does NOT catch:
//   - Appends that never spell `Append` — a differently named method on
//     Transcript that does the same job (`Add`, `Push`, `emit`), or direct
//     manipulation of `t.items`, which is unexported but reachable from
//     anywhere in the package.
//   - Appends reached through a function value or interface, where the
//     selector is not resolvable in the AST.
//   - Appends from another package, and appends in _test.go files: both are
//     outside the walk by construction.
//   - Whether appendBlock itself still increments the count. That is
//     behaviour, and TestNewSinceCountsArrivalsWhileUnpinned owns it.
//
// It is a spelling check with one true positive in the tree, not a proof that
// the count cannot be bypassed. Its value is that the obvious bypasses — the
// ones someone reaches for without thinking about the indicator — are all
// spelled `Append`.
func TestAppendBlockIsTheOnlyAppendPath(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package directory: %v", err)
	}

	fset := token.NewFileSet()
	callers := map[string][]string{} // enclosing declaration -> call positions
	scanned := 0

	// collect attributes every `.Append(` call under node to owner. Called once
	// per top-level declaration, so a call outside every function still lands
	// under the declaration that carries it rather than being skipped.
	collect := func(owner string, node ast.Node) {
		ast.Inspect(node, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			// Matches a call to any method named Append, on any receiver.
			// See the docblock for what that does and does not cover.
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Append" {
				return true
			}
			callers[owner] = append(callers[owner], fset.Position(call.Pos()).String())
			return true
		})
	}

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		scanned++

		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Body == nil {
					continue // external or assembly body: nothing to walk
				}
				collect(d.Name.Name, d.Body)
			case *ast.GenDecl:
				// var, const and type declarations. Their initialisers can
				// hold function literals, and a literal is a body like any
				// other.
				for _, spec := range d.Specs {
					collect(declName(d.Tok.String(), spec), spec)
				}
			}
		}
	}

	if scanned == 0 {
		t.Fatal("no non-test .go file was scanned; the walk is wrong and this test proves nothing")
	}
	if len(callers) == 0 {
		t.Fatalf("scanned %d file(s) and found no `.Append(` call at all; either the "+
			"method has been renamed or the only caller has gone, and this test is now vacuous",
			scanned)
	}

	for fn, at := range callers {
		if fn == "appendBlock" {
			continue
		}
		t.Errorf("%s calls Append directly at %s; route it through Model.appendBlock so the "+
			"block is counted against the \"↓ n new\" indicator (ui-spec §2.4)",
			fn, strings.Join(at, ", "))
	}
}

// declName labels a non-func declaration for the failure message.
//
// tok is the declaration keyword ("var", "const", "type", "import") and spec is
// one spec from that declaration. The result is a human-readable owner such as
// `var fakeTestOutput`, or the keyword alone when the spec carries no name that
// helps. It never returns "appendBlock", so a declaration can never be mistaken
// for the one allowed caller.
func declName(tok string, spec ast.Spec) string {
	switch s := spec.(type) {
	case *ast.ValueSpec:
		names := make([]string, 0, len(s.Names))
		for _, n := range s.Names {
			names = append(names, n.Name)
		}
		if len(names) > 0 {
			return tok + " " + strings.Join(names, ", ")
		}
	case *ast.TypeSpec:
		return tok + " " + s.Name.Name
	}
	return tok + " declaration"
}

// browsingUnpinned is scrolledUp, and composingUnpinned is the same viewport
// handed back to the composer.
//
// Esc is the route back because it is the only one that does not re-pin on the
// way — walking the selection past the last card returns focus too, but a
// transcript parked at the bottom is exactly the state these tests need to not
// be in. That Esc leaves the offset alone is this mission's own change, so these
// helpers stop working the moment it is reverted, which is intended.
func composingUnpinned(t *testing.T) Model {
	t.Helper()
	m := drive(scrolledUp(t), keyType(tea.KeyEsc))
	if m.mode() != ModeComposing {
		t.Fatalf("setup: mode after Esc = %v, want Composing", m.mode())
	}
	if m.scroll.Pinned {
		t.Fatal("setup: Esc re-pinned the transcript, so there is no unpinned state left to test")
	}
	return m
}

// typeCmd types a string into the composer one keypress at a time and submits.
func typeCmd(m Model, s string) Model {
	for _, r := range s {
		m = drive(m, key(r))
	}
	return drive(m, keyType(tea.KeyEnter))
}

// TestBrowsingCapitalGRePins covers `G` in the transcript pane.
//
// It is TestEndClearsTheNewCount's assertion set against the other key that
// means "go to the bottom". The two are deliberately separate tests rather than
// a table: End is a key type and G is a rune, so they reach keyBrowsing through
// different switches, and one switch losing its arm must not be hidden by the
// other still working.
func TestBrowsingCapitalGRePins(t *testing.T) {
	m := drive(scrolledUp(t), NoticeMsg{Text: "first"}, NoticeMsg{Text: "second"})
	if m.scroll.NewSince != 2 {
		t.Fatalf("precondition: NewSince = %d before G, want 2 — G has nothing to clear",
			m.scroll.NewSince)
	}
	if m.mode() != ModeBrowsing {
		t.Fatalf("precondition: mode = %v, want Browsing — G only scrolls there", m.mode())
	}

	m = drive(m, key('G'))
	if !m.scroll.Pinned {
		t.Error("G did not re-pin the transcript")
	}
	if m.scroll.NewSince != 0 {
		t.Errorf("G left NewSince = %d, want 0", m.scroll.NewSince)
	}
	if band := transcriptBand(t, m); hasIndicator(m, band) {
		t.Errorf("G left the indicator drawn on the band:\n%s", band)
	}
}

// TestBrowsingGJumpsToTop covers `g`, and asserts the offset rather than the
// pin: the top of an overflowing transcript is by definition not the bottom, so
// a `g` that quietly re-pinned would be the same defect as a `g` that did
// nothing, and only the offset separates them.
func TestBrowsingGJumpsToTop(t *testing.T) {
	m := drive(scrolledUp(t), key('G'))
	if !m.scroll.Pinned {
		t.Fatal("precondition: G did not re-pin, so g has nowhere to travel from")
	}

	m = drive(m, key('g'))
	if m.scroll.Offset != 0 {
		t.Errorf("g left the viewport at offset %d, want 0", m.scroll.Offset)
	}
	if m.scroll.Pinned {
		t.Errorf("g pinned a transcript of %d lines in a %d-line viewport; the top of an "+
			"overflowing transcript is not the bottom", len(m.lines), m.layout().TranscriptH)
	}
}

// TestGAndCapitalGAreLiteralInTheComposer is the containment half of the two
// tests above. `g` and `G` are ordinary characters while typing, and a binding
// added to the wrong switch would make the composer unusable for any word
// containing them — the same failure `?` is guarded against in keyComposing.
func TestGAndCapitalGAreLiteralInTheComposer(t *testing.T) {
	m := composingUnpinned(t)
	before := m.scroll.Offset

	m = drive(m, key('g'), key('G'))
	if v := m.comp.Value(); v != "gG" {
		t.Errorf("composer value = %q, want %q", v, "gG")
	}
	if m.scroll.Pinned || m.scroll.Offset != before {
		t.Errorf("typing moved the viewport to offset %d (pinned=%v), want offset %d unpinned",
			m.scroll.Offset, m.scroll.Pinned, before)
	}
}

// TestSlashCommandsRePin covers the operator's ruling that a slash command is a
// request, so its answer is brought on screen.
//
// Removing Esc's re-pin made "composer focused while scrolled up" reachable for
// the first time, and these four commands all answer into the transcript — so
// before this change their answer landed off-screen behind the "↓ n new"
// indicator, which is the one place a user is least likely to look for something
// they just asked for.
func TestSlashCommandsRePin(t *testing.T) {
	for _, cmd := range []string{"/status", "/files", "/compact", "/approvals"} {
		t.Run(cmd, func(t *testing.T) {
			m := composingUnpinned(t)
			before := m.tr.Len()

			m = typeCmd(m, cmd)
			if n := m.tr.Len(); n != before+1 {
				t.Fatalf("%s appended %d items, want 1: it answered with nothing, so there is "+
					"no answer here that could have been left off-screen", cmd, n-before)
			}
			if !m.scroll.Pinned {
				t.Errorf("%s did not re-pin; its answer is below the fold behind the "+
					"\"↓ n new\" indicator", cmd)
			}
			if band := transcriptBand(t, m); hasIndicator(m, band) {
				t.Errorf("%s left the new-content indicator on the band:\n%s", cmd, band)
			}
		})
	}
}

// TestHintOnlySlashCommandsDoNotRePin is the boundary of the rule above.
//
// Neither of these puts anything in the transcript: the answer is a dim hint in
// the composer, which is on screen wherever the viewport happens to be. Re-pinning
// for them would spend the reader's scroll position to report a typo, so the
// re-pin sits on runSlash's shared tail and both of these return before it.
func TestHintOnlySlashCommandsDoNotRePin(t *testing.T) {
	cases := []struct {
		name, cmd, wantHint string
		wire                bool // set RunTool, so the debug arm reaches its usage check
	}{
		{name: "unknown command", cmd: "/nosuchthing", wantHint: "unknown command /nosuchthing"},
		{name: "debug command missing its argument", cmd: "/read", wantHint: "usage: /read <path>", wire: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := composingUnpinned(t)
			if tc.wire {
				// Left nil, /read takes the "tools are not wired up" arm instead,
				// which answers with a notice and so re-pins like any other answer.
				m.RunTool = func(string, map[string]any) {}
			}
			before, offset := m.tr.Len(), m.scroll.Offset

			m = typeCmd(m, tc.cmd)
			if n := m.tr.Len(); n != before {
				t.Fatalf("%s appended %d item(s); it is no longer a hint-only path and this "+
					"test is guarding the wrong thing", tc.cmd, n-before)
			}
			if m.comp.Hint != tc.wantHint {
				t.Fatalf("hint = %q, want %q", m.comp.Hint, tc.wantHint)
			}
			if m.scroll.Pinned {
				t.Errorf("%s re-pinned the transcript: its answer is the composer hint %q, "+
					"which is on screen wherever the viewport is, so re-pinning only costs "+
					"the reader their place", tc.cmd, tc.wantHint)
			}
			if m.scroll.Offset != offset {
				t.Errorf("%s moved the viewport from offset %d to %d", tc.cmd, offset, m.scroll.Offset)
			}
		})
	}
}

// TestAHintKeepsThePinnedViewportAtTheBottom is the other half of the hint-only
// paths: they leave the viewport alone, and they change the geometry under it.
//
// A hint takes a row of the composer, so the transcript loses one. While pinned,
// the offset is derived from the viewport height — shorter viewport, larger
// offset — and the derivation happens in relayout, which neither hint-only path
// reaches. View computes its layout fresh, so without dispatchKey's layout
// comparison the frame is drawn one row short of where the model says it is: the
// bottom line of the transcript falls below the fold while Pinned still reads
// true, which is the one thing Pinned is supposed to rule out.
//
// Everything about it generalises past hints. Any handler that changes the
// composer's height and does not relayout leaves the same gap, which is why the
// fix is a comparison in dispatchKey rather than two extra calls in runSlash.
func TestAHintKeepsThePinnedViewportAtTheBottom(t *testing.T) {
	// `y` resolves the fixture's pending approval, which otherwise captures
	// every key. The viewport is deliberately short so the fixture overflows it.
	m := drive(newDrivenSize(t, 80, 12), key('y'))
	if !m.scroll.Pinned {
		t.Fatal("setup: not pinned, so there is no derived offset to get wrong")
	}
	before := m.layout()
	if len(m.lines) <= before.TranscriptH {
		t.Fatalf("setup: %d transcript lines fit a %d-row viewport, so nothing is below the "+
			"fold and the defect has nowhere to show", len(m.lines), before.TranscriptH)
	}

	m = typeCmd(m, "/nosuchthing")

	if m.comp.Hint == "" {
		t.Fatal("setup: no hint was set, so the composer never grew")
	}
	after := m.layout()
	if after.TranscriptH != before.TranscriptH-1 {
		t.Fatalf("setup: the hint row moved the transcript from %d rows to %d, want %d; this "+
			"test is guarding the wrong geometry", before.TranscriptH, after.TranscriptH,
			before.TranscriptH-1)
	}
	if !m.scroll.Pinned {
		t.Fatal("/nosuchthing unpinned the transcript; TestHintOnlySlashCommandsDoNotRePin owns that")
	}
	if want := maxInt(0, len(m.lines)-after.TranscriptH); m.scroll.Offset != want {
		t.Errorf("offset = %d, want %d: the hint shortened the transcript from %d rows to %d "+
			"and the offset was left measured against the taller one, so the last line is "+
			"drawn below the fold while the model reports Pinned",
			m.scroll.Offset, want, before.TranscriptH, after.TranscriptH)
	}
}

// TestModalHomeAndEndScrollLikeGAndCapitalG covers the pair that only started
// arriving once NormalizeInput was wired up.
//
// g/G exist in a modal because Home and End could not be relied on to arrive at
// all; now that they do, a key that jumps to the top of the transcript and does
// nothing in a modal reads as a broken modal. The two are asserted against the
// g/G outcomes rather than against literal offsets so that the pair cannot drift
// apart from the gesture they mirror.
func TestModalHomeAndEndScrollLikeGAndCapitalG(t *testing.T) {
	open := func(t *testing.T) Model {
		t.Helper()
		// `y` resolves the fixture's pending approval, which captures every key
		// including the one that opens a modal. A short frame is what makes the
		// help overlay taller than its viewport and so scrollable.
		m := typeCmd(drive(newDrivenSize(t, 80, 14), key('y')), "/help")
		if m.modal == nil {
			t.Fatal("setup: /help did not open the help modal")
		}
		if len(m.modal.Lines) < 2 {
			t.Fatalf("setup: the help modal has %d line(s); there is nowhere to scroll",
				len(m.modal.Lines))
		}
		return m
	}

	t.Run("End matches G", func(t *testing.T) {
		g := drive(open(t), key('G'))
		end := drive(open(t), keyType(tea.KeyEnd))
		if g.modal.Off == 0 {
			t.Fatalf("setup: G left the modal at offset 0, so End matching it proves nothing")
		}
		if end.modal.Off != g.modal.Off {
			t.Errorf("End left the modal at offset %d, G at %d; End is inert in modals while "+
				"working in the transcript", end.modal.Off, g.modal.Off)
		}
	})

	t.Run("Home matches g", func(t *testing.T) {
		scrolled := drive(open(t), key('G'))
		if scrolled.modal.Off == 0 {
			t.Fatal("setup: G left the modal at offset 0, so returning to the top proves nothing")
		}
		if home := drive(scrolled, keyType(tea.KeyHome)); home.modal.Off != 0 {
			t.Errorf("Home left the modal at offset %d, want 0", home.modal.Off)
		}
	})
}

// TestNormalizeSS3 covers the byte rewriting that makes Home and End reach the
// program at all on terminals whose terminfo encodes them as SS3.
//
// The join between this test and the running program is Bubble Tea's key table:
// it carries `\x1b[H` and `\x1b[F` and does not carry `\x1bOH` or `\x1bOF`, so
// rewriting the SS3 forms into the CSI ones is what turns one keypress into a
// tea.KeyHome or tea.KeyEnd instead of an alt+O followed by a stray rune. The
// arrow cases are the control: they are SS3 too, Bubble Tea already decodes
// them, and rewriting them would break keys that currently work.
func TestNormalizeSS3(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"SS3 Home becomes CSI Home", "\x1bOH", "\x1b[H"},
		{"SS3 End becomes CSI End", "\x1bOF", "\x1b[F"},
		{"alt-prefixed SS3 Home keeps its prefix", "\x1b\x1bOH", "\x1b\x1b[H"},
		{"two sequences in one read", "\x1bOH\x1bOF", "\x1b[H\x1b[F"},
		{"SS3 up is left alone", "\x1bOA", "\x1bOA"},
		{"SS3 down is left alone", "\x1bOB", "\x1bOB"},
		{"SS3 F1 is left alone", "\x1bOP", "\x1bOP"},
		{"CSI Home is already correct", "\x1b[H", "\x1b[H"},
		{"CSI End is already correct", "\x1b[F", "\x1b[F"},
		{"a bare Esc is untouched", "\x1b", "\x1b"},
		{"a truncated sequence is untouched", "\x1bO", "\x1bO"},
		{"the letters alone are not a sequence", "OH", "OH"},
		{"only the ESC-introduced run is rewritten", "\x1bOHOH", "\x1b[HOH"},
		{"typed text is untouched", "go to the end", "go to the end"},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := []byte(tc.in)
			normalizeSS3(b)
			if string(b) != tc.want {
				t.Errorf("normalizeSS3(%q) = %q, want %q", tc.in, string(b), tc.want)
			}
			if len(b) != len(tc.in) {
				t.Errorf("length changed from %d to %d; the rewrite must be in place so the "+
					"reader cannot short-read", len(tc.in), len(b))
			}
		})
	}
}

// pipeFile returns the read end of a pipe pre-loaded with s, closed for writing.
//
// A pipe rather than a strings.Reader because ss3File wraps an *os.File — the
// whole point of the type — and because it is the simplest way to get a real
// descriptor into a test without a terminal. The caller gets a reader that
// returns io.EOF after s.
func pipeFile(t *testing.T, s string) *os.File {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })
	go func() {
		defer w.Close()
		_, _ = w.Write([]byte(s))
	}()
	return r
}

// TestNormalizedInputRewritesTheStream checks the reader wrapper, and pins the
// documented gap: a sequence split across two reads is passed through unchanged
// rather than buffered. Holding a trailing ESC back to see what follows it would
// delay every bare Esc, and Esc cancels a turn, closes a modal and leaves the
// transcript pane — so the split case is left decoding exactly as it does today.
//
// ss3File is constructed directly rather than through NormalizeInput because
// NormalizeInput returns nil for anything that is not a terminal, and a pipe is
// not one. TestNormalizeInputDeclinesWhatItCannotHelp covers that half.
func TestNormalizedInputRewritesTheStream(t *testing.T) {
	r := &ss3File{File: pipeFile(t, "\x1bOH\x1bOFx")}
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if want := "\x1b[H\x1b[Fx"; string(got) != want {
		t.Errorf("stream = %q, want %q", string(got), want)
	}

	// Each Read sees only "\x1bO" then "H", so neither call holds a whole
	// sequence and nothing is rewritten. The two-byte buffer is what splits
	// them; the pipe hands over whatever is available.
	split := &ss3File{File: pipeFile(t, "\x1bOH")}
	buf := make([]byte, 2)
	n, err := split.Read(buf)
	if err != nil {
		t.Fatalf("split read: %v", err)
	}
	if string(buf[:n]) != "\x1bO" {
		t.Errorf("split read = %q, want %q unchanged", string(buf[:n]), "\x1bO")
	}
}

// TestNormalizedInputKeepsTheFileCapabilities is the regression test for the
// defect that made this whole wrapper inert.
//
// Bubble Tea does not ask an input what it can do; it type-asserts, and an
// assertion that fails picks a degraded path in silence. A first version of this
// wrapper was `struct{ r io.Reader }` with only a Read method, which compiled,
// rewrote the bytes correctly, passed every test above — and satisfied neither
// interface, so tty_unix.go never called term.MakeRaw and the terminal stayed
// canonical and echoing. Nothing in the package could see it: every other TUI
// test drives Model.Update directly and never builds a tea.Program.
//
// The checks below are the capabilities, in the order Bubble Tea reaches for
// them. `var _ term.File` and `var _ cancelreader.File` beside the type catch
// the same regression at compile time; this catches it with the failure text
// that explains what the user would have seen.
func TestNormalizedInputKeepsTheFileCapabilities(t *testing.T) {
	f := pipeFile(t, "")
	var in io.Reader = &ss3File{File: f}

	// 1. Raw mode. tty_unix.go asserts to term.File before term.MakeRaw, so
	//    failing this assertion means the terminal is never put into raw mode:
	//    line-buffered keys, characters echoed over the alt screen, Ctrl+C
	//    arriving as SIGINT instead of as a key.
	tf, ok := in.(term.File)
	if !ok {
		t.Fatalf("%T does not satisfy term.File; Bubble Tea will skip term.MakeRaw and "+
			"the terminal will stay canonical and echoing", in)
	}
	if tf.Fd() != f.Fd() {
		t.Errorf("Fd() = %d, want the wrapped file's %d; raw mode would be applied to the "+
			"wrong descriptor", tf.Fd(), f.Fd())
	}

	// 2. Cancellable reads. cancelreader.NewReader asserts to cancelreader.File
	//    — term.File plus Name — and otherwise hands back a fallback whose
	//    Cancel always reports failure, which is what makes every quit and
	//    suspend wait out Bubble Tea's read-loop timeout with the read goroutine
	//    still blocked. Cancel returning true is the behaviour, not the concrete
	//    type, so this does not pin an unexported name in someone else's
	//    package.
	cf, ok := in.(cancelreader.File)
	if !ok {
		t.Fatalf("%T does not satisfy cancelreader.File; reads will not be cancellable", in)
	}
	if cf.Name() != f.Name() {
		t.Errorf("Name() = %q, want the wrapped file's %q", cf.Name(), f.Name())
	}
	cr, err := cancelreader.NewReader(in)
	if err != nil {
		t.Fatalf("cancelreader.NewReader: %v", err)
	}
	defer cr.Close()
	if !cr.Cancel() {
		t.Errorf("Cancel() = false: cancelreader fell back to the uncancellable reader (%T), "+
			"so quitting and suspending will block until Bubble Tea's read-loop timeout", cr)
	}
}

// TestNormalizeInputDeclinesWhatItCannotHelp pins the nil contract.
//
// Returning nil is not a failure signal — it is "leave Bubble Tea's own input
// path alone". main.go must then omit the option entirely, because
// tea.WithInput(nil) means something else again: no input at all. The case that
// matters in practice is stdin redirected or piped, where Bubble Tea opens
// /dev/tty for itself and wrapping stdin would take that fallback away. A test
// process has no terminal on stdin, so os.Stdin exercises exactly that path.
func TestNormalizeInputDeclinesWhatItCannotHelp(t *testing.T) {
	if got := NormalizeInput(nil); got != nil {
		t.Errorf("NormalizeInput(nil) = %#v, want nil", got)
	}
	if term.IsTerminal(os.Stdin.Fd()) {
		t.Skip("stdin is a terminal in this run; the not-a-terminal case is unreachable here")
	}
	if got := NormalizeInput(os.Stdin); got != nil {
		t.Errorf("NormalizeInput(non-terminal stdin) = %#v, want nil so that main.go omits "+
			"tea.WithInput and Bubble Tea opens /dev/tty for itself", got)
	}
}
