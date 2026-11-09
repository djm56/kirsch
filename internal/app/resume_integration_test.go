package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/agent"
	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/policy"
	"github.com/djm56/kirsch/internal/provider"
	"github.com/djm56/kirsch/internal/session"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

// TestResumeCompletesPartialTurn is the m4-d5 integration test: a session with
// a partial turn (user message, assistant tool call, tool result, and a granted
// scope) is resumed, the TUI receives the reconstructed transcript, grants are
// restored, and the next Submit continues the conversation with a valid message
// array for the provider.
func TestResumeCompletesPartialTurn(t *testing.T) {
	tmp := t.TempDir()
	stateRoot := filepath.Join(tmp, "state")

	cfg := config.Defaults()
	cfg.Session.StorageDir = filepath.Join(stateRoot, "sessions")

	wsRoot, err := filepath.Abs(filepath.Join("..", "..", "testdata", "repo-small"))
	if err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Detect(wsRoot)
	if err != nil {
		t.Fatal(err)
	}

	// Create a partial session that ends mid-task: the assistant asked for a
	// tool and the user/tool layer answered it, then the process quit before the
	// assistant could continue.
	layout, err := session.NewLayout(stateRoot, ws.CanonicalRoot)
	if err != nil {
		t.Fatalf("NewLayout: %v", err)
	}
	path, _, err := layout.CreateSession(time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	s, err := session.New(path, session.WithCadence(time.Hour))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	s.Record(agent.RecordAppend, userTextMessage("plan the refactor"))
	s.Record(agent.RecordAppend, assistantTextToolMessage("I'll read main.go", "call-1", "read_file", `{"path":"main.go"}`))
	s.Record(agent.RecordAppend, userToolResultMessage("call-1", "package main\n", false))
	s.RecordScopeGranted("run_command", "go test ./...")
	if err := s.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Resume the session through the app.
	// Three messages: the notice, the transcript and the session totals.
	c := newCollector(3)
	a := New(ws, cfg, telemetry.Disabled())
	a.sendFn = c.send
	defer a.Close()

	if err := a.ResumeLastSession(); err != nil {
		t.Fatalf("ResumeLastSession: %v", err)
	}

	msgs := c.wait(t, 2*time.Second)
	var loaded tui.TranscriptLoadedMsg
	for _, m := range msgs {
		if lm, ok := m.(tui.TranscriptLoadedMsg); ok {
			loaded = lm
		}
	}
	if loaded.Items == nil {
		t.Fatalf("TranscriptLoadedMsg not received: got %+v", msgs)
	}
	if len(loaded.Items) != 3 {
		t.Fatalf("transcript items = %d, want 3", len(loaded.Items))
	}
	if loaded.Items[0].Kind != tui.KindUser {
		t.Errorf("item[0].Kind = %v, want KindUser", loaded.Items[0].Kind)
	}
	if loaded.Items[1].Kind != tui.KindAssistant {
		t.Errorf("item[1].Kind = %v, want KindAssistant", loaded.Items[1].Kind)
	}
	if loaded.Items[2].Kind != tui.KindTool {
		t.Errorf("item[2].Kind = %v, want KindTool", loaded.Items[2].Kind)
	}

	// The session-scoped grant must be restored into the policy.
	grants := a.Grants()
	if len(grants) != 1 || grants[0] != "go test ./..." {
		t.Fatalf("grants = %v, want [go test ./...]", grants)
	}

	// Now complete the turn. The fake provider records the request it receives.
	fake := provider.NewFake(provider.Turn{Text: "done"})
	a.providerBuilder = func(_ string, _ config.EndpointConfig, _, _, _ string, _ *telemetry.Logger) (provider.Provider, error) {
		return fake, nil
	}

	// The turn's message count has changed before (SessionUsageMsg added one),
	// and a count that is short returns while the turn goroutine is still
	// writing a.conv. So this wait is for the terminal message by type, and
	// then for the goroutine itself to exit. The collector's mutex orders every
	// send before the receive below.
	c2 := newCollector(0)
	a.sendFn = c2.send

	a.Submit("continue", func(string) string { return "fake-key" })

	waitForMsg[tui.TurnCompleteMsg](t, c2, 2*time.Second)
	a.inFlight.Wait()

	requests := fake.Requests()
	if len(requests) != 1 {
		t.Fatalf("provider requests = %d, want 1", len(requests))
	}
	req := requests[0]
	if len(req.Messages) != 3 {
		t.Fatalf("request messages = %d, want 3", len(req.Messages))
	}

	// The resumed conversation must be present and every tool call answered.
	if req.Messages[0].Role != provider.RoleUser {
		t.Errorf("message[0].Role = %v, want user", req.Messages[0].Role)
	}
	if req.Messages[1].Role != provider.RoleAssistant {
		t.Errorf("message[1].Role = %v, want assistant", req.Messages[1].Role)
	}
	if req.Messages[2].Role != provider.RoleUser {
		t.Errorf("message[2].Role = %v, want user", req.Messages[2].Role)
	}

	var callIDs []string
	var resultIDs []string
	for _, msg := range req.Messages {
		for _, b := range msg.Content {
			switch b.Kind {
			case provider.BlockToolCall:
				if b.ToolCall == nil {
					t.Fatal("tool call block with nil ToolCall")
				}
				callIDs = append(callIDs, b.ToolCall.ID)
			case provider.BlockToolResult:
				if b.ToolResult == nil {
					t.Fatal("tool result block with nil ToolResult")
				}
				resultIDs = append(resultIDs, b.ToolResult.CallID)
			case provider.BlockText, provider.BlockThinking:
				// Text and thinking blocks carry no call IDs to match.
			}
		}
	}
	if len(callIDs) != 1 || callIDs[0] != "call-1" {
		t.Fatalf("tool calls = %v, want [call-1]", callIDs)
	}
	if len(resultIDs) != 1 || resultIDs[0] != "call-1" {
		t.Fatalf("tool results = %v, want [call-1]", resultIDs)
	}

	// The in-memory conversation grew by the merged follow-up user message and
	// the assistant response, proving the resumed conversation was the pending
	// state.
	if len(a.conv.Messages) != 4 {
		t.Fatalf("conv.Messages = %d, want 4", len(a.conv.Messages))
	}
}

// TestResumeLastSessionNoSession verifies that ResumeLastSession surfaces the
// sentinel error when the workspace has no recorded session.
func TestResumeLastSessionNoSession(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.Defaults()
	cfg.Session.StorageDir = filepath.Join(tmp, "sessions")

	c := newCollector(0)
	a := testAppWithConfig(t, cfg, c)
	defer a.Close()

	err := a.ResumeLastSession()
	if !errors.Is(err, session.ErrNoSession) {
		t.Fatalf("ResumeLastSession error = %v, want ErrNoSession", err)
	}
}

// TestStartNewSessionCreatesStore verifies that StartNewSession opens a new
// session store and makes it available for subsequent turns.
func TestStartNewSessionCreatesStore(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.Defaults()
	cfg.Session.StorageDir = filepath.Join(tmp, "sessions")

	c := newCollector(0)
	a := testAppWithConfig(t, cfg, c)
	defer a.Close()

	if err := a.StartNewSession(); err != nil {
		t.Fatalf("StartNewSession: %v", err)
	}
	if a.store == nil {
		t.Fatal("store is nil after StartNewSession")
	}
}

func userTextMessage(text string) agent.Message {
	return agent.Message{
		Role:    agent.RoleUser,
		Content: []agent.Block{{Kind: agent.BlockText, Text: text}},
	}
}

func assistantTextToolMessage(text, callID, toolName, input string) agent.Message {
	content := []agent.Block{}
	if text != "" {
		content = append(content, agent.Block{Kind: agent.BlockText, Text: text})
	}
	content = append(content, agent.Block{
		Kind: agent.BlockToolCall,
		ToolCall: &agent.ToolCall{
			ID:    callID,
			Name:  toolName,
			Input: json.RawMessage(input),
		},
	})
	return agent.Message{Role: agent.RoleAssistant, Content: content}
}

func userToolResultMessage(callID, content string, isErr bool) agent.Message {
	return agent.Message{
		Role: agent.RoleUser,
		Content: []agent.Block{{
			Kind: agent.BlockToolResult,
			Result: &agent.ResultBlock{
				CallID:  callID,
				Content: content,
				IsError: isErr,
			},
		}},
	}
}

// clearGrantsApp builds an App whose state root is under tmp, so two Apps built
// in turn see the same session files.
func clearGrantsApp(t *testing.T, tmp string) *App {
	t.Helper()
	cfg := config.Defaults()
	cfg.Session.StorageDir = filepath.Join(tmp, "sessions")
	return testAppWithConfig(t, cfg, newCollector(0))
}

// currentSessionLog returns the path of a's open session log.
func currentSessionLog(a *App) string {
	return session.SessionPath(a.layout.Root, a.layout.Workspace, a.currentSessionID)
}

// resumeGrants resumes the last session in a fresh App and returns the grants
// it restored.
func resumeGrants(t *testing.T, tmp string) []string {
	t.Helper()
	a := clearGrantsApp(t, tmp)
	defer a.Close()
	if err := a.ResumeLastSession(); err != nil {
		t.Fatalf("ResumeLastSession: %v", err)
	}
	return a.Grants()
}

// TestClearGrantsRoundTripAndEvent drives App.ClearGrants on a session with a
// recorded grant. It checks that the log gains an approval.scopes_cleared line
// and that resuming restores no grants. The grant is recorded the way the
// approval path records it (policy.Grant plus Store.RecordScopeGranted), not
// through the approval dialog. A control read of the log with
// session.LoadGrants before the clear shows the grant is in the log, so it
// would otherwise be restored.
func TestClearGrantsRoundTripAndEvent(t *testing.T) {
	tmp := t.TempDir()
	a := clearGrantsApp(t, tmp)
	if err := a.StartNewSession(); err != nil {
		t.Fatalf("StartNewSession: %v", err)
	}
	if err := a.pol.Grant(policy.OperationCommand, []string{"go", "test"}); err != nil {
		t.Fatalf("Grant: %v", err)
	}
	a.store.RecordScopeGranted("run_command", "go test")
	if err := a.store.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	logPath := currentSessionLog(a)

	// Positive control: before the clear, the log holds the grant. LoadGrants
	// is the function resume uses to read grants back, called here directly on
	// the log file; no second App is started.
	before, err := session.LoadGrants(logPath)
	if err != nil || len(before) != 1 {
		t.Fatalf("control LoadGrants = %v, %v; want one grant", before, err)
	}

	a.ClearGrants()
	if got := a.Grants(); len(got) != 0 {
		t.Fatalf("Grants() after clear = %v, want none", got)
	}
	a.Close()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"type":"approval.scopes_cleared"`) {
		t.Fatalf("session log lacks approval.scopes_cleared:\n%s", data)
	}
	if got := resumeGrants(t, tmp); len(got) != 0 {
		t.Fatalf("grants after resume = %v, want none", got)
	}
}

// TestClearGrantsAfterResumeBeforeAnyTurn resumes a session holding a grant,
// calls ClearGrants before any turn runs, closes, and resumes again. The
// resumed store is open for append, so the clear must land in the same file.
func TestClearGrantsAfterResumeBeforeAnyTurn(t *testing.T) {
	tmp := t.TempDir()
	a := clearGrantsApp(t, tmp)
	if err := a.StartNewSession(); err != nil {
		t.Fatalf("StartNewSession: %v", err)
	}
	a.store.RecordScopeGranted("run_command", "go test")
	a.Close()

	// Control: the grant is restored on the first resume.
	b := clearGrantsApp(t, tmp)
	if err := b.ResumeLastSession(); err != nil {
		t.Fatalf("ResumeLastSession: %v", err)
	}
	if got := b.Grants(); len(got) != 1 || got[0] != "go test" {
		t.Fatalf("grants after first resume = %v, want [go test]", got)
	}
	b.ClearGrants()
	b.Close()

	if got := resumeGrants(t, tmp); len(got) != 0 {
		t.Fatalf("grants after clear and second resume = %v, want none", got)
	}
}

// jsonlFiles walks root and returns every .jsonl file under it. A walk error
// fails the test.
func jsonlFiles(t *testing.T, root string) []string {
	t.Helper()
	var found []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(p, ".jsonl") {
			found = append(found, p)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir %s: %v", root, err)
	}
	return found
}

// TestClearGrantsWithoutStoreCreatesNoSession verifies that a clear with no
// open session writes no session file. The absence check is paired with a
// positive control: StartNewSession on the same App, walked over the same
// root, produces exactly one .jsonl file.
func TestClearGrantsWithoutStoreCreatesNoSession(t *testing.T) {
	tmp := t.TempDir()
	a := clearGrantsApp(t, tmp)
	defer a.Close()
	a.ClearGrants()
	if a.store != nil {
		t.Fatal("ClearGrants created a store")
	}
	if found := jsonlFiles(t, tmp); len(found) != 0 {
		t.Fatalf("ClearGrants created session files: %v", found)
	}

	// Positive control: creating the session does produce a file, so the walk
	// above can see one when there is one.
	if err := a.StartNewSession(); err != nil {
		t.Fatalf("StartNewSession: %v", err)
	}
	if err := a.store.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if found := jsonlFiles(t, tmp); len(found) != 1 {
		t.Fatalf("after StartNewSession found %v, want exactly one .jsonl", found)
	}
}

// TestClearGrantsIsDurableWithoutClose records a grant, calls ClearGrants and,
// without calling Close, reads the session file. ClearGrants must have flushed
// the approval.scopes_cleared line itself rather than leaving it queued for
// the ticker or Close. Control: before the clear, the file already holds the
// grant line and not the cleared line.
func TestClearGrantsIsDurableWithoutClose(t *testing.T) {
	tmp := t.TempDir()
	a := clearGrantsApp(t, tmp)
	defer a.Close()
	if err := a.StartNewSession(); err != nil {
		t.Fatalf("StartNewSession: %v", err)
	}
	if err := a.pol.Grant(policy.OperationCommand, []string{"go", "test"}); err != nil {
		t.Fatalf("Grant: %v", err)
	}
	a.store.RecordScopeGranted("run_command", "go test")
	if err := a.store.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	logPath := currentSessionLog(a)

	before, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(before), `"type":"approval.scope_granted"`) ||
		strings.Contains(string(before), `"type":"approval.scopes_cleared"`) {
		t.Fatalf("control: log before clear should hold the grant only:\n%s", before)
	}

	a.ClearGrants()
	// No Close and no explicit Flush here: only what ClearGrants made durable.
	after, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), `"type":"approval.scopes_cleared"`) {
		t.Fatalf("log lacks approval.scopes_cleared before Close:\n%s", after)
	}
}

// grantViaResolve drives the real approval path: Request registers a pending
// command approval and Resolve(id, ApprovalOutcomeSession) grants it, which
// records approval.scope_granted through the store. It returns the outcome and
// grant error Request returned.
func grantViaResolve(t *testing.T, a *App) (ApprovalOutcome, error) {
	t.Helper()
	const id = 1
	type result struct {
		out ApprovalOutcome
		err error
	}
	done := make(chan result, 1)
	go func() {
		out, err := a.Request(a.rootCtx, ApprovalRequest{
			ID:          id,
			Description: "run go test",
			Operation:   policy.OperationCommand,
			Argv:        []string{"go", "test"},
		})
		done <- result{out, err}
	}()
	// Wait for Request to register the pending approval before resolving it.
	deadline := time.Now().Add(2 * time.Second)
	for {
		a.approveMu.Lock()
		_, ok := a.approvals[id]
		a.approveMu.Unlock()
		if ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("Request never registered the approval")
		}
		time.Sleep(time.Millisecond)
	}
	a.Resolve(id, ApprovalOutcomeSession)
	select {
	case r := <-done:
		return r.out, r.err
	case <-time.After(2 * time.Second):
		t.Fatal("Request did not return after Resolve")
		return 0, nil
	}
}

// TestResolveSessionGrantThenClearNotRestored grants through the real path
// (Request plus Resolve with ApprovalOutcomeSession), then calls ClearGrants
// and Close, and resumes in a fresh App. A control run does the same without
// the clear and must restore the grant.
func TestResolveSessionGrantThenClearNotRestored(t *testing.T) {
	run := func(t *testing.T, clear bool) []string {
		tmp := t.TempDir()
		a := clearGrantsApp(t, tmp)
		if err := a.StartNewSession(); err != nil {
			t.Fatalf("StartNewSession: %v", err)
		}
		out, err := grantViaResolve(t, a)
		if err != nil || out != ApprovalOutcomeSession {
			t.Fatalf("Request = %v, %v; want ApprovalOutcomeSession, nil", out, err)
		}
		if got := a.Grants(); len(got) != 1 || got[0] != "go test" {
			t.Fatalf("Grants() after Resolve = %v, want [go test]", got)
		}
		if clear {
			a.ClearGrants()
		}
		a.Close()
		return resumeGrants(t, tmp)
	}

	if got := run(t, false); len(got) != 1 || got[0] != "go test" {
		t.Fatalf("control (no clear): resumed grants = %v, want [go test]", got)
	}
	if got := run(t, true); len(got) != 0 {
		t.Fatalf("resumed grants after clear = %v, want none", got)
	}
}
