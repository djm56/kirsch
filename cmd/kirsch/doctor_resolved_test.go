package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/doctor"
	"github.com/djm56/kirsch/internal/provider"
)

// These tests drive run() with "doctor" (through doctorRun), so they cover what
// main passes in (os.Getenv for the key line, exec.LookPath for the git and rg
// presence probes, the config paths) and what the report prints for the key,
// backend, model and project-context lines. The backend line takes nothing from
// main: tool.Backend reads the process PATH itself.

// setEnv sets name to value for the test, or unsets it when value is empty. The
// previous value is restored when the test ends.
func setEnv(t *testing.T, name, value string) {
	t.Helper()
	t.Setenv(name, "") // registers the restore
	if value == "" {
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
		return
	}
	t.Setenv(name, value)
}

// writeUserConfig writes body as the global config under cfgHome.
func writeUserConfig(t *testing.T, cfgHome, body string) {
	t.Helper()
	path := filepath.Join(cfgHome, "kirsch", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// lineStarting returns the first stdout line beginning with prefix, or "".
func lineStarting(out, prefix string) string {
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, prefix) {
			return l
		}
	}
	return ""
}

// TestRunDoctorKeyLine sets KIRSCH_OPENCODE_API_KEY and OPENCODE_API_KEY each
// set and unset (all four combinations) and asserts the key line names the
// winner, or says none, and that no value appears in stdout or stderr. The
// failure messages print only the expected line or fixed text: not the output,
// not a line of it, not run's error, not a value. Positive control: the same leak check, with a probe that
// deliberately prints a value, does flag it.
func TestRunDoctorKeyLine(t *testing.T) {
	repo := gitRepoOrSkip(t)
	isolatedEnv(t)
	const v1, v2 = "run-level-value-aaa-111", "run-level-value-bbb-222"
	leaks := func(s string) bool { return strings.Contains(s, v1) || strings.Contains(s, v2) }

	for _, tc := range []struct {
		name, prefixed, plain, want string
	}{
		{"neither", "", "", "warn  key: endpoint opencode: no key found (set KIRSCH_OPENCODE_API_KEY or OPENCODE_API_KEY)"},
		{"prefixed only", v1, "", "ok  key: endpoint opencode: key found in KIRSCH_OPENCODE_API_KEY\n"},
		{"plain only", "", v2, "ok  key: endpoint opencode: key found in OPENCODE_API_KEY\n"},
		{"both", v1, v2, "ok  key: endpoint opencode: key found in KIRSCH_OPENCODE_API_KEY\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setEnv(t, "KIRSCH_OPENCODE_API_KEY", tc.prefixed)
			setEnv(t, "OPENCODE_API_KEY", tc.plain)
			out, errOut, err := doctorRun(t, "--workspace", repo)
			if err != nil {
				t.Fatal("run doctor returned an error, want nil (a missing key only warns; error withheld: a key variable may be set)")
			}
			if !strings.Contains(out, tc.want) {
				t.Errorf("stdout lacks the expected key line %q (actual output withheld: a key variable may be set)", tc.want)
			}
			if leaks(out) || leaks(errOut) {
				t.Error("a key value reached doctor's output (value withheld from this message)")
			}
		})
	}

	t.Run("control: the leak check sees a printed value", func(t *testing.T) {
		orig := doctorProbes
		t.Cleanup(func() { doctorProbes = orig })
		doctorProbes = func(doctor.Options) doctor.Outcome {
			return doctor.Outcome{Results: []doctor.Result{{Status: doctor.StatusOK, Name: "key", Detail: "found " + v1}}}
		}
		out, _, _ := doctorRun(t)
		if !leaks(out) {
			t.Fatal("control: the leak check did not see a value that doctor printed")
		}
	})
}

// TestRunDoctorModelLine: the default model prints its endpoint, id and window;
// a model id not in the table prints "not in the table" and the conservative
// fallback window, and run still returns nil.
func TestRunDoctorModelLine(t *testing.T) {
	repo := gitRepoOrSkip(t)
	cfgHome := isolatedEnv(t)

	// Expected lines come from the built-in default and the model table.
	cfg := config.Defaults()
	epName := cfg.Provider.Default
	info := provider.LookupModel(cfg.Provider.Endpoints[epName].Model)
	if !info.Known {
		t.Fatal("control: the default model is not in the table")
	}
	wantDefault := fmt.Sprintf("ok  model: endpoint %s, model %s: context window %d tokens\n", epName, info.ID, info.ContextWindow)
	ctl, _, err := doctorRun(t, "--workspace", repo)
	if err != nil || !strings.Contains(ctl, wantDefault) {
		t.Fatalf("control: stdout lacks %q (err %v):\n%s", wantDefault, err, ctl)
	}

	writeUserConfig(t, cfgHome, "[provider."+epName+"]\nmodel = \"no-such-model\"\n")
	out, _, err := doctorRun(t, "--workspace", repo)
	if err != nil {
		t.Fatalf("run doctor = %v, want nil (an unknown model only warns)\n%s", err, out)
	}
	want := fmt.Sprintf("warn  model: endpoint %s, model no-such-model: not in the table; conservative fallback, context window %d tokens\n",
		epName, provider.ConservativeBudget())
	if !strings.Contains(out, want) {
		t.Errorf("stdout lacks %q:\n%s", want, out)
	}
}

