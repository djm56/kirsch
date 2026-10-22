package app

import (
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/agent"
)

// applyPatchLogWait bounds how long the failure log waits for the agent turn to
// stop before it gives up reading the conversation.
const applyPatchLogWait = 2 * time.Second

// smokeLogger is the part of *testing.T the apply_patch log writes to. The unit
// tests pass a recording sink in its place.
type smokeLogger interface {
	Logf(format string, args ...any)
}

// logApplyPatchCalls logs every apply_patch tool call in msgs, in order. Each
// call is logged as its "diff" field verbatim, or as its raw input when the
// input is not a JSON object with a string "diff". Each call is followed by the
// result that answers it, matched on call ID, or by "no result". Tool calls of
// any other name are skipped. Only apply_patch inputs and results are logged.
//
// Compaction replaces the messages it summarises with one ordinary assistant
// text message. That message has no field that marks it as a summary, so the
// apply_patch calls inside the summarised range are absent from msgs and are
// not logged. The log does not say that calls were dropped.
func logApplyPatchCalls(log smokeLogger, msgs []agent.Message) {
	// Results are indexed by call ID, so each call finds its own answer whatever
	// order the results arrive in. The first result for an ID wins.
	results := make(map[string]*agent.ResultBlock)
	for _, m := range msgs {
		for _, b := range m.Content {
			if b.Kind != agent.BlockToolResult || b.Result == nil {
				continue
			}
			if _, seen := results[b.Result.CallID]; !seen {
				results[b.Result.CallID] = b.Result
			}
		}
	}

	n := 0
	for _, m := range msgs {
		for _, b := range m.Content {
			if b.Kind != agent.BlockToolCall || b.ToolCall == nil || b.ToolCall.Name != "apply_patch" {
				continue
			}
			n++
			tc := b.ToolCall

			// A pointer field tells a missing "diff" apart from an empty one.
			var in struct {
				Diff *string `json:"diff"`
			}
			if err := json.Unmarshal(tc.Input, &in); err == nil && in.Diff != nil {
				log.Logf("apply_patch call %d (id %q): diff=%q", n, tc.ID, *in.Diff)
			} else {
				log.Logf("apply_patch call %d (id %q): input=%q", n, tc.ID, string(tc.Input))
			}

			res, ok := results[tc.ID]
			if !ok {
				log.Logf("apply_patch call %d (id %q): no result", n, tc.ID)
				continue
			}
			log.Logf("apply_patch call %d (id %q): result is_error=%t error_kind=%q content=%q",
				n, tc.ID, res.IsError, res.ErrorKind, res.Content)
		}
	}
}

// noFinalTextLine is logged when the conversation holds no assistant text.
const noFinalTextLine = "final assistant text: none (no assistant message with text in the conversation)"

// logFinalText logs the text blocks of the last assistant message in msgs,
// joined in order, as one quoted line. It logs noFinalTextLine when msgs has no
// assistant message or that message has no text block with text.
func logFinalText(log smokeLogger, msgs []agent.Message) {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role != agent.RoleAssistant {
			continue
		}
		var text string
		for _, b := range msgs[i].Content {
			if b.Kind == agent.BlockText {
				text += b.Text
			}
		}
		if text == "" {
			break
		}
		log.Logf("final assistant text: %q", text)
		return
	}
	log.Logf("%s", noFinalTextLine)
}

// logApplyPatchOnFailure decides what the failure log writes. When failed is
// false it returns before it touches any seam. Otherwise it calls cancel, then
// calls wait with applyPatchLogWait. If wait reports true, the turn has stopped
// and the apply_patch calls in msgs() are logged, followed by the final
// assistant text from the same read. If wait reports false, the turn has not
// stopped, so msgs is not read and a skip line is logged instead; the final
// text is not logged on that branch. The seams let a test drive each branch
// without a live App.
func logApplyPatchOnFailure(failed bool, log smokeLogger, cancel func(), wait func(time.Duration) bool, msgs func() []agent.Message) {
	if !failed {
		return
	}
	// Cancelling first matches App.Close, which cancels the root context before
	// it waits on inFlight.
	cancel()
	if !wait(applyPatchLogWait) {
		log.Logf("apply_patch calls not logged: the agent turn had not stopped after %s, so its conversation was not read", applyPatchLogWait)
		return
	}
	conv := msgs()
	logApplyPatchCalls(log, conv)
	logFinalText(log, conv)
}

