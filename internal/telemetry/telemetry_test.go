package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNeverWritesToStdoutOrStderr is the load-bearing test in this package.
// Kirsch owns the alternate screen; a stray log line corrupts the rendering in
// a way that presents as a TUI bug and gets diagnosed as anything but logging.
func TestNeverWritesToStdoutOrStderr(t *testing.T) {
	dir := t.TempDir()

	outR, outW, _ := os.Pipe()
	errR, errW, _ := os.Pipe()
	origOut, origErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outW, errW
	defer func() { os.Stdout, os.Stderr = origOut, origErr }()

	log, err := New(Options{Path: filepath.Join(dir, "debug.log"), Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 500; i++ {
		log.Debug("burst", "i", i, "detail", strings.Repeat("x", 200))
		log.Info("burst info", "i", i)
		log.Warn("burst warn", "i", i)
		log.Error("burst error", "i", i)
	}
	_ = log.Close()

	_ = outW.Close()
	_ = errW.Close()
	os.Stdout, os.Stderr = origOut, origErr

	var buf [4096]byte
	n, _ := outR.Read(buf[:])
	if n > 0 {
		t.Errorf("wrote %d bytes to stdout: %q", n, buf[:n])
	}
	n, _ = errR.Read(buf[:])
	if n > 0 {
		t.Errorf("wrote %d bytes to stderr: %q", n, buf[:n])
	}
}

func TestEnabledWritesStructuredLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "debug.log")
	log, err := New(Options{Path: path, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	log.Info("tool completed", "tool", "read_file", "duration_ms", 4)
	_ = log.Close()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no log file created: %v", err)
	}
	s := string(body)
	for _, want := range []string{`"msg":"tool completed"`, `"tool":"read_file"`, `"duration_ms":4`} {
		if !strings.Contains(s, want) {
			t.Errorf("log missing %s:\n%s", want, s)
		}
	}
}

func TestDisabledCreatesNoFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "debug.log")
	log, err := New(Options{Path: path, Enabled: false})
	if err != nil {
		t.Fatal(err)
	}
	log.Info("should go nowhere", "k", "v")
	_ = log.Close()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("a disabled logger created a file")
	}
	if log.Enabled() {
		t.Error("Enabled() true for a disabled logger")
	}
}

// TestRedactionAtTheHelper proves the rules hold even when the call site does
// the wrong thing — redaction is enforced in the logger, not trusted to
// callers.
func TestRedactionAtTheHelper(t *testing.T) {
	path := filepath.Join(t.TempDir(), "debug.log")
	log, err := New(Options{Path: path, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}

	const secret = "sk-ant-super-secret-value"
	log.Info("careless call site", "api_key", secret)
	log.Info("careless again", "ANTHROPIC_TOKEN", secret)
	log.Info("and again", "user_password", secret)
	log.Info("env value", "env_value", secret)
	log.Info("proper helper", EnvName("var", "ANTHROPIC_API_KEY"))
	log.Info("content", Content("body", strings.Repeat("A", 5000)))
	_ = log.Close()

	body, _ := os.ReadFile(path)
	s := string(body)

	if strings.Contains(s, secret) {
		t.Error("a secret value reached the log despite the redaction rule")
	}
	if !strings.Contains(s, "<redacted>") {
		t.Error("no redaction marker in the log")
	}
	if strings.Contains(s, strings.Repeat("A", MaxContentBytes+1)) {
		t.Errorf("Content() logged more than %d bytes", MaxContentBytes)
	}
	if !strings.Contains(s, "ANTHROPIC_API_KEY=<redacted>") {
		t.Error("EnvName should log the variable name without its value")
	}
}

func TestNilLoggerIsSafe(t *testing.T) {
	var log *Logger
	log.Info("no panic please", "k", "v")
	if log.Enabled() {
		t.Error("nil logger reports enabled")
	}
	if log.Path() != "" {
		t.Error("nil logger reports a path")
	}
	if err := log.Close(); err != nil {
		t.Errorf("closing a nil logger: %v", err)
	}
}

func TestDefaultPath(t *testing.T) {
	if got := DefaultPath("/state", "/home/me"); got != "/state/kirsch/debug.log" {
		t.Errorf("XDG_STATE_HOME ignored: %q", got)
	}
	if got := DefaultPath("", "/home/me"); got != "/home/me/.local/state/kirsch/debug.log" {
		t.Errorf("home fallback wrong: %q", got)
	}
	if got := DefaultPath("", ""); got != "" {
		t.Errorf("no home should yield no path, got %q", got)
	}
}

func TestSizeCapTruncatesOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "debug.log")
	log, err := New(Options{Path: path, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	log.written = MaxLogBytes - 10 // pretend we are already at the cap
	log.Info("this crosses the limit", "pad", strings.Repeat("y", 100))
	log.Info("and this follows the truncation")
	_ = log.Close()

	body, _ := os.ReadFile(path)
	if !strings.Contains(string(body), "truncated") {
		t.Errorf("no truncation warning:\n%s", body)
	}
	if int64(len(body)) >= MaxLogBytes {
		t.Errorf("log is %d bytes, cap is %d", len(body), MaxLogBytes)
	}
}
