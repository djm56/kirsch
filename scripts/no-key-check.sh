#!/usr/bin/env bash
# Start Kirsch with no API key, in a clean environment, and check that it
# emitted its onboarding line naming the key variables to set.
#
#   bash scripts/no-key-check.sh        (or: npm run check:no-key)
#
# The script takes no arguments and refuses any. It builds bin/kirsch with
# scripts/build.sh, creates a temporary HOME and a temporary Git repository,
# runs
#
#   env -i HOME=<temporary home> PATH=$PATH TERM=xterm-256color bin/kirsch
#
# inside that repository under script(1), which gives Kirsch a pseudo-terminal,
# and checks that the terminal output script(1) captured carries
#
#   No API key: set KIRSCH_OPENCODE_API_KEY → OPENCODE_API_KEY
#
# which is the key pair of the default endpoint (internal/config, opencode).
# What is measured: the line was EMITTED to the terminal at some point in the
# run. The capture is a byte stream of everything Kirsch wrote, not a rendering
# of the final screen, so a line that was drawn and later erased or scrolled away
# still matches. It does not prove the line was on screen when the run ended.
# Nothing from the caller's environment reaches Kirsch: no API key, no XDG_*,
# no NO_COLOR, no LANG. (The NOKEY_* variables the util-linux form uses to hand
# its arguments over are removed by env -i as well.)
#
# This script spends no tokens: nothing is typed into the composer, so Kirsch
# sends no request, and the run has no key to send one with.
#
# Exit codes:
#   0    the onboarding line was emitted (the matched line is printed)
#   1    Kirsch drew frames (the frame marker below is in the capture) and the
#        line is not among them, or the run hit its time limit after Kirsch drew
#        a frame (the last 40 lines of the emitted text go to stderr)
#   2    the check could not complete, which is neither a pass nor a fail: an
#        argument was given, go.mod is missing, a required tool is missing, the
#        build failed, a temporary file or repository could not be made,
#        script(1) is the busybox one, the run hit its time limit before Kirsch
#        drew a frame, script(1) captured nothing or was killed by a signal
#        (exit status 128 or more), the capture holds no Kirsch frame (Kirsch
#        never started, for example because `stty` failed), grep itself failed
#        (exit status 2 or more), the run could not be stopped, or any other
#        command failed (an ERR trap turns that into 2)
#   130  interrupted (INT, TERM or HUP); the run is stopped and the temporary
#        files removed first
#
# Frame marker. Every frame Kirsch draws at the window size used here carries
# its header row: the word Kirsch, a space and a run of rule characters
# (headerRows in internal/tui/view.go; "-" under env -i, which has no UTF-8
# locale). The marker is "Kirsch" followed by 20 or more "-". The header does not
# depend on the onboarding, so it is there with a key supplied and with none. A
# missing onboarding line is a FAIL only when the marker is in the capture; with
# no marker the capture says nothing about Kirsch, and the result is 2.
#
# Required tools (each is looked up on PATH before anything is built or
# created): every external command this script runs, directly or inside the
# pty, plus go. sh, stty and env run inside the pty (sh runs the command string
# in both script(1) forms); ps must list parent process ids
# (`ps -A -o pid= -o ppid=`), because it is how the whole process tree of a
# stuck run is found. go is not run by this script but by scripts/build.sh; it
# is looked up here so that its absence is reported by name. Any other tool
# scripts/build.sh needs is not looked up here: if one is missing the build
# fails, and a failed build is exit 2.
#
# Terminal size. The pty that script(1) creates has no window size of its own
# when the caller has no terminal (CI, an agent shell): on macOS, measured on
# 2026-11-05 with no terminal on the caller's stdin or stdout,
# `script -q <file> sh -c 'stty -a' </dev/null` captured
# "speed 9600 baud; 0 rows; 0 columns;". The command run under script
# therefore starts with `stty rows 40 cols 120`, and then execs `env -i ...`,
# so Kirsch still
# gets the clean environment above and a window wide enough that the onboarding
# line does not wrap.
#
# The script(1) in use is one of two forms, chosen by asking
# `script --version`: util-linux answers with a line naming itself, the BSD
# script on macOS has no such option and fails. util-linux is run as
# `script -q -f -c "<fixed command string>" <file>` (-f flushes the capture
# after every write, so the quit-key feeder sees the text as it is drawn), BSD as
# `script -q <file> <command> <args...>`. The busybox script has the util-linux
# argument form but a different behaviour; it is detected from `script --help`
# and refused with exit 2.
#
# UNRUN BRANCH: the util-linux form has not been run on Linux; it has only been
# checked for shell syntax. Only the BSD form (macOS) has been run.
#
# Start-up time. In two timed runs of this setup on macOS (2026-11-05, BSD
# script form), the first byte reached the capture 29.66 s and 30.16 s after
# launch, and "No API key" was in that first read of the capture both times; no
# cause is claimed. RENDER_WAIT is set above that, RUN_LIMIT above RENDER_WAIT
# plus the two-second exit wait.
#
# The quit keys are sent once the text "No API key" is in the capture, or after
# RENDER_WAIT seconds if it never is. Two Ctrl+C within one second quit Kirsch
# from any mode. A run that has not ended after RUN_LIMIT seconds is sent TERM,
# given GRACE seconds, then sent KILL; it is then a FAIL (1) if Kirsch drew a
# frame first and 2 if it did not (see "Exit codes").
set -eEuo pipefail

