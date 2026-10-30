package doctor

import (
	"bytes"
	"strings"
	"testing"
)

// TestFailed covers the exit rule: all-ok and warn-only are not failures; any
// fail is, wherever it sits in the list.
func TestFailed(t *testing.T) {
	ok := Result{StatusOK, "a", "fine"}
	warn := Result{StatusWarn, "b", "meh"}
	fail := Result{StatusFail, "c", "broken"}
	tests := []struct {
		name string
		in   []Result
		want bool
	}{
		{"none", nil, false},
		{"all ok", []Result{ok, ok}, false},
		{"warn present", []Result{ok, warn, ok}, false},
		{"fail present", []Result{ok, fail, warn}, true},
		{"fail first", []Result{fail, ok}, true},
		// An unknown status must not fail open (DIR-002).
		{"unknown status alone", []Result{{Status(9), "d", "?"}}, true},
		{"unknown status among ok", []Result{ok, {Status(-1), "d", "?"}, warn}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Failed(tt.in); got != tt.want {
				t.Errorf("Failed = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestReportFormat pins the exact bytes: version line, one line per result with
// two spaces after the status, and the summary line.
func TestReportFormat(t *testing.T) {
	var buf bytes.Buffer
	err := Report(&buf, "1.2.3", []Result{
		{StatusOK, "git", "found"},
		{StatusWarn, "rg", "not on PATH"},
		{StatusFail, "config", "bad"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "kirsch 1.2.3\n" +
		"ok  git: found\n" +
		"warn  rg: not on PATH\n" +
		"fail  config: bad\n" +
		"doctor: 1 ok, 1 warn, 1 fail\n"
	if buf.String() != want {
		t.Errorf("report =\n%q\nwant\n%q", buf.String(), want)
	}
}

// TestReportNoResults is the skeleton case: version line and a zero summary.
func TestReportNoResults(t *testing.T) {
	var buf bytes.Buffer
	if err := Report(&buf, "9", nil); err != nil {
		t.Fatal(err)
	}
	if want := "kirsch 9\ndoctor: 0 ok, 0 warn, 0 fail\n"; buf.String() != want {
		t.Errorf("report = %q, want %q", buf.String(), want)
	}
}

// TestReportSummaryCounts checks the summary against differing counts per
// status, so a swapped or off-by-one counter shows up.
func TestReportSummaryCounts(t *testing.T) {
	var results []Result
	for i := 0; i < 3; i++ {
		results = append(results, Result{StatusOK, "o", "d"})
	}
	for i := 0; i < 2; i++ {
		results = append(results, Result{StatusWarn, "w", "d"})
	}
	results = append(results, Result{StatusFail, "f", "d"})
	var buf bytes.Buffer
	if err := Report(&buf, "v", results); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if got, want := lines[len(lines)-1], "doctor: 3 ok, 2 warn, 1 fail"; got != want {
		t.Errorf("summary = %q, want %q", got, want)
	}
	if len(lines) != 1+len(results)+1 {
		t.Errorf("got %d lines, want %d", len(lines), 1+len(results)+1)
	}
}

// TestStatusString pins the words, including the unknown fallback.
func TestStatusString(t *testing.T) {
	for s, want := range map[Status]string{StatusOK: "ok", StatusWarn: "warn", StatusFail: "fail", Status(9): "unknown"} {
		if got := s.String(); got != want {
			t.Errorf("Status(%d) = %q, want %q", int(s), got, want)
		}
	}
}

// TestReportCountsUnknownStatusAsFail: a result with an out-of-range status is
// printed with the word "unknown" and counted under fail, so the summary agrees
// with Failed. The control is the known statuses, counted as before.
func TestReportCountsUnknownStatusAsFail(t *testing.T) {
	results := []Result{
		{StatusOK, "a", "fine"},
		{StatusWarn, "b", "meh"},
		{Status(9), "c", "odd"},
	}
	var buf bytes.Buffer
	if err := Report(&buf, "v", results); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"unknown  c: odd\n", "doctor: 1 ok, 1 warn, 1 fail\n"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("report lacks %q:\n%s", want, buf.String())
		}
	}
	if !Failed(results) {
		t.Error("Failed = false with an unknown status")
	}
}

// TestReportKeepsEachResultOnOneLine feeds details (and a name) that span lines
// in every break style and checks the report keeps the grammar: one line per
// result, pieces joined with "; ", empty pieces dropped.
func TestReportKeepsEachResultOnOneLine(t *testing.T) {
	var buf bytes.Buffer
	err := Report(&buf, "v", []Result{
		{StatusFail, "config", "first\n\nsecond\r\nthird\rfourth\n"},
		{StatusWarn, "na\nme", "x"},
		{StatusOK, "plain", "unchanged detail"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "kirsch v\n" +
		"fail  config: first; second; third; fourth\n" +
		"warn  na; me: x\n" +
		"ok  plain: unchanged detail\n" +
		"doctor: 1 ok, 1 warn, 1 fail\n"
	if buf.String() != want {
		t.Errorf("report =\n%q\nwant\n%q", buf.String(), want)
	}
}
