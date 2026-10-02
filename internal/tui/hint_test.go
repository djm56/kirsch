package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestHintComputeLayout verifies overlay cases now expect HintH == 1 at h >= 24
// and that overlay state doesn't change the layout.
func TestHintComputeLayout(t *testing.T) {
	// Assert that computeLayout(w,h,c,true) equals computeLayout(w,h,c,false)
	// for h in 20..40 at every width
	for _, w := range []int{40, 60, 80, 120} {
		for h := 20; h <= 40; h++ {
			for c := 1; c <= 5; c++ {
				plainLay := computeLayout(w, h, c, false)
				overLay := computeLayout(w, h, c, true)

				if plainLay != overLay {
					t.Errorf("%dx%d composer=%d: opening overlay changed layout:\n"+
						"  plain:  %+v\n  overlay: %+v",
						w, h, c, plainLay, overLay)
				}

				// Test the boundary: HintH should be 1 at h >= 24, 0 below
				wantHintH := 0
				if h >= 24 {
					wantHintH = 1
				}
				if plainLay.HintH != wantHintH {
					t.Errorf("%dx%d: HintH=%d, want %d", w, h, plainLay.HintH, wantHintH)
				}
			}
		}
	}
}

// TestHintInView verifies that at 80×24 in idle Composing mode, the last row
// is the composing hint and the row above it is the composer prompt row.
func TestHintInView(t *testing.T) {
	m := New(Options{
		Caps:    Caps{Unicode: true},
		Session: SessionInfo{Project: "test"},
	})
	mm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = mm.(Model)

	// Ensure we're in composing mode
	m.setBase(BaseComposing)

	view := m.View()
	rows := strings.Split(view, "\n")

	// Should have exactly 24 rows
	if len(rows) != 24 {
		t.Errorf("view has %d rows, want 24", len(rows))
	}

	// Last row should be the hint (trimmed to check content)
	lastRow := stripSGR(rows[23])
	trimmedLast := strings.TrimSpace(lastRow)

	// The last row should contain composing hint elements
	if !strings.Contains(trimmedLast, "history") && !strings.Contains(trimmedLast, "cards") {
		t.Errorf("last row doesn't contain expected composing hint content: %q", trimmedLast)
	}

	// Row above should be the composer prompt row (not blank)
	secondLast := stripSGR(rows[22])
	if strings.TrimSpace(secondLast) == "" {
		t.Errorf("row 22 (above hint) is blank but should be composer prompt")
	}
}

// hintBlankCaps are the two glyph sets every blank-row test runs in.
var hintBlankCaps = []struct {
	name string
	caps Caps
}{
	{"Unicode", Caps{Unicode: true}},
	{"ASCII", Caps{Unicode: false}},
}

// assertHintBlankOnlyUnderOverlay checks, in one frame geometry, that the hint
// row is present with no overlay and is exactly a full-width run of spaces once
// the overlay opens. The control frame is what proves the blanking happened
// rather than the row never having had content.
func assertHintBlankOnlyUnderOverlay(t *testing.T, m Model, open func(*Model), isOpen func(Model) bool, label string) {
	t.Helper()
	const h = 24

	plain := strings.Split(m.View(), "\n")
	if len(plain) != h {
		t.Fatalf("no overlay: view has %d rows, want %d", len(plain), h)
	}
	if strings.TrimSpace(stripSGR(plain[h-1])) == "" {
		t.Fatalf("no overlay: hint row is blank, so the blanking assertion below proves nothing")
	}

	open(&m)
	if !isOpen(m) {
		t.Fatalf("%s did not open", label)
	}
	rows := strings.Split(m.View(), "\n")
	if len(rows) != h {
		t.Errorf("with %s open, view has %d rows, want %d", label, len(rows), h)
	}
	// The frame is the left margin plus the content width; the last terminal
	// column is deliberately unused, so "full frame width" is lay.Pad+lay.ContentW.
	lay := m.layout()
	frameW := lay.Pad + lay.ContentW
	if got, want := stripSGR(rows[len(rows)-1]), strings.Repeat(" ", frameW); got != want {
		t.Errorf("hint row under %s is not %d spaces: %q", label, frameW, got)
	}
}

