package tui

import (
	"os"
	"testing"
)

// Tests for DetectCaps and unicodeLocale. DetectCaps reads the process
// environment, so every test here uses t.Setenv (or os.Unsetenv with a
// restore) and none may call t.Parallel.

// openCharDevice returns /dev/null, a character device, for DetectCaps to Stat.
// DetectCaps reads the *os.File it is handed, not os.Stdout, so this stands in
// for a terminal.
func openCharDevice(t *testing.T) *os.File {
	t.Helper()
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

// TestDetectCapsColour covers the colour half of DetectCaps. The first case is
// the positive control: with TERM=xterm, NO_COLOR unset and a character device
// it must report colour on, or the colour-off cases prove nothing.
func TestDetectCapsColour(t *testing.T) {
	tests := []struct {
		name    string
		noColor *string // nil = unset
		term    string
		pipe    bool
		want    bool
	}{
		{"control: xterm, NO_COLOR unset, char device", nil, "xterm", false, true},
		{"NO_COLOR non-empty", ptr("1"), "xterm", false, false},
		{"NO_COLOR empty is treated as unset (O18)", ptr(""), "xterm", false, true},
		{"TERM dumb", nil, "dumb", false, false},
		{"TERM empty", nil, "", false, false},
		{"stdout is a pipe", nil, "xterm", true, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.noColor == nil {
				unsetEnv(t, "NO_COLOR")
			} else {
				t.Setenv("NO_COLOR", *tc.noColor)
			}
			t.Setenv("TERM", tc.term)
			out := openCharDevice(t)
			if tc.pipe {
				r, w, err := os.Pipe()
				if err != nil {
					t.Fatalf("pipe: %v", err)
				}
				t.Cleanup(func() { r.Close(); w.Close() })
				out = w
			}
			if got := DetectCaps(out).Colour; got != tc.want {
				t.Errorf("Colour = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestDetectCapsNoColourUnsetRestores checks that unsetEnv really removes
// NO_COLOR for the test body and puts the prior value back afterwards. The
// restore is asserted from the parent test once the subtest's cleanups ran.
func TestDetectCapsNoColourUnsetRestores(t *testing.T) {
	t.Setenv("NO_COLOR", "prior") // registers its own restore of the real value
	t.Run("unset", func(t *testing.T) {
		unsetEnv(t, "NO_COLOR")
		if v, ok := os.LookupEnv("NO_COLOR"); ok {
			t.Errorf("NO_COLOR still set to %q after unsetEnv", v)
		}
		t.Setenv("TERM", "xterm")
		if !DetectCaps(openCharDevice(t)).Colour {
			t.Errorf("Colour = false with NO_COLOR truly unset, want true")
		}
	})
	if v, ok := os.LookupEnv("NO_COLOR"); !ok || v != "prior" {
		t.Errorf("NO_COLOR after subtest = %q (set=%v), want %q restored", v, ok, "prior")
	}
}

// TestDetectCapsUnicode covers unicodeLocale through DetectCaps. All three
// variables are set explicitly in every row, so a host value can never leak in.
// The first variable in precedence order that is non-empty decides: LC_ALL, then
// LC_CTYPE, then LANG. Rows where a lower-precedence variable says the opposite
// of a higher one catch a wrong order; rows with only one variable set (LANG,
// LC_ALL, LC_CTYPE, each alone) catch dropping that variable from the list;
// en_US.utf8 is the UTF8-without-hyphen spelling and lowercase "utf-8" needs
// ToUpper; all-empty must be false. Each false row has a true counterpart in
// the table that swaps or supplies the UTF-8 value.
func TestDetectCapsUnicode(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		lcAll, lcCtype, lang string
		want                 bool
	}{
		{"control: LANG=en_US.UTF-8 only", "", "", "en_US.UTF-8", true},
		{"LANG=C only", "", "", "C", false},
		{"LC_ALL=C beats LANG=en_US.UTF-8", "C", "", "en_US.UTF-8", false},
		{"LC_ALL=en_US.UTF-8 beats LANG=C", "en_US.UTF-8", "", "C", true},
		{"LC_CTYPE=en_US.UTF-8 beats LANG=C", "", "en_US.UTF-8", "C", true},
		{"LC_CTYPE=C beats LANG=en_US.UTF-8", "", "C", "en_US.UTF-8", false},
		{"LC_ALL beats LC_CTYPE (C over UTF-8)", "C", "en_US.UTF-8", "", false},
		{"LC_ALL beats LC_CTYPE (UTF-8 over C)", "en_US.UTF-8", "C", "", true},
		{"LC_ALL=en_US.UTF-8 alone", "en_US.UTF-8", "", "", true},
		{"LC_CTYPE=en_US.UTF-8 alone", "", "en_US.UTF-8", "", true},
		{"UTF8 spelling without hyphen", "", "", "en_US.utf8", true},
		{"lowercase utf-8 needs ToUpper", "", "", "en_US.utf-8", true},
		{"all three empty", "", "", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("LC_ALL", tc.lcAll)
			t.Setenv("LC_CTYPE", tc.lcCtype)
			t.Setenv("LANG", tc.lang)
			if got := DetectCaps(openCharDevice(t)).Unicode; got != tc.want {
				t.Errorf("Unicode = %v, want %v", got, tc.want)
			}
		})
	}
}

// ptr returns a pointer to s, to tell "empty" from "unset" in a table.
func ptr(s string) *string { return &s }

// unsetEnv removes key from the environment for the rest of the test and
// registers a cleanup restoring the prior state (value, or unset). t.Setenv
// cannot express unset.
func unsetEnv(t *testing.T, key string) {
	t.Helper()
	prior, had := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset %s: %v", key, err)
	}
	t.Cleanup(func() {
		if had {
			os.Setenv(key, prior)
		} else {
			os.Unsetenv(key)
		}
	})
}
