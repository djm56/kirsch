#!/usr/bin/env bash
# Count Markdown checkboxes per section, straight from the tick marks.
#
#   bash scripts/plan-status.sh FILE...
#
# Why: status counts typed by hand drift from the ticks they describe. This
# builds the counts from the ticks each time it runs, so nothing needs
# maintaining and nothing it prints is committed. The rules are in
# plan/adr/0009-planning-layout-from-m6.md, Decision item 3.
#
# What counts as a checkbox line: optional indentation, `- ` or `* `, `[`, a
# marker of 1 to 4 UTF-8 characters, `]`. A `]` followed straight by `(` is a
# link, so `- [ab](url)` is not a box. A longer bracket such as `- [Benefit 1]`
# is not a box either.
#
# Markers (exact byte strings):
#   [ ]        open
#   [x] [✅]   done
#   [❌]       failed
#   any other  unrecognised (reported, never dropped)
#
# Lines inside a code fence (``` or ~~~) are ignored, headings included.
# Boxes group under the nearest preceding `## ` heading outside a fence;
# `### ` and deeper count under their `## `. Boxes before the first `## `
# group under (top). Sections with no boxes are not printed.
#
# Output, stdout, tab separated, files in argument order. For each file:
#   <file> <section> open=N done=N failed=N unrecognised=N   one per section
#   <file> TOTAL     open=N done=N failed=N unrecognised=N   sum for the file
#   failed <file>:<line>                                     one per failed box
#   unrecognised <file>:<line> [<marker>]                    one per such box
#   unclosed-fence <file>:<line>                             fence never closed
#
# Exit status: 0 when every file was read, whatever the counts. 2 for no
# arguments, a directory, or a file that is missing or unreadable; in that
# case nothing is printed to stdout. The script writes no files.
#
# Two `## ` lines with the same text are two separate sections, each printed on
# its own line with its own counts. They are never merged by name.
#
# Locale: LC_ALL=C is exported below, and the export is what protects the
# comparisons. Without it, awk's `==` is collation-dependent: on macOS awk
# 20200816 under en_ZA.UTF-8 or en_US.UTF-8, "✅" == "❌" is true, so the
# marker tests would fold the two. Counting is done in awk with no sort or
# uniq, because under a UTF-8 locale `sort | uniq -c` folds [✅] and [❌] into
# one row as well. The output is therefore the same in every locale.
#
# Deliberately dependency-free: bash and awk only.
set -euo pipefail
export LC_ALL=C

# Print a usage message on stderr and exit 2.
#   $1  what was wrong
usage_error() {
  echo "plan-status: $1" >&2
  echo "usage: plan-status.sh FILE..." >&2
  exit 2
}

# The awk program that reads one file on stdin. PS_FILE (environment) carries
# the file name used in output, so no quoting of the name reaches awk's parser.
IFS= read -r -d '' AWK_PROGRAM <<'EOF' || true
# Length of s in UTF-8 characters: the byte length minus the continuation bytes
# (0x80-0xBF). This assumes length() counts bytes. That was tested on macOS awk
# 20200816, where length("✅") is 3 under C, en_ZA.UTF-8 and en_US.UTF-8. An awk
# that counts characters would subtract too much; the lengths case in
# plan-status-test.sh is expected to fail on such an awk (not tried on one).
function nchars(s,   t, cont) {
  t = s
  cont = gsub(/[\200-\277]/, "", t)
  return length(s) - cont
}

# Number of leading characters of s that equal the single character c.
function run_length(s, c,   n) {
  n = 0
  while (substr(s, n + 1, 1) == c) n++
  return n
}

BEGIN {
  file = ENVIRON["PS_FILE"]
  seq = 0          # section number; 0 is (top), each "## " heading adds one
  name[0] = "(top)"
  infence = 0
}

{
  line = $0
  sub(/\r$/, "", line)
  body = line
  sub(/^[ \t]+/, "", body)
  c = substr(body, 1, 1)
  run = 0
  if (c == "`" || c == "~") run = run_length(body, c)

  # Inside a fence nothing counts. It closes on the same character, at least
  # as many of them as the opener, with nothing else on the line.
  if (infence) {
    if (c == fch && run >= flen && substr(body, run + 1) ~ /^[ \t]*$/) infence = 0
    next
  }
  if (run >= 3) {
    infence = 1
    fch = c
    flen = run
    fline = NR
    next
  }

  if (substr(line, 1, 3) == "## ") {
    h = substr(line, 4)
    sub(/[ \t]+$/, "", h)
    name[++seq] = h
    next
  }

  if (match(line, /^[ \t]*[-*] \[/)) {
    rest = substr(line, RLENGTH + 1)
    close_at = index(rest, "]")
    if (close_at < 2) next                       # no "]", or an empty marker
    marker = substr(rest, 1, close_at - 1)
    if (substr(rest, close_at + 1, 1) == "(") next   # a link, not a box
    if (nchars(marker) > 4) next                 # a longer bracket, not a box

    boxes[seq]++
    if (marker == " ") {
      open[seq]++
    } else if (marker == "x" || marker == "✅") {
      done[seq]++
    } else if (marker == "❌") {
      failed[seq]++
      flist = flist "failed\t" file ":" NR "\n"
    } else {
      unrec[seq]++
      ulist = ulist "unrecognised\t" file ":" NR "\t[" marker "]\n"
    }
  }
}

END {
  for (s = 0; s <= seq; s++) {
    if (boxes[s] == 0) continue
    printf "%s\t%s\topen=%d\tdone=%d\tfailed=%d\tunrecognised=%d\n", \
      file, name[s], open[s], done[s], failed[s], unrec[s]
    to += open[s]; td += done[s]; tf += failed[s]; tu += unrec[s]
  }
  printf "%s\tTOTAL\topen=%d\tdone=%d\tfailed=%d\tunrecognised=%d\n", \
    file, to, td, tf, tu
  printf "%s%s", flist, ulist
  if (infence) printf "unclosed-fence\t%s:%d\n", file, fline
}
EOF

[ "$#" -ge 1 ] || usage_error "no files given"

# Check every argument before printing anything, so a bad one gives no partial
# output.
for f in "$@"; do
  [ ! -d "$f" ] || usage_error "$f is a directory"
  { [ -f "$f" ] && [ -r "$f" ]; } || usage_error "cannot read $f"
done

for f in "$@"; do
  PS_FILE="$f" awk "$AWK_PROGRAM" < "$f"
done
