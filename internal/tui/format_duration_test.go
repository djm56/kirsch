package tui

import (
	"strings"
	"testing"
	"time"
)

// TestFormatDurationBoundaries: entry point formatDuration, called directly. A duration
// above zero and under a millisecond reads "<1ms", one millisecond reads "1ms", a
// duration under a second reads whole milliseconds and a second or more reads seconds to
// one decimal. The sub-millisecond rows are the ones a "0ms" rendering would fail.
func TestFormatDurationBoundaries(t *testing.T) {
	for _, c := range []struct {
		d    time.Duration
		want string
	}{
		{1 * time.Nanosecond, "<1ms"},
		{time.Millisecond - time.Nanosecond, "<1ms"},
		{time.Millisecond, "1ms"},
		{999 * time.Millisecond, "999ms"},
		{time.Second, "1.0s"},
		{2400 * time.Millisecond, "2.4s"},
	} {
		if got := formatDuration(c.d); got != c.want {
			t.Errorf("formatDuration(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

// TestToolCardDrawsSubMillisecondDurationAndNothingForZero: entry point Model.Update with
// a ToolStartedMsg and a ToolCompletedMsg, then View. A completion whose Elapsed is half a
// millisecond draws "<1ms" on the card head; the control, the same messages with Elapsed
// of exactly one millisecond, draws "1ms" on that head, so the head can show a duration; a
// zero Elapsed draws neither.
func TestToolCardDrawsSubMillisecondDurationAndNothingForZero(t *testing.T) {
	head := func(elapsed time.Duration) string {
		m := sizedModel(t, 100, 24, false)
		m = drive(m,
			ToolStartedMsg{ID: 1, Name: "apply_patch", Target: "x.go"},
			ToolCompletedMsg{ID: 1, Name: "apply_patch", Target: "x.go", OK: true, Summary: "done", Elapsed: elapsed})
		for _, line := range strings.Split(m.View(), "\n") {
			if strings.Contains(line, "apply_patch") {
				return line
			}
		}
		t.Fatalf("no apply_patch head in the frame:\n%s", m.View())
		return ""
	}
	if h := head(500 * time.Microsecond); !strings.Contains(h, " · <1ms") {
		t.Errorf("a half-millisecond card head draws no \"<1ms\": %q", h)
	}
	if h := head(time.Millisecond); !strings.Contains(h, " · 1ms") {
		t.Errorf("control: a one-millisecond card head draws no \"1ms\": %q", h)
	}
	if h := head(0); strings.Contains(h, "ms") || strings.Contains(h, "<1") {
		t.Errorf("a zero-duration card head draws a duration: %q", h)
	}
}
