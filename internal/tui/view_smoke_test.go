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
	m := NewWithFixture(Options{Version: "0.1.0", Caps: caps})
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
	m := NewWithFixture(Options{Caps: Caps{Unicode: true}})
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
	m := sgrRe.FindStringSubmatch(got)
	if m == nil {
		t.Fatalf("accent emitted no SGR at all: %q", got)
	}
	// Asserted against the palette rather than a literal index, so retuning a
	// colour does not require editing the test that guards colour working.
	if !PaletteSGR[m[1]] {
		t.Errorf("accent emitted SGR %q, which is not in the palette", m[1])
	}
	if !strings.HasPrefix(m[1], "38;5;") {
		t.Errorf("accent emitted %q, want a 256-colour foreground", m[1])
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
	m := NewWithFixture(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
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
//
// This test used to reach the composer through setBase rather than Esc, and
// said so: Esc was itself a re-pin trigger, so pressing it would have masked
// the thing under test. It is no longer one, and that is the point. Esc was the
// only route back to the composer that does not first walk the selection off
// the end of the transcript, so the rule held everywhere except on the path
// someone scrolling up to read actually takes — which is how the operator hit
// it in manual testing and reported the feature as broken. The walk below now
// goes the way a user goes, and the old trigger is asserted gone rather than
// avoided.
func TestTypingDoesNotRepin(t *testing.T) {
	// A viewport the fixture overflows, so there is something to scroll.
	m := drive(newDrivenSize(t, 80, 12), key('y'))
	m = drive(m, keyType(tea.KeyUp), keyType(tea.KeyPgUp))
	if m.scroll.Pinned {
		t.Fatalf("PgUp did not unpin (%d lines, viewport %d)", len(m.lines), m.layout().TranscriptH)
	}
	before := m.scroll.Offset

	m = drive(m, keyType(tea.KeyEsc))
	if m.mode() != ModeComposing {
		t.Fatalf("Esc did not return focus to the composer: mode = %v", m.mode())
	}
	if m.scroll.Pinned {
		t.Fatal("Esc re-pinned the transcript; it is no longer one of §2.4's triggers")
	}
	if m.scroll.Offset != before {
		t.Fatalf("Esc scrolled the transcript: %d -> %d", before, m.scroll.Offset)
	}

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

// newDrivenColour is newDrivenSize with the colour palette live.
//
// Colour is the point for the overlay tests below. Each of them hinges on a
// flattened transcript and a live one rendering to different bytes; under the
// identity palette they render to the same bytes, so a colourless fixture would
// satisfy those assertions whatever the code did.
//
// TestNewSessionResetsAndRebuilds calls this helper too, but nothing about the
// palette is load-bearing there: assertCacheRebuilt compares row counts and
// content, never colour, so it fails on a stale cache either way. It uses the same
// helper as its neighbours for consistency, not because it needs the palette.
func newDrivenColour(t *testing.T, w, h int) Model {
	t.Helper()
	m := NewWithFixture(Options{Version: "0.1.0", Caps: Caps{Colour: true, Unicode: true}})
	return drive(m, tea.WindowSizeMsg{Width: w, Height: h})
}

// transcriptBand slices the transcript rows out of a rendered frame.
//
// The flattening an overlay applies is confined to that band — the header,
// status bar and composer keep their live palette behind a modal — so comparing
// the band keeps the assertion on the thing under test, and lets a confirm whose
// action legitimately changes the status bar use exactly the same check.
func transcriptBand(t *testing.T, m Model) string {
	t.Helper()
	lay := m.layout()
	rows := strings.Split(m.View(), "\n")
	start := 0
	if lay.ShowHeader {
		start = 1
	}
	end := minInt(start+lay.TranscriptH, len(rows))
	if start >= end {
		t.Fatalf("no transcript band in a %dx%d frame", m.width, m.height)
	}
	return strings.Join(rows[start:end], "\n")
}

// assertSameBand compares two transcript bands and points at the first row that
// differs, since a whole-band diff of styled text is unreadable.
func assertSameBand(t *testing.T, what, want, got string) {
	t.Helper()
	if got == want {
		return
	}
	t.Error(what)
	wl, gl := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < len(wl) && i < len(gl); i++ {
		if wl[i] != gl[i] {
			t.Errorf("  first difference at transcript row %d:\n   want: %q\n   got:  %q", i+1, wl[i], gl[i])
			return
		}
	}
	t.Errorf("  row counts differ: want %d, got %d", len(wl), len(gl))
}

// TestClosingAModalRepaintsInTheSameFrame covers a transition rather than a
// state, which is why the goldens cannot see it: relayout flattens the
// transcript's palette while an overlay is open and caches the result, and it
// runs *before* the key is dispatched — so on the frame that closes an overlay
// the cache is still the flat one and the whole window renders dimmed with
// nothing drawn over it. The user sees it as "press Esc a second time for the
// colours to return".
//
// The assertion is on rendered bytes rather than on m.modal: m.modal is already
// nil on the broken frame, so checking it would pass and prove nothing.
func TestClosingAModalRepaintsInTheSameFrame(t *testing.T) {
	// Every key that closes the help overlay, the `?` toggle included.
	for _, tc := range []struct {
		name string
		k    tea.KeyMsg
	}{
		{"esc", keyType(tea.KeyEsc)},
		{"ctrl-c", keyType(tea.KeyCtrlC)},
		{"question-mark", key('?')},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Both keys are setup, not the thing under test: `y` approves the
			// fixture's pending approval, since NewWithFixture starts in
			// ModeApprovalPending and nothing else reaches the composer until it
			// is resolved; Up then moves focus to the transcript, because `?` is
			// a literal character in the composer and only opens help from
			// Browsing (§5.1).
			m := drive(newDrivenColour(t, 80, 24), key('y'), keyType(tea.KeyUp))
			if m.mode() != ModeBrowsing {
				t.Fatalf("mode = %v, want Browsing", m.mode())
			}
			before := transcriptBand(t, m)
			if !strings.Contains(before, "\x1b[") {
				t.Fatal("no escapes in the frame: the palette is inert and this test cannot fail")
			}

			open := drive(m, key('?'))
			if open.mode() != ModeModal {
				t.Fatalf("? did not open the help overlay: mode = %v", open.mode())
			}
			if transcriptBand(t, open) == before {
				t.Fatal("the help overlay changed nothing on screen")
			}

			closed := drive(open, tc.k)
			if closed.mode() != ModeBrowsing {
				t.Fatalf("closing the help overlay went to %v, want Browsing", closed.mode())
			}
			assertSameBand(t, "the transcript is still flat on the frame that closed the modal",
				before, transcriptBand(t, closed))
		})
	}
}

// TestClosingAConfirmRepaintsInTheSameFrame is the same defect reached through
// the confirm prompt, which keyConfirm clears on paths of its own. Fixing only
// the modal path would leave every one of them dimmed.
func TestClosingAConfirmRepaintsInTheSameFrame(t *testing.T) {
	for _, tc := range []struct {
		name string
		k    tea.KeyMsg
	}{
		{"esc", keyType(tea.KeyEsc)},
		{"reject", key('n')},
		// Not the same `y` as the setup keypress below: this one accepts the
		// confirm and runs ConfirmClearGrants, which is the arm that also
		// changes the status bar — hence the band-only comparison.
		{"accept", key('y')},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newDrivenColour(t, 80, 24)
			m.status.Grants = 1 // /approvals only prompts when there is something to clear
			// Setup, not the thing under test: NewWithFixture starts in
			// ModeApprovalPending, so `y` resolves that approval and is what lets
			// the keys below reach the composer at all.
			m = drive(m, key('y'))
			if m.mode() != ModeComposing {
				t.Fatalf("mode = %v, want Composing", m.mode())
			}
			before := transcriptBand(t, m)
			if !strings.Contains(before, "\x1b[") {
				t.Fatal("no escapes in the frame: the palette is inert and this test cannot fail")
			}

			open := m
			for _, r := range "/approvals" {
				open = drive(open, key(r))
			}
			open = drive(open, keyType(tea.KeyEnter))
			if open.mode() != ModeConfirm {
				t.Fatalf("/approvals raised no confirm prompt: mode = %v", open.mode())
			}
			if transcriptBand(t, open) == before {
				t.Fatal("the confirm prompt changed nothing on screen")
			}

			closed := drive(open, tc.k)
			if closed.mode() != ModeComposing {
				t.Fatalf("closing the confirm prompt went to %v, want Composing", closed.mode())
			}
			assertSameBand(t, "the transcript is still flat on the frame that closed the confirm prompt",
				before, transcriptBand(t, closed))
		})
	}
}

