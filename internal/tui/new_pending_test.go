package tui

// Tests for C48: while /new is pending, the messages the cancelled turn can
// still send must not clear busy or change its verb, and so Enter must not
// reach Submit (where the real app refuses it with a "busy" TurnErrorMsg that
// shows as a stray error card).
//
// Entry points, so each test name can be read against what it really calls:
// every test drives Model.Update only. /new is reached by the keypresses
// "/new" and Enter through keyComposing, submit, runSlash and applyConfirm;
// the tests never call applyConfirm or runSlash directly. The late messages
// are delivered as values to Update, the way the Bubble Tea loop would deliver
// what internal/app/app.go sends. The tests never run the Cmd /new returns, so
// NewSession never runs; where a test needs NewSessionDoneMsg it delivers that
// message to Update itself.

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// errorCards counts the error cards in the transcript.
func errorCards(m Model) int {
	n := 0
	for _, it := range m.tr.Items() {
		if it.Kind == KindError {
			n++
		}
	}
	return n
}

// TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages starts
// /new on an idle model (runSlash calls applyConfirm directly, with no confirm
// prompt), then delivers one late message, then types text and presses Enter.
// Every case but ManualCompactMsg is a message the cancelled turn can still
// send. The ManualCompactMsg case does not build the sequence that brings that
// message into the /new window: like every case here it starts /new from idle,
// with no /compact run and compacting false, and delivers the message straight
// after. It pins only that the busy hold covers ManualCompactMsg.
// TestManualCompactMsgDuringNewAfterBusyClearedMidCompaction drives the
// sequence itself.
//
// The app answers a Submit that arrives during /new with a TurnErrorMsg of
// Kind "busy" (App.Submit, the opMu.TryLock refusal). The Submit stub below
// only counts calls. When it was called, the test delivers that refusal
// itself, so a Submit that gets through shows up as the stray error card C48
// describes.
//
// Each case is checked twice: busy right after the late message (the cause),
// and the Submit count and error-card count after Enter (the symptom). The
// "control" subtest delivers a message that does not touch busy at all
// (UsageMsg); it shows only that the busy assertion holds when nothing
// interferes. The Submit and card counts are absence checks, so the last
// subtest ("positive control") is the presence case: with no /new pending, a
// ToolStartedMsg then a TurnCompleteMsg leave busy cleared, Enter reaches
// Submit once, and the busy refusal adds exactly one error card. That subtest
// stops the others passing because Enter or the card count can never move.
func TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages(t *testing.T) {
	cases := []struct {
		name string
		msg  tea.Msg
	}{
		{"control: UsageMsg touches no busy state", UsageMsg{InputTokens: 1}},
		{"TurnCompleteMsg", TurnCompleteMsg{}},
		{"TurnCancelledMsg", TurnCancelledMsg{}},
		{"TurnErrorMsg", TurnErrorMsg{Kind: "provider", Message: "stream cut"}},
		{"ErrorMsg", ErrorMsg{Kind: "session_store", Message: "flush failed"}},
		{"ManualCompactMsg (delivered directly, no /compact run)", ManualCompactMsg{Outcome: ManualCompactCancelled}},
		{"ToolStartedMsg", ToolStartedMsg{ID: 900, Name: "read_file", Target: "x.go"}},
		{"AssistantTextDeltaMsg", AssistantTextDeltaMsg{BlockID: 901, Delta: "late"}},
		{"CompactingMsg", CompactingMsg{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := readyModel(t)
			stub := &newSessionStub{release: make(chan struct{})}
			m.NewSession = stub.fn
			var submitted int
			m.Submit = func(string) { submitted++ }
			// The composer is idle, as it is after Esc cancelled a turn that is
			// still unwinding in the app: Enter is swallowed while busy, so
			// /new is only reachable from idle, where it runs with no confirm.
			// The Cmd it returns is never run, so NewSession stays pending.
			m, _ = submitSlash(t, m, "/new")
			if !m.newSessionPending {
				t.Fatalf("setup: newSessionPending is false after /new and Enter (mode %v)", m.mode())
			}

			m = drive(m, tc.msg)

			if !m.busy.Active || m.busy.Verb != "new session" {
				t.Errorf("after %T: busy = {%v %q}, want {true \"new session\"}", tc.msg, m.busy.Active, m.busy.Verb)
			}

			cardsBefore := errorCards(m)
			m = drive(m, key('h'), key('i'), keyType(tea.KeyEnter))
			if submitted != 0 {
				// What the real app does with a Submit during /new.
				m = drive(m, TurnErrorMsg{Kind: "busy", Message: "A session operation is in progress. Try again in a moment."})
			}
			if submitted != 0 {
				t.Errorf("after %T: Enter during /new reached Submit %d time(s), want 0", tc.msg, submitted)
			}
			if got := errorCards(m); got != cardsBefore {
				t.Errorf("after %T: Enter during /new left %d error card(s), want the %d already there", tc.msg, got, cardsBefore)
			}

			close(stub.release)
		})
	}

	t.Run("positive control: no /new pending, Enter reaches Submit", func(t *testing.T) {
		m := readyModel(t)
		var submitted int
		m.Submit = func(string) { submitted++ }
		// A tool message sets busy and a completion clears it, as in a turn that
		// ends normally: with no /new pending, busy must really end here.
		m = drive(m, ToolStartedMsg{ID: 900, Name: "read_file", Target: "x.go"})
		if !m.busy.Active {
			t.Fatalf("setup: busy is not active after ToolStartedMsg")
		}
		m = drive(m, TurnCompleteMsg{})
		if m.busy.Active {
			t.Fatalf("with no /new pending, TurnCompleteMsg left busy = {%v %q}, want idle", m.busy.Active, m.busy.Verb)
		}

		cardsBefore := errorCards(m)
		m = drive(m, key('h'), key('i'), keyType(tea.KeyEnter))
		if submitted != 1 {
			t.Fatalf("Enter with /new not pending reached Submit %d time(s), want 1", submitted)
		}
		// The refusal the real app would send if /new were running; here it
		// shows the card count can move.
		m = drive(m, TurnErrorMsg{Kind: "busy", Message: "A session operation is in progress. Try again in a moment."})
		if got := errorCards(m); got != cardsBefore+1 {
			t.Errorf("error cards = %d, want %d: the refusal's card was not counted", got, cardsBefore+1)
		}
	})
}

