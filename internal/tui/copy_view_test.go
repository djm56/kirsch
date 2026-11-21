package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// The tests in this file read what View draws. The copy-notice tests in
// copy_test.go and reply_select_test.go check m.comp.Hint, the model's state,
// which does not show whether the notice reaches the screen. Every case here
// drives the Model through Update with real key messages and then reads
// m.View() with the styling stripped.

const (
	cvWidth  = 80
	cvHeight = 28
)

// cvCaps are the two glyph sets the file runs under. The browsing key-hint row
// spells its arrows and Shift key differently in each.
var cvCaps = []struct {
	name string
	caps Caps
}{
	{"unicode", Caps{Unicode: true}},
	{"ascii", Caps{Unicode: false}},
}

// cvTranscript is a mixed transcript on an 80x28 model: a reply, a tool card,
// then a second reply, with a recording Copy wired. The tool card and the
// replies are selectable; ids are returned so a test can select one directly.
type cvTranscript struct {
	m            Model
	card, replyB ItemID
	rec          *copyRecorder
}

func newCVTranscript(t *testing.T, caps Caps) cvTranscript {
	t.Helper()
	x := cvTranscript{rec: &copyRecorder{}}
	m := New(Options{Caps: caps})
	m.Copy = x.rec.copy
	m.appendBlock(Item{Kind: KindAssistant, Text: &TextBlock{Lines: []string{"reply A"}, Source: "reply A"}})
	x.card = m.appendBlock(Item{Kind: KindTool, Tool: &ToolCard{
		Name: "read_file", Target: "a.go", State: StateOK, Out: []string{"package a"},
	}})
	x.replyB = m.appendBlock(Item{Kind: KindAssistant, Text: &TextBlock{Lines: []string{"reply B"}, Source: "reply B"}})
	x.m = drive(m, tea.WindowSizeMsg{Width: cvWidth, Height: cvHeight})
	if x.m.mode() != ModeComposing {
		t.Fatalf("setup: mode = %v, want Composing", x.m.mode())
	}
	return x
}

// cvFrame is a rendered View split into its composer region, the bottom
// key-hint row, and everything above the composer.
type cvFrame struct {
	all      []string // every row, styling stripped
	above    []string // rows before the composer region
	composer []string // the composer's rows (input rows plus any notice row)
	hintRow  string   // the bottom key-hint row
}

// cvRender renders m and slices the frame by the model's own layout: the key
// hint row is last when HintH is 1, and the composer is the ComposerH rows
// before it.
func cvRender(t *testing.T, m Model) cvFrame {
	t.Helper()
	lay := m.layout()
	rows := plain(strings.Split(m.View(), "\n"))
	if len(rows) != cvHeight {
		t.Fatalf("frame is %d rows, want %d", len(rows), cvHeight)
	}
	if lay.HintH != 1 {
		t.Fatalf("setup: HintH = %d at %dx%d, want 1", lay.HintH, cvWidth, cvHeight)
	}
	end := len(rows) - lay.HintH
	start := end - lay.ComposerH
	return cvFrame{
		all:      rows,
		above:    rows[:start],
		composer: rows[start:end],
		hintRow:  strings.TrimSpace(rows[end]),
	}
}

// rowsWith returns the indexes of rows in the slice that contain sub.
func rowsWith(rows []string, sub string) []int {
	var at []int
	for i, r := range rows {
		if strings.Contains(r, sub) {
			at = append(at, i)
		}
	}
	return at
}

// wantNoticeInComposer fails unless the notice is drawn on exactly one row, that
// row is in the composer region, and the row reads as the composer draws a hint:
// two spaces, then the notice. Rows outside the composer must not hold it.
func wantNoticeInComposer(t *testing.T, f cvFrame, notice string) {
	t.Helper()
	if n := len(rowsWith(f.composer, notice)); n != 1 {
		t.Errorf("composer region holds the notice %q on %d rows, want 1; frame:\n%s",
			notice, n, strings.Join(f.all, "\n"))
		return
	}
	found := false
	for _, r := range f.composer {
		if strings.TrimSpace(r) == notice && strings.HasSuffix(strings.TrimRight(r, " "), "  "+notice) {
			found = true
		}
	}
	if !found {
		t.Errorf("no composer row is the notice indented two spaces: %q; frame:\n%s", "  "+notice, strings.Join(f.all, "\n"))
	}
	if n := len(rowsWith(f.above, notice)); n != 0 {
		t.Errorf("notice %q also drawn above the composer on %d rows", notice, n)
	}
	if strings.Contains(f.hintRow, notice) {
		t.Errorf("notice %q drawn in the key-hint row %q", notice, f.hintRow)
	}
}