// assertBandMatchesRebuild fails unless m's rendered transcript band already
// equals the band a forced rebuild produces — that is, unless the cache in
// m.lines is current for the model it is about to be drawn from.
//
// The copy is safe to rebuild: relayout assigns freshly allocated slices to
// lines/plainLines/rows, and Scroll is a value field, so nothing it writes can
// reach back into m.
func assertBandMatchesRebuild(t *testing.T, what string, m Model) {
	t.Helper()
	forced := m
	forced.relayout(forced.layout())
	assertSameBand(t, what, transcriptBand(t, forced), transcriptBand(t, m))
}

// assertCacheRebuilt is assertBandMatchesRebuild's counterpart for the cases the
// frame cannot show: it compares m.lines directly against a forced rebuild.
//
// Needed because transcriptRows short-circuits to the onboarding screen whenever
// m.tr is empty, so after a session reset the rendered band is identical whether
// the cache was rebuilt or not.
func assertCacheRebuilt(t *testing.T, what string, m Model) {
	t.Helper()
	forced := m
	forced.relayout(forced.layout())
	if len(m.lines) != len(forced.lines) {
		t.Errorf("%s\n  the cache holds %d rows; a rebuild produces %d", what, len(m.lines), len(forced.lines))
		return
	}
	for i := range forced.lines {
		if m.lines[i] != forced.lines[i] {
			t.Errorf("%s\n  first difference at cached row %d:\n   want: %q\n   got:  %q",
				what, i+1, forced.lines[i], m.lines[i])
			return
		}
	}
}