# Any command that fails without its own guard below means the check could not
# complete: exit 2, not the 1 that errexit would give.
trap 'echo "no-key-check: a command failed at line $LINENO; the check could not complete" >&2; exit 2' ERR

if [ "$#" -ne 0 ]; then
  echo "usage: bash scripts/no-key-check.sh   (takes no arguments)" >&2
  exit 2
fi

RENDER_WAIT=45 # seconds to wait for the onboarding text before quitting anyway
RUN_LIMIT=90   # seconds before the run is stopped
GRACE=5        # seconds between TERM and KILL

# Every external command the script runs, directly or inside the pty, plus go,
# which scripts/build.sh runs (see "Required tools" above). bin/kirsch is not
# listed: it is built below and checked as a file. Builtins (cd, kill, printf,
# read, wait, trap, command) are not listed. Property: each name resolves on
# PATH now, so a later "command not found" cannot be mistaken for a result
# about Kirsch.
for tool in bash sh script git go sed tr grep tail sleep mktemp mkdir rm dirname env stty ps; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "no-key-check: required tool '$tool' not found on PATH" >&2
    exit 2
  fi
done
# Property: ps can list the parent of every process, which tree_of relies on.
if ! ps -A -o pid= -o ppid= >/dev/null 2>&1; then
  echo "no-key-check: 'ps -A -o pid= -o ppid=' does not work here" >&2
  exit 2
fi

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo="$(dirname "$here")"
bin="$repo/bin/kirsch"

# What the onboarding line must contain. The arrow is built from its UTF-8 bytes
# (e2 86 92) so the source file's encoding cannot change it. Kirsch's glyph table
# falls back to ASCII when no locale variable names UTF-8, which is the case
# under env -i, so the bullet in front of the line is "." here; the arrow is part
# of the literal text in the onboarding code and is not folded.
arrow="$(printf '\342\206\222')"
want="No API key: set KIRSCH_OPENCODE_API_KEY ${arrow} OPENCODE_API_KEY"

if [ ! -f "$repo/go.mod" ]; then
  echo "no-key-check: no go.mod in '$repo'; run the script from its place in the repository" >&2
  exit 2
fi

if ! build_out="$(cd "$repo" && bash scripts/build.sh 2>&1)"; then
  printf '%s\n' "$build_out" >&2
  echo "no-key-check: scripts/build.sh failed" >&2
  exit 2