// waitWithin reports whether wg reaches zero within d. It returns true when the
// group finishes first and false when d elapses first. After a false result the
// helper goroutine stays blocked in wg.Wait until the group does reach zero.
// Each turn calls inFlight.Done after its last write to conv, so a true result
// makes the conv reads that follow race-free.
func waitWithin(wg *sync.WaitGroup, d time.Duration) bool {
	idle := make(chan struct{})
	go func() {
		wg.Wait()
		close(idle)
	}()
	select {
	case <-idle:
		return true
	case <-time.After(d):
		return false
	}
}

// recordingLog is a smokeLogger that keeps each formatted line.
type recordingLog struct {
	lines []string
}

// Logf records one formatted line.
func (r *recordingLog) Logf(format string, args ...any) {
	r.lines = append(r.lines, fmt.Sprintf(format, args...))
}

// failureSeams is the set of fakes for logApplyPatchOnFailure. order records
// the cancel, wait and messages calls in the order they happen. reads counts
// the messages calls. waited is the duration wait was given.
type failureSeams struct {
	order  []string
	reads  int
	waited time.Duration
	waitOK bool
	msgs   []agent.Message
}

// cancel records a cancel call.
func (f *failureSeams) cancel() {
	f.order = append(f.order, "cancel")
}

// wait records a wait call and returns the configured result.
func (f *failureSeams) wait(d time.Duration) bool {
	f.order = append(f.order, "wait")
	f.waited = d
	return f.waitOK
}

// messages records a messages read and returns the configured conversation.
func (f *failureSeams) messages() []agent.Message {
	f.order = append(f.order, "messages")
	f.reads++
	return f.msgs
}

// toolCallBlock builds an assistant block that calls name with input.
func toolCallBlock(id, name, input string) agent.Block {
	return agent.Block{Kind: agent.BlockToolCall, ToolCall: &agent.ToolCall{
		ID: id, Name: name, Input: json.RawMessage(input),
	}}
}

// toolResultBlock builds a result block that answers callID.
func toolResultBlock(callID, content string, isError bool, errorKind string) agent.Block {
	return agent.Block{Kind: agent.BlockToolResult, Result: &agent.ResultBlock{
		CallID: callID, Content: content, IsError: isError, ErrorKind: errorKind,
	}}
}

// diffInput is the JSON input an apply_patch call carries for diff.
func diffInput(t *testing.T, diff string) string {
	t.Helper()
	b, err := json.Marshal(map[string]string{"diff": diff})
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	return string(b)
}

