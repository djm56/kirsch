package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/djm56/kirsch/internal/tool"
)

// The shape of a read_file modal's lines in the benchmark: source lines of
// modalBenchShort bytes, and every modalBenchLongEvery-th one modalBenchLong bytes
// (a minified bundle or a data row) that wraps onto many display rows.
const (
	modalBenchShort     = 100
	modalBenchLong      = 2000
	modalBenchLongEvery = 10
)

// modalBenchLine is line i of a read_file modal's content, in the shape the
// constants above give.
func modalBenchLine(i int) string {
	w := modalBenchShort
	if i%modalBenchLongEvery == 0 {
		w = modalBenchLong
	}
	return hlBenchLine(fmt.Sprintf("line %d: x := compute(a, \"s\", 42) + y // c ", i), w)
}

// modalBenchContent is the first n lines modalBenchLine makes.
func modalBenchContent(n int) []string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = modalBenchLine(i)
	}
	return lines
}

// modalBenchContentBytes is modalBenchLine's lines, as many as fit in size bytes
// with a newline after each, the way they would sit in a file, and the number of
// bytes they take. It stops at the first line that would pass size, so the bytes
// fall short of size by at most modalBenchLong.
func modalBenchContentBytes(size int) (lines []string, used int) {
	for i := 0; ; i++ {
		l := modalBenchLine(i)
		if used+len(l)+1 > size {
			return lines, used
		}
		lines = append(lines, l)
		used += len(l) + 1
	}
}

// modalBenchDiff is /diff output in the shape git prints it: file sections of a
// header block and then hunks, about 80 bytes a code line, grown until it holds
// about size bytes. size is /diff's cap (tool.MaxDiffBytes) for the largest case.
func modalBenchDiff(size int) []string {
	var lines []string
	total := 0
	add := func(s string) {
		lines = append(lines, s)
		total += len(s) + 1
	}
	for f := 0; total < size; f++ {
		add(fmt.Sprintf("diff --git a/pkg/f%d.go b/pkg/f%d.go", f, f))
		add("index 1111111..2222222 100644")
		add(fmt.Sprintf("--- a/pkg/f%d.go", f))
		add(fmt.Sprintf("+++ b/pkg/f%d.go", f))
		for h := 0; h < 20 && total < size; h++ {
			add(fmt.Sprintf("@@ -%d,7 +%d,8 @@ func compute()", h*20+1, h*20+1))
			add(" " + hlBenchLine("	ctx := context.Background() // unchanged ", 80))
			add("-" + hlBenchLine("	old := compute(a, \"s\", 42) + y // removed ", 80))
			add("+" + hlBenchLine("	next := compute(a, \"s\", 43) + y // added ", 80))
			add("+" + hlBenchLine("	extra := compute(a, \"t\", 44) + y // added ", 400))
			add(" " + hlBenchLine("	return ctx, old, next // unchanged ", 80))
		}
	}
	return lines
}

// modalBenchModel is a Model at w x 40 with a modal open on lines. The sizing is
// run once before the model is returned so a /diff modal's highlighting, which
// the first modalSizing works out and keeps, is not in the timed loop.
func modalBenchModel(b *testing.B, w int, md ModalState) Model {
	b.Helper()
	m := New(Options{
		Version: fixtureVersion,
		Caps:    Caps{Colour: true, Unicode: true},
	})
	m = drive(m, tea.WindowSizeMsg{Width: w, Height: 40})
	m.openModal(md)
	m.modalSizing(m.layout())
	return m
}

// BenchmarkModalSizing times modalSizing, which keyModal reaches through
// modalExtent on every scroll key and modalBox reaches on every View frame, and
// times View with the modal open. Each case runs three ways. "cold" empties the
// row-count cache before every call, so it is the cost of the first frame at a
// width and of the first one after a resize. "warm" is a call with the cache
// filled, which is every later frame and every scroll key at that width. "view"
// is a whole View frame, warm.
//
// The cases are the largest the producers allow. read_file: its default is 500
// lines (tool.DefaultMaxLines), the first case, and an explicit range is capped by
// bytes and not by lines (tool.MaxFileBytes), the second, which modalBenchContentBytes
// fills with the same lines up to that cap; the case fails if the lines it made
// pass the cap or stop short of it by more than modalBenchLong bytes. /diff:
// tool.MaxDiffBytes, 200 KB. The widths are 80 and 200 columns.
//
// Measurements and decision (review 7 W1 of mission-20261121-01, which found no
// cache and no measurement). With no cache, on an Apple M4, modalSizing took about
// 9 ms (500 lines, 80 columns), about 136 ms (2 MB, 80 columns) and about 12 ms
// (/diff, 80 columns), and every View frame and every scroll key paid it. With the
// counts cached on the ModalState, a warm call took about 0.2 µs and allocated
// nothing at each case and width. The decision was to cache the counts, so that a
// scroll key or a frame at an unchanged width no longer wraps every line. No
// threshold was set in advance; the decision rests on those figures. The no-cache
// figures were measured once, on the code as it was then and with the 2 MB case
// sized by a hand-kept average line length; "cold" is the cache-miss path today and
// prints its own. Run it with:
//
//	go test -run '^$' -bench BenchmarkModalSizing -benchmem ./internal/tui
func BenchmarkModalSizing(b *testing.B) {
	cases := []struct {
		name string
		md   func() ModalState
	}{
		{"read_file-500-lines", func() ModalState {
			return ModalState{Kind: ModalContent, Title: "read_file a.go", ReadFile: true, Path: "a.go", Lines: modalBenchContent(500)}
		}},
		{"read_file-2MB", func() ModalState {
			lines, used := modalBenchContentBytes(tool.MaxFileBytes)
			if used > tool.MaxFileBytes || tool.MaxFileBytes-used > modalBenchLong {
				b.Fatalf("read_file-2MB: %d lines take %d bytes, want at most the %d-byte cap and no more than %d short of it", len(lines), used, tool.MaxFileBytes, modalBenchLong)
			}
			b.Logf("read_file-2MB: %d lines, %d bytes of the %d-byte cap", len(lines), used, tool.MaxFileBytes)
			return ModalState{Kind: ModalContent, Title: "read_file a.go", ReadFile: true, Path: "a.go", Lines: lines}
		}},
		{"diff-200KB", func() ModalState {
			return ModalState{Kind: ModalDiff, Title: "working tree", WorkTree: true, Lines: modalBenchDiff(200 << 10)}
		}},
	}
	for _, c := range cases {
		for _, w := range []int{80, 200} {
			m := modalBenchModel(b, w, c.md())
			lay := m.layout()
			name := fmt.Sprintf("%s/%dcols", c.name, w)
			b.Run("cold/"+name, func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					m.modal.rc = rowCache{}
					m.modalSizing(lay)
				}
			})
			m.modalSizing(lay) // refill what cold emptied
			b.Run("warm/"+name, func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					m.modalSizing(lay)
				}
			})
			b.Run("view/"+name, func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					if !strings.Contains(m.View(), "close") {
						b.Fatal("the frame has no modal footer")
					}
				}
			})
		}
	}
}
