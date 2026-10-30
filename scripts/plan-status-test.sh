#!/usr/bin/env bash
# Tests for scripts/plan-status.sh.
#
#   bash scripts/plan-status-test.sh
#   PLAN_STATUS_BIN=path/to/copy.sh bash scripts/plan-status-test.sh
#   PLAN_STATUS_LOCALES="en_ZA.UTF-8 xx_XX.UTF-8" bash scripts/plan-status-test.sh
#
# PLAN_STATUS_BIN names the script under test, default scripts/plan-status.sh.
# Pointing it at a deliberately broken copy shows which case catches which
# fault. PLAN_STATUS_LOCALES lists the locales for the locale cases, default
# "en_ZA.UTF-8 en_US.UTF-8".
#
# Uses only the committed fixtures in scripts/testdata/plan-status/. It writes
# no files. Expected output is built here by hand (the sec/tot/... helpers),
# never by running the script under test.
#
# Runs from the repository root so file names in the output are the same
# relative paths the expectations use. Prints PASS, FAIL or SKIP per case, then
# one line with the passed, failed and skipped counts. Exits non-zero if any
# case fails, or if no case ran at all. A SKIP (a locale the machine does not
# have) is not a pass and does not by itself fail the run.
#
# Not tested:
#   - The unreadable-file branch of the script (a file that exists but cannot be
#     read). A committed fixture cannot carry mode 000 and this test writes no
#     files, so nothing here exercises it. Removing the `-r` check from the
#     script is not caught.
#   - CRLF line endings. A committed CRLF fixture would depend on the git
#     settings of whoever checks it out.
set -uo pipefail

cd "$(dirname "$0")/.."
BIN="${PLAN_STATUS_BIN:-scripts/plan-status.sh}"
D=scripts/testdata/plan-status
LOCALES="${PLAN_STATUS_LOCALES:-en_ZA.UTF-8 en_US.UTF-8}"

passed=0
failed=0
skipped=0

# Expected output helpers. The tab-separated layout is fixed here, not in the
# script under test.

# One section line.  $1 file  $2 section  $3 open  $4 done  $5 failed  $6 unrecognised
sec() { printf '%s\t%s\topen=%d\tdone=%d\tfailed=%d\tunrecognised=%d\n' "$1" "$2" "$3" "$4" "$5" "$6"; }
# The TOTAL line for a file; same arguments as sec without the section.
tot() { printf '%s\tTOTAL\topen=%d\tdone=%d\tfailed=%d\tunrecognised=%d\n' "$1" "$2" "$3" "$4" "$5"; }
# A failed-box line.  $1 file  $2 line number
fail_at() { printf 'failed\t%s:%d\n' "$1" "$2"; }
# An unrecognised-box line.  $1 file  $2 line number  $3 marker text
unrec_at() { printf 'unrecognised\t%s:%d\t[%s]\n' "$1" "$2" "$3"; }
# An unclosed-fence line.  $1 file  $2 line number
unclosed_at() { printf 'unclosed-fence\t%s:%d\n' "$1" "$2"; }

# Record one result.  $1 case name  $2 0 for pass, anything else for fail
record() {
  if [ "$2" -eq 0 ]; then
    echo "PASS $1"
    passed=$((passed + 1))
  else
    echo "FAIL $1"
    failed=$((failed + 1))
  fi
}

# Record a case that could not run. It is counted as skipped, never as a pass.
#   $1 case name  $2 why
skip() {
  echo "SKIP $1 ($2)"
  skipped=$((skipped + 1))
}

# True when `locale -a` lists the locale. Names are compared lower-case with
# hyphens removed, because Linux lists en_US.UTF-8 as en_US.utf8. The list is
# captured first so grep -q closing the pipe early cannot trip pipefail.
#   $1 locale name
locale_installed() {
  local want have
  want=$(printf '%s' "$1" | tr 'A-Z' 'a-z' | tr -d '-')
  have=$(locale -a 2>/dev/null | tr 'A-Z' 'a-z' | tr -d '-')
  grep -qx -- "$want" <<<"$have"
}

