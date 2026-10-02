package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestToolCardPreview25Lines verifies that a 25-line tool output shows exactly
// the first 10 lines in the preview box, with a marker showing 10 of 25 lines.
func TestToolCardPreview25Lines(t *testing.T) {
	// Build a tool card with 25 lines of output
	lines := make([]string, 25)
	for i := 0; i < 25; i++ {
		lines[i] = "line " + itoa(i+1)
	}
	content := strings.Join(lines, "\n")

	m := New(Options{
		Version: "0.1.0",
		Caps:    Caps{Colour: false, Unicode: true},
	})
	m = drive(m,
		tea.WindowSizeMsg{Width: 80, Height: 30},
		ToolStartedMsg{ID: 1, Name: "test_tool", Target: "test"},
		ToolCompletedMsg{
			ID:      1,
			Name:    "test_tool",
			Target:  "test",
			OK:      true,
			Content: content,
		},
	)
	view := m.View()

	// Verify exactly lines 1-10 are shown
	for i := 1; i <= 10; i++ {
		if !strings.Contains(view, "line "+itoa(i)) {
			t.Errorf("line %d should be visible in preview", i)
		}
	}

	// Verify line 11 is NOT shown
	if strings.Contains(view, "line 11") {
		t.Errorf("line 11 should not be visible in preview")
	}

	// Verify the marker is present
	if !strings.Contains(view, "‹10 of 25 lines") {
		t.Errorf("marker should show '‹10 of 25 lines'; view:\n%s", view)
	}
}

// TestToolCardPreview10Lines verifies that a 10-line tool output shows all
// 10 lines with no marker.
func TestToolCardPreview10Lines(t *testing.T) {
	lines := make([]string, 10)
	for i := 0; i < 10; i++ {
		lines[i] = "line " + itoa(i+1)
	}
	content := strings.Join(lines, "\n")

	m := New(Options{
		Version: "0.1.0",
		Caps:    Caps{Colour: false, Unicode: true},
	})
	m = drive(m,
		tea.WindowSizeMsg{Width: 80, Height: 30},
		ToolStartedMsg{ID: 2, Name: "test_tool", Target: "test"},
		ToolCompletedMsg{
			ID:      2,
			Name:    "test_tool",
			Target:  "test",
			OK:      true,
			Content: content,
		},
	)
	view := m.View()

	// Verify all 10 lines are shown
	for i := 1; i <= 10; i++ {
		if !strings.Contains(view, "line "+itoa(i)) {
			t.Errorf("line %d should be visible", i)
		}
	}

	// Verify no marker is present (since all lines fit)
	if strings.Contains(view, "‹10 of") {
		t.Errorf("marker should not be present when exactly 10 lines fit; view:\n%s", view)
	}
}

// TestToolCardPreview3Lines verifies that a 3-line tool output shows all
// 3 lines with no marker.
func TestToolCardPreview3Lines(t *testing.T) {
	m := New(Options{
		Version: "0.1.0",
		Caps:    Caps{Colour: false, Unicode: true},
	})
	m = drive(m,
		tea.WindowSizeMsg{Width: 80, Height: 30},
		ToolStartedMsg{ID: 3, Name: "test_tool", Target: "test"},
		ToolCompletedMsg{
			ID:      3,
			Name:    "test_tool",
			Target:  "test",
			OK:      true,
			Content: "line 1\nline 2\nline 3",
		},
	)
	view := m.View()

	// Verify all 3 lines are shown
	for i := 1; i <= 3; i++ {
		if !strings.Contains(view, "line "+itoa(i)) {
			t.Errorf("line %d should be visible", i)
		}
	}

	// Verify no marker is present
	if strings.Contains(view, "‹") {
		t.Errorf("marker should not be present for 3-line output; view:\n%s", view)
	}
}

// TestToolCardTogglePreviewAndCollapse verifies that Enter on a selected
// tool card toggles between preview and collapsed states.
func TestToolCardTogglePreviewAndCollapse(t *testing.T) {
	lines := make([]string, 25)
	for i := 0; i < 25; i++ {
		lines[i] = "line " + itoa(i+1)
	}
	content := strings.Join(lines, "\n")

	m := New(Options{
		Version: "0.1.0",
		Caps:    Caps{Colour: false, Unicode: true},
	})
	m = drive(m,
		tea.WindowSizeMsg{Width: 80, Height: 30},
		ToolStartedMsg{ID: 4, Name: "test_tool", Target: "test"},
		ToolCompletedMsg{
			ID:      4,
			Name:    "test_tool",
			Target:  "test",
			OK:      true,
			Content: content,
		},
		// Enter toggle only works in Browsing mode; navigate to it with Shift+Up
		tea.KeyMsg{Type: tea.KeyShiftUp},
	)

	// Initially in preview (showing first 10 lines)
	view := m.View()
	if !strings.Contains(view, "line 1") {
		t.Errorf("preview should show line 1 initially; view:\n%s", view)
	}
	if !strings.Contains(view, "line 10") {
		t.Errorf("preview should show line 10 initially; view:\n%s", view)
	}

	// Press Enter to collapse
	m = drive(m, tea.KeyMsg{Type: tea.KeyEnter})
	view = m.View()

	if strings.Contains(view, "line 1") {
		t.Errorf("collapsed card should not show line 1; view:\n%s", view)
	}
	if strings.Contains(view, "‹10 of") {
		t.Errorf("marker should not appear in collapsed state; view:\n%s", view)
	}

	// Press Enter again to restore preview
	m = drive(m, tea.KeyMsg{Type: tea.KeyEnter})
	view = m.View()

	if !strings.Contains(view, "line 1") {
		t.Errorf("preview should be restored after second Enter; view:\n%s", view)
	}
	if !strings.Contains(view, "line 10") {
		t.Errorf("preview should show line 10 after second Enter; view:\n%s", view)
	}
}

