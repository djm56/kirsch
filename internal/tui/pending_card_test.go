package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// setupApprovalCard creates a model with an initial approval card and sets dimensions.
func setupApprovalCard(t *testing.T) Model {
	t.Helper()
	m := New(Options{})
	// Set dimensions first
	um, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = um.(Model)

	// Request approval
	um, _ = m.Update(ApprovalRequestedMsg{
		ID:          1,
		Description: "test approval",
		Kind:        "command",
		Subject:     "echo hi",
		Detail:      []string{"test detail"},
		Argv:        []string{"echo", "hi"},
	})
	return um.(Model)
}

// TestPendingCardP1 — A fresh card. y resolves Approved immediately (1 call).
func TestPendingCardP1(t *testing.T) {
	m := setupApprovalCard(t)
	resolved := 0
	var resolvedOutcome ApprovalOutcome
	m.ResolveApproval = func(id int64, outcome ApprovalOutcome) {
		resolved++
		resolvedOutcome = outcome
	}

	// Press y
	um, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = um.(Model)

	if resolved != 1 {
		t.Errorf("resolved %d times, want 1", resolved)
	}
	if resolvedOutcome != Approved {
		t.Errorf("outcome %v, want Approved", resolvedOutcome)
	}
}

// TestPendingCardP2 — A fresh card. Type /run echo hi one rune at a time.
// 0 resolve calls, the card is still pending in the view, and the hint is visible.
func TestPendingCardP2(t *testing.T) {
	m := setupApprovalCard(t)
	resolved := 0
	m.ResolveApproval = func(id int64, outcome ApprovalOutcome) {
		resolved++
	}

	// Type /run echo hi one rune at a time
	for _, r := range "/run echo hi" {
		um, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = um.(Model)
	}

	if resolved != 0 {
		t.Errorf("resolved %d times, want 0", resolved)
	}

	// Verify card is still pending and hint is visible
	view := m.View()
	if !strings.Contains(view, "approval pending") {
		t.Errorf("hint not visible in view")
	}

	// Verify the approval card itself is drawn (title and action row visible)
	if !strings.Contains(view, "test approval") {
		t.Errorf("approval card not drawn in view (title missing)")
	}
	if !strings.Contains(view, "[y] approve") {
		t.Errorf("approval card not drawn in view (action row missing)")
	}
}

// TestPendingCardP3 — After p2, y and n give 0 calls. Tab clears the hint, then y gives 1 call, Approved.
func TestPendingCardP3(t *testing.T) {
	m := setupApprovalCard(t)
	resolved := 0
	var resolvedOutcome ApprovalOutcome
	m.ResolveApproval = func(id int64, outcome ApprovalOutcome) {
		resolved++
		resolvedOutcome = outcome
	}

	// Type /run echo hi one rune at a time to release
	for _, r := range "/run echo hi" {
		um, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = um.(Model)
	}

	// Press y - should do nothing because released
	um, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = um.(Model)
	if resolved != 0 {
		t.Errorf("y on released: resolved %d times, want 0", resolved)
	}

	// Press n - should do nothing because released
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = um.(Model)
	if resolved != 0 {
		t.Errorf("n on released: resolved %d times, want 0", resolved)
	}

	// Press Tab - should re-arm and clear hint
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = um.(Model)
	view := m.View()
	if strings.Contains(view, "approval pending") {
		t.Errorf("hint still visible after Tab")
	}

	// Press y - should now resolve
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = um.(Model)
	if resolved != 1 {
		t.Errorf("after Tab: resolved %d times, want 1", resolved)
	}
	if resolvedOutcome != Approved {
		t.Errorf("outcome %v, want Approved", resolvedOutcome)
	}
}

// TestPendingCardP4 — Released, then Shift+↑ re-arms (the hint disappears), then n gives 1 call, Rejected.
func TestPendingCardP4(t *testing.T) {
	m := setupApprovalCard(t)
	resolved := 0
	var resolvedOutcome ApprovalOutcome
	m.ResolveApproval = func(id int64, outcome ApprovalOutcome) {
		resolved++
		resolvedOutcome = outcome
	}

	// Release with a key
	um, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = um.(Model)

	// Press Shift+↑ to re-arm
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftUp})
	m = um.(Model)
	view := m.View()
	if strings.Contains(view, "approval pending") {
		t.Errorf("hint still visible after Shift+↑")
	}

	// Press n - should resolve
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = um.(Model)
	if resolved != 1 {
		t.Errorf("resolved %d times, want 1", resolved)
	}
	if resolvedOutcome != Rejected {
		t.Errorf("outcome %v, want Rejected", resolvedOutcome)
	}
}