fi
if [ ! -x "$bin" ]; then
  echo "no-key-check: scripts/build.sh did not produce '$bin'" >&2
  exit 2
fi

# tree_of PID...: prints each PID and every descendant of it, one per line, from
# one ps snapshot. A process whose parent has already died is no longer found
# through it, so callers take the list before they signal anything.
# shellcheck disable=SC2086 # $list is a space-separated pid list, split on purpose
tree_of() {
  local list=" $* " snap p pp grew=1
  snap="$(ps -A -o pid= -o ppid= 2>/dev/null || true)"
  while [ "$grew" -eq 1 ]; do
    grew=0
    while read -r p pp; do
      case "$list" in
      *" $pp "*)
        case "$list" in
        *" $p "*) ;;
        *)
          list="$list$p "
          grew=1
          ;;
        esac
        ;;
      esac
    done <<EOF
$snap
EOF
  done
  printf '%s\n' $list
}

# stop_children: TERM to every descendant of this shell (script(1), what it
# started, and the quit-key feeder), up to GRACE seconds for them to go, then KILL
# to whatever is left, including descendants that were found again. The list is
# taken before anything is signalled. Property: afterwards none of the processes
# that were descendants when it started is still alive, unless KILL itself had no
# effect. It returns at once when there are none.
# shellcheck disable=SC2086 # $victims is a pid list
stop_children() {
  local victims="" p i alive
  for p in $(tree_of "$$"); do
    [ "$p" = "$$" ] || victims="$victims $p"
  done
  [ -n "$victims" ] || return 0
  for p in $victims; do
    kill -TERM "$p" 2>/dev/null || true
  done
  i=0
  while [ "$i" -lt $((GRACE * 5)) ]; do
    alive=0
    for p in $victims; do
      if kill -0 "$p" 2>/dev/null; then alive=1; fi
    done
    [ "$alive" -eq 1 ] || return 0
    sleep 0.2
    i=$((i + 1))
  done
  for p in $victims $(tree_of $victims); do
    [ "$p" = "$$" ] || kill -KILL "$p" 2>/dev/null || true
  done
  sleep 0.2
}

# One temporary root holds the HOME, the repository and the capture.
# The trap removes it on every exit path, and the only path it removes is the one
# mktemp returned.
work=""
spid=""
# shellcheck disable=SC2329 # called by the EXIT trap below
cleanup() {
  stop_children || true
  if [ -n "$work" ]; then
    rm -rf -- "$work" || true
  fi
}
trap cleanup EXIT
trap 'exit 130' INT TERM HUP

if ! work="$(mktemp -d "${TMPDIR:-/tmp}/kirsch-no-key.XXXXXX")"; then
  echo "no-key-check: mktemp could not create a temporary directory" >&2
  exit 2
fi
tmp_home="$work/home"
tmp_repo="$work/repo"
capture="$work/capture.raw"
alive="$work/alive"
if ! mkdir "$tmp_home" || ! : >"$capture" || ! : >"$alive"; then
  echo "no-key-check: could not create the temporary home, capture file or marker file" >&2
  exit 2
fi
# env -i: no GIT_* variable from the caller (GIT_DIR, GIT_WORK_TREE, ...) can
# point git init somewhere else, and the temporary HOME hides the caller's git
# configuration.
if ! env -i HOME="$tmp_home" PATH="$PATH" git init -q "$tmp_repo"; then
  echo "no-key-check: git init failed in '$tmp_repo'" >&2
  exit 2
fi

# The command run under the pty: size it, then replace this shell with env -i so
# Kirsch sees only HOME, PATH and TERM. The values are not part of this text;
# they arrive as the positional parameters $1..$3 (BSD form) or as NOKEY_HOME,
# NOKEY_PATH and NOKEY_BIN in script's environment (util-linux form), so no
# quoting of them is ever needed.
# shellcheck disable=SC2016 # the $1..$3 are for the inner shell, not this one
inner_bsd='stty rows 40 cols 120 && exec env -i HOME="$1" PATH="$2" TERM=xterm-256color "$3"'
# shellcheck disable=SC2016 # the NOKEY_* expand in the inner shell, not this one
inner_linux='stty rows 40 cols 120 && exec env -i HOME="$NOKEY_HOME" PATH="$NOKEY_PATH" TERM=xterm-256color "$NOKEY_BIN"'

