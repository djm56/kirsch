package tui_test

import (
	"context"
	"path/filepath"
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

// Tests for the approval card of a request whose Operation is
// policy.OperationUnspecified (plan C36). They live in package tui_test, not
// tui, because the entry point is app.App.Request: that call is what turns a
// request into the tui.ApprovalRequestedMsg the model receives, and package app
// imports package tui, so an in-package test could not call it.

// unspecifiedRequestName is the text the request carries in Description. It is
// the only operation name an unspecified request has: Operation is the zero
// value and ApprovalRequest has no other name field.
const unspecifiedRequestName = "reticulate splines"

// unspecifiedApprovalView drives an unspecified-operation request through the
// real path and returns the message the TUI received and the rendered screen.
//
// Entry point: app.App.Request (internal/app/app.go, the call that builds
// ApprovalRequestedMsg), with the message captured through SetSendFn. The
// captured message is then fed to tui.Model.Update and the screen is read from
// Model.View. Nothing is hand-built, so the test sees what a real request
// produces, not what a fixture assumes it produces.
func unspecifiedApprovalView(t *testing.T) (tui.ApprovalRequestedMsg, string) {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "testdata", "repo-small"))
	if err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	a := app.New(ws, config.Defaults(), telemetry.Disabled())
	defer a.Close()

	sent := make(chan any, 4)
	a.SetSendFn(func(m any) { sent <- m })

	// Request blocks until resolved or cancelled; cancel it once the message is in hand.
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = a.Request(ctx, app.ApprovalRequest{
			Description: unspecifiedRequestName,
			Operation:   policy.OperationUnspecified,
		})
	}()
	defer func() {
		cancel()
		<-done
	}()

	var msg tui.ApprovalRequestedMsg
	select {
	case m := <-sent:
		var ok bool
		if msg, ok = m.(tui.ApprovalRequestedMsg); !ok {
			t.Fatalf("first message is %T, want tui.ApprovalRequestedMsg", m)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("App.Request sent no message")
	}

	m := tui.New(tui.Options{})
	um, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	um, _ = um.(tui.Model).Update(msg)
	return msg, um.(tui.Model).View()
}

// TestApprovalUnspecifiedCardShowsRequestName asserts that the card for an
// unspecified operation shows the name the request carries (its Description) as
// the card title, and that the message reaching the TUI still carries it.
func TestApprovalUnspecifiedCardShowsRequestName(t *testing.T) {
	msg, view := unspecifiedApprovalView(t)

	if msg.Description != unspecifiedRequestName {
		t.Errorf("message Description = %q, want %q", msg.Description, unspecifiedRequestName)
	}
	if !strings.Contains(view, "approval required") {
		t.Errorf("view has no approval card (control for the title check):\n%s", view)
	}
	if !strings.Contains(view, unspecifiedRequestName) {
		t.Errorf("card does not show the request name %q:\n%s", unspecifiedRequestName, view)
	}
}

// boxContent returns a rendered row with the box and gutter characters and
// spaces removed, so a row with nothing in it becomes "".
func boxContent(line string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case ' ', '|', '│', '┃':
			return -1
		}
		return r
	}, line)
}

// TestApprovalUnspecifiedCardBlankFieldsKnownDefect records what renders today
// for an unspecified operation. The milestone asks that the card show the
// operation name and no empty field; the name half is asserted above and holds,
// the no-empty-field half does NOT hold. This test therefore pins the current
// output, defect included, so the gap is visible and a fix has to flip it.
//
// DEFECT (product code, not changed here): for OperationUnspecified,
// App.Request leaves Subject and Detail empty (app.go, the
// `case policy.OperationUnspecified` branch computes nothing) and derives Kind
// "patch" (kindString). The card then renders
//   - a title row and two blank rows where Detail would be,
//   - the "[d] view diff" key, though there is no diff to view,
//
// and, once resolved, a collapsed line with an empty Subject.
// When the product code is fixed this test fails on purpose: replace the pins
// with assertions that Subject, Detail and the body rows are non-empty.
//
// Chosen over t.Skip so the defect stays executable: a skip cannot fail, so it
// would also pass if the fields were filled in or the card vanished.
func TestApprovalUnspecifiedCardBlankFieldsKnownDefect(t *testing.T) {
	msg, view := unspecifiedApprovalView(t)

	// Message-level pins: the fields the card is built from.
	if msg.Subject != "" {
		t.Errorf("Subject = %q; pinned empty (known defect) - if filled, update this test", msg.Subject)
	}
	if len(msg.Detail) != 0 {
		t.Errorf("Detail = %q; pinned empty (known defect) - if filled, update this test", msg.Detail)
	}

	// Screen-level pin: the rows between the title and the key row are blank.
	lines := strings.Split(view, "\n")
	title, keys := -1, -1
	for i, ln := range lines {
		switch {
		case title < 0 && strings.Contains(ln, unspecifiedRequestName):
			title = i
		case keys < 0 && strings.Contains(ln, "[y] approve"):
			keys = i
		}
	}
	if title < 0 || keys < 0 || keys <= title {
		t.Fatalf("could not locate title and key rows (title=%d keys=%d):\n%s", title, keys, view)
	}
	body := lines[title+1 : keys]
	if len(body) == 0 {
		t.Fatalf("no rows between title and key row; layout changed - update this test:\n%s", view)
	}
	for _, ln := range body {
		if boxContent(ln) != "" {
			t.Errorf("row %q between title and keys is non-empty; pinned blank (known defect) - if filled, update this test", ln)
		}
	}
	if !strings.Contains(lines[keys], "[d] view diff") {
		t.Errorf("key row %q lacks [d] view diff; pinned as shown today", lines[keys])
	}
}