// browsingKeyHint is the key-hint row the browsing mode shows for a selected
// card (reply false) or a selected reply (reply true), in the given glyph set.
func browsingKeyHint(caps Caps, reply bool) string {
	upDown, bullet := "Up/Down", "."
	if caps.Unicode {
		upDown, bullet = "↑↓", "·"
	}
	items := []string{upDown + " select", "Enter preview", "d detail", "Tab/Esc prompt"}
	if reply {
		items = []string{upDown + " select", "c copy", "Tab/Esc prompt"}
	}
	return strings.Join(items, " "+bullet+" ")
}

// composingKeyHint is the key-hint row Composing shows when idle.
func composingKeyHint(caps Caps) string {
	upDown, shiftUp, bullet := "Up/Down", "Shift+Up", "."
	if caps.Unicode {
		upDown, shiftUp, bullet = "↑↓", "⇧↑", "·"
	}
	return strings.Join([]string{upDown + " history", shiftUp + "/Tab cards", "/help"}, " "+bullet+" ")
}

// TestCopyViewCardNoticeIsDrawn: Shift+Up into Browsing, select the card, press
// c. Entry point: Update with real key messages. The notice must be drawn in the
// composer region and the key-hint row must be the browsing one for a card. The
// control is the frame before c, which has no notice anywhere.
func TestCopyViewCardNoticeIsDrawn(t *testing.T) {
	const notice = "c copies replies only"
	for _, c := range cvCaps {
		t.Run(c.name, func(t *testing.T) {
			x := newCVTranscript(t, c.caps)
			m := drive(x.m, keyType(tea.KeyShiftUp))
			m.sel = x.card
			before := cvRender(t, m)
			if n := len(rowsWith(before.all, notice)); n != 0 {
				t.Fatalf("control: the notice is drawn on %d rows before c", n)
			}
			m = drive(m, key('c'))
			if m.mode() != ModeBrowsing {
				t.Fatalf("mode = %v after c, want Browsing", m.mode())
			}
			f := cvRender(t, m)
			wantNoticeInComposer(t, f, notice)
			if want := browsingKeyHint(c.caps, false); f.hintRow != want {
				t.Errorf("key-hint row = %q, want %q", f.hintRow, want)
			}
		})
	}
}

// TestCopyViewReplyResultIsDrawn: Browsing, c on a selected reply. The Cmd Update
// returns is run and its CopyResultMsg is delivered through Update, as the
// runtime would. "copied N characters" must then be drawn in the composer
// region. The control is the frame before the result arrives, which has no copy
// notice.
func TestCopyViewReplyResultIsDrawn(t *testing.T) {
	for _, c := range cvCaps {
		t.Run(c.name, func(t *testing.T) {
			x := newCVTranscript(t, c.caps)
			m := drive(x.m, keyType(tea.KeyShiftUp))
			m.sel = x.replyB
			next, cmd := m.Update(key('c'))
			m = next.(Model)
			if cmd == nil {
				t.Fatal("c on a reply returned no Cmd")
			}
			notice := copiedNotice("reply B")
			if n := len(rowsWith(cvRender(t, m).all, notice)); n != 0 {
				t.Fatalf("control: the notice is drawn on %d rows before the result", n)
			}
			m = runCopyCmd(t, m, cmd)
			if len(x.rec.got) != 1 || x.rec.got[0] != "reply B" {
				t.Fatalf("Copy was called with %q, want [\"reply B\"]", x.rec.got)
			}
			f := cvRender(t, m)
			wantNoticeInComposer(t, f, notice)
			if want := browsingKeyHint(c.caps, true); f.hintRow != want {
				t.Errorf("key-hint row = %q, want %q", f.hintRow, want)
			}
		})
	}
}

