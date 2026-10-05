package anthropic

import (
	"fmt"
	"strings"
	"testing"
	"unicode"
)

// liveModelEnv names the environment variable that overrides the default model
// for the live smoke test.
const liveModelEnv = "KIRSCH_LIVE_MODEL"

// liveModel resolves the model name for the live smoke test. lookup follows
// os.LookupEnv's signature and is called once, for KIRSCH_LIVE_MODEL; def is
// the model returned when the variable is unset.
//
// Unset: returns def, the source "default" and a nil error.
//
// Set and empty: returns "", "" and an error that names KIRSCH_LIVE_MODEL,
// says it is empty, and shows the value with %q.
//
// Set and containing any rune for which unicode.IsSpace or unicode.IsControl
// reports true: returns "", "" and an error that names KIRSCH_LIVE_MODEL, says
// it contains invalid characters, and shows the value with %q.
//
// Otherwise: returns the value verbatim (no trimming), the source
// "KIRSCH_LIVE_MODEL" and a nil error. An invalid value never falls back to def.
func liveModel(lookup func(string) (string, bool), def string) (model, source string, err error) {
	val, ok := lookup(liveModelEnv)
	if !ok {
		return def, "default", nil
	}

	if val == "" {
		return "", "", fmt.Errorf("%s is empty: %q", liveModelEnv, val)
	}

	for _, r := range val {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return "", "", fmt.Errorf("%s contains invalid characters: %q", liveModelEnv, val)
		}
	}

	return val, liveModelEnv, nil
}

// TestLiveModel calls liveModel directly with a closure-backed fake lookup; it
// never reads the real environment and does not exercise the live request path
// (TestLiveOpencodeSmoke, its key and skip handling, or the adapter). Each
// rejection subtest is chosen so that one predicate of liveModel alone decides
// it, as its name says: "space-only" inputs are rejected by unicode.IsSpace
// and not by unicode.IsControl, "control-only" inputs by unicode.IsControl and
// not by unicode.IsSpace, and "space-and-control" inputs by both.
func TestLiveModel(t *testing.T) {
	const (
		testDefault   = "test-default-model"
		reasonEmpty   = "is empty"
		reasonInvalid = "contains invalid characters"
	)
	tests := []struct {
		name       string
		value      string
		set        bool
		wantModel  string
		wantSrc    string
		wantReason string // substring of the error; "" means no error expected
	}{
		{name: "unset-returns-default", set: false, wantModel: testDefault, wantSrc: "default"},
		{name: "valid-value-with-dot-hyphen-digits", value: "qwen3.7-plus", set: true, wantModel: "qwen3.7-plus", wantSrc: liveModelEnv},
		{name: "empty-value", value: "", set: true, wantReason: reasonEmpty},
		{name: "space-only-leading-ascii-space", value: " qwen3.7-plus", set: true, wantReason: reasonInvalid},
		{name: "space-only-interior-ascii-space", value: "qwen 3.7", set: true, wantReason: reasonInvalid},
		{name: "space-only-interior-no-break-space", value: "qwen\u00a03.7", set: true, wantReason: reasonInvalid},
		{name: "control-only-interior-nul", value: "qwen\x003.7", set: true, wantReason: reasonInvalid},
		{name: "control-only-interior-escape", value: "qwen\x1b[0m", set: true, wantReason: reasonInvalid},
		{name: "space-and-control-trailing-newline", value: "qwen3.7-plus\n", set: true, wantReason: reasonInvalid},
		{name: "space-and-control-interior-tab", value: "qwen\t3.7", set: true, wantReason: reasonInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookup := func(key string) (string, bool) {
				if key == liveModelEnv && tt.set {
					return tt.value, true
				}
				return "", false
			}

			got, src, err := liveModel(lookup, testDefault)

			if tt.wantReason == "" {
				if err != nil {
					t.Errorf("%s: unexpected error: %v", tt.name, err)
				}
			} else if err == nil {
				t.Errorf("%s: expected an error containing %q, got nil", tt.name, tt.wantReason)
			} else {
				msg := err.Error()
				if !strings.Contains(msg, liveModelEnv) {
					t.Errorf("%s: error does not name the variable %q: %q", tt.name, liveModelEnv, msg)
				}
				if want := fmt.Sprintf("%q", tt.value); !strings.Contains(msg, want) {
					t.Errorf("%s: error does not show the value as %%q (%s): %q", tt.name, want, msg)
				}
				if !strings.Contains(msg, tt.wantReason) {
					t.Errorf("%s: error does not give the reason %q: %q", tt.name, tt.wantReason, msg)
				}
			}
			if got != tt.wantModel {
				t.Errorf("%s: returned model: got %q, want %q", tt.name, got, tt.wantModel)
			}
			if src != tt.wantSrc {
				t.Errorf("%s: returned source: got %q, want %q", tt.name, src, tt.wantSrc)
			}
		})
	}
}
