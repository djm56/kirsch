// Package doctor implements the report behind `kirsch doctor`.
//
// Doctor only checks and never changes anything: it creates no folders, takes
// no lock and makes no network calls. This package holds the report grammar and
// the pass/fail rule (doctor.go) and the probes (probes.go): git, rg, the
// repository and the config file, then the lines that read the resolved
// configuration: the API key, the model and the project-context file, which the
// loaded config feeds, and the search_code backend, which takes no config. The
// caller hands Run its environment values, then passes the results to Report and
// Failed.
package doctor

import (
	"fmt"
	"io"
	"strings"
)

// Status is the outcome of one check.
type Status int

// The statuses a check can report. The zero value is StatusOK.
const (
	// StatusOK means the check passed.
	StatusOK Status = iota
	// StatusWarn means the check found something worth knowing; it does not
	// make doctor fail.
	StatusWarn
	// StatusFail means the check found a problem; it makes doctor fail.
	StatusFail
)

// String returns the word used in the report: "ok", "warn" or "fail". An
// unknown value renders as "unknown" so a bad Status is visible, not hidden.
func (s Status) String() string {
	switch s {
	case StatusOK:
		return "ok"
	case StatusWarn:
		return "warn"
	case StatusFail:
		return "fail"
	}
	return "unknown"
}

// Result is the outcome of one check: its status, the name of what was
// checked, and a one-line detail.
type Result struct {
	Status Status
	Name   string
	Detail string
}

// Failed reports whether any result is not OK and not Warn. Warnings do not
// count. A Status outside the three known values counts as a failure: a result
// whose meaning is unknown must not let the run pass.
func Failed(results []Result) bool {
	for _, r := range results {
		if r.Status != StatusOK && r.Status != StatusWarn {
			return true
		}
	}
	return false
}

// oneLine collapses s onto a single line. Every line break (LF, CR or CRLF)
// becomes "; ", and the pieces are trimmed with empty ones dropped, so a
// multi-line error cannot split a report line in two.
func oneLine(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '\n' || r == '\r' })
	kept := parts[:0]
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, "; ")
}

// Report writes the doctor report to w and returns the first write error.
//
// The first line is "kirsch <version>". Each result then gets one line,
// "<status>  <name>: <detail>", with two spaces after the status word. A name or
// detail that spans several lines is joined onto one with "; " (see oneLine).
// The last line is "doctor: <n> ok, <n> warn, <n> fail"; a result with an
// unknown status is counted under fail, matching Failed.
func Report(w io.Writer, version string, results []Result) error {
	if _, err := fmt.Fprintf(w, "kirsch %s\n", version); err != nil {
		return err
	}
	var ok, warn, fail int
	for _, r := range results {
		switch r.Status {
		case StatusOK:
			ok++
		case StatusWarn:
			warn++
		default:
			fail++
		}
		if _, err := fmt.Fprintf(w, "%s  %s: %s\n", r.Status, oneLine(r.Name), oneLine(r.Detail)); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "doctor: %d ok, %d warn, %d fail\n", ok, warn, fail)
	return err
}
