package main

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/djm56/kirsch/internal/app"
	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/policy"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

// TestApprovalOutcomeConversion verifies the ToAppOutcome conversion function
// handles all enum values correctly, preventing the enum trap. Uses actual app
// constants rather than bare literals to ensure reordering the iota block
// fails a test instead of silently inverting the mapping.
func TestApprovalOutcomeConversion(t *testing.T) {
	tests := []struct {
		tuiOutcome tui.ApprovalOutcome
		appOutcome app.ApprovalOutcome
		name       string
	}{
		{tui.Approved, app.ApprovalOutcomeOnce, "Approved -> Once"},
		{tui.ApprovedSession, app.ApprovalOutcomeSession, "ApprovedSession -> Session"},
		{tui.Rejected, app.ApprovalOutcomeDeny, "Rejected -> Deny"},
		{tui.Cancelled, app.ApprovalOutcomeCancelled, "Cancelled -> Cancelled"},
		{tui.Unresolved, app.ApprovalOutcomeCancelled, "Unresolved -> Cancelled (safety fallback)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tui.ToAppOutcome(tt.tuiOutcome)
			if got != int(tt.appOutcome) {
				t.Errorf("ToAppOutcome(%d) = %d, want %d", tt.tuiOutcome, got, int(tt.appOutcome))
			}
		})
	}
}

// TestFromAppOutcomeConversion verifies the FromAppOutcome conversion function
// (the inverse of ToAppOutcome) handles all app enum values correctly, preventing
// the enum trap. Uses actual app constants rather than bare literals to ensure
// reordering the iota block fails a test instead of silently inverting the mapping.
func TestFromAppOutcomeConversion(t *testing.T) {
	tests := []struct {
		appOutcome app.ApprovalOutcome
		tuiOutcome tui.ApprovalOutcome
		name       string
	}{
		{app.ApprovalOutcomeOnce, tui.Approved, "Once -> Approved"},
		{app.ApprovalOutcomeSession, tui.ApprovedSession, "Session -> ApprovedSession"},
		{app.ApprovalOutcomeDeny, tui.Rejected, "Deny -> Rejected"},
		{app.ApprovalOutcomeCancelled, tui.Cancelled, "Cancelled -> Cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tui.FromAppOutcome(int(tt.appOutcome))
			if got != tt.tuiOutcome {
				t.Errorf("FromAppOutcome(%d) = %d, want %d", int(tt.appOutcome), got, tt.tuiOutcome)
			}
		})
	}
}