// TestPendingCardP5 — Release card A and cancel it with Esc. Card B arrives and is armed: y gives 1 call for B.
func TestPendingCardP5(t *testing.T) {
	m := setupApprovalCard(t)
	resolved := 0
	var lastID int64
	m.ResolveApproval = func(id int64, outcome ApprovalOutcome) {
		resolved++
		lastID = id
	}

	// Release card A
	um, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = um.(Model)

	// Cancel with Esc - should resolve card A
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = um.(Model)
	if resolved != 1 || lastID != 1 {
		t.Errorf("cancel card A: resolved %d times (ID %d), want 1 time (ID 1)", resolved, lastID)
	}

	// Request card B
	resolved = 0
	um, _ = m.Update(ApprovalRequestedMsg{
		ID:          2,
		Description: "card B",
		Kind:        "command",
		Subject:     "echo B",
		Detail:      []string{},
		Argv:        []string{"echo", "B"},
	})
	m = um.(Model)

	// Press y on card B - should resolve B (and be armed)
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = um.(Model)
	if resolved != 1 || lastID != 2 {
		t.Errorf("approve card B: resolved %d times (ID %d), want 1 time (ID 2)", resolved, lastID)
	}
}

// TestPendingCardP6 — Released, then Esc gives 1 call, Cancelled.
func TestPendingCardP6(t *testing.T) {
	m := setupApprovalCard(t)
	resolved := 0
	var resolvedOutcome ApprovalOutcome
	m.ResolveApproval = func(id int64, outcome ApprovalOutcome) {
		resolved++
		resolvedOutcome = outcome
	}

	// Release with a key
	um, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = um.(Model)

	// Press Esc - should cancel even though released
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = um.(Model)
	if resolved != 1 {
		t.Errorf("resolved %d times, want 1", resolved)
	}
	if resolvedOutcome != Cancelled {
		t.Errorf("outcome %v, want Cancelled", resolvedOutcome)
	}
}

// TestPendingCardP7 — Released, then d opens the detail modal.
func TestPendingCardP7(t *testing.T) {
	m := setupApprovalCard(t)
	resolved := 0
	m.ResolveApproval = func(id int64, outcome ApprovalOutcome) {
		resolved++
	}

	// Release with a key
	um, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = um.(Model)

	// Press d - should open detail modal
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = um.(Model)
	if resolved != 0 {
		t.Errorf("resolved %d times, want 0", resolved)
	}

	// Check that modal is open
	if m.modal == nil {
		t.Errorf("modal not open after 'd'")
	}
}

// TestPendingCardP8 — A single multi-rune KeyRunes{"run"} releases the card. 0 calls.
func TestPendingCardP8(t *testing.T) {
	m := setupApprovalCard(t)
	resolved := 0
	m.ResolveApproval = func(id int64, outcome ApprovalOutcome) {
		resolved++
	}

	// Send multi-rune "run" - should release but not resolve
	um, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("run")})
	m = um.(Model)
	if resolved != 0 {
		t.Errorf("resolved %d times, want 0", resolved)
	}

	view := m.View()
	if !strings.Contains(view, "approval pending") {
		t.Errorf("hint not visible after multi-rune release")
	}
}

// TestPendingCardP9 — Released, then ↓ and PgUp neither re-arm nor resolve. The hint is still shown.
func TestPendingCardP9(t *testing.T) {
	m := setupApprovalCard(t)
	resolved := 0
	m.ResolveApproval = func(id int64, outcome ApprovalOutcome) {
		resolved++
	}

	// Release with a key
	um, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = um.(Model)

	// Press ↓ - should not re-arm, not resolve, hint still shown
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = um.(Model)
	if resolved != 0 {
		t.Errorf("after ↓: resolved %d times, want 0", resolved)
	}

	// Press PgUp - should not re-arm, not resolve, hint still shown
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = um.(Model)
	if resolved != 0 {
		t.Errorf("after PgUp: resolved %d times, want 0", resolved)
	}

	view := m.View()
	if !strings.Contains(view, "approval pending") {
		t.Errorf("hint not visible after navigation keys")
	}
}

