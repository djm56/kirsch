package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// newTestModel builds an 80x24 model with the Unicode glyph set.
//
// Unicode is not optional here. New(Options{}) defaults to the ASCII glyphs, in
// which the check mark renders as "[ok]" and the bullet as ".", so every
// assertion that a "✓ approved" line is absent passes trivially and every
// assertion that one is present can never pass. The guard below keeps an ASCII
// default from returning silently; TestOneCardHelperIsUnicode pins it too.
func newTestModel() Model {
	m := New(Options{Caps: Caps{Colour: false, Unicode: true}})
	if m.gly.OK != "\u2713" || m.gly.Bullet != "\u00b7" {
		panic("newTestModel: glyphs are not the Unicode set; one-card assertions would be vacuous")
	}
	updatedModel, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return updatedModel.(Model)
}

// send feeds one message through Update and returns the resulting model.
func send(m Model, msg tea.Msg) Model {
	updated, _ := m.Update(msg)
	return updated.(Model)
}

// headLine returns the first rendered line containing sub, or "" if none.
func headLine(view, sub string) string {
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, sub) {
			return line
		}
	}
	return ""
}

// itemOf returns the transcript item with the given ID, or nil.
func itemOf(m Model, id ItemID) *Item {
	it, _, ok := m.tr.Find(id)
	if !ok {
		return nil
	}
	return &it
}

// TestOneCardHelperIsUnicode pins the helper's glyph set. The one-card tests
// assert on "\u2713 approved" and " \u00b7 approved"; under ASCII neither exists.
func TestOneCardHelperIsUnicode(t *testing.T) {
	m := newTestModel()
	if m.gly.OK != "\u2713" {
		t.Errorf("gly.OK = %q, want the Unicode check mark", m.gly.OK)
	}
	if m.gly.Bullet != "\u00b7" {
		t.Errorf("gly.Bullet = %q, want the Unicode bullet", m.gly.Bullet)
	}
}

// TestOneCardLinking verifies that tool and approval cards are correctly linked
// when an approval is requested with a tool ID.
func TestOneCardLinking(t *testing.T) {
	m := newTestModel()

	// Start a tool
	toolMsg := ToolStartedMsg{ID: 7, Name: "run_command", Target: "echo test"}
	updatedModel, _ := m.Update(toolMsg)
	m = updatedModel.(Model)

	// Request approval linked to that tool
	approvalMsg := ApprovalRequestedMsg{
		ID:     1,
		ToolID: 7,
		Kind:   "command",
	}
	updatedModel, _ = m.Update(approvalMsg)
	m = updatedModel.(Model)

	// Verify linking
	var toolCard, approvalCard *Item
	for _, item := range m.tr.Items() {
		switch item.Kind {
		case KindTool:
			toolCard = &item
		case KindApproval:
			approvalCard = &item
		case KindUser, KindAssistant, KindError, KindNotice, KindThinking:
			// Other item kinds are not relevant to this test.
		}
	}

	if toolCard == nil {
		t.Fatal("tool card not found")
	}
	if approvalCard == nil {
		t.Fatal("approval card not found")
	}

	if toolCard.Tool.ApprovalItem == 0 {
		t.Error("tool card should have ApprovalItem set")
	}
	if approvalCard.Approval.ToolItem == 0 {
		t.Error("approval card should have ToolItem set")
	}
	if toolCard.Tool.ApprovalItem != approvalCard.ID {
		t.Errorf("tool.ApprovalItem = %d, want %d", toolCard.Tool.ApprovalItem, approvalCard.ID)
	}
	if approvalCard.Approval.ToolItem != toolCard.ID {
		t.Errorf("approval.ToolItem = %d, want %d", approvalCard.Approval.ToolItem, toolCard.ID)
	}
}