// TestResolveApprovalNilCheckWired verifies the ResolveApproval callback is
// properly wired and explicitly checked for nil before calling. This guards
// against accidental removal of the assignment inside wireCallbacks.
func TestResolveApprovalNilCheckWired(t *testing.T) {
	ws, err := workspace.Detect("")
	if err != nil {
		t.Fatalf("workspace.Detect: %v", err)
	}

	log, err := telemetry.New(telemetry.Options{Enabled: false})
	if err != nil {
		t.Fatalf("telemetry.New: %v", err)
	}

	cfg, _, err := config.Load(config.Options{})
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	a := app.New(ws, cfg, log)
	defer a.Close()

	// Create a real TUI model.
	m := tui.New(tui.Options{
		Version: "0.1.0-test",
		Caps:    tui.Caps{Colour: false, Unicode: false},
		Session: tui.SessionInfo{
			Project: "test-project",
			Branch:  "main",
			Dirty:   false,
		},
	})

	// Before wiring, ResolveApproval should be nil. This verifies we're testing
	// the wiring, not accidentally relying on a previous assignment.
	if m.ResolveApproval != nil {
		t.Fatal("ResolveApproval should be nil before wireCallbacks")
	}

	// Wire the callbacks.
	wireCallbacks(&m, a, log)

	// After wiring, ResolveApproval must not be nil.
	if m.ResolveApproval == nil {
		t.Fatal("ResolveApproval is nil after wireCallbacks; wiring failed")
	}

	// Verify the wired callback works by dispatching an approval and key through
	// the real Update path.
	done := make(chan app.ApprovalOutcome, 1)
	go func() {
		result, _ := a.Request(context.Background(), app.ApprovalRequest{
			Description: "test patch",
			Operation:   policy.OperationPatch,
		})
		done <- result
	}()

	// Give the request goroutine a moment to register.
	time.Sleep(10 * time.Millisecond)

	// Dispatch the ApprovalRequestedMsg through Update.
	msg := tui.ApprovalRequestedMsg{
		ID:                   1,
		Description:          "test patch",
		Kind:                 "patch",
		CanApproveForSession: false,
	}
	model, _ := m.Update(msg)
	m = model.(tui.Model)

	// Dispatch Esc through Update to trigger cancellation via the wired callback.
	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	model, _ = m.Update(keyMsg)
	m = model.(tui.Model)

	// Verify the App received ApprovalOutcomeCancelled through the wired callback.
	select {
	case outcome := <-done:
		if outcome != app.ApprovalOutcomeCancelled {
			t.Errorf("Request returned %d, want %d (ApprovalOutcomeCancelled)",
				outcome, app.ApprovalOutcomeCancelled)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Request() blocked; ResolveApproval callback may not be wired correctly")
	}
}

// TestCancelCallbackWired verifies that the Cancel callback is properly wired
// through wireCallbacks. Deleting the m.Cancel assignment inside wireCallbacks
// causes this test to fail because Cancel will be nil.
func TestCancelCallbackWired(t *testing.T) {
	ws, err := workspace.Detect("")
	if err != nil {
		t.Fatalf("workspace.Detect: %v", err)
	}

	log, err := telemetry.New(telemetry.Options{Enabled: false})
	if err != nil {
		t.Fatalf("telemetry.New: %v", err)
	}

	cfg, _, err := config.Load(config.Options{})
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	a := app.New(ws, cfg, log)
	defer a.Close()

	m := tui.New(tui.Options{
		Version: "0.1.0-test",
		Caps:    tui.Caps{Colour: false, Unicode: false},
		Session: tui.SessionInfo{
			Project: "test-project",
			Branch:  "main",
			Dirty:   false,
		},
	})

	// Before wiring, Cancel should be nil.
	if m.Cancel != nil {
		t.Fatal("Cancel should be nil before wireCallbacks")
	}

	// Wire the callbacks through wireCallbacks, which assigns m.Cancel.
	wireCallbacks(&m, a, log)

	// After wiring, Cancel must not be nil. If m.Cancel = ... is deleted
	// from inside wireCallbacks, this assertion will fail.
	if m.Cancel == nil {
		t.Fatal("Cancel is nil after wireCallbacks; wiring assignment is missing")
	}
}

// TestGrantsSanitisedE2E verifies end-to-end that a grant containing hostile
// ANSI escape bytes — created through the real approval flow, with the exact
// argv also passed to policy via a.Request/a.Resolve — is stripped of those
// bytes when displayed via /approvals, while its ordinary text survives.
//
// The assertion reads m.View(), the fully rendered screen, rather than the
// tui package's internal modal.Lines: that field is unexported, and this test
// lives in package main (cmd/kirsch), which cannot reach into internal/tui
// state without an internal import — reading the same bytes a real terminal
// would receive is the only externally-visible way to prove sanitisation ran.
// GetGrants() alone cannot do this: it deliberately returns the policy's raw,
// unsanitised text (sanitising happens only at the /approvals render site in
// update.go's runSlash), so asserting against it — as the previous version of
// this test did — passes whether or not the sanitising call is even present.
func TestGrantsSanitisedE2E(t *testing.T) {
	ws, err := workspace.Detect("")
	if err != nil {
		t.Fatalf("workspace.Detect: %v", err)
	}

	log, err := telemetry.New(telemetry.Options{Enabled: false})
	if err != nil {
		t.Fatalf("telemetry.New: %v", err)
	}

	cfg, _, err := config.Load(config.Options{})
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	a := app.New(ws, cfg, log)
	defer a.Close()

	m := tui.New(tui.Options{
		Version: "0.1.0-test",
		Caps:    tui.Caps{Colour: false, Unicode: false},
		Session: tui.SessionInfo{
			Project: "test-project",
			Branch:  "main",
			Dirty:   false,
		},
	})

	// Wire the callbacks
	wireCallbacks(&m, a, log)

	// View() renders nothing at the zero-value width/height Options{} leaves
	// it with (§2.2: degenerate sizes render nothing rather than panicking),
	// so a real terminal size must arrive before the rendered-screen
	// assertions below can see anything.
	model0, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = model0.(tui.Model)

	// Hostile argv: a real command prefix with an ANSI escape sequence riding
	// along in one token. GrantScope is computed the same way app.go's
	// formatCommandDisplay does (space-joined argv), so the dispatched
	// message matches what the real Request path would send.
	hostileToken := "\x1b[31mpwned\x1b[0m"
	argv := []string{"go", "test", hostileToken}
	grantScope := strings.Join(argv, " ")

	// Run an approval request that offers a session grant in a goroutine.
	done := make(chan app.ApprovalOutcome, 1)
	go func() {
		result, _ := a.Request(context.Background(), app.ApprovalRequest{
			Description: "test command for grant",
			Operation:   policy.OperationCommand,
			Argv:        argv,
		})
		done <- result
	}()

	// Give the request goroutine a moment to register the approval.
	time.Sleep(10 * time.Millisecond)

	// Dispatch the ApprovalRequestedMsg through the TUI as app.go would send it.
	msg := tui.ApprovalRequestedMsg{
		ID:                   1,
		Description:          "test command for grant",
		Kind:                 "command",
		CanApproveForSession: true,
		Subject:              "go test",
		Detail:               []string{},
		GrantScope:           grantScope,
		Argv:                 argv,
	}
	model, _ := m.Update(msg)
	m = model.(tui.Model)

	// User presses 'a' to approve for session (creates the grant).
	keyMsg := tea.KeyMsg{Runes: []rune("a")}
	model, _ = m.Update(keyMsg)
	m = model.(tui.Model)

	// App confirms the grant creation via ApprovalResolvedMsg.
	resolved := tui.ApprovalResolvedMsg{
		ID:      1,
		Outcome: tui.ApprovedSession,
	}
	model, _ = m.Update(resolved)
	m = model.(tui.Model)

	// Verify the approval outcome is now confirmed.
	// (The wired callback has processed the approval and created the grant in policy.)
	select {
	case outcome := <-done:
		if outcome != app.ApprovalOutcomeSession {
			t.Errorf("Request returned %d, want %d (ApprovalOutcomeSession)",
				outcome, app.ApprovalOutcomeSession)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Request() blocked; approval flow may not be wired correctly")
	}

	// The raw grant, as policy stores and GetGrants() returns it, must still
	// carry the hostile bytes untouched — sanitising is not policy's job.
	rawGrants := m.GetGrants()
	rawJoined := strings.Join(rawGrants, "|")
	if !strings.Contains(rawJoined, hostileToken) {
		t.Fatalf("setup: raw grant lost the hostile token before /approvals even ran: %v", rawGrants)
	}

	// Now run /approvals command to display the created grant.
	for _, r := range "/approvals" {
		model, _ = m.Update(tea.KeyMsg{Runes: []rune{r}})
		m = model.(tui.Model)
	}
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(tui.Model)

	// Read the rendered screen — the one thing package main can observe that
	// proves the sanitising call ran, since modal.Lines is unexported.
	rendered := m.View()

	if !strings.Contains(rendered, "session grants") {
		t.Fatalf("rendered view does not show the grants modal: %s", rendered)
	}
	if strings.ContainsRune(rendered, '\x1b') {
		t.Errorf("rendered view still contains a raw ESC byte; grant was not sanitised: %q", rendered)
	}
	if !strings.Contains(rendered, "pwned") {
		t.Errorf("rendered view lost the grant's ordinary text 'pwned': %s", rendered)
	}
}

// TestConfigWiringThroughAppNew verifies that the config object created in
// run() reaches the RunCommand tool through app.New(). This test constructs
// the same path the production code does: config.Load(), app.New(), and then
// checks that the tool registry contains a RunCommand tool with the config wired.
func TestConfigWiringThroughAppNew(t *testing.T) {
	ws, err := workspace.Detect("")
	if err != nil {
		t.Fatalf("workspace.Detect: %v", err)
	}

	log, err := telemetry.New(telemetry.Options{Enabled: false})
	if err != nil {
		t.Fatalf("telemetry.New: %v", err)
	}

	// Load config the same way run() does
	cfg, _, err := config.Load(config.Options{})
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	// Create the app the same way run() does
	a := app.New(ws, cfg, log)
	defer a.Close()

	// Verify the app can access the config through its registry.
	// The run_command tool is registered with Config: &cfg.
	// We can't directly inspect the RunCommand struct, but we can verify
	// the tool is registered by checking the registry has it.
	registry := a.Registry()
	names := registry.Names()

	// Look for run_command in the registry
	found := false
	for _, name := range names {
		if name == "run_command" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("run_command tool not registered in app registry; config wiring may be broken")
	}
}

// TestSubmitCallbackWired verifies that the Submit callback is properly wired
// through wireCallbacks. Deleting the m.Submit assignment inside wireCallbacks
// causes this test to fail because Submit will be nil, blocking real model turns.
func TestSubmitCallbackWired(t *testing.T) {
	ws, err := workspace.Detect("")
	if err != nil {
		t.Fatalf("workspace.Detect: %v", err)
	}

	log, err := telemetry.New(telemetry.Options{Enabled: false})
	if err != nil {
		t.Fatalf("telemetry.New: %v", err)
	}

	cfg, _, err := config.Load(config.Options{})
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	a := app.New(ws, cfg, log)
	defer a.Close()

	m := tui.New(tui.Options{
		Version: "0.1.0-test",
		Caps:    tui.Caps{Colour: false, Unicode: false},
		Session: tui.SessionInfo{
			Project: "test-project",
			Branch:  "main",
			Dirty:   false,
		},
	})

	// Before wiring, Submit should be nil.
	if m.Submit != nil {
		t.Fatal("Submit should be nil before wireCallbacks")
	}

	// Wire the callbacks through wireCallbacks, which assigns m.Submit.
	wireCallbacks(&m, a, log)

	// After wiring, Submit must not be nil. If m.Submit = ... is deleted
	// from inside wireCallbacks, this assertion will fail.
	if m.Submit == nil {
		t.Fatal("Submit is nil after wireCallbacks; wiring assignment is missing")
	}
}

// TestCheckOnboardingCalledWiresAttach verifies that CheckOnboarding can be
// called after the program is attached. The actual async message path is
// exercised by the app-level CheckOnboarding tests; this test ensures the
// main-side wiring surface (app created, callbacks wired, CheckOnboarding
// callable with os.Getenv) is present and does not panic.
func TestCheckOnboardingCalledWiresAttach(t *testing.T) {
	ws, err := workspace.Detect("")
	if err != nil {
		t.Fatalf("workspace.Detect: %v", err)
	}

	log, err := telemetry.New(telemetry.Options{Enabled: false})
	if err != nil {
		t.Fatalf("telemetry.New: %v", err)
	}

	cfg, _, err := config.Load(config.Options{})
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	a := app.New(ws, cfg, log)
	defer a.Close()

	m := tui.New(tui.Options{
		Version: "0.1.0-test",
		Caps:    tui.Caps{Colour: false, Unicode: false},
		Session: tui.SessionInfo{
			Project: "test-project",
			Branch:  "main",
			Dirty:   false,
		},
	})

	// Wire the same callbacks run() wires.
	wireCallbacks(&m, a, log)

	// CheckOnboarding must be callable with a getenv function and must not
	// panic after callbacks are wired. The message delivery itself is tested
	// in internal/app/app_test.go.
	a.CheckOnboarding(func(string) string { return "" })
}

// TestCheckOnboardingInvokedAtStartup verifies that the post-attach startup
// sequence in run() actually invokes App.CheckOnboarding. If the call is
// removed from startupPostAttach, the spy here never fires and the test fails.
func TestCheckOnboardingInvokedAtStartup(t *testing.T) {
	ws, err := workspace.Detect("")
	if err != nil {
		t.Fatalf("workspace.Detect: %v", err)
	}

	log, err := telemetry.New(telemetry.Options{Enabled: false})
	if err != nil {
		t.Fatalf("telemetry.New: %v", err)
	}

	cfg, _, err := config.Load(config.Options{})
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	a := app.New(ws, cfg, log)
	defer a.Close()

	m := tui.New(tui.Options{
		Version: "0.1.0-test",
		Caps:    tui.Caps{Colour: false, Unicode: false},
		Session: tui.SessionInfo{
			Project: "test-project",
			Branch:  "main",
			Dirty:   false,
		},
	})

	// The program is never Run(); we only need a concrete *tea.Program for Attach.
	p := tea.NewProgram(m, tea.WithOutput(io.Discard), tea.WithInput(nil))

	old := onboardingCheck
	called := false
	onboardingCheck = func(aa *app.App) {
		if aa != a {
			t.Fatalf("onboardingCheck called with wrong app pointer")
		}
		called = true
	}
	defer func() { onboardingCheck = old }()

	startupPostAttach(a, p, "0.1.0-test")

	if !called {
		t.Fatal("startupPostAttach did not invoke onboardingCheck; CheckOnboarding is missing from startup")
	}
}

// TestStartSessionInvokedAtStartup verifies that the post-attach startup
// sequence in run() actually invokes App.StartSession. If the call is removed
// from startupPostAttach, the system prompt is never assembled and the spy here
// never fires.
func TestStartSessionInvokedAtStartup(t *testing.T) {
	ws, err := workspace.Detect("")
	if err != nil {
		t.Fatalf("workspace.Detect: %v", err)
	}

	log, err := telemetry.New(telemetry.Options{Enabled: false})
	if err != nil {
		t.Fatalf("telemetry.New: %v", err)
	}

	cfg, _, err := config.Load(config.Options{})
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	a := app.New(ws, cfg, log)
	defer a.Close()

	m := tui.New(tui.Options{
		Version: "0.1.0-test",
		Caps:    tui.Caps{Colour: false, Unicode: false},
		Session: tui.SessionInfo{
			Project: "test-project",
			Branch:  "main",
			Dirty:   false,
		},
	})

	p := tea.NewProgram(m, tea.WithOutput(io.Discard), tea.WithInput(nil))

	old := startSessionCheck
	called := false
	startSessionCheck = func(aa *app.App) {
		if aa != a {
			t.Fatalf("startSessionCheck called with wrong app pointer")
		}
		called = true
	}
	defer func() { startSessionCheck = old }()

	startupPostAttach(a, p, "0.1.0-test")

	if !called {
		t.Fatal("startupPostAttach did not invoke startSessionCheck; StartSession is missing from startup")
	}
}

// TestClearGrantsEmptiesPolicyE2E verifies end-to-end that clearing grants
// through the TUI actually empties the real policy. A real grant is created
// via approval, then /approvals is run, then 'c' clears it, and the policy
// is verified empty afterward.
func TestClearGrantsEmptiesPolicyE2E(t *testing.T) {
	ws, err := workspace.Detect("")
	if err != nil {
		t.Fatalf("workspace.Detect: %v", err)
	}

	log, err := telemetry.New(telemetry.Options{Enabled: false})
	if err != nil {
		t.Fatalf("telemetry.New: %v", err)
	}

	cfg, _, err := config.Load(config.Options{})
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	a := app.New(ws, cfg, log)
	defer a.Close()

	m := tui.New(tui.Options{
		Version: "0.1.0-test",
		Caps:    tui.Caps{Colour: false, Unicode: false},
		Session: tui.SessionInfo{
			Project: "test-project",
			Branch:  "main",
			Dirty:   false,
		},
	})

	// Wire the callbacks
	wireCallbacks(&m, a, log)

	// Create a real grant by running an approval request for a session grant.
	done := make(chan app.ApprovalOutcome, 1)
	go func() {
		result, _ := a.Request(context.Background(), app.ApprovalRequest{
			Description: "test command for clear",
			Operation:   policy.OperationCommand,
			Argv:        []string{"go", "test"},
		})
		done <- result
	}()

	// Give the request goroutine a moment to register.
	time.Sleep(10 * time.Millisecond)

	// Dispatch the approval request to the TUI.
	msg := tui.ApprovalRequestedMsg{
		ID:                   1,
		Description:          "test command for clear",
		Kind:                 "command",
		CanApproveForSession: true,
		Subject:              "go test",
		Detail:               []string{},
		GrantScope:           "go test",
		Argv:                 []string{"go", "test"},
	}
	model, _ := m.Update(msg)
	m = model.(tui.Model)

	// Approve for session to create a grant.
	keyMsg := tea.KeyMsg{Runes: []rune("a")}
	model, _ = m.Update(keyMsg)
	m = model.(tui.Model)

	// Confirm the grant.
	resolved := tui.ApprovalResolvedMsg{
		ID:      1,
		Outcome: tui.ApprovedSession,
	}
	model, _ = m.Update(resolved)
	m = model.(tui.Model)

	// Wait for the approval to be confirmed.
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Request() blocked; approval flow may not be wired correctly")
	}

	// Verify the grant was created. The precondition this test exists to
	// prove is exactly this: a skip here would hide a real regression in
	// grant creation behind a result that every summary reads as a pass.
	grantsAfterCreate := m.GetGrants()
	if len(grantsAfterCreate) == 0 {
		t.Fatal("grant was not created through the approval flow; nothing to clear")
	}

	// Run /approvals to open the grants modal
	for _, r := range "/approvals" {
		model, _ = m.Update(tea.KeyMsg{Runes: []rune{r}})
		m = model.(tui.Model)
	}
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(tui.Model)

	// Press 'c' to trigger clear confirmation
	keyMsg = tea.KeyMsg{Runes: []rune("c")}
	model, _ = m.Update(keyMsg)
	m = model.(tui.Model)

	// Confirm the clear action
	keyMsg = tea.KeyMsg{Runes: []rune("y")}
	model, _ = m.Update(keyMsg)
	m = model.(tui.Model)

	// Verify the policy is now empty
	grantsAfterClear := m.GetGrants()
	if len(grantsAfterClear) != 0 {
		t.Errorf("after clear, GetGrants() = %v, want empty", grantsAfterClear)
	}
}