// TestNewSessionDoneEndsBusyAfterLateTurnMessages is the other side of the
// fence: after late turn messages arrived during /new, NewSessionDoneMsg must
// still clear busy and the pending flag, and the late messages' own effects
// (here, an error card) are kept rather than dropped. On success the
// transcript is reset to empty; on failure (NewSessionDoneMsg with an Err) it
// is not reset, which the test checks by finding the late error card still
// there. It drives Model.Update only, from an idle composer.
func TestNewSessionDoneEndsBusyAfterLateTurnMessages(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"success", nil},
		{"failure", errors.New("disk full")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := readyModel(t)
			m.NewSession = (&newSessionStub{release: make(chan struct{})}).fn
			m, _ = submitSlash(t, m, "/new")
			m = drive(m, TurnErrorMsg{Kind: "provider", Message: "stream cut"}, TurnCompleteMsg{})
			if !m.newSessionPending || !m.busy.Active {
				t.Fatalf("setup: pending=%v busy=%v after the late messages, want both true", m.newSessionPending, m.busy.Active)
			}
			if errorCards(m) != 1 {
				t.Fatalf("the late TurnErrorMsg's card was dropped: %d error cards, want 1", errorCards(m))
			}

			m = drive(m, NewSessionDoneMsg{Err: tc.err})
			if m.busy.Active || m.newSessionPending {
				t.Errorf("after NewSessionDoneMsg: busy=%v pending=%v, want both false", m.busy.Active, m.newSessionPending)
			}
			if tc.err == nil {
				if n := len(m.tr.Items()); n != 0 {
					t.Errorf("transcript holds %d items after a successful /new, want 0", n)
				}
			} else if got := errorCards(m); got != 1 {
				t.Errorf("after a failed /new: %d error cards, want the late one kept", got)
			}
		})
	}
}