// TestEnterOnNonToolCardChangesNothing verifies that pressing Enter on a
// non-tool card (such as a resolved approval) does not change the view.
func TestEnterOnNonToolCardChangesNothing(t *testing.T) {
	lines := make([]string, 25)
	for i := 0; i < 25; i++ {
		lines[i] = "line " + itoa(i+1)
	}
	content := strings.Join(lines, "\n")

	m := New(Options{
		Version: "0.1.0",
		Caps:    Caps{Colour: false, Unicode: true},
	})
	m = drive(m,
		tea.WindowSizeMsg{Width: 80, Height: 30},
		ToolStartedMsg{ID: 1, Name: "test_tool", Target: "test"},
		ToolCompletedMsg{
			ID:      1,
			Name:    "test_tool",
			Target:  "test",
			OK:      true,
			Content: content,
		},
		ApprovalRequestedMsg{
			ID:          1,
			Description: "test approval",
			Kind:        "patch",
			Subject:     "2 files changed",
			Detail:      []string{"detail line 1", "detail line 2", "detail line 3"},
		},
		// Resolve the approval with 'y'
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}},
	)

	// Navigate to Browsing mode by pressing Shift+Up (from Composing to Browsing)
	m = drive(m, tea.KeyMsg{Type: tea.KeyShiftUp})

	// Now navigate to the approval card using ↓ from Browsing
	// After the first Up, we should be in Browsing with the tool card selected
	// The approval card is after the tool card, so we press Down to move to it
	m = drive(m, tea.KeyMsg{Type: tea.KeyDown})

	// Verify the gutter marker is on the approval card's line (look for the gutter character)
	view := m.View()
	viewLines := strings.Split(view, "\n")
	gutterFound := false
	for _, line := range viewLines {
		if strings.Contains(line, "apply_patch") && strings.Contains(line, "┃") {
			gutterFound = true
			break
		}
	}
	if !gutterFound {
		t.Errorf("gutter marker should be on approval card line; view:\n%s", view)
	}

	// Verify the tool card still shows its preview rows before pressing Enter
	if !strings.Contains(view, "line 1") || !strings.Contains(view, "line 10") {
		t.Errorf("tool card preview should be visible before Enter; view:\n%s", view)
	}

	// Capture the view
	capturedView := m.View()

	// Press Enter (should not change the view for non-tool cards)
	m = drive(m, tea.KeyMsg{Type: tea.KeyEnter})
	viewAfterEnter := m.View()

	// Assert the view is byte-identical
	if capturedView != viewAfterEnter {
		t.Errorf("view should not change when Enter is pressed on non-tool card\nBefore:\n%s\nAfter:\n%s",
			capturedView, viewAfterEnter)
	}

	// Verify the tool card still shows its preview rows after pressing Enter
	if !strings.Contains(viewAfterEnter, "line 1") || !strings.Contains(viewAfterEnter, "line 10") {
		t.Errorf("tool card preview should still be visible after Enter; view:\n%s", viewAfterEnter)
	}
}

// TestToolCardDOpensFull verifies that pressing 'd' opens the full output modal
// regardless of the preview state.
func TestToolCardDOpensFull(t *testing.T) {
	lines := make([]string, 25)
	for i := 0; i < 25; i++ {
		lines[i] = "line " + itoa(i+1)
	}
	content := strings.Join(lines, "\n")

	m := New(Options{
		Version: "0.1.0",
		Caps:    Caps{Colour: false, Unicode: true},
	})
	m = drive(m,
		tea.WindowSizeMsg{Width: 80, Height: 30},
		ToolStartedMsg{ID: 5, Name: "test_tool", Target: "test"},
		ToolCompletedMsg{
			ID:      5,
			Name:    "test_tool",
			Target:  "test",
			OK:      true,
			Content: content,
		},
		// Navigate to browsing mode for 'd' to work
		tea.KeyMsg{Type: tea.KeyShiftUp},
		// Press 'd' to open the modal
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}},
	)

	// Verify the modal is open
	if m.modal == nil {
		t.Errorf("modal should be open after 'd'")
	}

	// Verify all 25 lines are in the modal
	if m.modal != nil && len(m.modal.Lines) < 25 {
		t.Errorf("modal should contain all 25 lines, got %d", len(m.modal.Lines))
	}
}