// TestHintBlankUnderModal verifies that at 80x24 with /help open the last row is
// exactly the full frame width of spaces, in both glyph sets, and that the same
// frame without the modal carries a hint.
func TestHintBlankUnderModal(t *testing.T) {
	for _, c := range hintBlankCaps {
		t.Run(c.name, func(t *testing.T) {
			m := New(Options{Caps: c.caps, Session: SessionInfo{Project: "test"}})
			mm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
			m = mm.(Model)
			assertHintBlankOnlyUnderOverlay(t, m,
				func(m *Model) { m.openModal(ModalState{Kind: ModalHelp, Title: "help", Lines: helpLines()}) },
				func(m Model) bool { return m.modal != nil }, "modal")
		})
	}
}

// TestHintBlankUnderConfirm verifies the same with a confirm prompt reached
// through a real path (/new with transcript content).
func TestHintBlankUnderConfirm(t *testing.T) {
	for _, c := range hintBlankCaps {
		t.Run(c.name, func(t *testing.T) {
			m := New(Options{Caps: c.caps, Session: SessionInfo{Project: "test"}})
			mm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
			m = mm.(Model)
			m.loadFixture()
			m.pendingApproval = 0
			m.approvalReleased = false
			m.setBase(BaseComposing)
			m.busy.Active = true
			assertHintBlankOnlyUnderOverlay(t, m,
				func(m *Model) {
					next, _ := m.runSlash("new", "", m.layout())
					*m = next.(Model)
				},
				func(m Model) bool { return m.confirm != nil }, "confirm")
		})
	}
}

// TestHintWhileBusy verifies that after a message submission setting busy,
// the last row is "⇧↑/Tab cards · Esc cancel turn" and contains neither
// "history" nor "/help".
func TestHintWhileBusy(t *testing.T) {
	m := New(Options{
		Caps:    Caps{Unicode: true},
		Session: SessionInfo{Project: "test"},
	})
	mm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = mm.(Model)

	// Set to composing mode
	m.setBase(BaseComposing)

	// Simulate message submission by setting busy
	m.busy.Active = true

	view := m.View()
	rows := strings.Split(view, "\n")

	lastRow := stripSGR(rows[23])
	trimmedLast := strings.TrimSpace(lastRow)

	// Should contain the busy composing hint
	if !strings.Contains(trimmedLast, "⇧↑/Tab") && !strings.Contains(trimmedLast, "Shift+Up/Tab") {
		t.Errorf("hint under busy should contain shift+up/tab, got: %q", trimmedLast)
	}

	if !strings.Contains(trimmedLast, "Esc cancel turn") {
		t.Errorf("hint under busy should contain 'Esc cancel turn', got: %q", trimmedLast)
	}

	// Should NOT contain history or /help
	if strings.Contains(trimmedLast, "history") {
		t.Errorf("hint under busy should not contain 'history', got: %q", trimmedLast)
	}

	if strings.Contains(trimmedLast, "/help") {
		t.Errorf("hint under busy should not contain '/help', got: %q", trimmedLast)
	}
}