# Sends the quit keys. It polls the capture for the onboarding text, waits a
# moment so the frame is complete, then sends two Ctrl+C 0.3 s apart. It stops
# polling, and sends nothing, as soon as the marker file is gone: cleanup removes
# the temporary directory, and a feeder that was started by script(1)'s own
# subshell is not a descendant of this shell once script has died, so it cannot
# be found and signalled; the marker is how it learns the check is over.
feed_quit() {
  local i=0 limit=$((RENDER_WAIT * 5))
  while [ "$i" -lt "$limit" ]; do
    [ -e "$alive" ] || return 0
    if LC_ALL=C grep -a -q -F 'No API key' "$capture" 2>/dev/null; then
      break
    fi
    sleep 0.2
    i=$((i + 1))
  done
  sleep 0.5
  [ -e "$alive" ] || return 0
  printf '\003'
  sleep 0.3
  printf '\003'
  sleep 2
}

cd "$tmp_repo"

# util-linux script names itself on --version; BSD script has no such option and
# fails. Output goes to a variable and not into a pipe, so no early reader exit
# can fail the pipeline. Run inside the temporary repository, with stdin from
# /dev/null, so nothing can wait on the caller's terminal or leave a file behind.
ver_out="$(script --version </dev/null 2>&1 || true)"
help_out="$(script --help </dev/null 2>&1 || true)"
case "$help_out" in
*[Bb]usy[Bb]ox*)
  echo "no-key-check: the busybox script(1) is not supported; install util-linux or BSD script" >&2
  exit 2
  ;;
esac
case "$ver_out" in
*util-linux*) script_form=util-linux ;;
*) script_form=bsd ;;
esac

# script's stdin is a pipe from feed_quit, made with process substitution. Not a
# fifo: script(1) on macOS fails its terminal-attribute call (tcgetattr) on one.
# Not a pipeline either: `wait` on the last process of a pipeline also waits for
# the first, so a feeder still sleeping would hold the wait up. With this form
# the background job is script alone, and spid is its pid.
if [ "$script_form" = util-linux ]; then
  # SHELL is the sh found on PATH (looked up above): util-linux runs the -c
  # string with $SHELL, and the fixed string is sh syntax.
  NOKEY_HOME="$tmp_home" NOKEY_PATH="$PATH" NOKEY_BIN="$bin" SHELL="$(command -v sh)" \
    script -q -f -c "$inner_linux" "$capture" < <(feed_quit) >/dev/null 2>&1 &
else
  script -q "$capture" sh -c "$inner_bsd" sh "$tmp_home" "$PATH" "$bin" < <(feed_quit) >/dev/null 2>&1 &
fi
spid=$!

# Watchdog: poll for the end of the run; at RUN_LIMIT stop everything it started.
waited=0
timed_out=0
while kill -0 "$spid" 2>/dev/null; do
  if [ "$waited" -ge "$RUN_LIMIT" ]; then
    timed_out=1
    stop_children
    break
  fi
  sleep 1
  waited=$((waited + 1))
done

# The wait is only made on a process that has already ended, so it cannot block.
if kill -0 "$spid" 2>/dev/null; then
  echo "no-key-check: script(1) (pid $spid) is still running after KILL; the check could not complete" >&2
  exit 2
fi
script_status=0
wait "$spid" 2>/dev/null || script_status=$?
spid=""