// TestOpeningAnOverlayRepaintsInTheSameFrame is the mirror image of the closing
// tests above, and the half the goldens are least able to see.
//
// relayout caches the transcript with the live palette *before* the key is
// dispatched, so on the frame that opens an overlay the cache is the unflattened
// one: the strip of transcript beside the modal box stays at full brightness
// until the next keypress happens to rebuild it. The user sees a box appear over
// an undimmed transcript, then watches it dim a keystroke later for no reason
// they pressed.
//
// The assertion is deliberately not "the band changed": the overlay box is blit
// over the band, so it changes whether or not the palette was rebuilt. It is
// "the band already equals a forced rebuild", which is only true if dispatchKey
// repainted on the opening transition as well as on the closing one.
func TestOpeningAnOverlayRepaintsInTheSameFrame(t *testing.T) {
	t.Run("modal", func(t *testing.T) {
		// Setup, as in the closing test: `y` resolves the fixture's pending
		// approval, Up moves focus off the composer so `?` is the help key
		// rather than a literal character (§5.1).
		m := drive(newDrivenColour(t, 80, 24), key('y'), keyType(tea.KeyUp))
		if m.mode() != ModeBrowsing {
			t.Fatalf("mode = %v, want Browsing", m.mode())
		}

		open := drive(m, key('?'))
		if open.mode() != ModeModal {
			t.Fatalf("? did not open the help overlay: mode = %v", open.mode())
		}
		band := transcriptBand(t, open)
		if !strings.Contains(band, "\x1b[") {
			t.Fatal("no escapes in the frame: the palette is inert and this test cannot fail")
		}
		assertBandMatchesRebuild(t,
			"the transcript is still live on the frame that opened the modal", open)
	})

	t.Run("confirm", func(t *testing.T) {
		// The oversized-paste prompt rather than /approvals, and the choice is
		// load-bearing: runSlash relayouts on its way out, so a confirm raised by
		// a slash command is repainted whether dispatchKey opens-repaints or not
		// and could never fail here. handlePaste raises its prompt and returns,
		// which leaves dispatchKey as the only thing holding the frame up.
		//
		// `y` is setup again — it resolves the fixture's pending approval, and a
		// paste is ignored outside ModeComposing (§7.2).
		m := drive(newDrivenColour(t, 80, 24), key('y'))
		if m.mode() != ModeComposing {
			t.Fatalf("mode = %v, want Composing", m.mode())
		}

		// Bracketed paste arrives as a KeyMsg with Paste set; there is no
		// tea.PasteMsg in bubbletea v1.
		open := drive(m, tea.KeyMsg{
			Type:  tea.KeyRunes,
			Paste: true,
			Runes: []rune(strings.Repeat("x", PasteWarnBytes+1)),
		})
		if open.mode() != ModeConfirm {
			t.Fatalf("an oversized paste raised no confirm prompt: mode = %v", open.mode())
		}
		band := transcriptBand(t, open)
		if !strings.Contains(band, "\x1b[") {
			t.Fatal("no escapes in the frame: the palette is inert and this test cannot fail")
		}
		assertBandMatchesRebuild(t,
			"the transcript is still live on the frame that opened the confirm prompt", open)
	})
}

