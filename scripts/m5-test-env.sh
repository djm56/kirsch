# One-line setup for the Milestone 5 manual tests. SOURCE it, do not run it.
#
# Use, from the kirsch folder, in zsh or bash:
#   source scripts/m5-test-env.sh
#
# Supported shells: zsh and bash only. Any other shell has neither ZSH_VERSION
# nor BASH_VERSION, so the sourced-or-run check below takes the "run" path: it
# prints the "must be sourced" message and calls exit 1, which would end that
# shell if the file was sourced there.
#
# It prints "Ready." and a short summary, or "Not ready:" and what to do.
# Then use these in the same terminal:
#   m5_check   re-run the checks (after a config change or a long break)
#   m5_done    forget the test setup and print the line that deletes its folders
#
# What it does: first checks that the external programs it runs are on PATH
# (see _m5_need), then unsets the API key variables (and KIRSCH_LIVE_MODEL,
# MY_TOKEN, T_BROKEN), makes new temporary folders (settings, saved sessions, a
# practice project), points XDG_CONFIG_HOME and XDG_STATE_HOME at the first
# two, makes the practice project a Git repository with a .kirsch folder,
# writes a test settings file that keeps saved sessions in the test folder, and
# defines the helper functions the manual test sections use: storage_ok,
# keycount, keycount_control, guard.
#
# Variables it sets and exports: T_CFG, T_STATE, WORKSPACE, KIRSCH_BIN,
# XDG_CONFIG_HOME, XDG_STATE_HOME, and its own bookkeeping: M5_OLD_DIRS,
# M5_BROKEN_DIRS, M5_XDG_SAVED, M5_ORIG_XDG_CONFIG_HOME, M5_ORIG_XDG_CONFIG_SET,
# M5_ORIG_XDG_STATE_HOME, M5_ORIG_XDG_STATE_SET. m5_done unsets all of them,
# except XDG_CONFIG_HOME and XDG_STATE_HOME, which it puts back (see below).
#
# It follows "Setup", "Guard functions" and "Cleanup" in
# plan/testing/manual/milestone-5.md. Where it differs from them:
#   - guard first checks that awk, env and grep are on PATH (GUARD FAIL: <tool>
#     was not found, return 2); requires T_CFG and T_STATE to be non-empty, so
#     an all-unset terminal does not pass; and runs keycount before
#     keycount_control, so a set key is reported as a set key.
#   - The folders are made with mktemp -d and a full template under TMPDIR (or
#     /tmp), named m5-test.XXXXXX, not with plain mktemp -d.
#   - git init is run with -q, and the two mkdir -p calls are one call.
#   - It does not cd; it finds the kirsch folder with git rev-parse.
#   - It also exports KIRSCH_BIN, records earlier test folders, and saves the
#     XDG variables for m5_done.
#   - The checks (_m5_checks) also check that the tools are on PATH, that the
#     test folders exist, that the practice project has .git and .kirsch, and
#     that the settings file exists. They run keycount before keycount_control,
#     and they do not run the two grep lines at the start of "Guard" (storage_ok
#     checks the same line).
#   - m5_done replaces "Cleanup": it puts XDG back instead of unsetting it, does
#     not run keycount_control or keycount, and prints a delete line that can
#     include the section 7 folder (T_BROKEN).
#
# XDG variables: the first time a source in this terminal has made the test
# folders, it saves whether XDG_CONFIG_HOME and XDG_STATE_HOME
# were set, and their values; it does not change them before that point. m5_done
# puts them back as they were, and leaves them unset if they were unset before.
# If they were never saved, m5_done leaves them alone.
#
# What it does not do: no network, no API keys, no build. Deletions are all
# non-recursive and all of folders or files this file made itself: rm -f and
# rmdir in _m5_refuses (its throwaway check files), and rmdir in _m5_setup
# when making the test folders fails part way. m5_done only prints the delete
# line, and running it is the operator's call.
#
# This file is sourced, so it never calls exit while sourced and never sets
# errexit or nounset: either would close or break the operator's terminal.