# Run the script on the given files and compare its stdout, byte for byte, with
# the output of the named expectation function. Also requires exit status 0.
# The script runs in the caller's locale.
#   $1 case name  $2 name of a function that prints the expected output
#   $3... files to pass to the script
check() {
  local name="$1" want="$2" rc=0
  shift 2
  if cmp -s <(bash "$BIN" "$@" 2>/dev/null) <("$want"); then
    bash "$BIN" "$@" >/dev/null 2>&1 || rc=$?
    record "$name" "$rc"
  else
    record "$name" 1
    diff <("$want") <(bash "$BIN" "$@" 2>&1) | sed 's/^/    /'
  fi
}

# Run the script expecting a usage error: exit 2, nothing on stdout, something
# on stderr.
#   $1 case name  $2... arguments
check_usage_error() {
  local name="$1" out err rc=0
  shift
  out=$(bash "$BIN" "$@" 2>/dev/null) || rc=$?
  err=$(bash "$BIN" "$@" 2>&1 >/dev/null) || true
  if [ "$rc" -eq 2 ] && [ -z "$out" ] && [ -n "$err" ]; then
    record "$name" 0
  else
    record "$name" 1
    echo "    exit=$rc stdout=[$out] stderr=[$err]"
  fi
}

# Expectations, one function per fixture.

want_markers() {
  local f=$D/markers.md
  sec "$f" Markers 2 2 1 2
  tot "$f" 2 2 1 2
  fail_at "$f" 5
  unrec_at "$f" 9 '✅ '
  unrec_at "$f" 10 X
}
want_indent() {
  local f=$D/indent.md
  sec "$f" Indent 1 2 1 0
  tot "$f" 1 2 1 0
  fail_at "$f" 5
}
want_notboxes() {
  local f=$D/notboxes.md
  sec "$f" 'Not boxes' 1 0 0 0
  tot "$f" 1 0 0 0
}
want_fences() {
  local f=$D/fences.md
  sec "$f" Fences 3 1 0 0
  sec "$f" After 0 1 0 0
  tot "$f" 3 2 0 0
}
want_unclosed() {
  local f=$D/unclosed.md
  sec "$f" Open 1 0 0 0
  tot "$f" 1 0 0 0
  unclosed_at "$f" 3
}
want_grouping() {
  local f=$D/grouping.md
  sec "$f" '(top)' 2 1 0 0
  sec "$f" Alpha 1 1 1 0
  sec "$f" 'Beta 🚀 héllo' 1 1 0 0
  tot "$f" 4 3 1 0
  fail_at "$f" 10
}
want_dupes() {
  local f=$D/dupes.md
  sec "$f" Same 1 0 0 0
  sec "$f" Other 0 1 0 0
  sec "$f" Same 0 1 1 0
  tot "$f" 1 2 1 0
  fail_at "$f" 7
}
want_spaced() {
  local f="$D/with space.md"
  sec "$f" 'Spaced name' 1 0 1 0
  tot "$f" 1 0 1 0
  fail_at "$f" 3
}
want_lengths() {
  local f=$D/lengths.md
  sec "$f" Lengths 1 0 0 4
  tot "$f" 1 0 0 4
  unrec_at "$f" 2 abcd
  unrec_at "$f" 3 '✅✅✅✅'
  unrec_at "$f" 6 ab
  unrec_at "$f" 7 é
}
# Two files in argument order: markers, then notboxes.
want_two_files() { want_markers; want_notboxes; }
# The same two files the other way round.
want_two_files_reversed() { want_notboxes; want_markers; }
# Every fixture, in the order the all_fixtures array lists them.
want_all() {
  want_dupes; want_fences; want_grouping; want_indent; want_lengths
  want_markers; want_notboxes; want_unclosed; want_spaced
}

all_fixtures=("$D/dupes.md" "$D/fences.md" "$D/grouping.md" "$D/indent.md"
  "$D/lengths.md" "$D/markers.md" "$D/notboxes.md" "$D/unclosed.md"
  "$D/with space.md")

