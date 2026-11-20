package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/djm56/kirsch/internal/session"
	"github.com/djm56/kirsch/internal/tui"
)

// TestResumeSetsSourceOnAssistantText resumes a stored session through
// session.Resume and a.transcriptToItems and checks the rebuilt assistant block:
// Source is the stored text exactly as the model sent it, markdown source and
// escape sequence included, while Lines is the sanitised rendering. The user
// block beside it is the control: it is built from the same log and has no
// Source, so a Source on every block would fail here.
func TestResumeSetsSourceOnAssistantText(t *testing.T) {
	a := testApp(t, "repo-small", newCollector(-1))
	defer a.Close()

	const reply = "Here:\n\n```go\nx := 1\n```\n\x1b[31mred\x1b[0m"
	path := filepath.Join(t.TempDir(), "session.jsonl")
	log := `{"v":1,"type":"user.message","content":"hi"}` + "\n" +
		`{"v":1,"type":"assistant.message","content":"Here:\n\n` + "```go\\nx := 1\\n```" + `\n\u001b[31mred\u001b[0m"}` + "\n"
	if err := os.WriteFile(path, []byte(log), 0o600); err != nil {
		t.Fatal(err)
	}
	_, tr, err := session.Resume(path)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}

	var user, assistant *tui.TextBlock
	for _, it := range a.transcriptToItems(tr) {
		switch it.Kind {
		case tui.KindUser:
			user = it.Text
		case tui.KindAssistant:
			assistant = it.Text
		default: // tool cards and notices are not under test
		}
	}
	if user == nil || assistant == nil {
		t.Fatalf("setup: user block %v, assistant block %v", user, assistant)
	}
	if assistant.Source != reply {
		t.Errorf("assistant Source = %q, want the stored text %q", assistant.Source, reply)
	}
	if strings.Contains(strings.Join(assistant.Lines, "\n"), "\x1b") {
		t.Errorf("assistant Lines %q hold an escape; Lines must stay sanitised", assistant.Lines)
	}
	if user.Source != "" {
		t.Errorf("user Source = %q, want none: only assistant replies are copied", user.Source)
	}
}
