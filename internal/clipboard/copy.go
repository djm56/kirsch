package clipboard

import (
	"encoding/base64"
	"io"
	"strconv"
	"time"
	"unicode/utf8"
)

// DefaultTimeout is how long the local clipboard tool gets when Options.Timeout
// is not positive. ui-spec §6: 3 seconds.
const DefaultTimeout = 3 * time.Second

// Options are the routes Copy tries. A nil route is simply not tried.
type Options struct {
	// WriteLocal hands the text to the host's clipboard tool. Production passes
	// WriteAll from github.com/atotto/clipboard. It runs in its own goroutine.
	WriteLocal func(string) error
	// OSC is the terminal output the OSC 52 sequence is written to, in one
	// Write call. Nil means no OSC 52 route.
	OSC io.Writer
	// Timeout is how long WriteLocal gets before its route counts as failed.
	// Zero or negative means DefaultTimeout.
	Timeout time.Duration
}

// Result is what Copy achieved.
type Result struct {
	Chars int  // code points in the text, not bytes
	Local bool // the local tool returned without error inside the timeout
	OSC   bool // the OSC 52 sequence was written without error
}

// Copy sends text by every route in opts and reports which worked.
//
// The local tool runs in a goroutine; the OSC 52 write happens while it runs, so
// a slow tool does not hold the terminal write back. If the tool has not
// returned when the timeout expires, the route counts as failed and Copy
// returns. The goroutine is left to finish on its own, because a stuck external
// process cannot be cancelled from here; its error goes to a buffered channel so
// it never blocks.
//
// OSC 52 is written as exactly one Write of ESC ] 52 ; c ; <base64 of text> BEL.
// A successful write says the bytes left the program, not that the terminal
// acted on them.
//
// The order is: start the local tool, write OSC 52, then wait up to the timeout
// for the tool. The timer starts after the OSC write returns, so a stalled
// terminal write adds to the total. Copy can therefore block for the write time
// plus the timeout, so callers run it in a tea.Cmd, never on the update loop.
func Copy(text string, opts Options) Result {
	res := Result{Chars: utf8.RuneCountInString(text)}

	var localDone chan error
	if opts.WriteLocal != nil {
		localDone = make(chan error, 1)
		go func() { localDone <- opts.WriteLocal(text) }()
	}

	if opts.OSC != nil {
		seq := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\a"
		if _, err := opts.OSC.Write([]byte(seq)); err == nil {
			res.OSC = true
		}
	}

	if localDone != nil {
		timeout := opts.Timeout
		if timeout <= 0 {
			timeout = DefaultTimeout
		}
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		select {
		case err := <-localDone:
			res.Local = err == nil
		case <-timer.C:
		}
	}
	return res
}

// Notice returns the composer hint for r. ui-spec §6, notice table.
func Notice(r Result) string {
	c := strconv.Itoa(r.Chars)
	switch {
	case r.Local:
		return "copied " + c + " characters"
	case r.OSC:
		return "sent " + c + " characters (OSC 52)"
	default:
		return "copy failed: no clipboard route"
	}
}
