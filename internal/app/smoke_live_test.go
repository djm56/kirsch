//go:build smoke

package app

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/agent"
	"github.com/djm56/kirsch/internal/tui"
)

// TestSmokeLive runs the smoke test against a real configured provider.
// It requires a valid API key to be configured via environment variables.
// The test has a 5-minute timeout and will skip if no API key is available.
// Entry point: App.Submit. This test uses the live context with 5-minute deadline
// and auto-approves tool requests. Live mode uses real config loading and os.Getenv.
// A live run auto-approves every tool request, and it does so in a temp copy.
func TestSmokeLive(t *testing.T) {
	// Skip if no API key is configured.
	// Check for at least one of the supported keys.
	if os.Getenv("ANTHROPIC_API_KEY") == "" &&
		os.Getenv("KIRSCH_ANTHROPIC_API_KEY") == "" &&
		os.Getenv("OPENCODE_API_KEY") == "" &&
		os.Getenv("KIRSCH_OPENCODE_API_KEY") == "" {
		t.Skip("no API key configured; set ANTHROPIC_API_KEY, KIRSCH_ANTHROPIC_API_KEY, OPENCODE_API_KEY, or KIRSCH_OPENCODE_API_KEY")
	}

	e := newSmokeEnv(t, os.Getenv, nil)
	// On failure, log every apply_patch call and its result, and the final
	// assistant text. This is registered here so it also runs when a t.Fatalf
	// below ends the test early.
	e.t.Cleanup(func() {
		logApplyPatchOnFailure(e.t.Failed(), e.t, e.a.rootCancel,
			func(d time.Duration) bool { return waitWithin(&e.a.inFlight, d) },
			func() []agent.Message { return e.a.conv.Messages })
	})

	// Create context with 5-minute deadline for live test.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Allow every approval request once; the helper stops with the test.
	e.autoApprove()

	msgs := e.submit(ctx, os.Getenv)

	// Verify the expected outcomes.
	if !findMsgType[tui.TurnCompleteMsg](msgs) {
		t.Fatalf("no TurnCompleteMsg; got %#v", msgs)
	}

	// Collect and check result using the same checker as offline test.
	result := collectSmokeResult(t, e)
	errs := checkSmokeRun(t, e, result)

	// Report all errors found.
	for _, errMsg := range errs {
		t.Errorf("smoke check failed: %s", errMsg)
	}
}