// TestCopyViewNoticeAfterHelpCloses: Shift+Up into Browsing on a card, ? opens
// help, a close key closes it, then c. The notice must be drawn. Entry point:
// Update with real key messages; the close keys are ? and Esc. Two controls: the
// same sequence without c draws no notice, and the help overlay is confirmed
// open and then closed by checking the mode between keys.
func TestCopyViewNoticeAfterHelpCloses(t *testing.T) {
	const notice = "c copies replies only"
	closers := []struct {
		name string
		k    tea.KeyMsg
	}{
		{"? closes help", key('?')},
		{"Esc closes help", keyType(tea.KeyEsc)},
	}
	for _, c := range cvCaps {
		for _, cl := range closers {
			t.Run(c.name+"/"+cl.name, func(t *testing.T) {
				x := newCVTranscript(t, c.caps)
				m := drive(x.m, keyType(tea.KeyShiftUp))
				m.sel = x.card
				m = drive(m, key('?'))
				if m.mode() != ModeModal {
					t.Fatalf("setup: mode = %v after ?, want Modal", m.mode())
				}
				m = drive(m, cl.k)
				if m.mode() != ModeBrowsing {
					t.Fatalf("setup: mode = %v after the close key, want Browsing", m.mode())
				}

				// Control: closed help, no c. Nothing drawn.
				ctl := cvRender(t, m)
				if n := len(rowsWith(ctl.all, notice)); n != 0 {
					t.Errorf("control: notice drawn on %d rows without c", n)
				}
				if len(ctl.composer) != m.layout().ComposerH {
					t.Fatalf("control: composer region is %d rows, layout says %d", len(ctl.composer), m.layout().ComposerH)
				}

				m = drive(m, key('c'))
				f := cvRender(t, m)
				wantNoticeInComposer(t, f, notice)
				if want := browsingKeyHint(c.caps, false); f.hintRow != want {
					t.Errorf("key-hint row = %q, want %q", f.hintRow, want)
				}
				if len(f.composer) != len(ctl.composer)+1 {
					t.Errorf("composer region is %d rows after c, want %d: one row for the notice", len(f.composer), len(ctl.composer)+1)
				}
			})
		}
	}
}

// TestCopyViewUsageIsDrawn: Composing, "/copy foo" typed and submitted with
// Enter. The usage hint must be drawn in the composer region, and the key-hint
// row is the Composing one.
func TestCopyViewUsageIsDrawn(t *testing.T) {
	const notice = "usage: /copy [code [N]]"
	for _, c := range cvCaps {
		t.Run(c.name, func(t *testing.T) {
			x := newCVTranscript(t, c.caps)
			m, cmd := submitSlash(t, x.m, "/copy foo")
			if cmd != nil {
				t.Fatal("/copy foo returned a Cmd; it must start no copy")
			}
			f := cvRender(t, m)
			wantNoticeInComposer(t, f, notice)
			if want := composingKeyHint(c.caps); f.hintRow != want {
				t.Errorf("key-hint row = %q, want %q", f.hintRow, want)
			}
		})
	}
}

// TestCopyViewUnknownCommandHintIsDrawn is the positive control for the drawing
// path: a hint that has nothing to do with copy, set by submitting /nosuch with
// Enter, is drawn the same way. Its negative control is a fresh model with no
// hint: the composer region is the input row alone, and every row below it is
// empty.
func TestCopyViewUnknownCommandHintIsDrawn(t *testing.T) {
	const notice = "unknown command /nosuch"
	for _, c := range cvCaps {
		t.Run(c.name, func(t *testing.T) {
			x := newCVTranscript(t, c.caps)

			fresh := cvRender(t, x.m)
			if x.m.comp.Hint != "" {
				t.Fatalf("setup: a fresh model has hint %q", x.m.comp.Hint)
			}
			if len(fresh.composer) < 1 {
				t.Fatal("composer region is empty")
			}
			for i, r := range fresh.composer[1:] {
				if strings.TrimSpace(r) != "" {
					t.Errorf("fresh model: composer row %d below the input row is %q, want empty", i+1, r)
				}
			}

			m, _ := submitSlash(t, x.m, "/nosuch")
			f := cvRender(t, m)
			wantNoticeInComposer(t, f, notice)
			if want := composingKeyHint(c.caps); f.hintRow != want {
				t.Errorf("key-hint row = %q, want %q", f.hintRow, want)
			}
			if len(f.composer) != len(fresh.composer)+1 {
				t.Errorf("composer region is %d rows with a hint, want %d", len(f.composer), len(fresh.composer)+1)
			}
		})
	}
}
