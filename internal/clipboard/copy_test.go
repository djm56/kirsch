package clipboard

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"
)

// countingWriter records every Write call, so a test can tell one write of the
// whole sequence from several writes of its pieces.
type countingWriter struct {
	calls [][]byte
	err   error
}

func (w *countingWriter) Write(p []byte) (int, error) {
	w.calls = append(w.calls, append([]byte(nil), p...))
	return len(p), w.err
}

func localOK(string) error   { return nil }
func localFail(string) error { return errors.New("no tool") }

// TestCopyRoutes covers each route outcome and the notice it produces.
//
// The "local times out" row uses a tool that never returns inside a short
// timeout; its OSC write succeeds, so the notice is the OSC one.
func TestCopyRoutes(t *testing.T) {
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	hangs := func(string) error { <-release; return nil }

	const short, long = 50 * time.Millisecond, time.Second
	cases := []struct {
		name      string
		local     func(string) error
		osc       bool
		timeout   time.Duration
		wantLocal bool
		wantOSC   bool
		wantNote  string
	}{
		{"local ok, no OSC", localOK, false, long, true, false, "copied 5 characters"},
		{"local ok and OSC both written", localOK, true, long, true, true, "copied 5 characters"},
		{"local fails, OSC written", localFail, true, long, false, true, "sent 5 characters (OSC 52)"},
		{"local fails, no OSC writer", localFail, false, long, false, false, "copy failed: no clipboard route"},
		{"no local tool, OSC written", nil, true, long, false, true, "sent 5 characters (OSC 52)"},
		{"local times out, OSC written", hangs, true, short, false, true, "sent 5 characters (OSC 52)"},
		{"no routes at all", nil, false, long, false, false, "copy failed: no clipboard route"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := &countingWriter{}
			opts := Options{WriteLocal: c.local, Timeout: c.timeout}
			if c.osc {
				opts.OSC = w
			}
			got := Copy("héllo", opts) // 5 code points, 6 bytes
			if got.Local != c.wantLocal || got.OSC != c.wantOSC || got.Chars != 5 {
				t.Errorf("Copy = %+v, want Chars 5 Local %v OSC %v", got, c.wantLocal, c.wantOSC)
			}
			if n := Notice(got); n != c.wantNote {
				t.Errorf("Notice = %q, want %q", n, c.wantNote)
			}
		})
	}
}

// TestCopyOSCBytesAndSingleWrite checks the exact sequence and that it is one
// Write call, with a counting writer.
func TestCopyOSCBytesAndSingleWrite(t *testing.T) {
	w := &countingWriter{}
	text := "line one\nline two ✓"
	Copy(text, Options{WriteLocal: localFail, OSC: w, Timeout: time.Second})
	if len(w.calls) != 1 {
		t.Fatalf("OSC writer saw %d Write calls, want 1", len(w.calls))
	}
	want := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\a"
	if got := string(w.calls[0]); got != want {
		t.Errorf("OSC bytes = %q, want %q", got, want)
	}
	// A literal for the same sequence, so the expected value is not computed
	// with the code under test's own recipe.
	if got, lit := string(w.calls[0]), "\x1b]52;c;bGluZSBvbmUKbGluZSB0d28g4pyT\a"; got != lit {
		t.Errorf("OSC bytes = %q, want literal %q", got, lit)
	}
}

// TestCopyOSCWriteErrorIsNotSent: a writer that fails is not a route.
func TestCopyOSCWriteErrorIsNotSent(t *testing.T) {
	w := &countingWriter{err: errors.New("closed")}
	got := Copy("x", Options{WriteLocal: localFail, OSC: w, Timeout: time.Second})
	if got.OSC {
		t.Errorf("OSC = true after the writer returned an error")
	}
	if len(w.calls) != 1 {
		t.Errorf("writer saw %d calls, want 1 (the attempt)", len(w.calls))
	}
}

// TestCopyLocalTimeout: a tool that blocks longer than the injected timeout
// counts as failed, and Copy returns without waiting for it. The control is the
// same call with a tool that returns at once.
func TestCopyLocalTimeout(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	blocking := func(string) error { <-release; return nil }

	start := time.Now()
	got := Copy("x", Options{WriteLocal: blocking, Timeout: 50 * time.Millisecond})
	elapsed := time.Since(start)
	if got.Local {
		t.Errorf("a tool that never returned counted as a success")
	}
	if elapsed < 40*time.Millisecond || elapsed > 2*time.Second {
		t.Errorf("Copy took %v with a 50ms timeout", elapsed)
	}
	if ctl := Copy("x", Options{WriteLocal: localOK, Timeout: 50 * time.Millisecond}); !ctl.Local {
		t.Errorf("control: a tool that returns at once did not count as a success")
	}
}

// TestCopyNonPositiveTimeoutUsesDefault: a Timeout of zero or below gives the
// local tool DefaultTimeout. A tool that answers after 200ms counts as a success
// with Timeout 0 and with Timeout -1, which a literal zero or negative timer
// would not allow; the control is the same tool with an explicit 50ms timeout,
// which must fail. A tool that never returns, with Timeout 0, holds Copy for
// DefaultTimeout and no longer.
func TestCopyNonPositiveTimeoutUsesDefault(t *testing.T) {
	slow := func(string) error { time.Sleep(200 * time.Millisecond); return nil }
	for _, d := range []time.Duration{0, -1} {
		if got := Copy("x", Options{WriteLocal: slow, Timeout: d}); !got.Local {
			t.Errorf("Timeout %v: a tool answering after 200ms counted as failed", d)
		}
	}
	if ctl := Copy("x", Options{WriteLocal: slow, Timeout: 50 * time.Millisecond}); ctl.Local {
		t.Errorf("control: a tool answering after 200ms succeeded inside a 50ms timeout")
	}

	release := make(chan struct{})
	defer close(release)
	hangs := func(string) error { <-release; return nil }
	start := time.Now()
	got := Copy("x", Options{WriteLocal: hangs, Timeout: 0})
	elapsed := time.Since(start)
	if got.Local {
		t.Errorf("a tool that never returned counted as a success")
	}
	if elapsed < DefaultTimeout-100*time.Millisecond || elapsed > DefaultTimeout+2*time.Second {
		t.Errorf("Copy with Timeout 0 took %v, want about DefaultTimeout (%v)", elapsed, DefaultTimeout)
	}
}

// TestCopyCountsCodePoints: Chars is code points of the text, not bytes.
func TestCopyCountsCodePoints(t *testing.T) {
	cases := []struct {
		text string
		want int
	}{
		{"", 0},
		{"abc", 3},
		{"é", 1},    // 2 bytes
		{"✓✓", 2},   // 3 bytes each
		{"😀", 1},    // 4 bytes
		{"a😀é✓", 4}, // 1+4+2+3 = 10 bytes
	}
	for _, c := range cases {
		if got := Copy(c.text, Options{}).Chars; got != c.want {
			t.Errorf("Copy(%q).Chars = %d, want %d", c.text, got, c.want)
		}
	}
}

// TestNoticeTexts pins the three result notices byte for byte.
func TestNoticeTexts(t *testing.T) {
	if got, want := Notice(Result{Chars: 12, Local: true, OSC: true}), "copied 12 characters"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := Notice(Result{Chars: 12, OSC: true}), "sent 12 characters (OSC 52)"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := Notice(Result{Chars: 12}), "copy failed: no clipboard route"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
