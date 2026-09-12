// Package tool defines the tool contract and registry.
//
// Every tool returns the same envelope. Per-tool result shapes were rejected
// deliberately: the model sees one structure whatever it calls, and the TUI has
// one card renderer rather than one per tool.
package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Kind is a tool error kind. plan §3.3.
type Kind string

// The full plan §3.3 set. Kinds beyond M1's five are declared here because the
// set is the contract; adding them later means the model has seen an
// incomplete vocabulary in the interim.
const (
	KindWorkspaceViolation Kind = "workspace_violation"
	KindPolicyDenied       Kind = "policy_denied"
	KindToolInputInvalid   Kind = "tool_input_invalid"
	KindFileNotFound       Kind = "file_not_found"
	KindFileTooLarge       Kind = "file_too_large"
	KindBinaryFile         Kind = "binary_file"
	KindPatchConflict      Kind = "patch_conflict"
	KindCommandTimeout     Kind = "command_timeout"
	KindCommandFailed      Kind = "command_failed"
	KindCancelled          Kind = "cancelled"
	KindProviderError      Kind = "provider_error"
	KindContextOverflow    Kind = "context_overflow"
	KindMaxTurnsExceeded   Kind = "max_turns_exceeded"
	KindInternal           Kind = "internal_error"
)

// Error is a tool failure. It is data returned to the model, not a Go error
// travelling up a stack — a failing tool is information, never a crash.
type Error struct {
	Kind    Kind   `json:"kind"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return string(e.Kind) + ": " + e.Message }

// Result is the uniform tool envelope. plan §3.
type Result struct {
	OK             bool
	Content        string // what the model sees
	DisplaySummary string // the collapsed card line in the UI
	Truncated      bool
	Error          *Error
	DurationMS     int64
}

// Fail builds a failed result.
func Fail(kind Kind, format string, args ...any) Result {
	msg := fmt.Sprintf(format, args...)
	return Result{
		OK:             false,
		Error:          &Error{Kind: kind, Message: msg},
		DisplaySummary: msg,
	}
}

// OK builds a successful result.
func OKResult(content, summary string, truncated bool) Result {
	return Result{OK: true, Content: content, DisplaySummary: summary, Truncated: truncated}
}

// Tool is one callable capability.
//
// Invoke never returns an error and never panics: the registry recovers panics
// and converts them, so one malformed tool cannot take down the TUI.
type Tool interface {
	Name() string
	Description() string
	Schema() json.RawMessage
	Invoke(ctx context.Context, raw json.RawMessage) Result
}

// Registry holds the available tools.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry { return &Registry{tools: map[string]Tool{}} }

// Register adds a tool, replacing any of the same name.
func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

// Get returns a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// List returns every tool in deterministic name order.
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

// Invoke runs a tool by name, timing it and containing its failures.
//
// This wrapper is the only place tools are called. It measures duration so no
// tool has to, converts a panic into a result so a bug in one tool cannot kill
// the program, and turns an unknown name into tool_input_invalid so a model
// that hallucinates a tool can read the error and correct itself.
func (r *Registry) Invoke(ctx context.Context, name string, raw json.RawMessage) (res Result) {
	start := time.Now()
	defer func() {
		if rec := recover(); rec != nil {
			res = Fail(KindInternal, "tool %q panicked: %v", name, rec)
		}
		res.DurationMS = time.Since(start).Milliseconds()
	}()

	t, ok := r.Get(name)
	if !ok {
		return Fail(KindToolInputInvalid,
			"unknown tool %q; available tools are: %s", name, strings.Join(r.Names(), ", "))
	}
	if err := ctx.Err(); err != nil {
		return Fail(KindCancelled, "cancelled before %s started", name)
	}
	return t.Invoke(ctx, raw)
}

// Names returns the registered tool names, sorted.
func (r *Registry) Names() []string {
	tools := r.List()
	out := make([]string, len(tools))
	for i, t := range tools {
		out[i] = t.Name()
	}
	return out
}

// DecodeInput unmarshals tool input strictly.
//
// Unknown fields are rejected rather than ignored, and the message names the
// offending field: from M3 this text is what lets the model correct itself, so
// it is written for a reader rather than as a status code.
func DecodeInput(raw json.RawMessage, dst any) *Error {
	if len(raw) == 0 {
		raw = json.RawMessage("{}")
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return &Error{
			Kind:    KindToolInputInvalid,
			Message: humaniseDecodeError(err),
		}
	}
	return nil
}

func humaniseDecodeError(err error) string {
	msg := err.Error()
	if strings.Contains(msg, "unknown field") {
		return msg + " — check the tool's schema for the accepted fields"
	}
	var typeErr *json.UnmarshalTypeError
	if ok := asTypeError(err, &typeErr); ok {
		return fmt.Sprintf("field %q expects %s, got %s",
			typeErr.Field, typeErr.Type.String(), typeErr.Value)
	}
	return msg
}

func asTypeError(err error, dst **json.UnmarshalTypeError) bool {
	t, ok := err.(*json.UnmarshalTypeError)
	if ok {
		*dst = t
	}
	return ok
}

// Truncate cuts s to maxBytes, preferring a line boundary and always
// respecting rune boundaries.
//
// Cutting mid-rune emits an invalid byte sequence, which corrupts the terminal
// — the failure looks like a rendering bug and is diagnosed as anything but a
// truncation. For large output the head and tail are both kept, because the
// interesting part of a long build log is usually at one end or the other.
func Truncate(s string, maxBytes int) (string, bool) {
	if maxBytes <= 0 || len(s) <= maxBytes {
		return s, false
	}
	const marker = "\n\n‹… output truncated …›\n\n"
	budget := maxBytes - len(marker)
	if budget < 16 {
		return safeCut(s, maxBytes), true
	}
	head := budget * 2 / 3
	tail := budget - head
	return safeCut(s, head) + marker + safeCutTail(s, tail), true
}

// safeCut returns the first n bytes of s, backing off to the previous line
// boundary where one is close, and never splitting a rune.
func safeCut(s string, n int) string {
	if n >= len(s) {
		return s
	}
	cut := n
	if idx := strings.LastIndexByte(s[:cut], '\n'); idx > cut-200 && idx > 0 {
		return s[:idx]
	}
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}

// safeCutTail returns the last n bytes of s under the same rules.
func safeCutTail(s string, n int) string {
	if n >= len(s) {
		return s
	}
	cut := len(s) - n
	if idx := strings.IndexByte(s[cut:], '\n'); idx >= 0 && idx < 200 {
		return s[cut+idx+1:]
	}
	for cut < len(s) && !utf8.RuneStart(s[cut]) {
		cut++
	}
	return s[cut:]
}