# storage_ok FILE: exit 0 only when the line storage_dir = "$T_STATE/sessions"
# appears exactly once under [session] (copies under other sections are not
# counted); exit 1 when it does not; awk's own non-zero status (2 here) when
# awk cannot read FILE.
storage_ok() {
  awk -v want="storage_dir = \"$T_STATE/sessions\"" '
    /^\[/ { sec = $0 }
    $0 == want && sec == "[session]" { n++ }
    END { exit (n == 1 ? 0 : 1) }' "$1"
}

# keycount: print how many lines of env output start with one of the four API
# key names (ANTHROPIC_API_KEY, KIRSCH_ANTHROPIC_API_KEY, OPENCODE_API_KEY,
# KIRSCH_OPENCODE_API_KEY) followed by "=". env lists exported variables only,
# so a key set but not exported is not counted. A key exported as the empty
# string is counted.
keycount() {
  env | grep -c -E '^(KIRSCH_)?(ANTHROPIC|OPENCODE)_API_KEY='
}

# keycount_control: in a subshell, export OPENCODE_API_KEY=planted and require
# keycount there to print 1. It passes when no key is exported, and also when
# OPENCODE_API_KEY is the only one exported (the plant overwrites it, so it
# still counts 1). It fails when any of the other three is exported (the count
# is then 2 or more).
keycount_control() {
  local got
  got=$(export OPENCODE_API_KEY=planted; keycount)
  if [ "$got" = 1 ]; then echo "CONTROL PASS: a planted key line counts as 1"; return 0; fi
  echo "CONTROL FAIL: a planted key line counted $got, not 1"; return 1
}

# guard: print GUARD PASS (return 0) or GUARD FAIL (return 1) for the test
# setup. When awk, env or grep is not on PATH it prints "GUARD FAIL: <tool> was
# not found" and returns 2, because then it could not check.
guard() {
  local t
  t=$(_m5_missing awk env grep) || { echo "GUARD FAIL: $t was not found"; return 2; }
  [ -n "$T_CFG" ] && [ -n "$T_STATE" ] && [ "$XDG_CONFIG_HOME" = "$T_CFG" ] && [ "$XDG_STATE_HOME" = "$T_STATE" ] || { echo "GUARD FAIL: XDG_CONFIG_HOME or XDG_STATE_HOME is not the scratch path"; return 1; }
  storage_ok "$T_CFG/kirsch/config.toml" || { echo "GUARD FAIL: storage_dir is not under [session] in the scratch config"; return 1; }
  [ "$(keycount)" = 0 ] || { echo "GUARD FAIL: a key variable is set"; return 1; }
  keycount_control || { echo "GUARD FAIL: the key-count control did not print 1"; return 1; }
  echo "GUARD PASS"
}

# _m5_missing TOOL...: print the name of the first TOOL that command -v cannot
# find and return 1; print nothing and return 0 when all are found.
_m5_missing() {
  local t
  for t in "$@"; do
    command -v "$t" >/dev/null 2>&1 || { printf '%s\n' "$t"; return 1; }
  done
  return 0
}

# _m5_need: check that each external program the functions in this file need
# for setup and checks (the names passed to _m5_missing below) is on PATH.
# Print nothing and return 0 when all are found; otherwise print one
# "Not ready: <tool> was not found. ..." line naming the first one missing,
# and return 1. Shell builtins are not listed. m5_done's optional getconf is
# not listed: it works without it.
_m5_need() {
  local t
  t=$(_m5_missing git mktemp awk env grep mkdir rm rmdir) || {
    printf '%s\n' "Not ready: $t was not found. Install it or put it on your PATH, then type: source scripts/m5-test-env.sh"
    return 1
  }
  return 0
}

# _m5_mkdir: make a new empty folder named m5-test.XXXXXX (random suffix) under
# TMPDIR, or under /tmp when TMPDIR is empty or unset, and print its path. A
# trailing slash on TMPDIR is dropped. mktemp is given the full template, so
# the folder is created exactly there. Return mktemp's status.
_m5_mkdir() {
  local base="${TMPDIR:-/tmp}"
  base="${base%/}"
  mktemp -d "$base/m5-test.XXXXXX"
}

# _m5_ours PATH: return 0 when the last part of PATH starts with "m5-test.",
# the name _m5_mkdir gives its folders, else 1. It looks at the name only, so a
# folder someone else names m5-test.something would also pass.
_m5_ours() {
  local p="${1%/}"
  case "${p##*/}" in
    m5-test.*) return 0 ;;
  esac
  return 1
}