// TestNewSessionResetsAndRebuilds covers the route into applyConfirm's
// ConfirmNewSession arm that is reachable today: `/new` typed while idle, which
// resets in place and never raises a prompt. That arm's own comment carries the
// argument for why the other branch into it cannot currently execute.
// applyConfirm does not relayout, so the only thing rebuilding the cache on this
// path is runSlash's own trailing relayout.
//
// What that assertion is worth, stated plainly — so nobody deletes it for the
// wrong reason and nobody trusts it for the wrong one:
//
// On `/new` itself a stale cache is invisible, and it does not last. Invisible
// because transcriptRows short-circuits to emptyStateRows whenever m.tr is empty,
// so the reset frame is byte-identical whether the rebuild ran or not — which is
// exactly why this test reads m.lines directly instead of the rendered band.
// Short-lived because Update relayouts at the top of every tea.KeyMsg, before
// dispatch, so the next keypress of any kind repairs it. No scroll key exposes it
// in between either, and for a structural reason rather than a measured one: End,
// PgUp, PgDown and Home all live in keyBrowsing, /new leaves the model in
// Composing — applyConfirm's reset arm does not touch m.base — and the keypress
// that moves focus to Browsing has itself already run Update's top-of-key relayout
// before keyComposing saw it. The cache is therefore current before any scroll key
// can be pressed at all. End would survive a stale cache regardless, since it
// rebuilds outright rather than sizing against len(m.lines); PgUp, PgDown and Home
// would not, because setOffset derives both its clamp and its re-pin flag from the
// total it is handed, and that total is len(m.lines). Measuring them against this
// fixture would only show whether its transcript happens to be shorter than the
// viewport.
//
// So this is a canary for runSlash's trailing relayout, not a `/new` guard. That
// one line is load-bearing for every slash command that leaves a card behind:
// each of them loses its notice row off the frame without it, which is the
// visible defect and what TestSlashNoticeCommandsRebuildTheCache covers — its
// case table is that list, and its docblock says why the table is the whole
// product surface. This test watches the same line from the cheaper side —
// it fails on the cache rather than on the pixels, and it is the only coverage of
// the ConfirmNewSession arm.
func TestNewSessionResetsAndRebuilds(t *testing.T) {
	// `y` resolves the fixture's pending approval; /new is only reachable once
	// keys reach the composer.
	m := drive(newDrivenColour(t, 80, 24), key('y'))
	if m.mode() != ModeComposing {
		t.Fatalf("mode = %v, want Composing", m.mode())
	}
	if m.busy.Active {
		t.Fatal("the fixture is mid-turn: /new would prompt rather than reset in place")
	}
	if m.tr.Len() == 0 {
		t.Fatal("the fixture transcript is empty: a reset would be unobservable")
	}

	after := m
	for _, r := range "/new" {
		after = drive(after, key(r))
	}
	after = drive(after, keyType(tea.KeyEnter))

	if after.confirm != nil {
		t.Fatalf("/new while idle raised a confirm prompt (%q); ui-spec §4.3 reserves that for a live turn",
			after.confirm.Prompt)
	}
	if n := after.tr.Len(); n != 0 {
		t.Errorf("transcript holds %d items after /new, want 0", n)
	}
	if after.sel != 0 || after.pendingApproval != 0 {
		t.Errorf("sel/pendingApproval = %d/%d after /new, want 0/0", after.sel, after.pendingApproval)
	}
	if !after.scroll.Pinned {
		t.Error("/new left the transcript unpinned")
	}
	assertCacheRebuilt(t, "the transcript cache still describes the discarded session after /new", after)

	// And the reset is visible: the onboarding screen is back.
	if band := stripSGR(transcriptBand(t, after)); !strings.Contains(band, Suggestions[0]) {
		t.Error("/new did not return the onboarding screen")
	}
}