// TestOneCardApprovalLineAbsent verifies that a folded approval does not show
// the "✓ approved" line and that the tool head itself carries the badge.
func TestOneCardApprovalLineAbsent(t *testing.T) {
	m := newTestModel()
	m = send(m, ToolStartedMsg{ID: 7, Name: "run_command", Target: "echo test"})
	m = send(m, ApprovalRequestedMsg{ID: 1, ToolID: 7, Kind: "command"})
	m = send(m, key('y'))
	m = send(m, ToolCompletedMsg{ID: 7, Name: "run_command", Target: "echo test", OK: true, Summary: "done"})

	view := m.View()

	head := headLine(view, "run_command")
	if head == "" {
		t.Fatalf("no tool head line in view:\n%s", view)
	}
	if !strings.Contains(head, " \u00b7 approved") {
		t.Errorf("tool head should carry the \u00b7 approved badge: %q", head)
	}

	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "\u2713 approved") {
			t.Errorf("folded approval must not render a \u2713 approved line: %q", line)
		}
	}
}

// TestOneCardNavigateAcrossFolded verifies that navigation skips a folded
// approval in both directions.
func TestOneCardNavigateAcrossFolded(t *testing.T) {
	m := newTestModel()

	// Items in order: tool A, its folded approval, tool B.
	m = send(m, ToolStartedMsg{ID: 1, Name: "toolA", Target: "a"})
	m = send(m, ApprovalRequestedMsg{ID: 1, ToolID: 1, Kind: "command"})
	m = send(m, key('y'))
	m = send(m, ToolCompletedMsg{ID: 1, Name: "toolA", Target: "a", OK: true})
	m = send(m, ToolStartedMsg{ID: 2, Name: "toolB", Target: "b"})
	m = send(m, ToolCompletedMsg{ID: 2, Name: "toolB", Target: "b", OK: true})

	m = send(m, tea.KeyMsg{Type: tea.KeyShiftUp})
	if m.mode() != ModeBrowsing {
		t.Fatalf("mode = %v, want Browsing", m.mode())
	}
	selName := func() string {
		it := itemOf(m, m.sel)
		if it == nil {
			return "<none>"
		}
		if it.Kind != KindTool {
			return "<kind " + itoa(int(it.Kind)) + ">"
		}
		return it.Tool.Name
	}
	if got := selName(); got != "toolB" {
		t.Fatalf("on entry want toolB selected, got %s", got)
	}

	// One step up from B must land on A, not on the folded approval between.
	m = send(m, tea.KeyMsg{Type: tea.KeyUp})
	if got := selName(); got != "toolA" {
		t.Fatalf("after up, want toolA, got %s", got)
	}

	// One step down from A must land on B.
	m = send(m, tea.KeyMsg{Type: tea.KeyDown})
	if got := selName(); got != "toolB" {
		t.Fatalf("after down, want toolB, got %s", got)
	}
}

// TestOneCardFirstEntrySkipsFolded verifies the real first-entry path: with
// m.sel still zero, entering Browsing calls lastSelectable, which must skip the
// folded approval that is the last item in the transcript.
//
// ToolCompletedMsg sets m.sel, so it must not run before Shift+Up or setBase
// never reaches lastSelectable.
func TestOneCardFirstEntrySkipsFolded(t *testing.T) {
	m := newTestModel()
	m = send(m, ToolStartedMsg{ID: 1, Name: "toolA", Target: "a"})
	m = send(m, ApprovalRequestedMsg{ID: 1, ToolID: 1, Kind: "command"})
	m = send(m, key('y'))

	if m.sel != 0 {
		t.Fatalf("precondition: m.sel = %d, want 0 so Shift+Up exercises lastSelectable", m.sel)
	}

	m = send(m, tea.KeyMsg{Type: tea.KeyShiftUp})
	if m.mode() != ModeBrowsing {
		t.Fatalf("mode = %v, want Browsing", m.mode())
	}

	it := itemOf(m, m.sel)
	if it == nil {
		t.Fatalf("m.sel = %d matches no item", m.sel)
	}
	if it.Kind != KindTool {
		t.Fatalf("first entry should select the tool, got item kind %d", it.Kind)
	}
	if it.Tool.Name != "toolA" {
		t.Fatalf("want toolA, got %s", it.Tool.Name)
	}

	// The gutter is drawn on the tool head line.
	head := headLine(m.View(), "toolA")
	if !strings.HasPrefix(strings.TrimLeft(head, " "), m.gly.Gutter) {
		t.Errorf("gutter should be on the tool head, got %q", head)
	}

	// d opens the tool's content modal, not the approval detail.
	m = send(m, key('d'))
	if m.modal == nil {
		t.Fatal("modal should be open")
	}
	if m.modal.Kind != ModalContent {
		t.Errorf("modal kind = %v, want ModalContent", m.modal.Kind)
	}
	if m.modal.Title == "approval detail" || m.modal.Source != it.ID {
		t.Errorf("modal is not the tool's: title %q source %d (tool %d)", m.modal.Title, m.modal.Source, it.ID)
	}
}