# Cases.

# [ ], [x], [✅], [❌] and [✅ ] sit on adjacent lines. Each must be counted
# on its own: failed=1 and done=2 with an unrecognised [✅ ] means nothing was
# folded into its neighbour. The fixture also holds [X], which is unrecognised.
check "markers: open, x, tick, cross, tick-space counted separately" want_markers "$D/markers.md"

check "indentation (spaces, tab) and * bullets" want_indent "$D/indent.md"

# Positive control: the fixture ends in one real box, so "not counted" cannot
# pass by the script finding nothing at all.
check "longer bracket, link, empty marker, no space, ordered, mid-line are not boxes" want_notboxes "$D/notboxes.md"

check "fences: backtick, tilde, ## inside, longer close, shorter and other-char non-close" want_fences "$D/fences.md"

check "unclosed fence: later lines uncounted, opener reported" want_unclosed "$D/unclosed.md"

# The fixture has a box between its `# Title` line and the first `## `: it must
# stay in (top), so a script that made `# ` a section would move it.
check "grouping: (top) incl. after # Title, ### and #### under ##, empty section hidden, TOTAL sums" want_grouping "$D/grouping.md"

# Two `## Same` lines with a different section between them stay two lines.
check "duplicate ## headings stay separate sections" want_dupes "$D/dupes.md"

# The file name holds a space; it must appear whole in every line that names it.
check "file name containing a space is reported whole" want_spaced "$D/with space.md"

# [abcd] and [✅✅✅✅] are 4 characters: unrecognised. [abcde] and five ticks are
# 5 characters: ignored. [✅✅✅✅] is 12 bytes, so a byte count would drop it.
check "marker length: 4 characters unrecognised, 5 ignored, counted in characters" want_lengths "$D/lengths.md"

check "several files: reported in argument order" want_two_files "$D/markers.md" "$D/notboxes.md"
check "several files: reversed arguments reverse the report" want_two_files_reversed "$D/notboxes.md" "$D/markers.md"

check_usage_error "usage error: no arguments"
check_usage_error "usage error: a directory" "$D"
check_usage_error "usage error: a missing file" "$D/does-not-exist.md"
check_usage_error "usage error: a missing file after a good one prints nothing" "$D/markers.md" "$D/does-not-exist.md"

# Every fixture, run under LC_ALL=C and compared with the hand-built expected
# output (want_all) followed by the exit line. A locale case below can only say
# "same as C", so this case is what makes C itself correct.
base=$(LC_ALL=C bash "$BIN" "${all_fixtures[@]}" 2>&1; echo "exit=$?")
want_base="$(want_all)"$'\n'"exit=0"
if [ "$base" = "$want_base" ]; then
  record "all fixtures together under LC_ALL=C match the expected output" 0
else
  record "all fixtures together under LC_ALL=C match the expected output" 1
  diff <(echo "$want_base") <(echo "$base") | sed 's/^/    /'
fi

# Every fixture under the caller's own locale, compared with want_all.
check "all fixtures together under the caller's locale match the expected output" want_all "${all_fixtures[@]}"

# Output under each listed locale compared with the LC_ALL=C output ($base),
# over every fixture. A locale `locale -a` does not list would silently fall
# back to C and pass without testing anything, so it is skipped instead.
for loc in $LOCALES; do
  name="locale: LC_ALL=$loc output identical to LC_ALL=C"
  if ! locale_installed "$loc"; then
    skip "$name" "locale not installed"
    continue
  fi
  other=$(LC_ALL=$loc bash "$BIN" "${all_fixtures[@]}" 2>&1; echo "exit=$?")
  if [ "$base" = "$other" ]; then
    record "$name" 0
  else
    record "$name" 1
    diff <(echo "$base") <(echo "$other") | sed 's/^/    /'
  fi
done

# A run that executed no cases proves nothing.
if [ "$((passed + failed))" -eq 0 ]; then
  echo "FAIL no cases ran"
  exit 1
fi

echo "$passed passed, $failed failed, $skipped skipped"
[ "$failed" -eq 0 ]