// TestPendingCardP10 — An armed card. Tab does not resolve and does not release:
// the hint is absent, and a following y gives 1 call, Approved.
func TestPendingCardP10(t *testing.T) {
	m := setupApprovalCard(t)
	resolved := 0
	var resolvedOutcome ApprovalOutcome
	m.ResolveApproval = func(id int64, outcome ApprovalOutcome) {
		resolved++
		resolvedOutcome = outcome
	}

	// Press Tab on armed card - should do nothing
	um, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = um.(Model)
	if resolved != 0 {
		t.Errorf("resolved %d times, want 0", resolved)
	}

	view := m.View()
	if strings.Contains(view, approvalReleaseHint) {
		t.Errorf("hint should not appear on armed card")
	}

	// The card is still armed: y resolves it.
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = um.(Model)
	if resolved != 1 {
		t.Errorf("after Tab on armed card: resolved %d times, want 1", resolved)
	}
	if resolvedOutcome != Approved {
		t.Errorf("outcome %v, want Approved", resolvedOutcome)
	}
}

// TestPendingCardP11 — Release with x, press Esc → 1 call, Cancelled, and the hint text
// is absent from m.View(). Repeat the check after a released card is approved via Tab then y.
func TestPendingCardP11(t *testing.T) {
	m := setupApprovalCard(t)
	resolved := 0
	var resolvedOutcome ApprovalOutcome
	m.ResolveApproval = func(id int64, outcome ApprovalOutcome) {
		resolved++
		resolvedOutcome = outcome
	}

	// Release with x
	um, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = um.(Model)

	// Press Esc - should cancel
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = um.(Model)
	if resolved != 1 {
		t.Errorf("resolved %d times, want 1", resolved)
	}
	if resolvedOutcome != Cancelled {
		t.Errorf("outcome %v, want Cancelled", resolvedOutcome)
	}

	// Check hint is absent from view
	view := m.View()
	if strings.Contains(view, "approval pending") {
		t.Errorf("hint visible after Esc on released card")
	}

	// Now request another card and test the repeat. Reset both recorders so the
	// final Approved assertion cannot be satisfied by the Cancelled half above.
	resolved = 0
	resolvedOutcome = Unresolved
	um, _ = m.Update(ApprovalRequestedMsg{
		ID:          2,
		Description: "second card",
		Kind:        "command",
		Subject:     "echo 2",
		Detail:      []string{},
		Argv:        []string{"echo", "2"},
	})
	m = um.(Model)

	// Release it
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = um.(Model)

	// Press Tab to re-arm; the hint must go with it.
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = um.(Model)
	if strings.Contains(m.View(), approvalReleaseHint) {
		t.Errorf("hint visible after Tab re-armed the released card")
	}

	// Press y - should resolve as Approved
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = um.(Model)
	if resolved != 1 {
		t.Errorf("after re-arm: resolved %d times, want 1", resolved)
	}
	if resolvedOutcome != Approved {
		t.Errorf("outcome %v, want Approved", resolvedOutcome)
	}
}

// TestPendingCardP12 — a on a command card that offers a grant.
// Armed: a → 1 call, ApprovedSession. Released: a → 0 calls. After Tab, a → 1 call.
func TestPendingCardP12(t *testing.T) {
	m := New(Options{})
	um, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = um.(Model)

	// Request a grant card (CanApproveForSession=true, GrantScope set, Argv set)
	um, _ = m.Update(ApprovalRequestedMsg{
		ID:                   1,
		Description:          "test grant",
		Kind:                 "command",
		Subject:              "grant me",
		Detail:               []string{},
		CanApproveForSession: true,
		GrantScope:           "/read",
		Argv:                 []string{"read_file", "/tmp/test"},
	})
	m = um.(Model)

	resolved := 0
	var resolvedOutcome ApprovalOutcome
	m.ResolveApproval = func(id int64, outcome ApprovalOutcome) {
		resolved++
		resolvedOutcome = outcome
	}

	// Armed: press a - should resolve as ApprovedSession
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = um.(Model)
	if resolved != 1 {
		t.Errorf("armed: resolved %d times, want 1", resolved)
	}
	if resolvedOutcome != ApprovedSession {
		t.Errorf("outcome %v, want ApprovedSession", resolvedOutcome)
	}

	// Request another grant card
	resolved = 0
	um, _ = m.Update(ApprovalRequestedMsg{
		ID:                   2,
		Description:          "test grant 2",
		Kind:                 "command",
		Subject:              "grant me 2",
		Detail:               []string{},
		CanApproveForSession: true,
		GrantScope:           "/read",
		Argv:                 []string{"read_file", "/tmp/test"},
	})
	m = um.(Model)

	// Release with x
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = um.(Model)

	// Released: press a - should do nothing (0 calls)
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = um.(Model)
	if resolved != 0 {
		t.Errorf("released: resolved %d times, want 0", resolved)
	}

	// Press Tab to re-arm
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = um.(Model)

	// Armed again: press a - should resolve as ApprovedSession (1 call)
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = um.(Model)
	if resolved != 1 {
		t.Errorf("after Tab: resolved %d times, want 1", resolved)
	}
	if resolvedOutcome != ApprovedSession {
		t.Errorf("outcome %v, want ApprovedSession", resolvedOutcome)
	}
}

