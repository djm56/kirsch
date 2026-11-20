package main

import (
	"os"
	"testing"

	"github.com/djm56/kirsch/internal/app"
	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

// TestOSCAllowed: OSC 52 goes only to a character device, and never when TERM is
// "dumb". The rows share one device so each condition is the only difference:
// /dev/null is a character device and is the positive control; a pipe is not
// one; TERM=dumb turns the control off.
func TestOSCAllowed(t *testing.T) {
	dev, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	cases := []struct {
		name string
		out  *os.File
		term string
		want bool
	}{
		{"character device, xterm (control)", dev, "xterm-256color", true},
		{"character device, dumb", dev, "dumb", false},
		{"pipe, xterm", w, "xterm-256color", false},
		{"character device, empty TERM (allowed per ui-spec §6)", dev, "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("TERM", c.term)
			if got := oscAllowed(c.out); got != c.want {
				t.Errorf("oscAllowed = %v, want %v", got, c.want)
			}
		})
	}
}

// TestWireCallbacksWiresCopy: the production wiring sets Model.Copy. It is not
// called here, because calling it would write to the host clipboard and the
// test's stdout.
func TestWireCallbacksWiresCopy(t *testing.T) {
	ws, err := workspace.Detect(t.TempDir())
	if err != nil {
		t.Fatalf("workspace.Detect: %v", err)
	}
	log := telemetry.Disabled()
	a := app.New(ws, config.Config{}, log)
	t.Cleanup(func() { a.Close() })

	m := tui.New(tui.Options{Caps: tui.Caps{Colour: false, Unicode: false}})
	if m.Copy != nil {
		t.Fatal("setup: a fresh model already has Copy wired, so this test proves nothing")
	}
	wireCallbacks(&m, a, log)
	if m.Copy == nil {
		t.Error("wireCallbacks left Model.Copy nil")
	}
}