// TestLogApplyPatchCalls drives the logging helper directly with a fake
// conversation. It does not go through App.Submit, so it checks the formatting
// and pairing only. logApplyPatchOnFailure also reaches the helper, and
// TestLogApplyPatchOnFailure covers that path with fakes. Only the wiring
// closure in smoke_live_test.go is exercised solely by a failed TestSmokeLive.
func TestLogApplyPatchCalls(t *testing.T) {
	t.Run("diff logged verbatim with tabs and newlines", func(t *testing.T) {
		diff := "--- a/x\n+++ b/x\n+\tif b == 0 {\n"
		msgs := []agent.Message{
			{Role: agent.RoleAssistant, Content: []agent.Block{
				toolCallBlock("c1", "apply_patch", diffInput(t, diff)),
			}},
			{Role: agent.RoleUser, Content: []agent.Block{
				toolResultBlock("c1", "applied", false, ""),
			}},
		}
		rec := &recordingLog{}
		logApplyPatchCalls(rec, msgs)
		want := []string{
			`apply_patch call 1 (id "c1"): diff="--- a/x\n+++ b/x\n+\tif b == 0 {\n"`,
			`apply_patch call 1 (id "c1"): result is_error=false error_kind="" content="applied"`,
		}
		checkLines(t, rec.lines, want)
	})

	t.Run("result paired by call ID not position", func(t *testing.T) {
		msgs := []agent.Message{
			{Role: agent.RoleAssistant, Content: []agent.Block{
				toolCallBlock("c1", "apply_patch", diffInput(t, "A")),
				toolCallBlock("c2", "apply_patch", diffInput(t, "B")),
			}},
			{Role: agent.RoleUser, Content: []agent.Block{
				// Results arrive in the reverse order of the calls.
				toolResultBlock("c2", "result-two", true, "invalid_patch"),
				toolResultBlock("c1", "result-one", false, ""),
			}},
		}
		rec := &recordingLog{}
		logApplyPatchCalls(rec, msgs)
		want := []string{
			`apply_patch call 1 (id "c1"): diff="A"`,
			`apply_patch call 1 (id "c1"): result is_error=false error_kind="" content="result-one"`,
			`apply_patch call 2 (id "c2"): diff="B"`,
			`apply_patch call 2 (id "c2"): result is_error=true error_kind="invalid_patch" content="result-two"`,
		}
		checkLines(t, rec.lines, want)
	})

	t.Run("first result for an ID wins", func(t *testing.T) {
		msgs := []agent.Message{
			{Role: agent.RoleAssistant, Content: []agent.Block{
				toolCallBlock("c1", "apply_patch", diffInput(t, "A")),
			}},
			{Role: agent.RoleUser, Content: []agent.Block{
				toolResultBlock("c1", "first", false, ""),
				toolResultBlock("c1", "second", true, "invalid_patch"),
			}},
		}
		rec := &recordingLog{}
		logApplyPatchCalls(rec, msgs)
		want := []string{
			`apply_patch call 1 (id "c1"): diff="A"`,
			`apply_patch call 1 (id "c1"): result is_error=false error_kind="" content="first"`,
		}
		checkLines(t, rec.lines, want)
	})

	t.Run("missing result logged as no result", func(t *testing.T) {
		msgs := []agent.Message{
			{Role: agent.RoleAssistant, Content: []agent.Block{
				toolCallBlock("c1", "apply_patch", diffInput(t, "A")),
			}},
		}
		rec := &recordingLog{}
		logApplyPatchCalls(rec, msgs)
		want := []string{
			`apply_patch call 1 (id "c1"): diff="A"`,
			`apply_patch call 1 (id "c1"): no result`,
		}
		checkLines(t, rec.lines, want)
	})

	t.Run("malformed input falls back to raw input", func(t *testing.T) {
		msgs := []agent.Message{
			{Role: agent.RoleAssistant, Content: []agent.Block{
				toolCallBlock("c1", "apply_patch", `{"diff": `),
			}},
			{Role: agent.RoleUser, Content: []agent.Block{
				toolResultBlock("c1", "bad input", true, "invalid_input"),
			}},
		}
		rec := &recordingLog{}
		logApplyPatchCalls(rec, msgs)
		want := []string{
			`apply_patch call 1 (id "c1"): input="{\"diff\": "`,
			`apply_patch call 1 (id "c1"): result is_error=true error_kind="invalid_input" content="bad input"`,
		}
		checkLines(t, rec.lines, want)
	})

	t.Run("null diff falls back to raw input", func(t *testing.T) {
		msgs := []agent.Message{
			{Role: agent.RoleAssistant, Content: []agent.Block{
				toolCallBlock("c1", "apply_patch", `{"diff":null}`),
			}},
		}
		rec := &recordingLog{}
		logApplyPatchCalls(rec, msgs)
		want := []string{
			`apply_patch call 1 (id "c1"): input="{\"diff\":null}"`,
			`apply_patch call 1 (id "c1"): no result`,
		}
		checkLines(t, rec.lines, want)
	})

	t.Run("non-string diff falls back to raw input", func(t *testing.T) {
		msgs := []agent.Message{
			{Role: agent.RoleAssistant, Content: []agent.Block{
				toolCallBlock("c1", "apply_patch", `{"diff":5}`),
			}},
		}
		rec := &recordingLog{}
		logApplyPatchCalls(rec, msgs)
		want := []string{
			`apply_patch call 1 (id "c1"): input="{\"diff\":5}"`,
			`apply_patch call 1 (id "c1"): no result`,
		}
		checkLines(t, rec.lines, want)
	})

	t.Run("input without a diff field falls back to raw input", func(t *testing.T) {
		msgs := []agent.Message{
			{Role: agent.RoleAssistant, Content: []agent.Block{
				toolCallBlock("c1", "apply_patch", `{"description":"x"}`),
			}},
		}
		rec := &recordingLog{}
		logApplyPatchCalls(rec, msgs)
		want := []string{
			`apply_patch call 1 (id "c1"): input="{\"description\":\"x\"}"`,
			`apply_patch call 1 (id "c1"): no result`,
		}
		checkLines(t, rec.lines, want)
	})

	t.Run("other tool calls skipped, apply_patch in the same slice logged", func(t *testing.T) {
		msgs := []agent.Message{
			{Role: agent.RoleAssistant, Content: []agent.Block{
				toolCallBlock("r1", "run_command", `{"argv":["go","test"]}`),
				toolCallBlock("a1", "apply_patch", diffInput(t, "D")),
			}},
			{Role: agent.RoleUser, Content: []agent.Block{
				toolResultBlock("r1", "ok", false, ""),
				toolResultBlock("a1", "applied", false, ""),
			}},
		}
		rec := &recordingLog{}
		logApplyPatchCalls(rec, msgs)
		want := []string{
			`apply_patch call 1 (id "a1"): diff="D"`,
			`apply_patch call 1 (id "a1"): result is_error=false error_kind="" content="applied"`,
		}
		checkLines(t, rec.lines, want)
	})
}