# _m5_sec7_why PATH BASE...: decide whether PATH (a section 7 T_BROKEN folder)
# may go in m5_done's delete line. Return 0 and print nothing when PATH,
# trailing slashes dropped, is not /, is not HOME, is not a symbolic link, does
# not end in . or .., and its parent folder is one of the BASE folders.
# Otherwise print the reason it is left out (naming the non-empty BASE folders
# when the parent is not one of them) and return 1.
_m5_sec7_why() {
  local p="$1" parent home="$HOME" b list=""
  shift
  while :; do case "$p" in ?*/) p="${p%/}" ;; *) break ;; esac; done
  while :; do case "$home" in ?*/) home="${home%/}" ;; *) break ;; esac; done
  if [ "$p" = / ]; then echo "it is the root folder"; return 1; fi
  if [ -n "$home" ] && [ "$p" = "$home" ]; then echo "it is your home folder"; return 1; fi
  if [ -L "$p" ]; then echo "it is a symbolic link"; return 1; fi
  case "${p##*/}" in
    .|..) echo "its name is \".\" or \"..\""; return 1 ;;
  esac
  parent="${p%/*}"
  while :; do case "$parent" in ?*/) parent="${parent%/}" ;; *) break ;; esac; done
  for b in "$@"; do
    [ -n "$b" ] || continue
    [ "$parent" = "$b" ] && return 0
    if [ -n "$list" ]; then list="$list or $b"; else list="$b"; fi
  done
  echo "it is not directly inside $list, where section 7's mktemp -d makes its folder"
  return 1
}

# _m5_refuses HEADER LINE: write a settings file with HEADER, then LINE, in a
# new temporary folder and run storage_ok on it. Callers pass HEADER with its
# brackets, for example '[policy]'. Return 0 if storage_ok refused the file
# (exit exactly 1), 1 if it accepted it (exit 0), 2 if the temporary file could
# not be made, and 3 if storage_ok ended some other way (it could not run). The
# temporary file and folders are removed with non-recursive calls.
_m5_refuses() {
  local dir s
  dir=$(_m5_mkdir) || return 2
  mkdir -p "$dir/kirsch" || { rmdir "$dir"; return 2; }
  printf '%s\n%s\n' "$1" "$2" > "$dir/kirsch/config.toml" || { rm -f "$dir/kirsch/config.toml"; rmdir "$dir/kirsch" "$dir"; return 2; }
  storage_ok "$dir/kirsch/config.toml"; s=$?
  rm -f "$dir/kirsch/config.toml"; rmdir "$dir/kirsch" "$dir"
  if [ "$s" = 1 ]; then return 0; fi
  if [ "$s" = 0 ]; then return 1; fi
  return 3
}

