#!/usr/bin/env bash
# Print the directory `go install` writes binaries into.
#
# One fact, one place. `go install` honours GOBIN when it is set and falls back
# to GOPATH/bin otherwise. Two scripts need that answer — scripts/go-tool.sh to
# find a tool, scripts/install-tools.sh to report where it just put one — and
# while they each stated it themselves they disagreed: the install assumed
# GOPATH/bin unconditionally, so with GOBIN set it named the binary it had just
# written as a foreign copy shadowing the install, and advised removing it.
#
# It asks `go` rather than reading the environment directly, because GOBIN and
# GOPATH can both come from the go env config file rather than from exported
# variables. A caller that can carry on without an answer checks for `go`
# before calling; a caller that cannot lets the missing-go error through.
#
# Usage: bin="$(bash scripts/go-bin-dir.sh)"
set -euo pipefail

bin="$(go env GOBIN)"
if [ -z "$bin" ]; then
  bin="$(go env GOPATH)/bin"
fi

printf '%s\n' "$bin"