// TestOneCardSelectionMoveByKeys verifies that selection moves via keyboard and not assignments.
func TestOneCardSelectionMoveByKeys(t *testing.T) {
	m := newTestModel()

	// Send ToolStartedMsg, then the linked ApprovalRequestedMsg
	updatedModel, _ := m.Update(ToolStartedMsg{ID: 1, Name: "run_command", Target: "test"})
	m = updatedModel.(Model)
	updatedModel, _ = m.Update(ApprovalRequestedMsg{ID: 1, ToolID: 1, Kind: "command"})
	m = updatedModel.(Model)

	// Press d to set focus on pending approval card (open it)
	updatedModel, _ = m.Update(key('d'))
	m = updatedModel.(Model)

	if m.modal == nil {
		t.Fatal("modal should be open after 'd'")
	}

	// Escape to close the modal
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updatedModel.(Model)

	// Press y to approve
	updatedModel, _ = m.Update(key('y'))
	m = updatedModel.(Model)

	// Navigate to Browsing if needed and check that selection is on tool
	if m.mode() != ModeBrowsing {
		updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftUp})
		m = updatedModel.(Model)
	}

	if m.mode() != ModeBrowsing {
		t.Fatalf("mode = %v, want Browsing", m.mode())
	}

	// The gutter should be on the tool's head line
	var selectedItem *Item
	for _, item := range m.tr.Items() {
		if item.ID == m.sel {
			selectedItem = &item
			break
		}
	}

	if selectedItem == nil || selectedItem.Kind != KindTool {
		t.Fatal("selection should be on tool card")
	}
}

// TestOneCardNoDoubleBlank verifies no extra blank lines between cards.
func TestOneCardNoDoubleBlank(t *testing.T) {
	m := newTestModel()

	// Create tool A, a linked approved card for A, and tool B
	updatedModel, _ := m.Update(ToolStartedMsg{ID: 1, Name: "toolA", Target: "a"})
	m = updatedModel.(Model)
	updatedModel, _ = m.Update(ApprovalRequestedMsg{ID: 1, ToolID: 1, Kind: "command"})
	m = updatedModel.(Model)
	updatedModel, _ = m.Update(key('y'))
	m = updatedModel.(Model)
	updatedModel, _ = m.Update(ToolCompletedMsg{ID: 1, Name: "toolA", Target: "a", OK: true})
	m = updatedModel.(Model)
	updatedModel, _ = m.Update(ToolStartedMsg{ID: 2, Name: "toolB", Target: "b"})
	m = updatedModel.(Model)
	updatedModel, _ = m.Update(ToolCompletedMsg{ID: 2, Name: "toolB", Target: "b", OK: true})
	m = updatedModel.(Model)

	view := m.View()
	lines := strings.Split(view, "\n")

	// Look for two consecutive empty lines between toolA and toolB
	for i := 0; i < len(lines)-1; i++ {
		if strings.TrimSpace(lines[i]) == "" && strings.TrimSpace(lines[i+1]) == "" {
			// Check if these are between toolA and toolB
			before := ""
			after := ""
			if i > 0 {
				before = lines[i-1]
			}
			if i+2 < len(lines) {
				after = lines[i+2]
			}
			if strings.Contains(before, "toolA") && strings.Contains(after, "toolB") {
				t.Errorf("double blank line found between toolA and toolB at line %d", i)
			}
		}
	}
}