# _m5_checks: run every check. Print nothing and return 0 when all pass;
# otherwise print one "Not ready: ..." line with what to do, and return 1.
# The only error output it hides is command -v's, in _m5_missing; error text
# from every program it runs is shown.
_m5_checks() {
  local fix="Type: source scripts/m5-test-env.sh" r
  _m5_need || return 1
  if [ -z "$T_CFG" ] || [ -z "$T_STATE" ] || [ -z "$WORKSPACE" ] \
    || [ "$XDG_CONFIG_HOME" != "$T_CFG" ] || [ "$XDG_STATE_HOME" != "$T_STATE" ]; then
    echo "Not ready: this terminal is not set up for testing. $fix"
    return 1
  fi
  if [ ! -d "$T_CFG" ] || [ ! -d "$T_STATE" ] || [ ! -d "$WORKSPACE" ]; then
    echo "Not ready: a test folder is missing. $fix"
    return 1
  fi
  if [ ! -e "$WORKSPACE/.git" ] || [ ! -d "$WORKSPACE/.kirsch" ]; then
    echo "Not ready: the practice project is not a Git repository with a .kirsch folder. $fix"
    return 1
  fi
  if [ ! -f "$T_CFG/kirsch/config.toml" ]; then
    echo "Not ready: the test settings file is missing, so saved sessions might go to your real folder. $fix to rebuild it."
    return 1
  fi
  storage_ok "$T_CFG/kirsch/config.toml"; r=$?
  if [ "$r" = 1 ]; then
    echo "Not ready: the test settings file was changed, so saved sessions might go to your real folder. $fix to rebuild it."
    return 1
  elif [ "$r" != 0 ]; then
    echo "Not ready: the test settings file could not be checked; the error message is above. $fix to rebuild it."
    return 1
  fi
  # keycount runs before keycount_control. With ANTHROPIC_API_KEY,
  # KIRSCH_ANTHROPIC_API_KEY or KIRSCH_OPENCODE_API_KEY exported, the control
  # counts 2 and fails, so checking keycount first makes the key message the
  # reason. (An exported OPENCODE_API_KEY does not fail the control: the
  # control's export overwrites it, so it counts 1.)
  if [ "$(keycount)" != 0 ]; then
    echo "Not ready: an API key is set in this terminal, so a test could reach a paid model. $fix (it clears the keys)."
    return 1
  fi
  if ! keycount_control >/dev/null; then
    echo "Not ready: the API key check is not working. Do not test; tell whoever maintains this script."
    return 1
  fi
  _m5_refuses '[policy]' "storage_dir = \"$T_STATE/sessions\""; r=$?
  if [ "$r" = 2 ]; then
    echo "Not ready: could not make a temporary file for the settings check; the error message is above. Check that TMPDIR (or /tmp) exists, is writable and has space, then type: m5_check"
    return 1
  elif [ "$r" = 3 ]; then
    echo "Not ready: the settings check could not run; the error message is above. Do not test; tell whoever maintains this script."
    return 1
  elif [ "$r" != 0 ]; then
    echo "Not ready: the settings check accepted a line in the wrong section. Do not test; tell whoever maintains this script."
    return 1
  fi
  _m5_refuses '[session]' "# storage_dir = \"$T_STATE/sessions\""; r=$?
  if [ "$r" = 2 ]; then
    echo "Not ready: could not make a temporary file for the settings check; the error message is above. Check that TMPDIR (or /tmp) exists, is writable and has space, then type: m5_check"
    return 1
  elif [ "$r" = 3 ]; then
    echo "Not ready: the settings check could not run; the error message is above. Do not test; tell whoever maintains this script."
    return 1
  elif [ "$r" != 0 ]; then
    echo "Not ready: the settings check accepted a commented-out line. Do not test; tell whoever maintains this script."
    return 1
  fi
  return 0
}

# m5_check: re-run the checks. Print Ready. and return 0, or print the
# "Not ready: ..." line and return 1.
m5_check() {
  local out
  out=$(_m5_checks) || { printf '%s\n' "$out"; return 1; }
  echo "Ready."
}