// TestPendingCardP13 — Overlapping cards. Card A is pending and released (press x).
// Card B arrives while A is still pending. y → 1 call for B's ID, and the hint is absent.
// This guards the flag reset (clearApprovalRelease) in receiveApprovalRequest: without it
// B arrives released and y gives 0 calls. The unconditional m.comp.Hint = "" beside it
// is guarded by TestPendingCardT5ForeignHint, not by this test.
func TestPendingCardP13(t *testing.T) {
	m := setupApprovalCard(t)
	resolved := 0
	var lastID int64
	m.ResolveApproval = func(id int64, outcome ApprovalOutcome) {
		resolved++
		lastID = id
	}

	// Release card A
	um, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = um.(Model)

	// Card B arrives without resolving A
	um, _ = m.Update(ApprovalRequestedMsg{
		ID:          2,
		Description: "card B",
		Kind:        "command",
		Subject:     "echo B",
		Detail:      []string{},
		Argv:        []string{"echo", "B"},
	})
	m = um.(Model)

	// Reset counter
	resolved = 0
	lastID = 0

	// Check hint is absent (should be cleared on new card arrival)
	view := m.View()
	if strings.Contains(view, "approval pending") {
		t.Errorf("hint visible after new card arrival")
	}

	// Press y - should resolve B only (B's ID)
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = um.(Model)
	if resolved != 1 {
		t.Errorf("resolved %d times, want 1", resolved)
	}
	if lastID != 2 {
		t.Errorf("resolved ID %d, want 2", lastID)
	}
}

// TestPendingCardP14 — Armed card. Navigation keys do not release; after pressing each one,
// the hint is absent and y gives 1 call, Approved. Table-driven over all 8 navigation keys.
func TestPendingCardP14(t *testing.T) {
	navigationKeys := []struct {
		name string
		key  tea.KeyMsg
	}{
		{"Up", tea.KeyMsg{Type: tea.KeyUp}},
		{"Down", tea.KeyMsg{Type: tea.KeyDown}},
		{"Left", tea.KeyMsg{Type: tea.KeyLeft}},
		{"Right", tea.KeyMsg{Type: tea.KeyRight}},
		{"PgUp", tea.KeyMsg{Type: tea.KeyPgUp}},
		{"PgDown", tea.KeyMsg{Type: tea.KeyPgDown}},
		{"Home", tea.KeyMsg{Type: tea.KeyHome}},
		{"End", tea.KeyMsg{Type: tea.KeyEnd}},
	}

	for _, tt := range navigationKeys {
		t.Run(tt.name, func(t *testing.T) {
			m := setupApprovalCard(t)
			resolved := 0
			var resolvedOutcome ApprovalOutcome
			m.ResolveApproval = func(id int64, outcome ApprovalOutcome) {
				resolved++
				resolvedOutcome = outcome
			}

			// Press the navigation key
			um, _ := m.Update(tt.key)
			m = um.(Model)

			// Check hint is absent
			view := m.View()
			if strings.Contains(view, "approval pending") {
				t.Errorf("hint visible after navigation key")
			}

			// Press y - should resolve as Approved (1 call)
			um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
			m = um.(Model)
			if resolved != 1 {
				t.Errorf("resolved %d times, want 1", resolved)
			}
			if resolvedOutcome != Approved {
				t.Errorf("outcome %v, want Approved", resolvedOutcome)
			}
		})
	}
}