// TestOneCardDiffViaToolCard verifies that pressing 'd' on a tool with linked patch approval opens diff modal.
func TestOneCardDiffViaToolCard(t *testing.T) {
	m := newTestModel()

	// Use a patch approval (Kind:"patch", DiffFilename:"a.go", DiffLines:[...]), linked and approved, with tool completed.
	updatedModel, _ := m.Update(ToolStartedMsg{ID: 1, Name: "apply_patch", Target: ""})
	m = updatedModel.(Model)
	updatedModel, _ = m.Update(ApprovalRequestedMsg{
		ID:           1,
		ToolID:       1,
		Kind:         "patch",
		DiffFilename: "a.go",
		DiffLines:    []string{"@@ -1 +1 @@", "-old", "+new"},
	})
	m = updatedModel.(Model)
	updatedModel, _ = m.Update(key('y'))
	m = updatedModel.(Model)
	updatedModel, _ = m.Update(ToolCompletedMsg{ID: 1, Name: "apply_patch", Target: "", OK: true})
	m = updatedModel.(Model)

	// Enter Browsing
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftUp})
	m = updatedModel.(Model)

	// Navigate and press 'd' to open the diff
	updatedModel, _ = m.Update(key('d'))
	m = updatedModel.(Model)

	// The modal should be open and show the diff filename
	if m.modal == nil {
		t.Fatal("modal should be open")
	}

	if m.modal.Kind != ModalDiff {
		t.Errorf("modal kind = %v, want ModalDiff", m.modal.Kind)
	}

	view := m.View()
	if !strings.Contains(view, "a.go") {
		t.Error("modal should show diff filename a.go")
	}
}

// TestOneCardScopeShown verifies that the grant scope is shown on the tool head
// for an approved-for-session call. The tool Target deliberately differs from
// GrantScope so the scope text cannot be satisfied by the Target.
func TestOneCardScopeShown(t *testing.T) {
	m := newTestModel()
	m = send(m, ToolStartedMsg{ID: 1, Name: "run_command", Target: "make ci"})
	m = send(m, ApprovalRequestedMsg{
		ID:                   1,
		ToolID:               1,
		Kind:                 "command",
		CanApproveForSession: true,
		GrantScope:           "go test",
		Subject:              "test command",
	})
	m = send(m, key('a')) // approve for session
	m = send(m, ToolCompletedMsg{ID: 1, Name: "run_command", Target: "make ci", OK: true})

	head := headLine(m.View(), "run_command")
	if head == "" {
		t.Fatal("no tool head line in view")
	}
	if !strings.Contains(head, " \u00b7 approved for session") {
		t.Errorf("head should show the session badge: %q", head)
	}
	if !strings.Contains(head, "session grant: go test") {
		t.Errorf("head should show the scope note: %q", head)
	}
}

