package tui

import (
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Milestone 0 smoke coverage. Task 6 replaces this with the full golden and
// synthetic-key suites; until then these are the invariants worth protecting
// while the layout is still churning.

func render(t *testing.T, caps Caps, w, h int) string {
	t.Helper()
	m := New(Options{Version: "0.1.0", Caps: caps})
	mm, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return mm.(Model).View()
}

// TestFrameInvariants is the "no visual corruption" check: the frame is exactly
// as tall as the terminal and never wider, at every size in ui-spec §2.2.
func TestFrameInvariants(t *testing.T) {
	widths := []int{200, 120, 80, 79, 60, 59, 40, 39, 38, 20, 2, 1, 0, -1}
	heights := []int{60, 24, 11, 10, 9, 6, 5, 4, 2, 1, 0, -1}
	for _, w := range widths {
		for _, h := range heights {
			out := render(t, Caps{Unicode: true}, w, h)
			if w <= 0 || h <= 0 {
				if out != "" {
					t.Errorf("%dx%d: want empty string, got %d bytes", w, h, len(out))
				}
				continue
			}
			lines := strings.Split(out, "\n")
			if len(lines) != h {
				t.Errorf("%dx%d: %d rows, want %d", w, h, len(lines), h)
			}
			for i, l := range lines {
				if cw := cellWidth(l); cw > w {
					t.Errorf("%dx%d row %d: %d cols, want <= %d", w, h, i+1, cw, w)
				}
			}
		}
	}
}

// TestNoColourEmitsNoEscapes is one half of the ui-spec §13 escape property:
// with colour off, not a single escape byte is written.
func TestNoColourEmitsNoEscapes(t *testing.T) {
	for _, uni := range []bool{true, false} {
		out := render(t, Caps{Colour: false, Unicode: uni}, 80, 24)
		if i := strings.IndexByte(out, 0x1b); i >= 0 {
			t.Errorf("unicode=%v: escape byte at %d", uni, i)
		}
	}
}

// TestSanitizeStripsControlSequences covers the §7.1 transformations. The CRLF
// case is first because collapsing \r runs before normalising CRLF would reduce
// a whole CRLF file to its last line.
func TestSanitizeSequences(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"crlf", "a\r\nb", "a\nb"},
		{"crlf-mixed", "a\r\n1%\r2%\r\nb", "a\n2%\nb"},
		{"sgr", "\x1b[31mred\x1b[0m", "red"},
		{"cursor", "a\x1b[2Jb\x1b[1;1Hc", "abc"},
		{"osc", "\x1b]0;title\x07rest", "rest"},
		{"progress", "10%\r50%\r100%", "100%"},
		{"tabs", "a\tb", "a   b"},
		{"tab-leading", "\tif x {", "    if x {"},
		{"nul", "a\x00b", "a·b"},
		{"keep-newline", "a\nb", "a\nb"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Sanitize(c.in); got != c.want {
				t.Errorf("Sanitize(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestSanitizeCapsLineWidth proves a minified bundle on one line cannot hang
// the wrapper, and that the cap counts display cells rather than runes.
func TestSanitizeCapsLineWidth(t *testing.T) {
	for _, in := range []string{
		strings.Repeat("x", 5000),
		strings.Repeat("漢", 1500), // 3000 cells from 1500 runes
	} {
		got := Sanitize(in)
		if w := cellWidth(got); w > MaxRenderedLineWidth {
			t.Errorf("capped width %d, want <= %d", w, MaxRenderedLineWidth)
		}
	}
}

// TestApplyPatchNeverOffersSessionGrant is ADR 0006's rule, checked on the
// model rather than the rendering: a keyboard user must not be able to grant a
// patch even if the key row were wrong.
func TestApplyPatchNeverOffersSessionGrant(t *testing.T) {
	patch := ApprovalCard{Kind: ApprovalPatch, GrantScope: "anything"}
	if patch.OffersSessionGrant() {
		t.Error("apply_patch offered a session grant")
	}
	cmd := ApprovalCard{Kind: ApprovalCommand, GrantScope: "go test"}
	if !cmd.OffersSessionGrant() {
		t.Error("run_command with a scope did not offer a session grant")
	}
}

// TestModeDerivationPrecedence checks ui-spec §5.1's precedence, including the
// subtlety that an approval arriving during an open modal must not take capture
// until the modal closes.
func TestModeDerivationPrecedence(t *testing.T) {
	m := New(Options{Caps: Caps{Unicode: true}})
	m.pendingApproval = 0
	if got := m.mode(); got != ModeComposing {
		t.Errorf("mode = %v, want Composing", got)
	}
	m.base = BaseBrowsing
	if got := m.mode(); got != ModeBrowsing {
		t.Errorf("mode = %v, want Browsing", got)
	}
	m.confirm = &ConfirmState{}
	if got := m.mode(); got != ModeConfirm {
		t.Errorf("mode = %v, want Confirm", got)
	}
	m.pendingApproval = 1
	if got := m.mode(); got != ModeApprovalPending {
		t.Errorf("mode = %v, want ApprovalPending", got)
	}
	m.modal = &ModalState{}
	if got := m.mode(); got != ModeModal {
		t.Errorf("approval during an open modal took capture: mode = %v, want Modal", got)
	}
	m.modal = nil
	if got := m.mode(); got != ModeApprovalPending {
		t.Errorf("closing the modal did not return to the approval: mode = %v", got)
	}
}

// TestRendererEmitsPaletteSGR guards the trap that makes colour testing
// worthless if missed: lipgloss ignores a termenv profile option unless
// SetColorProfile is called, falling back to inspecting the writer. The writer
// here is io.Discard, which is not a TTY — so without the explicit call every
// style resolves to Ascii, the "coloured" output is byte-identical to the plain
// output, and every colour test passes while proving nothing.
func TestRendererEmitsPaletteSGR(t *testing.T) {
	sty := NewStyles(NewRenderer(true), true)
	got := sty.Accent("X")
	if !strings.Contains(got, "\x1b[38;5;111m") {
		t.Fatalf("accent emitted no 256-colour SGR: %q", got)
	}
}

// sgrRe matches the SGR sequences the renderer is allowed to emit.
var sgrRe = regexp.MustCompile("\x1b\\[([0-9;]*)m")

// TestStyledStripsToPlain is ui-spec §13's central property: colour is
// decoration and structure is the contract, so removing every escape from the
// coloured frame must yield exactly the uncoloured frame — byte for byte, at
// every size. This is what makes the no-colour fallback a downgrade in styling
// rather than a loss of information (§12).
func TestStyledStripsToPlain(t *testing.T) {
	sizes := [][2]int{{80, 24}, {80, 18}, {100, 30}, {60, 12}, {40, 11}, {38, 6}}
	for _, uni := range []bool{true, false} {
		for _, s := range sizes {
			w, h := s[0], s[1]
			styled := render(t, Caps{Colour: true, Unicode: uni}, w, h)
			plain := render(t, Caps{Colour: false, Unicode: uni}, w, h)
			if got := sgrRe.ReplaceAllString(styled, ""); got != plain {
				t.Errorf("unicode=%v %dx%d: stripped styled output != plain output", uni, w, h)
				gl, pl := strings.Split(got, "\n"), strings.Split(plain, "\n")
				for i := 0; i < len(gl) && i < len(pl); i++ {
					if gl[i] != pl[i] {
						t.Errorf("  first difference at row %d:\n   styled: %q\n   plain:  %q", i+1, gl[i], pl[i])
						break
					}
				}
			}
		}
	}
}

// TestOnlyPaletteSGRReachesView is the other half of the §13 escape property,
// and doubles as the automated form of "no raw colour numbers outside
// styles.go": any escape that is not an SGR drawn from the palette means either
// a colour escaped its home or tool output reached the terminal unfiltered.
func TestOnlyPaletteSGRReachesView(t *testing.T) {
	out := render(t, Caps{Colour: true, Unicode: true}, 80, 24)
	for i := 0; i < len(out); i++ {
		if out[i] != 0x1b {
			continue
		}
		loc := sgrRe.FindStringIndex(out[i:])
		if loc == nil || loc[0] != 0 {
			t.Fatalf("non-SGR escape at byte %d: %q", i, out[i:minInt(i+24, len(out))])
		}
		params := sgrRe.FindStringSubmatch(out[i:])[1]
		if !PaletteSGR[params] {
			t.Errorf("SGR %q is not in the styles.go palette", params)
		}
		i += loc[1] - 1
	}
}

// key builds a rune keypress.
func key(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

// keyType builds a non-rune keypress.
func keyType(t tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: t} }

// drive applies a sequence of messages, discarding commands. Commands are
// deliberately not executed: tea.Tick returns a closure that really sleeps, so
// running them would make the suite slow and reintroduce wall-clock flakiness.
func drive(m Model, msgs ...tea.Msg) Model {
	for _, msg := range msgs {
		next, _ := m.Update(msg)
		m = next.(Model)
	}
	return m
}

func newDriven(t *testing.T) Model { return newDrivenSize(t, 80, 24) }

func newDrivenSize(t *testing.T, w, h int) Model {
	t.Helper()
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	return drive(m, tea.WindowSizeMsg{Width: w, Height: h})
}

// TestApprovalCapturesInputExclusively checks ui-spec §5.2's rule that keys
// other than the approval's own are swallowed and never reach the composer.
func TestApprovalCapturesInputExclusively(t *testing.T) {
	m := newDriven(t)
	if m.mode() != ModeApprovalPending {
		t.Fatalf("fixture should start with an approval pending, got %v", m.mode())
	}
	for r := rune(0x20); r <= 0x7e; r++ {
		if strings.ContainsRune("yand?", r) {
			continue
		}
		got := drive(m, key(r))
		if v := got.comp.Value(); v != "" {
			t.Fatalf("key %q leaked into the composer (%q)", r, v)
		}
		if got.mode() != ModeApprovalPending {
			t.Fatalf("key %q escaped ApprovalPending", r)
		}
	}
}

// TestApproveResolvesAndReleasesCapture covers the y path end to end.
func TestApproveResolvesAndReleasesCapture(t *testing.T) {
	m := drive(newDriven(t), key('y'))
	if m.mode() != ModeComposing {
		t.Errorf("mode after approve = %v, want Composing", m.mode())
	}
	if m.status.Grants != 0 {
		t.Errorf("plain approve recorded %d grants, want 0", m.status.Grants)
	}
	if out := m.View(); strings.Contains(out, "approval required") {
		t.Error("resolved approval still renders its box")
	}
}

// TestSessionGrantRefusedOnPatch is ADR 0006 enforced in Update, not merely
// hidden in the renderer: pressing `a` on a patch must do nothing at all.
func TestSessionGrantRefusedOnPatch(t *testing.T) {
	m := drive(newDriven(t), key('a'))
	if m.mode() != ModeApprovalPending {
		t.Error("`a` resolved an apply_patch approval")
	}
	if m.status.Grants != 0 {
		t.Errorf("`a` on apply_patch recorded %d grants, want 0", m.status.Grants)
	}
}

// TestQuestionMarkIsLiteralInComposer is the §5.1 rule that a help overlay must
// not fire mid-sentence. Both directions are checked: only asserting the
// literal case would let a regression where `?` is always literal pass.
func TestQuestionMarkIsLiteralInComposer(t *testing.T) {
	m := drive(newDriven(t), key('y'), key('w'), key('h'), key('y'), key('?'))
	if m.mode() != ModeComposing {
		t.Fatalf("mode = %v, want Composing", m.mode())
	}
	if got := m.comp.Value(); got != "why?" {
		t.Errorf("composer = %q, want %q", got, "why?")
	}

	// ...and in ApprovalPending it does open help.
	if h := drive(newDriven(t), key('?')); h.mode() != ModeModal {
		t.Errorf("? in ApprovalPending: mode = %v, want Modal", h.mode())
	}
}

// TestModalEscReturnsToPreviousMode is the binding most likely to be written as
// "always go back to the composer". From an approval it must return to the
// pending approval. ui-spec §4.
func TestModalEscReturnsToPreviousMode(t *testing.T) {
	m := drive(newDriven(t), key('d'))
	if m.mode() != ModeModal {
		t.Fatalf("d did not open a modal: mode = %v", m.mode())
	}
	m = drive(m, keyType(tea.KeyEsc))
	if m.mode() != ModeApprovalPending {
		t.Errorf("Esc from a modal opened on an approval went to %v, want ApprovalPending", m.mode())
	}
}

// TestTypingDoesNotRepin is ui-spec §2.4. Typing while scrolled up must not
// yank the view to the bottom — the user is reading deliberately.
func TestTypingDoesNotRepin(t *testing.T) {
	// A viewport the fixture overflows, so there is something to scroll.
	m := drive(newDrivenSize(t, 80, 12), key('y'))
	m = drive(m, keyType(tea.KeyUp), keyType(tea.KeyPgUp))
	if m.scroll.Pinned {
		t.Fatalf("PgUp did not unpin (%d lines, viewport %d)", len(m.lines), m.layout().TranscriptH)
	}

	// Return to the composer without using Esc: Esc is itself a re-pin trigger
	// (§2.4), so it would mask the thing under test. Focus is moved directly,
	// which is the state a user reaches by pressing ↓ past the last card.
	m.setBase(BaseComposing)
	before := m.scroll.Offset

	m = drive(m, key('h'), key('e'), key('l'), key('l'), key('o'))
	if got := m.comp.Value(); got != "hello" {
		t.Fatalf("composer = %q, want %q", got, "hello")
	}
	if m.scroll.Pinned {
		t.Error("typing re-pinned the transcript")
	}
	if m.scroll.Offset != before {
		t.Errorf("typing scrolled the transcript: %d -> %d", before, m.scroll.Offset)
	}

	// Esc, by contrast, must re-pin.
	if m = drive(m, keyType(tea.KeyUp), keyType(tea.KeyEsc)); !m.scroll.Pinned {
		t.Error("Esc in Browsing did not re-pin")
	}
}

// TestPgUpDoesNotMoveSelection is the other half of §2.4: scrolling and
// selection are separate concerns.
func TestPgUpDoesNotMoveSelection(t *testing.T) {
	m := drive(newDrivenSize(t, 80, 12), key('y'), keyType(tea.KeyUp))
	sel := m.sel
	m = drive(m, keyType(tea.KeyPgUp), keyType(tea.KeyPgDown))
	if m.sel != sel {
		t.Errorf("PgUp/PgDn moved the selection: %d -> %d", sel, m.sel)
	}
}

// TestDoubleCtrlCForceQuits checks the §5.3 window from any mode.
func TestDoubleCtrlCForceQuits(t *testing.T) {
	base := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	now := base
	m := New(Options{Caps: Caps{Unicode: true}, Now: func() time.Time { return now }})
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	m2, _ := m.Update(keyType(tea.KeyCtrlC))
	now = base.Add(999 * time.Millisecond)
	_, cmd := m2.(Model).Update(keyType(tea.KeyCtrlC))
	if cmd == nil {
		t.Error("second Ctrl+C inside the window did not quit")
	}

	now = base
	m3, _ := m.Update(keyType(tea.KeyCtrlC))
	now = base.Add(1500 * time.Millisecond)
	if _, cmd := m3.(Model).Update(keyType(tea.KeyCtrlC)); cmd != nil {
		t.Error("second Ctrl+C outside the window quit anyway")
	}
}

// TestUnknownSlashCommandHints checks §6: a dim inline hint, never an error
// card, and never sent onward.
func TestUnknownSlashCommandHints(t *testing.T) {
	m := drive(newDriven(t), key('y'))
	for _, r := range "/nope" {
		m = drive(m, key(r))
	}
	m = drive(m, keyType(tea.KeyEnter))
	if m.comp.Hint == "" {
		t.Error("unknown command produced no hint")
	}
	for _, it := range m.tr.Items() {
		if it.Kind == KindError {
			t.Error("unknown command produced an error card")
		}
	}
}
