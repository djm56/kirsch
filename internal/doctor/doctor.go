// Package doctor implements the report behind `kirsch doctor`.
//
// Doctor only checks and never changes anything: it creates no folders, takes
// no lock and makes no network calls. This package holds the report grammar and
// the pass/fail rule. A probe is any code that returns []Result; the caller
// gathers the results from its probes and hands them to Report.
package doctor

import (
	"fmt"
	"io"
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

// Failed reports whether any result has StatusFail. Warnings do not count.
func Failed(results []Result) bool {
	for _, r := range results {
		if r.Status == StatusFail {
			return true
		}
	}
	return false
}

// Report writes the doctor report to w and returns the first write error.
//
// The first line is "kirsch <version>". Each result then gets one line,
// "<status>  <name>: <detail>", with two spaces after the status word. The last
// line is "doctor: <n> ok, <n> warn, <n> fail".
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
		case StatusFail:
			fail++
		}
		if _, err := fmt.Fprintf(w, "%s  %s: %s\n", r.Status, r.Name, r.Detail); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "doctor: %d ok, %d warn, %d fail\n", ok, warn, fail)
	return err
}