# m5_done: if the XDG variables were saved, put XDG_CONFIG_HOME and
# XDG_STATE_HOME back as they were (unset if they were unset); if not, leave
# them alone. Unset the variables this file set, then list the folders that
# still exist and print the one line that deletes them. Listed: folders named
# m5-test.* that are in T_CFG, T_STATE, WORKSPACE or M5_OLD_DIRS; and folders
# in M5_BROKEN_DIRS or T_BROKEN (section 7) that _m5_sec7_why accepts, checked
# against TMPDIR (or /tmp) and, where getconf has it, DARWIN_USER_TEMP_DIR
# (where macOS's plain mktemp -d makes folders). A section 7 folder it does not
# accept gets a line saying it was left out and why. Deletes nothing itself.
# Finally removes the functions this file defined.
m5_done() {
  local cfg="$T_CFG" state="$T_STATE" work="$WORKSPACE" old="$M5_OLD_DIRS"
  local sec7="$M5_BROKEN_DIRS" tb="$T_BROKEN"
  local all="" p kind why b1="${TMPDIR:-/tmp}" b2="" seen="
" line="" n=0 left=0
  while :; do case "$b1" in ?*/) b1="${b1%/}" ;; *) break ;; esac; done
  # getconf's error output is hidden: DARWIN_USER_TEMP_DIR is a macOS name, and
  # where getconf does not know it, it fails and only TMPDIR (or /tmp) is used.
  if command -v getconf >/dev/null 2>&1; then
    b2=$(getconf DARWIN_USER_TEMP_DIR 2>/dev/null) || b2=""
    while :; do case "$b2" in ?*/) b2="${b2%/}" ;; *) break ;; esac; done
  fi
  if [ -n "$M5_XDG_SAVED" ]; then
    if [ "$M5_ORIG_XDG_CONFIG_SET" = 1 ]; then export XDG_CONFIG_HOME="$M5_ORIG_XDG_CONFIG_HOME"; else unset XDG_CONFIG_HOME; fi
    if [ "$M5_ORIG_XDG_STATE_SET" = 1 ]; then export XDG_STATE_HOME="$M5_ORIG_XDG_STATE_HOME"; else unset XDG_STATE_HOME; fi
  fi
  unset ANTHROPIC_API_KEY KIRSCH_ANTHROPIC_API_KEY OPENCODE_API_KEY KIRSCH_OPENCODE_API_KEY KIRSCH_LIVE_MODEL MY_TOKEN T_BROKEN M5_OLD_DIRS M5_BROKEN_DIRS
  unset M5_XDG_SAVED M5_ORIG_XDG_CONFIG_HOME M5_ORIG_XDG_CONFIG_SET M5_ORIG_XDG_STATE_HOME M5_ORIG_XDG_STATE_SET
  unset T_CFG T_STATE WORKSPACE KIRSCH_BIN
  echo "Test setup forgotten in this terminal."
  # Build a newline-separated list of "o:PATH" (must be named m5-test.*) and
  # "b:PATH" (section 7, must pass _m5_sec7_why) entries.
  for p in "$cfg" "$state" "$work"; do
    [ -n "$p" ] && all="${all}o:$p
"
  done
  while IFS= read -r p; do
    [ -n "$p" ] && all="${all}o:$p
"
  done <<EOF
$old
EOF
  while IFS= read -r p; do
    [ -n "$p" ] && all="${all}b:$p
"
  done <<EOF
$sec7
$tb
EOF
  while IFS= read -r p; do
    [ -n "$p" ] || continue
    kind="${p%%:*}"
    p="${p#*:}"
    [ -d "$p" ] || continue
    if [ "$kind" = o ]; then _m5_ours "$p" || continue; fi
    case "$seen" in
      *"
$p
"*) continue ;;
    esac
    seen="$seen$p
"
    if [ "$kind" = b ]; then
      why=$(_m5_sec7_why "$p" "$b1" "$b2") || {
        printf '%s\n' "Left out of the delete line: $p (named by T_BROKEN, section 7): $why."
        left=1
        continue
      }
    fi
    if [ "$n" = 0 ]; then
      echo "These test folders are still on disk:"
    fi
    n=1
    if [ "$kind" = b ]; then
      printf '%s\n' "  $p  (named by T_BROKEN, section 7)"
    else
      printf '%s\n' "  $p"
    fi
    line="$line$(printf ' %q' "$p")"
  done <<EOF