// TestHintContentStrings asserts the exact full trimmed last row (hint row) for each
// mode in both Unicode and ASCII. The brief requires == equality, not substring matching.
func TestHintContentStrings(t *testing.T) {
	cases := []struct {
		name        string
		setup       func(*Model)
		wantUnicode string
		wantASCII   string
	}{
		{
			name: "Composing",
			setup: func(m *Model) {
				m.setBase(BaseComposing)
			},
			wantUnicode: "↑↓ history · ⇧↑/Tab cards · /help",
			wantASCII:   "Up/Down history . Shift+Up/Tab cards . /help",
		},
		{
			name: "Composing busy",
			setup: func(m *Model) {
				m.setBase(BaseComposing)
				m.busy.Active = true
			},
			wantUnicode: "⇧↑/Tab cards · Esc cancel turn",
			wantASCII:   "Shift+Up/Tab cards . Esc cancel turn",
		},
		{
			name: "Browsing",
			setup: func(m *Model) {
				m.loadFixture()
				// Clear pending approval to be in pure browsing mode
				m.pendingApproval = 0
				m.approvalReleased = false
				m.setBase(BaseBrowsing)
			},
			wantUnicode: "↑↓ select · Enter preview · d detail · Tab/Esc prompt",
			wantASCII:   "Up/Down select . Enter preview . d detail . Tab/Esc prompt",
		},
		{
			name: "Armed, no grant",
			setup: func(m *Model) {
				m.loadFixture()
				// Create approval without grant
				approval := &ApprovalCard{Kind: ApprovalPatch}
				id := m.appendBlock(Item{Kind: KindApproval, Approval: approval})
				m.pendingApproval = id
				m.approvalReleased = false
			},
			wantUnicode: "y approve · n reject · d detail · Esc cancel",
			wantASCII:   "y approve . n reject . d detail . Esc cancel",
		},
		{
			name: "Armed, with grant",
			setup: func(m *Model) {
				m.loadFixture()
				// Create approval with grant
				approval := &ApprovalCard{Kind: ApprovalCommand, GrantScope: "session"}
				id := m.appendBlock(Item{Kind: KindApproval, Approval: approval})
				m.pendingApproval = id
				m.approvalReleased = false
			},
			wantUnicode: "y approve · a session · n reject · d detail · Esc cancel",
			wantASCII:   "y approve . a session . n reject . d detail . Esc cancel",
		},
		{
			name: "Released",
			setup: func(m *Model) {
				m.loadFixture()
				approval := &ApprovalCard{Kind: ApprovalCommand}
				id := m.appendBlock(Item{Kind: KindApproval, Approval: approval})
				m.pendingApproval = id
				m.approvalReleased = true
			},
			wantUnicode: "Tab back to the card · Esc cancel",
			wantASCII:   "Tab back to the card . Esc cancel",
		},
	}

	for _, c := range cases {
		// Test Unicode
		t.Run(c.name+" (Unicode)", func(t *testing.T) {
			m := New(Options{
				Caps:    Caps{Unicode: true},
				Session: SessionInfo{Project: "test"},
			})
			mm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
			m = mm.(Model)

			c.setup(&m)

			view := m.View()
			rows := strings.Split(view, "\n")
			lastRow := stripSGR(rows[23])
			trimmedLast := strings.TrimSpace(lastRow)

			if trimmedLast != c.wantUnicode {
				t.Errorf("hint mismatch:\n  got:  %q\n  want: %q", trimmedLast, c.wantUnicode)
			}
		})

		// Test ASCII
		t.Run(c.name+" (ASCII)", func(t *testing.T) {
			m := New(Options{
				Caps:    Caps{Unicode: false},
				Session: SessionInfo{Project: "test"},
			})
			mm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
			m = mm.(Model)

			c.setup(&m)

			view := m.View()
			rows := strings.Split(view, "\n")
			lastRow := stripSGR(rows[23])
			trimmedLast := strings.TrimSpace(lastRow)

			if trimmedLast != c.wantASCII {
				t.Errorf("hint mismatch:\n  got:  %q\n  want: %q", trimmedLast, c.wantASCII)
			}

			// Verify every rune is ASCII
			for i, r := range trimmedLast {
				if r > 0x7f {
					t.Errorf("rune %d is non-ASCII: %q (U+%04X)", i, r, r)
				}
			}
		})
	}
}

