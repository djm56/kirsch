package clipboard

import "strings"

// tabStop is the width a tab advances to, for measuring indentation. ui-spec §6.
const tabStop = 4

// minFence is the shortest run of fence characters that opens or closes a block.
const minFence = 3

// fence is an opening fence found by parseOpen.
type fence struct {
	char   byte // '`' or '~'
	length int  // length of the opening run
	indent int  // indentation of the opening line, in columns
}

// CodeBlocks returns the copied text of every fenced code block in cleaned, in
// order. Empty blocks are included as empty strings, so the index of a block is
// its number minus one. ui-spec §6, "Code blocks".
//
// cleaned must already have been through Clean: fences are found after
// cleaning, not in the source, so it holds only LF line ends.
//
// A fence opens on a line whose first characters that are not spaces or tabs
// are three or more backticks or tildes, at any indentation. A backtick fence
// whose info string contains a backtick does not open. A block closes on a line
// of the same character, at least as long as the opener, indented or not and
// followed only by spaces or tabs; a block still open at the end runs to the
// end of the text. A content line loses leading spaces and tabs up to the
// opener's indentation, measured in columns with a tab advancing to the next
// multiple of 4; a tab that straddles that cut becomes spaces for the columns
// past it. The copied text is the content lines joined with LF: no fence lines,
// no info string, no trailing newline. The newline that ends the text does not
// add an empty content line to an unclosed block.
func CodeBlocks(cleaned string) []string {
	lines := strings.Split(cleaned, "\n")
	// A text that ends in a newline splits into one trailing empty piece that
	// is not a line.
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}

	var blocks []string
	var open *fence
	var content []string
	for _, line := range lines {
		if open == nil {
			if f, ok := parseOpen(line); ok {
				open = &f
				content = nil
			}
			continue
		}
		if closesFence(line, *open) {
			blocks = append(blocks, strings.Join(content, "\n"))
			open = nil
			continue
		}
		content = append(content, stripIndent(line, open.indent))
	}
	if open != nil {
		blocks = append(blocks, strings.Join(content, "\n"))
	}
	return blocks
}

// leadingWidth returns the number of leading space and tab bytes in line and
// the column they reach, a tab advancing to the next multiple of tabStop.
func leadingWidth(line string) (n, col int) {
	for n < len(line) {
		switch line[n] {
		case ' ':
			col++
		case '\t':
			col += tabStop - col%tabStop
		default:
			return n, col
		}
		n++
	}
	return n, col
}

// fenceRun returns the length of the run of ch at the start of s.
func fenceRun(s string, ch byte) int {
	n := 0
	for n < len(s) && s[n] == ch {
		n++
	}
	return n
}

// parseOpen reports whether line opens a fence, and which.
func parseOpen(line string) (fence, bool) {
	n, col := leadingWidth(line)
	rest := line[n:]
	if rest == "" || (rest[0] != '`' && rest[0] != '~') {
		return fence{}, false
	}
	ch := rest[0]
	run := fenceRun(rest, ch)
	if run < minFence {
		return fence{}, false
	}
	if ch == '`' && strings.Contains(rest[run:], "`") {
		return fence{}, false // a backtick in the info string: inline code, not a fence
	}
	return fence{char: ch, length: run, indent: col}, true
}

// closesFence reports whether line closes the fence f.
func closesFence(line string, f fence) bool {
	n, _ := leadingWidth(line)
	rest := line[n:]
	if rest == "" || rest[0] != f.char {
		return false
	}
	run := fenceRun(rest, f.char)
	if run < f.length {
		return false
	}
	return strings.Trim(rest[run:], " \t") == ""
}

// stripIndent removes leading spaces and tabs from line until indent columns
// are gone. A tab that straddles the cut is replaced by spaces for the columns
// past it, and the rest of the line is kept as it is.
func stripIndent(line string, indent int) string {
	col, i := 0, 0
	for i < len(line) && col < indent {
		switch line[i] {
		case ' ':
			col++
		case '\t':
			next := col + tabStop - col%tabStop
			if next > indent {
				return strings.Repeat(" ", next-indent) + line[i+1:]
			}
			col = next
		default:
			return line[i:]
		}
		i++
	}
	return line[i:]
}