// TestRunDoctorContextLine: with AGENTS.md in the workspace the context line
// names it and its size; without it the line says none. The control pair is the
// same repository before and after the file is written.
func TestRunDoctorContextLine(t *testing.T) {
	repo := gitRepoOrSkip(t)
	isolatedEnv(t)

	out, _, err := doctorRun(t, "--workspace", repo)
	if err != nil {
		t.Fatalf("run doctor = %v\n%s", err, out)
	}
	if l := lineStarting(out, "ok  context: "); !strings.HasPrefix(l, "ok  context: none") {
		t.Errorf("without AGENTS.md the context line = %q, want one starting %q", l, "ok  context: none")
	}

	body := "# rules\nbe brief\n"
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	out, _, err = doctorRun(t, "--workspace", repo)
	if err != nil {
		t.Fatalf("run doctor = %v\n%s", err, out)
	}
	if want := fmt.Sprintf("ok  context: AGENTS.md, %d bytes\n", len(body)); !strings.Contains(out, want) {
		t.Errorf("with AGENTS.md stdout lacks %q:\n%s", want, out)
	}
}

// TestRunDoctorBackendLine: under a PATH shim without rg the backend line reads
// pure-go; after a stub rg is added to the shim it reads ripgrep. The shim keeps
// the rest of PATH (DIR-009). Controls: exec.LookPath agrees with each case.
func TestRunDoctorBackendLine(t *testing.T) {
	repo := gitRepoOrSkip(t)
	isolatedEnv(t)
	shimPathWithout(t, "rg")
	if _, err := exec.LookPath("rg"); err == nil {
		t.Fatal("control: rg found under the shim")
	}
	out, _, err := doctorRun(t, "--workspace", repo)
	if err != nil {
		t.Fatalf("run doctor = %v\n%s", err, out)
	}
	if want := "warn  backend: search_code uses pure-go (rg is not on PATH)\n"; !strings.Contains(out, want) {
		t.Errorf("without rg stdout lacks %q:\n%s", want, out)
	}

	stubDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(stubDir, "rg"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil { //nolint:gosec // a stub executable the test must be able to run
		t.Fatal(err)
	}
	t.Setenv("PATH", stubDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if _, err := exec.LookPath("rg"); err != nil {
		t.Fatalf("control: stub rg not found: %v", err)
	}
	out, _, err = doctorRun(t, "--workspace", repo)
	if err != nil {
		t.Fatalf("run doctor = %v\n%s", err, out)
	}
	if want := "ok  backend: search_code uses ripgrep\n"; !strings.Contains(out, want) {
		t.Errorf("with rg stdout lacks %q:\n%s", want, out)
	}
}

// TestRunDoctorSkippedLinesWhenConfigDidNotLoad: with a config that does not
// parse, the key, model and context lines each read "skipped: config did not
// load" even with a key variable set and AGENTS.md present, and run fails on the
// config line. A key variable is set throughout, so no failure message prints
// the output, a line of it or run's error.
func TestRunDoctorSkippedLinesWhenConfigDidNotLoad(t *testing.T) {
	repo := gitRepoOrSkip(t)
	cfgHome := isolatedEnv(t)
	setEnv(t, "OPENCODE_API_KEY", "skipped-run-level-value")
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("# rules\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	ctl, _, err := doctorRun(t, "--workspace", repo)
	if err != nil || strings.Contains(ctl, "skipped:") {
		t.Fatal("control: good config printed a skipped line or failed (error and output withheld: a key variable is set)")
	}

	writeUserConfig(t, cfgHome, "this is = not [valid toml\n")
	out, errOut, err := doctorRun(t, "--workspace", repo)
	if !errors.Is(err, errDoctorFailed) {
		t.Fatal("run doctor did not return errDoctorFailed (error and output withheld: a key variable is set)")
	}
	for _, name := range []string{"key", "model", "context"} {
		want := "warn  " + name + ": skipped: config did not load\n"
		if !strings.Contains(out, want) {
			t.Errorf("stdout lacks %q (actual output withheld)", want)
		}
	}
	if strings.Contains(out+errOut, "skipped-run-level-value") {
		t.Error("a key value reached doctor's output (value withheld from this message)")
	}
}
