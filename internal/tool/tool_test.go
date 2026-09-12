package tool

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

type fakeTool struct {
	name   string
	invoke func(context.Context, json.RawMessage) Result
}

func (f fakeTool) Name() string            { return f.name }
func (f fakeTool) Description() string     { return "fake" }
func (f fakeTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (f fakeTool) Invoke(ctx context.Context, raw json.RawMessage) Result {
	return f.invoke(ctx, raw)
}

// TestPanickingToolDoesNotEscape is the whole reason Invoke has a recover: one
// malformed tool must not take down the TUI.
func TestPanickingToolDoesNotEscape(t *testing.T) {
	r := NewRegistry()
	r.Register(fakeTool{name: "boom", invoke: func(context.Context, json.RawMessage) Result {
		panic("deliberate")
	}})

	res := r.Invoke(context.Background(), "boom", nil)
	if res.OK {
		t.Fatal("a panicking tool reported success")
	}
	if res.Error == nil || res.Error.Kind != KindInternal {
		t.Fatalf("kind = %v, want %s", res.Error, KindInternal)
	}
	if !strings.Contains(res.Error.Message, "deliberate") {
		t.Errorf("message loses the panic value: %q", res.Error.Message)
	}
	if !strings.Contains(res.Error.Message, "boom") {
		t.Errorf("message does not name the tool: %q", res.Error.Message)
	}
}

// TestUnknownToolSelfCorrects: from M3 this message is what a model reads to
// fix its own mistake, so it must list what is actually available.
func TestUnknownToolSelfCorrects(t *testing.T) {
	r := NewRegistry()
	r.Register(fakeTool{name: "read_file", invoke: okInvoke})
	r.Register(fakeTool{name: "list_files", invoke: okInvoke})

	res := r.Invoke(context.Background(), "reed_file", nil)
	if res.Error == nil || res.Error.Kind != KindToolInputInvalid {
		t.Fatalf("kind = %v, want %s", res.Error, KindToolInputInvalid)
	}
	for _, want := range []string{"reed_file", "read_file", "list_files"} {
		if !strings.Contains(res.Error.Message, want) {
			t.Errorf("message does not mention %q: %q", want, res.Error.Message)
		}
	}
}

func okInvoke(context.Context, json.RawMessage) Result {
	return OKResult("content", "summary", false)
}

func TestDurationMeasuredByRegistry(t *testing.T) {
	r := NewRegistry()
	r.Register(fakeTool{name: "t", invoke: okInvoke})
	res := r.Invoke(context.Background(), "t", nil)
	if res.DurationMS < 0 {
		t.Errorf("DurationMS = %d", res.DurationMS)
	}
	// Also set on the failure path, which is easy to forget.
	res = r.Invoke(context.Background(), "missing", nil)
	if res.DurationMS < 0 {
		t.Errorf("failed call has DurationMS = %d", res.DurationMS)
	}
}

func TestCancelledBeforeStart(t *testing.T) {
	r := NewRegistry()
	r.Register(fakeTool{name: "t", invoke: okInvoke})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res := r.Invoke(ctx, "t", nil)
	if res.Error == nil || res.Error.Kind != KindCancelled {
		t.Fatalf("kind = %v, want %s", res.Error, KindCancelled)
	}
}

func TestListIsDeterministic(t *testing.T) {
	r := NewRegistry()
	for _, n := range []string{"zebra", "alpha", "middle"} {
		r.Register(fakeTool{name: n, invoke: okInvoke})
	}
	want := []string{"alpha", "middle", "zebra"}
	for i := 0; i < 10; i++ {
		got := r.Names()
		for j := range want {
			if got[j] != want[j] {
				t.Fatalf("List order = %v, want %v", got, want)
			}
		}
	}
}

func TestDecodeInputRejectsUnknownFields(t *testing.T) {
	var in struct {
		Path string `json:"path"`
	}
	err := DecodeInput(json.RawMessage(`{"path":"a.txt","pathh":"typo"}`), &in)
	if err == nil {
		t.Fatal("unknown field accepted")
	}
	if err.Kind != KindToolInputInvalid {
		t.Errorf("kind = %s", err.Kind)
	}
	if !strings.Contains(err.Message, "pathh") {
		t.Errorf("message does not name the offending field: %q", err.Message)
	}
	if !strings.Contains(err.Message, "schema") {
		t.Errorf("message does not point at the schema: %q", err.Message)
	}
}

func TestDecodeInputTypeErrorNamesField(t *testing.T) {
	var in struct {
		MaxDepth int `json:"max_depth"`
	}
	err := DecodeInput(json.RawMessage(`{"max_depth":"two"}`), &in)
	if err == nil {
		t.Fatal("string accepted for an int field")
	}
	if !strings.Contains(err.Message, "max_depth") {
		t.Errorf("message does not name the field: %q", err.Message)
	}
}

func TestDecodeInputEmptyIsValid(t *testing.T) {
	var in struct {
		Path string `json:"path"`
	}
	if err := DecodeInput(nil, &in); err != nil {
		t.Errorf("empty input should decode to defaults: %v", err)
	}
}

// TestTruncateNeverSplitsARune: an invalid byte sequence corrupts the terminal,
// and the failure presents as a rendering bug rather than a truncation one.
func TestTruncateNeverSplitsARune(t *testing.T) {
	inputs := []string{
		strings.Repeat("日本語テキスト", 500),
		strings.Repeat("🚀", 800),
		strings.Repeat("é", 900),
		strings.Repeat("ascii line\n", 900),
	}
	for _, in := range inputs {
		for _, max := range []int{16, 100, 1001, 4096} {
			got, truncated := Truncate(in, max)
			if !utf8.ValidString(got) {
				t.Fatalf("truncating to %d produced invalid UTF-8", max)
			}
			if !truncated && len(in) > max {
				t.Errorf("truncated flag false for an over-long input")
			}
			if len(got) > max+64 { // marker overhead
				t.Errorf("result is %d bytes for a %d cap", len(got), max)
			}
		}
	}
}

func TestTruncateKeepsHeadAndTail(t *testing.T) {
	var b strings.Builder
	b.WriteString("FIRST-LINE\n")
	for i := 0; i < 5000; i++ {
		b.WriteString("filler line\n")
	}
	b.WriteString("LAST-LINE\n")

	got, truncated := Truncate(b.String(), 2000)
	if !truncated {
		t.Fatal("not truncated")
	}
	if !strings.Contains(got, "FIRST-LINE") {
		t.Error("head lost; the interesting part of a long log is often at the start")
	}
	if !strings.Contains(got, "LAST-LINE") {
		t.Error("tail lost; the interesting part of a long log is often at the end")
	}
	if !strings.Contains(got, "truncated") {
		t.Error("no truncation marker")
	}
}

func TestTruncateShortInputUntouched(t *testing.T) {
	in := "short\n"
	got, truncated := Truncate(in, 1000)
	if truncated || got != in {
		t.Errorf("short input was altered: %q", got)
	}
}