// TestManualCompactMsgDuringNewAfterBusyClearedMidCompaction drives, through
// Model.Update only, the sequence the Update docblock describes for a
// ManualCompactMsg arriving while /new is pending: "/compact" and Enter, then
// a session_store ErrorMsg (the message flushStore sends for a failed flush),
// then "/new" and Enter, then the ManualCompactMsg, then NewSessionDoneMsg.
// Neither the Cmd /compact returns nor the one /new returns is run, so
// ManualCompact and NewSession never run; the test delivers their messages to
// Update itself.
//
// The ErrorMsg clearing busy mid-compaction is current behaviour, asserted as
// a setup step so the test fails if the sequence stops reaching /new from
// idle. After the ManualCompactMsg the test checks that /new is still pending
// and busy is held at "new session" (the hold), and that the message's other
// effects are kept: compacting cleared and its notice in the transcript. Enter
// in that window must not reach Submit (an absence check; the positive
// control in TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages
// shows Enter can reach Submit once busy is cleared). NewSessionDoneMsg must
// then end busy.
func TestManualCompactMsgDuringNewAfterBusyClearedMidCompaction(t *testing.T) {
	m := readyModel(t)
	m.ManualCompact = func() ManualCompactMsg { return ManualCompactMsg{Outcome: ManualCompactApplied} }
	m.NewSession = (&newSessionStub{release: make(chan struct{})}).fn
	var submitted int
	m.Submit = func(string) { submitted++ }

	m, _ = submitSlash(t, m, "/compact")
	if !m.compacting || !m.busy.Active || m.busy.Verb != "compacting" {
		t.Fatalf("setup: after /compact compacting=%v busy={%v %q}, want true and {true \"compacting\"}", m.compacting, m.busy.Active, m.busy.Verb)
	}

	m = drive(m, ErrorMsg{Kind: "session_store", Message: "Failed to flush session log: disk full"})
	if m.busy.Active || !m.compacting {
		t.Fatalf("setup: after the flush ErrorMsg busy=%v compacting=%v, want busy cleared and compacting still set", m.busy.Active, m.compacting)
	}

	m, _ = submitSlash(t, m, "/new")
	if !m.newSessionPending {
		t.Fatalf("setup: newSessionPending is false after /new and Enter (mode %v)", m.mode())
	}

	m = drive(m, ManualCompactMsg{Outcome: ManualCompactApplied})
	if !m.newSessionPending {
		t.Errorf("after ManualCompactMsg: newSessionPending is false, want /new still pending")
	}
	if !m.busy.Active || m.busy.Verb != "new session" {
		t.Errorf("after ManualCompactMsg: busy = {%v %q}, want {true \"new session\"}", m.busy.Active, m.busy.Verb)
	}
	if m.compacting {
		t.Errorf("after ManualCompactMsg: compacting is still set, want it cleared")
	}
	if n := countNotices(m, "compaction complete"); n != 1 {
		t.Errorf("after ManualCompactMsg: %d \"compaction complete\" notice(s), want 1", n)
	}

	m = drive(m, key('h'), key('i'), keyType(tea.KeyEnter))
	if submitted != 0 {
		t.Errorf("Enter after ManualCompactMsg during /new reached Submit %d time(s), want 0", submitted)
	}

	m = drive(m, NewSessionDoneMsg{})
	if m.busy.Active || m.newSessionPending {
		t.Errorf("after NewSessionDoneMsg: busy=%v pending=%v, want both false", m.busy.Active, m.newSessionPending)
	}
}