# Emitted text: drop CSI sequences (ESC [, parameter bytes 0x30-0x3F, intermediate
# bytes 0x20-0x2F, one final byte 0x40-0x7E), OSC sequences ended by ESC \
# (Kirsch's background-colour query), and CRs. The parameter-byte range is
# written [0-?] (0x30 to 0x3F: digits, : ; < = > ?). The intermediate-byte range
# is written [ -/] and the s command uses # as its delimiter so that the / inside
# the bracket needs no escape; a backslash there would be a literal in a POSIX
# bracket expression and widen the range.
esc="$(printf '\033')"
emitted="$work/emitted.txt"
LC_ALL=C sed \
  -e "s#${esc}\[[0-?]*[ -/]*[@-~]##g" \
  -e "s#${esc}\][^${esc}]*${esc}\\\\##g" \
  "$capture" | tr -d '\r' >"$emitted"

# grep_emitted ARGS...: runs grep over the emitted text, writes the first matching
# line to $first_match (empty when none) and returns 0 for a match, 1 for none.
# grep's own exit status 2 or more is an error, not "none": it ends the check
# with exit 2. The status is read directly, never folded into a pipeline or an
# `|| true`, so an error and "no match" cannot look alike.
first_match=""
grep_emitted() {
  local rc=0 all="$work/grep.out"
  LC_ALL=C grep -a "$@" "$emitted" >"$all" || rc=$?
  if [ "$rc" -ge 2 ]; then
    echo "no-key-check: grep failed (exit status $rc); the check could not complete" >&2
    exit 2
  fi
  first_match=""
  if [ "$rc" -eq 0 ]; then
    IFS= read -r first_match <"$all" || true
  fi
  return "$rc"
}

# A run stopped by the watchdog. This comes before the empty-capture and signal
# checks below: the watchdog itself sent script(1) TERM (and KILL if needed), so
# script's exit status is not evidence about Kirsch. The frame-marker rule (see
# "Frame marker" above) decides it: a hang after Kirsch drew a frame is a FAIL;
# a hang with no Kirsch frame in the capture (an empty capture included) says
# nothing about Kirsch, and the check could not complete.
if [ "$timed_out" -eq 1 ]; then
  if grep_emitted -E -- 'Kirsch -{20,}'; then
    echo "no-key-check: FAIL (Kirsch drew frames, then the run did not end within ${RUN_LIMIT}s and was stopped)" >&2
    tail -n 40 "$emitted" >&2
    exit 1
  fi
  echo "no-key-check: the run did not end within ${RUN_LIMIT}s and was stopped, and the capture holds no Kirsch frame; the check could not complete" >&2
  tail -n 40 "$emitted" >&2
  exit 2
fi

# Could script(1) run the command at all? No bytes captured means nothing was
# ever drawn, and a status of 128 or more means script (or what it ran) was
# killed by a signal; either way the emitted text says nothing about Kirsch. Any
# other status is Kirsch's own or script's own, and the text decides: the
# onboarding line is what is being checked, not Kirsch's exit status.
if [ ! -s "$capture" ]; then
  echo "no-key-check: script(1) captured nothing (exit status $script_status); the check could not complete" >&2
  exit 2
fi
if [ "$script_status" -ge 128 ]; then
  echo "no-key-check: script(1) was killed by a signal (exit status $script_status); the check could not complete" >&2
  exit 2
fi

if grep_emitted -F -- "$want"; then
  printf '%s\n' "$first_match"
  echo "no-key-check: PASS (the line was emitted to the terminal)"
  exit 0
fi

# The line is missing. That is a FAIL only if Kirsch drew a frame: its header row
# (see "Frame marker" above) is positive evidence it ran. Without it the run
# never reached Kirsch and there is nothing to judge.
if ! grep_emitted -E -- 'Kirsch -{20,}'; then
  echo "no-key-check: the capture holds no Kirsch frame (no header row); Kirsch did not start or drew nothing; the check could not complete" >&2
  tail -n 40 "$emitted" >&2
  exit 2
fi
echo "no-key-check: FAIL (Kirsch drew frames, header row seen: ${first_match:0:30}; the terminal output never carried: $want)" >&2
tail -n 40 "$emitted" >&2
exit 1