// TestToolCardMarkerFormat verifies the exact marker text format.
func TestToolCardMarkerFormat(t *testing.T) {
	lines := make([]string, 25)
	for i := 0; i < 25; i++ {
		lines[i] = "line " + itoa(i+1)
	}
	content := strings.Join(lines, "\n")

	m := New(Options{
		Version: "0.1.0",
		Caps:    Caps{Colour: false, Unicode: true},
	})
	m = drive(m,
		tea.WindowSizeMsg{Width: 80, Height: 30},
		ToolStartedMsg{ID: 6, Name: "test_tool", Target: "test"},
		ToolCompletedMsg{
			ID:      6,
			Name:    "test_tool",
			Target:  "test",
			OK:      true,
			Content: content,
		},
	)
	view := m.View()

	// Verify the marker format matches the brief's requirement
	if !strings.Contains(view, "‹10 of 25 lines — d full output · Enter collapse›") {
		t.Errorf("marker format should be '‹10 of 25 lines — d full output · Enter collapse›'; view:\n%s", view)
	}
}

// TestEmptyOutputRendersNoBox verifies that empty output renders a head and no box.
func TestEmptyOutputRendersNoBox(t *testing.T) {
	m := New(Options{
		Version: "0.1.0",
		Caps:    Caps{Colour: false, Unicode: true},
	})
	m = drive(m,
		tea.WindowSizeMsg{Width: 80, Height: 30},
		ToolStartedMsg{ID: 7, Name: "test_tool", Target: "test"},
		ToolCompletedMsg{
			ID:        7,
			Name:      "test_tool",
			Target:    "test",
			OK:        false,
			ErrorKind: "not_found",
			ErrorMsg:  "file not found",
			Content:   "",
		},
	)
	view := m.View()

	// Verify the head is present
	if !strings.Contains(view, "test_tool") {
		t.Errorf("tool name should be visible; view:\n%s", view)
	}

	// Verify no box border is rendered (no ┌ or │ inside a box for empty output)
	// The head line is present but the output box should not be
	lines := strings.Split(view, "\n")
	boxFound := false
	for _, line := range lines {
		// Look for box borders that would indicate an output box
		if strings.Contains(line, "┌") || strings.Contains(line, "└") || strings.Contains(line, "│") {
			boxFound = true
			break
		}
	}
	if boxFound {
		t.Errorf("no output box should be rendered for empty content; view:\n%s", view)
	}
}

// TestCollapseClampScrollOffset verifies that after collapsing a card while
// unpinned, the scroll offset is clamped so no blank rows pad the pane.
func TestCollapseClampScrollOffset(t *testing.T) {
	lines := make([]string, 25)
	for i := 0; i < 25; i++ {
		lines[i] = "line " + itoa(i+1)
	}
	content := strings.Join(lines, "\n")

	m := New(Options{
		Version: "0.1.0",
		Caps:    Caps{Colour: false, Unicode: true},
	})
	// Use a small pane height to make scrolling more visible
	m = drive(m,
		tea.WindowSizeMsg{Width: 80, Height: 12},
		ToolStartedMsg{ID: 8, Name: "test_tool", Target: "test"},
		ToolCompletedMsg{
			ID:      8,
			Name:    "test_tool",
			Target:  "test",
			OK:      true,
			Content: content,
		},
	)

	// Navigate to Browsing mode first by pressing Shift+Up (from Composing to Browsing)
	m = drive(m, tea.KeyMsg{Type: tea.KeyShiftUp})

	// Scroll down a bit to ensure we're past the top
	m = drive(m, tea.KeyMsg{Type: tea.KeyPgDown})

	// Then scroll back up to create an offset that we can then clamp
	// After scrolling down then up, we should have an offset > 0
	m = drive(m, tea.KeyMsg{Type: tea.KeyPgUp})

	lay := m.layout()
	preCollapseLines := len(m.lines)

	// Collapse the tool card to see the effect on line count
	// We'll store the model state before collapse
	preCollapseOffset := m.scroll.Offset

	// Precondition 1: verify the view is unpinned
	if m.scroll.Pinned {
		t.Fatalf("scroll should be unpinned before collapse")
	}

	// Precondition 2: verify offset is positive (scrolled up from bottom)
	if preCollapseOffset == 0 {
		t.Fatalf("offset should be non-zero to test clamping; need to scroll up more")
	}

	// Collapse the card (press Enter)
	m = drive(m, tea.KeyMsg{Type: tea.KeyEnter})

	lay = m.layout()
	postCollapseOffset := m.scroll.Offset
	postCollapseMaxOffset := maxInt(0, len(m.lines)-lay.TranscriptH)

	// Assert the offset is clamped
	// After collapsing, the offset should not exceed the maximum for the new line count
	if postCollapseOffset > postCollapseMaxOffset {
		t.Errorf("offset %d should be clamped to maximum %d after collapse; "+
			"pre-collapse offset=%d, pre-collapse lines=%d, post-collapse lines=%d",
			postCollapseOffset, postCollapseMaxOffset, preCollapseOffset, preCollapseLines, len(m.lines))
	}
}