// TestLogApplyPatchOnFailure drives the failure decision with fakes for every
// seam. The guard subtest checks that a passed test touches no seam and logs
// nothing, and it hands over a conversation holding a call, so a dropped or
// inverted guard shows up as logged lines or seam calls. The success subtest is
// its mirror on the same conversation. The timeout and order subtests cover the
// other branch and the seam order. The final-text subtests cover the last
// assistant text logged after the calls on a failed test, and nothing logged on
// a passed one.
func TestLogApplyPatchOnFailure(t *testing.T) {
	// call is a conversation with one apply_patch call and its result.
	call := []agent.Message{
		{Role: agent.RoleAssistant, Content: []agent.Block{
			toolCallBlock("c1", "apply_patch", diffInput(t, "A")),
		}},
		{Role: agent.RoleUser, Content: []agent.Block{
			toolResultBlock("c1", "applied", false, ""),
		}},
	}
	// callLines is what logApplyPatchCalls writes for call.
	// The conversation has no assistant text, so the no-text line follows.
	callLines := []string{
		`apply_patch call 1 (id "c1"): diff="A"`,
		`apply_patch call 1 (id "c1"): result is_error=false error_kind="" content="applied"`,
		noFinalTextLine,
	}
	// skipLine is the line the timeout branch writes instead of the calls.
	skipLine := fmt.Sprintf("apply_patch calls not logged: the agent turn had not stopped after %s, so its conversation was not read", applyPatchLogWait)

	t.Run("guard: passed test touches no seam and logs nothing", func(t *testing.T) {
		rec := &recordingLog{}
		f := &failureSeams{waitOK: true, msgs: call}
		logApplyPatchOnFailure(false, rec, f.cancel, f.wait, f.messages)
		checkLines(t, rec.lines, nil)
		if len(f.order) != 0 {
			t.Errorf("seams called on a passed test: %q", f.order)
		}
	})

	t.Run("success: failed test with the turn stopped logs the calls", func(t *testing.T) {
		rec := &recordingLog{}
		f := &failureSeams{waitOK: true, msgs: call}
		logApplyPatchOnFailure(true, rec, f.cancel, f.wait, f.messages)
		checkLines(t, rec.lines, callLines)
		if f.reads != 1 {
			t.Errorf("messages read %d times, want 1", f.reads)
		}
	})

	t.Run("timeout: turn not stopped logs only the skip line", func(t *testing.T) {
		rec := &recordingLog{}
		f := &failureSeams{waitOK: false, msgs: call}
		logApplyPatchOnFailure(true, rec, f.cancel, f.wait, f.messages)
		// The skip line is the presence check. checkLines also rules out any diff
		// line beside it, on the same path that would have logged the calls.
		checkLines(t, rec.lines, []string{skipLine})
		if f.reads != 0 {
			t.Errorf("messages read %d times on timeout, want 0", f.reads)
		}
		// Cancel runs before the wait, and nothing follows the wait on timeout.
		want := []string{"cancel", "wait"}
		if fmt.Sprint(f.order) != fmt.Sprint(want) {
			t.Errorf("seam order %q on timeout, want %q", f.order, want)
		}
	})

	t.Run("final text: failed test logs the last assistant text", func(t *testing.T) {
		conv := append(slices.Clone(call), agent.Message{Role: agent.RoleAssistant, Content: []agent.Block{
			{Kind: agent.BlockText, Text: "**Summary:** done"},
		}})
		rec := &recordingLog{}
		f := &failureSeams{waitOK: true, msgs: conv}
		logApplyPatchOnFailure(true, rec, f.cancel, f.wait, f.messages)
		want := append(slices.Clone(callLines[:2]), `final assistant text: "**Summary:** done"`)
		checkLines(t, rec.lines, want)
	})

	t.Run("final text: passed test logs no final text", func(t *testing.T) {
		conv := append(slices.Clone(call), agent.Message{Role: agent.RoleAssistant, Content: []agent.Block{
			{Kind: agent.BlockText, Text: "**Summary:** done"},
		}})
		rec := &recordingLog{}
		f := &failureSeams{waitOK: true, msgs: conv}
		logApplyPatchOnFailure(false, rec, f.cancel, f.wait, f.messages)
		checkLines(t, rec.lines, nil)
	})

	t.Run("order: cancel then wait then read", func(t *testing.T) {
		rec := &recordingLog{}
		f := &failureSeams{waitOK: true, msgs: call}
		logApplyPatchOnFailure(true, rec, f.cancel, f.wait, f.messages)
		want := []string{"cancel", "wait", "messages"}
		if fmt.Sprint(f.order) != fmt.Sprint(want) {
			t.Errorf("seam order %q, want %q", f.order, want)
		}
		if f.waited != applyPatchLogWait {
			t.Errorf("wait given %s, want %s", f.waited, applyPatchLogWait)
		}
	})
}