// TestOneCardTerminalToolUnlinked verifies that approvals for completed tools are not folded.
func TestOneCardTerminalToolUnlinked(t *testing.T) {
	m := newTestModel()

	// Complete tool 7 first, then send ApprovalRequestedMsg{ToolID:7} and approve with 'y'.
	updatedModel, _ := m.Update(ToolStartedMsg{ID: 7, Name: "run_command", Target: "test"})
	m = updatedModel.(Model)
	updatedModel, _ = m.Update(ToolCompletedMsg{ID: 7, Name: "run_command", Target: "test", OK: true})
	m = updatedModel.(Model)
	updatedModel, _ = m.Update(ApprovalRequestedMsg{ID: 1, ToolID: 7, Kind: "command"})
	m = updatedModel.(Model)
	updatedModel, _ = m.Update(key('y'))
	m = updatedModel.(Model)

	view := m.View()

	// The approval card's ✓ approved line should be present (terminal tool, so not folded)
	if !strings.Contains(view, "✓ approved") {
		t.Error("approval for terminal tool should show ✓ approved line (unfolded)")
	}

	// The tool head should not have the · approved badge (approval can't fold when tool is terminal)
	lines := strings.Split(view, "\n")
	for _, line := range lines {
		if strings.Contains(line, "run_command") && strings.Contains(line, "· approved") {
			t.Errorf("tool head should not have badge for terminal tool's approval: %s", line)
		}
	}
}

// TestOneCardRefusedGrantStaysFolded verifies that a refused session grant shows allow-once.
func TestOneCardRefusedGrantStaysFolded(t *testing.T) {
	m := newTestModel()

	// Press 'a' on a grant card, then send ApprovalResolvedMsg with Approved (allow-once), then complete the tool.
	updatedModel, _ := m.Update(ToolStartedMsg{ID: 1, Name: "run_command", Target: "test"})
	m = updatedModel.(Model)
	updatedModel, _ = m.Update(ApprovalRequestedMsg{
		ID:                   1,
		ToolID:               1,
		Kind:                 "command",
		CanApproveForSession: true,
		GrantScope:           "test scope",
	})
	m = updatedModel.(Model)
	updatedModel, _ = m.Update(key('a'))
	m = updatedModel.(Model)
	updatedModel, _ = m.Update(ApprovalResolvedMsg{ID: 1, Outcome: Approved}) // Grant was refused, falls back to allow-once
	m = updatedModel.(Model)
	updatedModel, _ = m.Update(ToolCompletedMsg{ID: 1, Name: "run_command", Target: "test", OK: true})
	m = updatedModel.(Model)

	view := m.View()

	// The head should show · approved and not "for session"
	if !strings.Contains(view, "· approved") {
		t.Error("head should show · approved badge")
	}

	if strings.Contains(view, "for session") {
		t.Error("head should not show 'for session' when grant was refused")
	}

	// No ✓ approved line should be present (it's folded)
	if strings.Contains(view, "✓ approved") {
		t.Error("folded approval should not show ✓ approved line")
	}
}

// TestOneCardLateConfirmationRelayouts verifies that a confirmation arriving
// with no following message still refreshes the view: the refused grant falls
// back from "approved for session" to "approved" on the next frame.
func TestOneCardLateConfirmationRelayouts(t *testing.T) {
	m := newTestModel()
	m = send(m, ToolStartedMsg{ID: 1, Name: "run_command", Target: "make ci"})
	m = send(m, ApprovalRequestedMsg{
		ID: 1, ToolID: 1, Kind: "command",
		CanApproveForSession: true, GrantScope: "go test",
	})
	m = send(m, key('a'))
	m = send(m, ToolCompletedMsg{ID: 1, Name: "run_command", Target: "make ci", OK: true})

	if head := headLine(m.View(), "run_command"); !strings.Contains(head, "approved for session") {
		t.Fatalf("precondition: head should show the optimistic session badge: %q", head)
	}

	// The confirmation is the last message; nothing after it relayouts.
	m = send(m, ApprovalResolvedMsg{ID: 1, Outcome: Approved})

	head := headLine(m.View(), "run_command")
	if strings.Contains(head, "for session") {
		t.Errorf("stale view: head still shows the session badge: %q", head)
	}
	if !strings.Contains(head, " \u00b7 approved") {
		t.Errorf("head should show the allow-once badge: %q", head)
	}
}