// TestHintTruncation asserts the exact truncated strings at a real 40-column terminal.
func TestHintTruncation(t *testing.T) {
	cases := []struct {
		name        string
		setup       func(*Model)
		wantUnicode string
		wantASCII   string
	}{
		{
			name: "Browsing",
			setup: func(m *Model) {
				// Clear pending approval to be in pure browsing mode
				m.pendingApproval = 0
				m.approvalReleased = false
				m.setBase(BaseBrowsing)
			},
			wantUnicode: "↑↓ select · Enter preview · d detail",
			wantASCII:   "Up/Down select . Enter preview",
		},
		{
			name: "Armed with grant",
			setup: func(m *Model) {
				// Clear default fixture approvals first
				m.pendingApproval = 0
				approval := &ApprovalCard{Kind: ApprovalCommand, GrantScope: "session"}
				id := m.appendBlock(Item{Kind: KindApproval, Approval: approval})
				m.pendingApproval = id
				m.approvalReleased = false
			},
			wantUnicode: "y approve · a session · n reject",
			wantASCII:   "y approve . a session . n reject",
		},
		{
			name: "Busy",
			setup: func(m *Model) {
				m.pendingApproval = 0
				m.approvalReleased = false
				m.setBase(BaseComposing)
				m.busy.Active = true
			},
			// Fits whole at 40 columns: no item is dropped.
			wantUnicode: "⇧↑/Tab cards · Esc cancel turn",
			wantASCII:   "Shift+Up/Tab cards . Esc cancel turn",
		},
		{
			name: "Released",
			setup: func(m *Model) {
				id := m.appendBlock(Item{Kind: KindApproval, Approval: &ApprovalCard{Kind: ApprovalCommand}})
				m.pendingApproval = id
				m.approvalReleased = true
			},
			wantUnicode: "Tab back to the card · Esc cancel",
			wantASCII:   "Tab back to the card . Esc cancel",
		},
		{
			name: "Armed, no grant",
			setup: func(m *Model) {
				id := m.appendBlock(Item{Kind: KindApproval, Approval: &ApprovalCard{Kind: ApprovalPatch}})
				m.pendingApproval = id
				m.approvalReleased = false
			},
			wantUnicode: "y approve · n reject · d detail",
			wantASCII:   "y approve . n reject . d detail",
		},
		{
			name: "Composing",
			setup: func(m *Model) {
				m.pendingApproval = 0
				m.approvalReleased = false
				m.setBase(BaseComposing)
			},
			wantUnicode: "↑↓ history · ⇧↑/Tab cards · /help",
			wantASCII:   "Up/Down history . Shift+Up/Tab cards",
		},
	}

	for _, c := range cases {
		// Test Unicode at 40 columns
		t.Run(c.name+" (Unicode at 40)", func(t *testing.T) {
			m := New(Options{
				Caps:    Caps{Unicode: true},
				Session: SessionInfo{Project: "test"},
			})
			mm, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 24})
			m = mm.(Model)
			m.loadFixture()

			c.setup(&m)

			lay := m.layout()
			hint := m.hintRow(lay)
			plainHint := stripSGR(hint)
			trimmed := strings.TrimSpace(plainHint)

			if trimmed != c.wantUnicode {
				t.Errorf("truncated hint mismatch:\n  got:  %q\n  want: %q", trimmed, c.wantUnicode)
			}
		})

		// Test ASCII at 40 columns
		t.Run(c.name+" (ASCII at 40)", func(t *testing.T) {
			m := New(Options{
				Caps:    Caps{Unicode: false},
				Session: SessionInfo{Project: "test"},
			})
			mm, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 24})
			m = mm.(Model)
			m.loadFixture()

			c.setup(&m)

			lay := m.layout()
			hint := m.hintRow(lay)
			plainHint := stripSGR(hint)
			trimmed := strings.TrimSpace(plainHint)

			if trimmed != c.wantASCII {
				t.Errorf("truncated hint mismatch:\n  got:  %q\n  want: %q", trimmed, c.wantASCII)
			}

			// Verify every rune is ASCII
			for i, r := range trimmed {
				if r > 0x7f {
					t.Errorf("rune %d is non-ASCII in ASCII mode: %q (U+%04X)", i, r, r)
				}
			}
		})
	}
}

// TestHintASCII verifies ASCII mode uses the Unicode flag correctly.
func TestHintASCII(t *testing.T) {
	m := New(Options{Caps: Caps{Unicode: false}})
	m.width = 80
	m.height = 24
	m.setBase(BaseComposing)

	lay := m.layout()
	hint := m.hintRow(lay)
	plainHint := stripSGR(hint)

	// Should use ASCII arrows from the Unicode flag
	if m.gly.Unicode {
		t.Error("ASCII model should have gly.Unicode=false")
	}

	// Should contain ASCII versions
	if !strings.Contains(plainHint, "Shift+Up") {
		t.Errorf("ASCII hint should contain 'Shift+Up', got: %q", plainHint)
	}

	if !strings.Contains(plainHint, "Up/Down") {
		t.Errorf("ASCII hint should contain 'Up/Down', got: %q", plainHint)
	}

	// Should not contain Unicode arrows
	if strings.Contains(plainHint, "↑") || strings.Contains(plainHint, "↓") ||
		strings.Contains(plainHint, "⇧") {
		t.Errorf("ASCII hint should not contain Unicode arrows, got: %q", plainHint)
	}
}

// TestTruncateHintNothingFits covers the path where even the first item is wider
// than the row: the hint is dropped to "" rather than clipped mid-word.
func TestTruncateHintNothingFits(t *testing.T) {
	m := New(Options{Caps: Caps{Unicode: true}})
	hint := "Enter preview " + m.gly.Bullet + " d detail"
	if got := m.truncateHint(hint, 5); got != "" {
		t.Errorf("truncateHint at width 5 = %q, want \"\"", got)
	}
	// One item exactly fits: it is kept, the rest dropped.
	if got := m.truncateHint(hint, cellWidth("Enter preview")); got != "Enter preview" {
		t.Errorf("truncateHint at one-item width = %q, want %q", got, "Enter preview")
	}
}