// TestLogFinalText drives logFinalText with fake conversations. Only the last
// assistant message counts, and only its text blocks.
func TestLogFinalText(t *testing.T) {
	text := func(s string) agent.Block { return agent.Block{Kind: agent.BlockText, Text: s} }
	t.Run("last assistant message wins, text blocks joined, tool blocks skipped", func(t *testing.T) {
		msgs := []agent.Message{
			{Role: agent.RoleAssistant, Content: []agent.Block{text("early")}},
			{Role: agent.RoleUser, Content: []agent.Block{text("user text")}},
			{Role: agent.RoleAssistant, Content: []agent.Block{
				text("part one "), toolCallBlock("c1", "run_command", `{}`), text("part two"),
			}},
		}
		rec := &recordingLog{}
		logFinalText(rec, msgs)
		checkLines(t, rec.lines, []string{`final assistant text: "part one part two"`})
	})

	t.Run("no assistant message logs the no-text line", func(t *testing.T) {
		rec := &recordingLog{}
		logFinalText(rec, []agent.Message{{Role: agent.RoleUser, Content: []agent.Block{text("hi")}}})
		checkLines(t, rec.lines, []string{noFinalTextLine})
	})

	t.Run("last assistant message without text logs the no-text line", func(t *testing.T) {
		msgs := []agent.Message{
			{Role: agent.RoleAssistant, Content: []agent.Block{text("older text")}},
			{Role: agent.RoleAssistant, Content: []agent.Block{toolCallBlock("c1", "run_command", `{}`)}},
		}
		rec := &recordingLog{}
		logFinalText(rec, msgs)
		checkLines(t, rec.lines, []string{noFinalTextLine})
	})
}

// TestWaitWithin checks both results of waitWithin on a real wait group.
func TestWaitWithin(t *testing.T) {
	t.Run("true once the group reaches zero", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		go wg.Done()
		if !waitWithin(&wg, 5*time.Second) {
			t.Error("waitWithin = false, want true once the group reached zero")
		}
	})

	t.Run("false when the group has not reached zero", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		if waitWithin(&wg, 10*time.Millisecond) {
			t.Error("waitWithin = true, want false while the group has not reached zero")
		}
		// Release the helper goroutine blocked in Wait.
		wg.Done()
	})
}

// checkLines reports an error with both slices when got and want differ.
func checkLines(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d\ngot:  %q\nwant: %q", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d:\ngot:  %s\nwant: %s", i, got[i], want[i])
		}
	}
}
