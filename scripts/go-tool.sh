#!/usr/bin/env bash
# Run one of the Go tools that `npm run tools` installs, from wherever it landed.
#
# Why this exists: `go install` puts binaries in $(go env GOPATH)/bin, which is
# not on the PATH of a non-interactive shell — which is the shell npm gives a
# script. So `npm run check` used to die at the lint stage with
#
#     sh: golangci-lint: command not found
#
# A missing tool then reads as a failing check, and the two need different
# responses. This resolves the path first, and when the tool genuinely is not
# installed it says which one and names the command that installs it.
#
# It also refuses a binary whose major version this repository cannot work with,
# for the same reason: "unknown command: fmt" is a worse error than being told
# which version was found and which is needed.
#
# Usage: bash scripts/go-tool.sh <tool> [args...]
set -euo pipefail

if [ "$#" -lt 1 ]; then
  echo "usage: bash scripts/go-tool.sh <tool> [args...]" >&2
  exit 2
fi

tool="$1"
shift

# Major versions below which a tool is not old but wrong. golangci-lint v1 has
# no `fmt` subcommand and cannot read a `version: "2"` config, so on v1 both
# `npm run fmt` and `npm run lint` die on a flag or config error that says
# nothing about the real cause. Tools absent from this case statement pay
# nothing for it — no floor, no extra process.
min_major=""
min_why=""
case "$tool" in
  golangci-lint)
    min_major=2
    min_why='.golangci.yml is a version: "2" config, CI pins a v2 release, and the
formatting step behind npm run fmt is a subcommand v1 does not have. A binary
below that major fails on all three.'
    ;;
esac

# Resolution order mirrors go's own, with one deliberate constraint: PATH is
# checked BEFORE anything asks `go` a question. A tool already on PATH then
# needs no Go installation at all — and, more to the point, a missing `go`
# cannot turn this script into the bare "command not found" it exists to
# replace.
resolved="$(command -v "$tool" 2>/dev/null || true)"

if [ -z "$resolved" ]; then
  # Only now does the fallback need go, and only to ask where `go install`
  # writes. A missing `go` is a different failure from a missing tool, and gets
  # its own message rather than being reported as the tool being absent.
  if ! command -v go >/dev/null 2>&1; then
    cat >&2 <<EOF
$tool is not on PATH, and neither is go.

This script falls back to the directory 'go install' writes to, and it asks go
itself where that is — so with no go on PATH there is nowhere left to look.

Install Go (https://go.dev/dl/), then install the project's tooling with:
    npm run tools
EOF
    exit 127
  fi

  # GOBIN wins when it is set; otherwise `go install` uses GOPATH/bin. That
  # rule lives in go-bin-dir.sh so this and install-tools.sh cannot answer it
  # differently — they did, and the install then called its own freshly
  # written binary a foreign copy shadowing itself. Resolving the helper
  # relative to this file keeps this script runnable from any directory.
  here="$(dirname "${BASH_SOURCE[0]}")"
  gobin="$(bash "$here/go-bin-dir.sh")"

  if [ -x "$gobin/$tool" ]; then
    resolved="$gobin/$tool"
  else
    cat >&2 <<EOF
$tool is not installed.

Looked on PATH, and in $gobin

Install the project's tooling with:
    npm run tools
EOF
    exit 127
  fi
fi

# The floor is checked on the binary about to run, whichever directory it came
# from: the install directory can hold a v1 left over from before
# install-tools.sh was last run, exactly as PATH can.
if [ -n "$min_major" ]; then
  # Bare `version` is the one invocation both majors accept — v1 rejects v2's
  # --short, v2 rejects v1's --format — and both answer "<tool> has version
  # X.Y.Z ...". Anchoring the parse on that phrase avoids matching the Go
  # version printed later on the same line. The optional 'v' is not cosmetic:
  # v1.64.8 prints "has version v1.64.8" and 2.13.2 prints "has version 2.13.2",
  # so a pattern demanding a digit immediately after the phrase reads every v1
  # as unparseable and lets it through.
  version_line="$("$resolved" version 2>&1 | head -1 || true)"
  found_version="$(printf '%s\n' "$version_line" |
    sed -n 's/^.*has version v\{0,1\}\([0-9][0-9]*\.[^[:space:]]*\).*$/\1/p')"

  # A `0.0.0-<timestamp>-<commit>` module pseudo-version is not version zero.
  # It is what a build carrying no reachable semver tag reports — an install
  # from @master or @<commit>, which is almost always NEWER than the pinned
  # release rather than two majors older. The three zeroes are a placeholder,
  # so ordering them against the floor would hard-block a current binary while
  # naming a version it does not have. Both flavours of "the version cannot be
  # determined" are then treated alike: this one and the `(devel)` that reaches
  # us down the same buildinfo path warn and continue. The one thing neither
  # may do is assert an ordering the string does not carry.
  case "$found_version" in
    0.0.0-*) found_version="" ;;
  esac

  found_major="${found_version%%.*}"

  if [ -z "$found_major" ]; then
    # An unreadable version string is not evidence of a wrong version, so it
    # warns and continues rather than blocking work on a parser. The fail-open
    # is bounded: this script ends by exec'ing the tool, so the tool's own exit
    # code still decides the check. The worst case is a floor that is absent,
    # never one that lies.
    echo "warning: could not read $tool version (needs v$min_major or newer); got: $version_line" >&2
  elif [ "$found_major" -lt "$min_major" ]; then
    cat >&2 <<EOF
$tool is too old for this repository.

    found:  v$found_version  ($version_line)
    at:     $resolved
    needs:  v$min_major or newer

$min_why

Upgrade with:
    npm run tools

If that leaves this message in place, the copy above is earlier on your PATH
than the one it installs.
EOF
    exit 1
  fi
fi

exec "$resolved" "$@"
