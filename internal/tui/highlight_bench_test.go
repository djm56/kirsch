package tui

import (
	"fmt"
	"strings"
	"testing"
)

// hlBenchLine builds a line of exactly n bytes, all ASCII, by repeating unit and
// cutting the result to n.
func hlBenchLine(unit string, n int) string {
	return strings.Repeat(unit, n/len(unit)+1)[:n]
}

// hlBenchCases are the lines BenchmarkHighlightLine lexes: one ordinary-looking
// line per language the ux-d6 measurement names, and adversarial lines built to
// make chroma's backtracking regular expressions work hard: an unterminated
// string or comment that a rule must scan to the end of the line and then give
// up on.
func hlBenchCases(n int) []struct{ name, path, line string } {
	return []struct{ name, path, line string }{
		{"go", "a.go", hlBenchLine(`x := compute(a, "s", 42) + y // c `, n)},
		{"js-minified", "a.js", hlBenchLine(`var a=function(b,c){return b.map(function(d){return d+c*2})};`, n)},
		{"python", "a.py", hlBenchLine(`def f(a, b=1): return [x for x in a if x > b]  # c `, n)},
		{"json", "a.json", hlBenchLine(`{"key": [1, 2.5, true, null, "v"]}, `, n)},
		{"markdown", "a.md", hlBenchLine("**bold** _it_ `code` [l](http://x) ", n)},
		{"adv-go-unterminated-string", "a.go", `"` + hlBenchLine(`\\a`, n-1)},
		{"adv-js-unterminated-regex", "a.js", "x = /" + hlBenchLine(`[\/\\a`, n-5)},
		{"adv-md-unclosed-emphasis", "a.md", hlBenchLine("*_`[", n)},
		{"adv-py-unterminated-triple", "a.py", `"""` + hlBenchLine(`\"a`, n-3)},
	}
}

// BenchmarkHighlightLine times hlLexNow, the per-line lexing, on one line of
// each case at several lengths. It lexes whatever it is given: it calls
// hlLexNow, which neither the line-length cutoff nor hlLexDeadline and its
// breaker gate, so the cost above the cutoff is measured too, and so is the full
// cost of an adversarial line that hlLexLine would give up on. Run it with:
//
//	go test -run '^$' -bench BenchmarkHighlightLine ./internal/tui
func BenchmarkHighlightLine(b *testing.B) {
	for _, n := range []int{256, 512, 1024, 2000} {
		for _, c := range hlBenchCases(n) {
			lx := hlLexer(hlPath, c.path)
			if lx == nil {
				b.Fatalf("%s: no lexer for %s", c.name, c.path)
			}
			b.Run(fmt.Sprintf("%s/%d", c.name, n), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					hlLexNow(lx, c.line)
				}
			})
		}
	}
}
