package clipboard

import (
	"reflect"
	"testing"
)

// TestCodeBlocks runs the §6 fence grammar through CodeBlocks, one rule per
// case. Each case that says a fence does not open or close has a sibling that
// says it does, so a CodeBlocks that found nothing, or everything, fails one of
// them.
func TestCodeBlocks(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"backtick fence", "a\n```go\nx\ny\n```\nb\n", []string{"x\ny"}},
		{"tilde fence", "~~~\nx\n~~~\n", []string{"x"}},
		{"no fence", "plain\ntext\n", nil},
		{"indented 4 spaces in a list", "- item\n\n    ```\n    code\n      more\n    ```\n", []string{"code\n  more"}},
		{"closing fence longer than the opener closes", "```\nx\n`````\ny\n", []string{"x"}},
		{"closing fence shorter than the opener does not close", "````\nx\n```\ny\n````\n", []string{"x\n```\ny"}},
		{"closing fence of the other character does not close", "```\nx\n~~~\ny\n```\n", []string{"x\n~~~\ny"}},
		{"tilde fence is not closed by backticks", "~~~\nx\n```\n~~~\n", []string{"x\n```"}},
		{"backtick opener with a backtick in the info string is not a fence", "```a`b\nx\n```\n", []string{""}},
		{"tilde opener may have a backtick in the info string", "~~~a`b\nx\n~~~\n", []string{"x"}},
		{"tab-indented opener strips the tab from content", "\t```\n\tx\n\t\ty\n\t```\n", []string{"x\n\ty"}},
		{"tab straddling the cut becomes spaces past it", "  ```\n\tx\n  ```\n", []string{"  x"}},
		{"tab ending exactly at the cut is removed whole", "    ```\n\tx\n    ```\n", []string{"x"}},
		{"unclosed fence runs to the end", "```\nx\ny", []string{"x\ny"}},
		{"unclosed fence plus trailing newline adds no empty line", "```\nx\ny\n", []string{"x\ny"}},
		{"unclosed fence, blank line before trailing newline keeps one empty line", "```\nx\n\n", []string{"x\n"}},
		{"block with no content lines", "```\n```\n", []string{""}},
		{"block whose only line is empty", "```\n\n```\n", []string{""}},
		{"block of spaces is not empty", "```\n  \n```\n", []string{"  "}},
		{"several blocks in order", "```\na\n```\ntext\n~~~\nb\n~~~\n```\n\n```\n", []string{"a", "b", ""}},
		{"closing fence followed by a tab closes", "```\nx\n```\t\ny\n", []string{"x"}},
		{"closing fence followed by spaces closes", "```\nx\n```   \ny\n", []string{"x"}},
		{"closing fence followed by text does not close", "```\nx\n```y\nz\n```\n", []string{"x\n```y\nz"}},
		{"closing fence indented deeper than the opener closes", "```\nx\n      ```\n", []string{"x"}},
		{"two backticks are not a fence", "``\nx\n``\n", nil},
		{"fence after a non-space character on the line is not a fence", "a ```\nx\n", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := CodeBlocks(c.in)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("CodeBlocks(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