$all
EOF
  if [ "$n" = 0 ]; then
    if [ "$left" = 1 ]; then
      echo "Nothing to put in a delete line."
    else
      echo "Nothing to clean up: no test folders are on disk."
    fi
  else
    echo "To delete them, paste this line (nothing here deletes them for you):"
    printf '%s\n' "rm -rf --$line"
  fi
  unset -f storage_ok keycount keycount_control guard _m5_missing _m5_need _m5_mkdir _m5_ours _m5_sec7_why _m5_checks _m5_refuses m5_check m5_done
}

# _m5_setup: the work of sourcing the file. Return 0 when ready, 1 when not.
_m5_setup() {
  local root built p had
  _m5_need || return 1
  root=$(git rev-parse --show-toplevel) || {
    printf '%s\n' "Not ready: git could not find the kirsch folder from here; its message is above. Run this from inside the kirsch folder (cd /Volumes/DATA/Github/kirsch), then type: source scripts/m5-test-env.sh"
    return 1
  }
  if [ ! -f "$root/scripts/m5-test-env.sh" ]; then
    printf '%s\n' "Not ready: $root is a Git folder but has no scripts/m5-test-env.sh. Run this from inside the kirsch folder (cd /Volumes/DATA/Github/kirsch), then type: source scripts/m5-test-env.sh"
    return 1
  fi

  # T_BROKEN is made by manual section 7, not by this script, and is about to
  # be unset. When it names a folder, add it to M5_BROKEN_DIRS (one path per
  # line) so a re-source does not lose it. Nothing is checked here; m5_done
  # decides (with _m5_sec7_why) whether it goes in the delete line.
  if [ -n "$T_BROKEN" ] && [ -d "$T_BROKEN" ]; then
    if [ -n "$M5_BROKEN_DIRS" ]; then M5_BROKEN_DIRS="$M5_BROKEN_DIRS
$T_BROKEN"; else M5_BROKEN_DIRS="$T_BROKEN"; fi
    export M5_BROKEN_DIRS
  fi

  unset ANTHROPIC_API_KEY KIRSCH_ANTHROPIC_API_KEY OPENCODE_API_KEY KIRSCH_OPENCODE_API_KEY KIRSCH_LIVE_MODEL MY_TOKEN T_BROKEN

  # Sourced again: T_CFG, T_STATE and WORKSPACE are about to be replaced. Record
  # the ones that are folders named m5-test.* (one path per line) in
  # M5_OLD_DIRS for m5_done to list, and say so once. Any other value those
  # variables held is not recorded.
  had=0
  for p in "$T_CFG" "$T_STATE" "$WORKSPACE"; do
    if [ -n "$p" ] && [ -d "$p" ] && _m5_ours "$p"; then
      if [ -n "$M5_OLD_DIRS" ]; then M5_OLD_DIRS="$M5_OLD_DIRS
$p"; else M5_OLD_DIRS="$p"; fi
      had=1
    fi
  done
  if [ "$had" = 1 ]; then
    export M5_OLD_DIRS
    echo "Your previous test folders are still on disk; m5_done will list them."
  fi

  # Make the folders one at a time. If one fails, rmdir the empty ones made in
  # this run (non-recursive) and leave T_CFG, T_STATE and WORKSPACE unset.
  unset T_CFG T_STATE WORKSPACE
  T_CFG=$(_m5_mkdir) || {
    unset T_CFG
    echo "Not ready: could not make temporary folders; the error message is above. Check that TMPDIR (or /tmp) exists, is writable and has space, then type: source scripts/m5-test-env.sh"
    return 1
  }
  T_STATE=$(_m5_mkdir) || {
    rmdir "$T_CFG"; unset T_CFG T_STATE
    echo "Not ready: could not make temporary folders; the error message is above. Check that TMPDIR (or /tmp) exists, is writable and has space, then type: source scripts/m5-test-env.sh"
    return 1
  }
  WORKSPACE=$(_m5_mkdir) || {
    rmdir "$T_CFG" "$T_STATE"; unset T_CFG T_STATE WORKSPACE
    echo "Not ready: could not make temporary folders; the error message is above. Check that TMPDIR (or /tmp) exists, is writable and has space, then type: source scripts/m5-test-env.sh"
    return 1
  }
  export T_CFG T_STATE WORKSPACE

  # Not saved yet in this terminal: save whether the XDG variables were set
  # and what they held, for m5_done to restore. ${VAR+1} is 1 only when VAR is
  # set, even to the empty string.
  if [ -z "$M5_XDG_SAVED" ]; then
    M5_ORIG_XDG_CONFIG_SET="${XDG_CONFIG_HOME+1}"; M5_ORIG_XDG_CONFIG_HOME="$XDG_CONFIG_HOME"
    M5_ORIG_XDG_STATE_SET="${XDG_STATE_HOME+1}"; M5_ORIG_XDG_STATE_HOME="$XDG_STATE_HOME"
    M5_XDG_SAVED=1
    export M5_XDG_SAVED M5_ORIG_XDG_CONFIG_HOME M5_ORIG_XDG_CONFIG_SET M5_ORIG_XDG_STATE_HOME M5_ORIG_XDG_STATE_SET
  fi
  export XDG_CONFIG_HOME="$T_CFG" XDG_STATE_HOME="$T_STATE"
  git -C "$WORKSPACE" init -q || {
    echo "Not ready: could not make the practice project a Git repository; git's message is above. Check that git works (type: git --version), then type: source scripts/m5-test-env.sh"
    return 1
  }
  mkdir -p "$WORKSPACE/.kirsch" "$T_CFG/kirsch" || {
    echo "Not ready: could not make the test folders inside the temporary folders; the error message is above. Check that your disk has space, then type: source scripts/m5-test-env.sh"
    return 1
  }
  printf '[session]\nstorage_dir = "%s"\n' "$T_STATE/sessions" > "$T_CFG/kirsch/config.toml" || {
    echo "Not ready: could not write the test settings file; the error message is above. Check that your disk has space, then type: source scripts/m5-test-env.sh"
    return 1
  }
  export KIRSCH_BIN="$root/bin/kirsch"

  _m5_checks || return 1

  if [ -e "$KIRSCH_BIN" ]; then built="(built)"; else built="(not built yet; the first test step builds it)"; fi
  echo "Ready."
  printf '%s\n' "  Test area:      $T_CFG  (settings for this test session)"
  printf '%s\n' "  Test project:   $WORKSPACE  (a practice project for kirsch to work in)"
  printf '%s\n' "  Sessions will be saved in: $T_STATE/sessions"
  echo "  API keys:       none of the four API key variables is set"
  printf '%s\n' "  kirsch:         $KIRSCH_BIN  $built"
  echo "Opened a new terminal window? Paste the setup line again there."
  echo "When you have finished testing, type: m5_done"
}

# Sourced or executed? zsh puts ":file" in ZSH_EVAL_CONTEXT when sourced;
# bash sets BASH_SOURCE[0] to this file, which differs from $0 only when sourced.
# Any other shell leaves _m5_sourced at 0 and takes the exit 1 path below.
_m5_sourced=0
if [ -n "${ZSH_VERSION:-}" ]; then
  case "${ZSH_EVAL_CONTEXT:-}" in *:file*) _m5_sourced=1 ;; esac
elif [ -n "${BASH_VERSION:-}" ]; then
  [ "${BASH_SOURCE[0]:-}" != "$0" ] && _m5_sourced=1
fi
if [ "$_m5_sourced" != 1 ]; then
  echo "This file must be sourced, not run. Type: source scripts/m5-test-env.sh"
  exit 1
fi
unset _m5_sourced

if _m5_setup; then
  unset -f _m5_setup
  return 0 2>/dev/null
else
  unset -f _m5_setup
  return 1 2>/dev/null
fi
