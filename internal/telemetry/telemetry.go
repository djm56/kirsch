// Package telemetry provides Kirsch's structured debug log.
//
// The single most important property here is that nothing is ever written to
// stdout or stderr. Kirsch owns the alternate screen; a stray line of log
// output corrupts the rendering in a way that looks like a TUI bug and is
// diagnosed as anything but a logging mistake. There is a test that captures
// both streams during a logging burst and asserts they stay empty.
package telemetry

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// MaxLogBytes caps the debug log. There is no rotation in v0.1: at this size
// the file is truncated and a single warning is logged.
const MaxLogBytes = 10 << 20

// MaxContentBytes caps any file content that reaches the log. Redaction is
// enforced here rather than trusted to call sites.
const MaxContentBytes = 256

// Logger wraps slog with Kirsch's redaction rules and file destination.
//
// The zero value is usable and discards everything, so callers never nil-check.
type Logger struct {
	mu      sync.Mutex
	log     *slog.Logger
	file    *os.File
	written int64
	enabled bool
	warned  bool
}

// Disabled returns a logger that formats nothing. Used when --debug is absent,
// so the hot path costs a boolean test rather than a discarded format.
func Disabled() *Logger {
	return &Logger{log: slog.New(slog.NewJSONHandler(io.Discard, nil))}
}

// Options configures New. Paths are supplied by the caller; nothing here reads
// the environment, matching internal/config.
type Options struct {
	Path    string // full path to the log file
	Enabled bool
}

// New opens the debug log. A logger that cannot open its file is still usable
// — it degrades to discarding rather than failing startup, because losing the
// debug log is never worth refusing to run.
func New(o Options) (*Logger, error) {
	if !o.Enabled || o.Path == "" {
		return Disabled(), nil
	}
	// 0700, not 0755: the debug log carries truncated file content and tool
	// summaries from the user's repository. It is their data, and nobody
	// else's business. gosec G301.
	if err := os.MkdirAll(filepath.Dir(o.Path), 0o700); err != nil {
		return Disabled(), err
	}
	f, err := os.OpenFile(o.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return Disabled(), err
	}
	st, _ := f.Stat()
	l := &Logger{
		file:    f,
		enabled: true,
		written: st.Size(),
	}
	l.log = slog.New(slog.NewJSONHandler(&sizeCounter{l: l, w: f}, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	return l, nil
}

// Enabled reports whether anything is actually written.
func (l *Logger) Enabled() bool { return l != nil && l.enabled }

// Close releases the log file.
func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

// Path returns the log file's location, or "" when disabled.
func (l *Logger) Path() string {
	if l == nil || l.file == nil {
		return ""
	}
	return l.file.Name()
}

// Debug logs at debug level.
func (l *Logger) Debug(msg string, args ...any) { l.emit(slog.LevelDebug, msg, args...) }

// Info logs at info level.
func (l *Logger) Info(msg string, args ...any) { l.emit(slog.LevelInfo, msg, args...) }

// Warn logs at warn level.
func (l *Logger) Warn(msg string, args ...any) { l.emit(slog.LevelWarn, msg, args...) }

// Error logs at error level.
func (l *Logger) Error(msg string, args ...any) { l.emit(slog.LevelError, msg, args...) }

func (l *Logger) emit(level slog.Level, msg string, args ...any) {
	if l == nil || !l.enabled {
		return
	}
	l.log.Log(nil, level, msg, redactArgs(args)...) //nolint:staticcheck // nil ctx is fine here
}

// Content returns a log attribute for file or tool content, truncated to
// MaxContentBytes.
//
// A helper rather than a convention: the rule "never log a full tool Content
// field" cannot be enforced at every call site, so the only way to log content
// is through something that has already truncated it.
func Content(key, body string) slog.Attr {
	if len(body) > MaxContentBytes {
		body = body[:MaxContentBytes] + "…(truncated)"
	}
	return slog.String(key, body)
}

// EnvName returns a log attribute naming an environment variable without its
// value. Logging the value is never correct — that is how a key ends up in a
// file the user later pastes into a bug report.
func EnvName(key, name string) slog.Attr {
	return slog.String(key, name+"=<redacted>")
}

// redactArgs is the backstop for the two rules above: any attribute whose key
// looks like a credential has its value replaced, whatever the call site did.
func redactArgs(args []any) []any {
	out := make([]any, len(args))
	copy(out, args)
	for i := 0; i+1 < len(out); i += 2 {
		key, ok := out[i].(string)
		if !ok {
			continue
		}
		if looksSecret(key) {
			out[i+1] = "<redacted>"
		}
	}
	for i, a := range out {
		if attr, ok := a.(slog.Attr); ok && looksSecret(attr.Key) {
			out[i] = slog.String(attr.Key, "<redacted>")
		}
	}
	return out
}

func looksSecret(key string) bool {
	k := strings.ToLower(key)
	for _, needle := range []string{"api_key", "apikey", "token", "secret", "password", "env_value"} {
		if strings.Contains(k, needle) {
			return true
		}
	}
	return false
}

// sizeCounter enforces the 10MB cap. At the limit the file is truncated once
// and a single warning recorded, rather than growing without bound.
type sizeCounter struct {
	l *Logger
	w *os.File
}

func (s *sizeCounter) Write(p []byte) (int, error) {
	s.l.mu.Lock()
	defer s.l.mu.Unlock()

	if s.l.written+int64(len(p)) > MaxLogBytes {
		if !s.l.warned {
			s.l.warned = true
			_ = s.w.Truncate(0)
			if _, err := s.w.Seek(0, io.SeekStart); err == nil {
				_, _ = s.w.WriteString(
					`{"level":"WARN","msg":"debug log reached 10MB and was truncated; no rotation in v0.1"}` + "\n")
			}
			s.l.written = 0
		}
	}
	n, err := s.w.Write(p)
	s.l.written += int64(n)
	return n, err
}

// DefaultPath returns the debug log location for the given environment.
// Resolved by the caller, never inside New.
func DefaultPath(xdgStateHome, home string) string {
	if xdgStateHome != "" {
		return filepath.Join(xdgStateHome, "kirsch", "debug.log")
	}
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".local", "state", "kirsch", "debug.log")
}