// TestPendingCardP15 — Table test for release keys.
// One fresh card per key: Space, Enter, Backspace, Ctrl+J.
// Each releases (the hint is shown and a following y gives 0 calls).
func TestPendingCardP15(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyMsg
	}{
		{"Space", tea.KeyMsg{Type: tea.KeySpace}},
		{"Enter", tea.KeyMsg{Type: tea.KeyEnter}},
		{"Backspace", tea.KeyMsg{Type: tea.KeyBackspace}},
		{"Ctrl+J", tea.KeyMsg{Type: tea.KeyCtrlJ}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := setupApprovalCard(t)
			resolved := 0
			m.ResolveApproval = func(id int64, outcome ApprovalOutcome) {
				resolved++
			}

			// Press the key - should release
			um, _ := m.Update(tt.key)
			m = um.(Model)

			// Check hint is shown
			view := m.View()
			if !strings.Contains(view, "approval pending") {
				t.Errorf("%s: hint not visible after key", tt.name)
			}

			// Press y - should do nothing (0 calls, card is released)
			um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
			m = um.(Model)
			if resolved != 0 {
				t.Errorf("%s: resolved %d times, want 0", tt.name, resolved)
			}
		})
	}
}

// TestPendingCardT5ForeignHint — A hint produced through the real path (typing an
// unknown slash command and pressing Enter) is not the approval hint, so
// clearApprovalRelease leaves it alone. receiveApprovalRequest's unconditional
// m.comp.Hint = "" is what removes it when a card arrives.
func TestPendingCardT5ForeignHint(t *testing.T) {
	m := New(Options{})
	um, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = um.(Model)

	for _, r := range "/zz" {
		um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = um.(Model)
	}
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = um.(Model)
	if !strings.Contains(m.View(), "unknown command /zz") {
		t.Fatalf("foreign hint not shown before the approval request:\n%s", m.View())
	}

	um, _ = m.Update(ApprovalRequestedMsg{
		ID:          1,
		Description: "test",
		Kind:        "command",
		Subject:     "echo hi",
		Detail:      []string{},
		Argv:        []string{"echo", "hi"},
	})
	m = um.(Model)

	view := m.View()
	if strings.Contains(view, "unknown command") {
		t.Errorf("foreign hint still visible after approval request")
	}
	if !strings.Contains(view, "[y] approve") {
		t.Errorf("approval card not drawn after the request")
	}
}

// TestPendingCardT6Paste — Bracketed paste on armed and released approval cards.
// Armed: paste neither releases nor resolves, then y → 1 call.
// Released: paste is dropped; nothing observable changes and y still gives 0 calls.
func TestPendingCardT6Paste(t *testing.T) {
	// Test 1: Paste on armed card
	t.Run("armed", func(t *testing.T) {
		m := setupApprovalCard(t)
		resolved := 0
		m.ResolveApproval = func(id int64, outcome ApprovalOutcome) {
			resolved++
		}

		// Send a bracketed paste on the armed card
		um, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("run"), Paste: true})
		m = um.(Model)

		// Paste should not have resolved or released
		if resolved != 0 {
			t.Errorf("resolved %d times, want 0 (paste should not resolve)", resolved)
		}
		view := m.View()
		if strings.Contains(view, "approval pending") {
			t.Errorf("hint should not appear (paste should not release)")
		}

		// Press y - should now resolve (1 call)
		um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
		m = um.(Model)
		if resolved != 1 {
			t.Errorf("resolved %d times, want 1", resolved)
		}
	})

	// Test 2: Paste on released card. A paste is dropped by handlePaste outside
	// Composing. Pasting "d" makes the drop observable: if the paste reached
	// keyApproval it would open the detail modal, which a released card still
	// honours. Neither the composer value nor the hint may change.
	t.Run("released", func(t *testing.T) {
		m := setupApprovalCard(t)
		resolved := 0
		m.ResolveApproval = func(id int64, outcome ApprovalOutcome) {
			resolved++
		}

		// Release the card with a key
		um, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
		m = um.(Model)
		if m.comp.Hint != approvalReleaseHint {
			t.Fatalf("card not released: hint %q", m.comp.Hint)
		}
		valueBefore := m.comp.Value()

		um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d"), Paste: true})
		m = um.(Model)

		if m.modal != nil {
			t.Errorf("paste reached keyApproval: detail modal opened")
		}
		if got := m.comp.Value(); got != valueBefore {
			t.Errorf("composer value changed by paste: %q -> %q", valueBefore, got)
		}
		if m.comp.Hint != approvalReleaseHint {
			t.Errorf("hint changed by paste: %q", m.comp.Hint)
		}
		if resolved != 0 {
			t.Errorf("resolved %d times, want 0", resolved)
		}

		// Still released: y does nothing.
		um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
		m = um.(Model)
		if resolved != 0 {
			t.Errorf("resolved %d times, want 0 (y on released card should do nothing)", resolved)
		}
	})
}