// TestSlashNoticeCommandsRebuildTheCache pins runSlash's trailing relayout on the
// paths where dropping it is visible to the user.
//
// The commands in the case table below all end the same way: append a notice
// card, fall through to the relayout at the bottom of runSlash, return. /status,
// /files and /compact do it unconditionally; /approvals does it on the branch it
// takes when there are no session grants to clear — the only one of them whose
// notice is conditional at all, and conditional on runtime session state at that,
// which makes it the one most likely to be restructured into something that skips
// the relayout. Nothing else rebuilds the cache for any of them. Update's relayout
// runs at the *top* of the keypress, before the card exists, and dispatchKey
// repaints only when the keypress changed the overlay state — which submitting a
// notice command does not. So without that one line the frame drawn for the
// keypress that ran the command is one row short: the notice the user just asked
// for is missing from the transcript until some later keypress happens to rebuild.
//
// The table is the whole product-surface set, and is meant to be read as complete
// rather than as a sample. The debug commands — /read, /ls, /search, /gitstatus and
// /gitdiff — reach the same trailing relayout by the same route: with RunTool nil
// they notice "tools are not wired up in this build" and fall through. That notice
// is conditional too, so /approvals is not the only conditional arm in runSlash —
// it is the only one conditional on session state rather than on build wiring.
// The debug arm is left uncovered deliberately: it is M1 scaffolding the code marks
// for removal in M3, its notice exists only in a build with no tools wired, and
// coverage of it should disappear when it does.
//
// Asserted through the rendered band rather than against m.lines, because here the
// frame really can show it — unlike /new, whose empty transcript sends
// transcriptRows to the onboarding screen and hides the same staleness entirely.
func TestSlashNoticeCommandsRebuildTheCache(t *testing.T) {
	// want is a fragment of the notice each command appends — short enough to
	// survive the card's wrapping at 80 columns.
	//
	// pre asserts the model state a command's notice branch depends on, and exists
	// for the one command that has such a dependency. Without it a fixture that
	// started with a session grant would send /approvals down the confirm branch,
	// and the failure would surface in the first vacuity guard as "it appended no
	// notice card" — true, but pointing at the transcript rather than at the
	// precondition that actually moved.
	cases := []struct {
		cmd, want string
		pre       func(*testing.T, Model)
	}{
		{cmd: "/status", want: "branch"},
		{cmd: "/files", want: "files touched this session"},
		{cmd: "/compact", want: "nothing to compact yet"},
		{cmd: "/approvals", want: "no active session grants", pre: func(t *testing.T, m Model) {
			t.Helper()
			if m.status.Grants != 0 {
				t.Fatalf("the fixture holds %d session grant%s, so /approvals raises the "+
					"clear-grants confirm instead of appending a notice",
					m.status.Grants, plural(m.status.Grants))
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.cmd, func(t *testing.T) {
			// `y` resolves the fixture's pending approval; slash commands are only
			// reachable once keys reach the composer.
			m := drive(newDriven(t), key('y'))
			if m.mode() != ModeComposing {
				t.Fatalf("mode = %v, want Composing", m.mode())
			}
			if tc.pre != nil {
				tc.pre(t, m)
			}
			before := m.tr.Len()

			after := m
			for _, r := range tc.cmd {
				after = drive(after, key(r))
			}
			after = drive(after, keyType(tea.KeyEnter))

			// Three vacuity guards before the real assertion. Without them this
			// subtest would also pass against a command that appended nothing —
			// runSlash's unknown-command arm returns early, adds no card, and needs
			// no relayout — or against a notice that landed outside the visible
			// window, where a missing row could never be seen.
			//
			// Two of the three can fire against runSlash as it stands; the kind
			// check cannot. The count guard already catches every arm that appends
			// nothing, and no arm appends anything but a notice, so there is no
			// current path where the count is right and the kind is wrong. It is
			// kept as forward-looking defence rather than as an exercised check: the
			// day an arm appends some other card, `want` could match text that is
			// not the notice this test believes it is watching.
			if n := after.tr.Len(); n != before+1 {
				t.Fatalf("%s left the transcript at %d items, want %d: it appended no notice card, "+
					"so there is nothing here a stale cache could drop", tc.cmd, n, before+1)
			}
			items := after.tr.Items()
			if last := items[len(items)-1]; last.Kind != KindNotice {
				t.Fatalf("%s appended a %v, not a notice card", tc.cmd, last.Kind)
			}
			fresh := after
			fresh.relayout(fresh.layout())
			if band := stripSGR(transcriptBand(t, fresh)); !strings.Contains(band, tc.want) {
				t.Fatalf("%s's notice (%q) is off the frame even after a forced rebuild: "+
					"a stale cache would be unobservable here", tc.cmd, tc.want)
			}

			assertBandMatchesRebuild(t,
				"the transcript cache is missing the notice row "+tc.cmd+" just appended", after)
		})
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

// TestQuittingHasNoKeyBinding is the operator's ruling from the Milestone 0
// manual test: a bare `q` ended the session, so the first character of "query"
// ended it by accident.
//
// Both sites are checked, because they failed differently. The composing one
// was guarded — idle, empty composer — and documented in ui-spec §5.2; the
// browsing one was neither, and appeared in no table in the spec or in
// bindingGroups. A fix that only removed the documented half would leave the
// undocumented half doing exactly what the operator reported.
func TestQuittingHasNoKeyBinding(t *testing.T) {
	t.Run("composing", func(t *testing.T) {
		// `y` resolves the fixture's pending approval, which is what lets any
		// key reach the composer at all. The composer is then empty and the
		// session idle — the exact state the old guard fired in.
		m := drive(newDriven(t), key('y'))
		if m.mode() != ModeComposing {
			t.Fatalf("mode = %v, want Composing", m.mode())
		}
		if m.comp.Value() != "" {
			t.Fatalf("composer = %q, want empty: the old guard only fired on an empty one",
				m.comp.Value())
		}
		m = drive(m, key('q'))
		if got := m.comp.Value(); got != "q" {
			t.Errorf("composer = %q after `q`, want \"q\": the key must reach the textarea "+
				"as a literal character, not quit", got)
		}
	})

	t.Run("browsing", func(t *testing.T) {
		m := drive(newDriven(t), key('y'), keyType(tea.KeyUp))
		if m.mode() != ModeBrowsing {
			t.Fatalf("mode = %v, want Browsing", m.mode())
		}
		// The assertion is on the command rather than on the model, because
		// tea.Quit changes nothing a Model comparison could see: it is returned
		// as a command and acted on by the runtime.
		next, cmd := m.Update(key('q'))
		if cmd != nil {
			t.Errorf("`q` in Browsing returned a command (%T); it is bound to nothing "+
				"and must return nil", cmd())
		}
		after := next.(Model)
		if got := after.mode(); got != ModeBrowsing {
			t.Errorf("mode = %v after `q`, want Browsing unchanged", got)
		}
	})
}

// TestQuitAndExitBothQuit covers the replacement surface. /quit already existed;
// /exit is the alias added when the key went away, and an alias that is listed
// in the help overlay but wired to nothing is worse than no alias at all.
func TestQuitAndExitBothQuit(t *testing.T) {
	for _, cmdText := range []string{"/quit", "/exit"} {
		t.Run(cmdText, func(t *testing.T) {
			m := drive(newDriven(t), key('y'))
			for _, r := range cmdText {
				m = drive(m, key(r))
			}
			_, cmd := m.Update(keyType(tea.KeyEnter))
			if cmd == nil {
				t.Fatalf("%s returned no command, so it did not quit", cmdText)
			}
			if msg := cmd(); !isQuitMsg(msg) {
				t.Errorf("%s produced %T, want tea.QuitMsg", cmdText, msg)
			}
		})
	}
}

// isQuitMsg reports whether a command's message is Bubble Tea's quit signal.
//
// tea.Quit is a func, and Go compares funcs to nil and nothing else, so the
// command itself cannot be identified. Running it can: tea.Quit's body returns
// QuitMsg and touches nothing, unlike tea.Tick's, which really sleeps.
func isQuitMsg(msg tea.Msg) bool {
	_, ok := msg.(tea.QuitMsg)
	return ok
}

// TestEverySlashCommandCompletesToItself pins the property Tab completion needs
// from the command set: no name may be a prefix of another.
//
// It is a structural test over SlashCommands and DebugCommands, not a list of
// names — so a command added later is covered the moment it is added. The rule
// it checks is "completing a command's full name returns that command and
// reports a unique match". A name that is a strict prefix of another breaks it:
// /exit next to a hypothetical /exitnow would make /exit uncompletable, and the
// longer name unreachable by typing the shorter one in full.
//
// What it does not check: that every command has a *short* unique prefix, which
// is a convenience rather than a correctness property, and that the two lists
// do not collide with each other beyond the prefix rule — completeSlash searches
// their concatenation, so a duplicate name across the two would be caught here
// as a non-unique match, but a duplicate within one list would not be reported
// as the duplicate it is.
func TestEverySlashCommandCompletesToItself(t *testing.T) {
	all := append(append([]string{}, SlashCommands...), trimSlashes(DebugCommands)...)
	for _, name := range all {
		got, unique := completeSlash(name)
		if !unique || got != name {
			t.Errorf("completeSlash(%q) = (%q, %v), want (%q, true): "+
				"some other command has %q as a prefix", name, got, unique, name, name)
		}
	}
}

// TestExitIsTabCompletable pins the half of /exit that runSlash's arm does not
// reach: its entry in SlashCommands, which is what puts it in Tab completion and
// in the help overlay's commands column. Wiring the arm and forgetting the entry
// gives a command that works only for someone who already knows it is there.
func TestExitIsTabCompletable(t *testing.T) {
	got, unique := completeSlash("ex")
	if !unique || got != "exit" {
		t.Errorf("completeSlash(%q) = (%q, %v), want (\"exit\", true): /exit is missing from "+
			"SlashCommands, so Tab does not complete it and /help does not list it", "ex", got, unique)
	}
}

// TestScriptedTurnHoldsTheRunningCard covers the second manual-test failure:
// the tester could not find the ◐ running glyph anywhere.
//
// Nothing was wrong with the glyph or the state machine — screen 02 renders both
// correctly. The card was simply mutated out of StateRunning by the very next
// tick, so the state existed for one 50ms frame.
//
// The dwell itself is a duration on a tea.Tick, and drive discards commands
// rather than sleeping through them, so the elapsed time is not observable from
// the message loop. What is observable is checked here: the card really passes
// through StateRunning, the constant really is longer than a stream frame, and
// the card's reported Elapsed is that same constant — which is what makes the
// docblock's "as long as it says it took" mechanical rather than a claim.
func TestScriptedTurnHoldsTheRunningCard(t *testing.T) {
	if fakeRunDwell <= StreamCoalesce {
		t.Fatalf("fakeRunDwell = %v, which is no longer than one %v stream frame: "+
			"the running state is back to being invisible", fakeRunDwell, StreamCoalesce)
	}

	m := newScriptedTurn(t)
	var scheduled tea.Cmd
	m = advanceUntil(t, m, "a running run_command card", func(m Model) bool {
		c, ok := lastToolCard(m)
		return ok && c.Name == "run_command" && c.State == StateRunning
	}, &scheduled)

	// The dwell is real only if it is wired to the tick that ends it. The
	// constant above proves the value; this proves the schedule. Running the
	// command and watching it *not* fire is one-sided on purpose — a slow
	// machine can only make the tick later, which is the passing direction, so
	// the check fails when the delay is genuinely short and never otherwise.
	// The margin is several stream frames rather than a fraction of the dwell,
	// which keeps the test fast and the mutation it catches unambiguous.
	if scheduled == nil {
		t.Fatal("the tick that started the run_command card scheduled nothing, " +
			"so the turn cannot reach the card's terminal state at all")
	}
	fired := make(chan struct{})
	go func() {
		scheduled()
		close(fired)
	}()
	select {
	case <-fired:
		t.Errorf("the tick behind the running card fired within %v: the card was scheduled "+
			"to end on a stream frame, not on the %v dwell", 3*StreamCoalesce, fakeRunDwell)
	case <-time.After(3 * StreamCoalesce):
	}

	m = drive(m, streamTickMsg{})
	c, ok := lastToolCard(m)
	if !ok {
		t.Fatal("the run_command card vanished from the transcript")
	}
	if c.State != StateError {
		t.Fatalf("card state = %v after the tick behind the dwell, want StateError", c.State)
	}
	if c.Elapsed != fakeRunDwell {
		t.Errorf("the card reports Elapsed %v but was held for %v; the time it claims to "+
			"have taken and the time it visibly took must be the same value", c.Elapsed, fakeRunDwell)
	}
}

// TestScriptedTurnOffersBothApprovalKinds covers the third manual-test failure:
// `[a] approve for session` never appeared.
//
// Both the renderer and the key handler were already complete and already
// tested — TestApplyPatchNeverOffersSessionGrant covers the predicate — but the
// only approval the running app ever built was the patch variant, which ADR 0006
// says must never offer [a]. So [a] rendered in the golden tests and nowhere a
// person could reach it.
//
// The waiting half matters as much as the second card. Model.pendingApproval
// holds one ID, so queueing the second approval while the first is unanswered
// would silently replace it, and the card on screen would stop being the card
// the keys resolve.
func TestScriptedTurnOffersBothApprovalKinds(t *testing.T) {
	m := newScriptedTurn(t)
	m = advanceUntil(t, m, "an approval", func(m Model) bool { return m.pendingApproval != 0 })

	patch, _, ok := m.tr.Find(m.pendingApproval)
	if !ok {
		t.Fatal("pendingApproval names an item that is not in the transcript")
	}
	if patch.Approval.Kind != ApprovalPatch {
		t.Fatalf("first approval is %v, want ApprovalPatch", patch.Approval.Kind)
	}

	// The turn must not walk past an unanswered card, however long it is left.
	held := m
	for i := 0; i < 20; i++ {
		held = drive(held, streamTickMsg{})
	}
	if held.pendingApproval != m.pendingApproval {
		t.Fatalf("20 ticks moved the pending approval from %d to %d: the scripted turn "+
			"ran on behind a card the operator had not answered",
			m.pendingApproval, held.pendingApproval)
	}

	m = drive(held, key('y'))
	if m.pendingApproval != 0 {
		t.Fatal("`y` did not resolve the patch approval")
	}
	m = advanceUntil(t, m, "a second approval", func(m Model) bool { return m.pendingApproval != 0 })

	second, _, ok := m.tr.Find(m.pendingApproval)
	if !ok {
		t.Fatal("the second pendingApproval names an item that is not in the transcript")
	}
	if !second.Approval.OffersSessionGrant() {
		t.Errorf("the turn's second approval is %v with GrantScope %q and offers no [a]; "+
			"the manual test has no way to reach the session-grant key",
			second.Approval.Kind, second.Approval.GrantScope)
	}
}

// newScriptedTurn starts the fake turn from an empty session, so that nothing
// the fixture pre-loads can be mistaken for something the turn produced.
func newScriptedTurn(t *testing.T) Model {
	t.Helper()
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = drive(m, key('h'), key('i'), keyType(tea.KeyEnter))
	if !m.busy.Active {
		t.Fatal("sending a message did not start a turn")
	}
	return m
}

// advanceUntil ticks the scripted turn until cond holds.
//
// The cap is a stuck-turn guard, not a step budget: the streamed sentence is
// one tick per word, so the count is well above anything the script reaches.
// Passing last captures the command the tick that satisfied cond returned, which
// is the only way to see the interval that tick scheduled; pass nil to ignore it.
func advanceUntil(t *testing.T, m Model, what string, cond func(Model) bool, last ...*tea.Cmd) Model {
	t.Helper()
	for i := 0; i < 200; i++ {
		if cond(m) {
			return m
		}
		next, cmd := m.Update(streamTickMsg{})
		m = next.(Model)
		for _, out := range last {
			if out != nil {
				*out = cmd
			}
		}
	}
	t.Fatalf("the scripted turn never reached %s", what)
	return m
}

// lastToolCard returns the most recent tool card in the transcript.
func lastToolCard(m Model) (ToolCard, bool) {
	items := m.tr.Items()
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].Kind == KindTool {
			return *items[i].Tool, true
		}
	}
	return ToolCard{}, false
}

// TestEnteringBrowsingDrawsTheGutterInTheSameFrame is dispatchKey's third cache
// invalidator, and it is not theoretical.
//
// relayout draws the selection gutter beside m.sel and takes its columns out of
// that item's content width, so moving the selection invalidates the cache. Most
// handlers that move it relayout by hand — but keyComposing's Up arm moves it
// through setBase, which does not, and neither does the revealSelection call
// beside it. With a selection of zero and a selectable card on screen, that arm
// selects the card and leaves the frame showing no gutter until some later
// keypress happens to rebuild.
//
// The setup reaches that state the way a user does: send a message, let one tool
// card land, then press Up. Nothing in that path assigns m.sel.
func TestEnteringBrowsingDrawsTheGutterInTheSameFrame(t *testing.T) {
	m := newScriptedTurn(t)
	m = advanceUntil(t, m, "a tool card", func(m Model) bool {
		_, ok := lastToolCard(m)
		return ok
	})
	if m.sel != 0 {
		t.Fatalf("m.sel = %d before Up, want 0: this test needs the state where the Up arm "+
			"is the thing that assigns the selection", m.sel)
	}

	m = drive(m, keyType(tea.KeyUp))
	if m.mode() != ModeBrowsing {
		t.Fatalf("mode = %v after Up, want Browsing", m.mode())
	}
	if m.sel == 0 {
		t.Fatal("Up did not select a card, so there is no gutter for this test to look for")
	}
	assertBandMatchesRebuild(t,
		"the frame that entered Browsing is missing the selection gutter: the cache was "+
			"built against the previous selection and dispatchKey did not rebuild it", m)
}
