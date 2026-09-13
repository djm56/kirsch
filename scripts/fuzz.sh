#!/usr/bin/env bash
# Run every fuzz target for a bounded time.
#
# `go test` runs each target's seed corpus as ordinary tests, which is what CI
# does. This script runs the fuzzing engine itself, which is what finds new
# inputs — and which found the case-sensitivity bypass in the path denylist
# (plan §11 amendment 51).
#
#   bash scripts/fuzz.sh          # 30s per target
#   FUZZTIME=5m bash scripts/fuzz.sh
set -euo pipefail

FUZZTIME="${FUZZTIME:-30s}"
failed=0

# Two things to get right here.
#
# `go test -list` prints a package's targets BEFORE its own "ok <pkg>" line, so
# a parser that carries the previous package forward pairs every target with the
# wrong one. The target then does not exist in the package it is run against,
# Go reports "ok" because nothing matched, and the whole script passes without
# fuzzing anything.
#
# And the pattern must be anchored: `-fuzz=FuzzSanitize` also matches
# FuzzSanitizeIsIdempotent, and Go refuses to fuzz more than one target at once.
targets=$(go test ./... -list='Fuzz.*' 2>/dev/null | awk '
  /^Fuzz/           { pending[n++] = $0 }
  /^ok[ \t]/        { for (i = 0; i < n; i++) print $2, pending[i]; n = 0 }
  /^(FAIL|---)/     { n = 0 }
' | sort -u || true)

if [ -z "$targets" ]; then
  echo "No fuzz targets found." >&2
  exit 1
fi

echo "Fuzzing each target for $FUZZTIME"
echo
while read -r pkg target; do
  [ -z "$target" ] && continue
  printf '  %-32s %-28s ' "${pkg##*/kirsch/}" "$target"
  # Guard against the mis-pairing above: a target Go cannot find is a broken
  # script, not a passing test.
  # Captured before matching, not piped into `grep -q`. Under `set -o pipefail`
  # grep -q exits on its first match, go test takes SIGPIPE, and the pipeline
  # reports failure — so the guard fired on every target that was actually
  # present, which is precisely backwards.
  listing=$(go test "$pkg" -list='Fuzz' </dev/null 2>/dev/null || true)
  if ! grep -qxF "$target" <<< "$listing"; then
    echo "SKIPPED — $target not found in $pkg (fuzz.sh parsing bug)"
    failed=1
    continue
  fi
  # </dev/null on every command in this loop: a `while read` loop feeds its
  # body the same stdin it is reading from, so any child that touches stdin
  # eats the remaining target list and the loop silently stops early.
  if out=$(go test "$pkg" -run="^${target}\$" -fuzz="^${target}\$" \
           -fuzztime="$FUZZTIME" </dev/null 2>&1); then
    echo "ok"
  else
    echo "FAIL"
    echo "$out" | sed 's/^/      /'
    failed=1
  fi
done <<< "$targets"

echo
if [ "$failed" -ne 0 ]; then
  echo "Failing inputs are written to the package's testdata/fuzz/ directory."
  echo "Commit them: they become regression tests that run on every go test."
  exit 1
fi
echo "All fuzz targets survived $FUZZTIME each."
