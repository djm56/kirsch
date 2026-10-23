#!/usr/bin/env bash
# Run what .github/workflows/ci.yml runs, locally, so a push does not have to
# be the first time CI's checks see a change.
#
#   bash scripts/ci.sh lint        gofmt, go vet, golangci-lint
#   bash scripts/ci.sh test        race tests, architecture rules, golden files
#   bash scripts/ci.sh security    govulncheck, gosec, secret scan, fuzz corpus
#   bash scripts/ci.sh screens     screen reference grid lint
#   bash scripts/ci.sh all         every job above; fails if any job fails
#
# The workflow's jobs call this same script (`bash scripts/ci.sh <job>`), so the
# commands live here once and the workflow cannot drift from a local run. The
# npm entries are `ci:local` (all) and `ci:lint`, `ci:test`, `ci:security`,
# `ci:screens`. They are separate from `npm run ci`, which does not run every CI
# step. `ci:lint` green is not `npm run fmt:check` green: the latter also runs
# gofumpt and goimports, which CI does not.
#
# Two things are pinned so a local run sees what CI sees:
#
#   Go toolchain. CI's actions/setup-go@v5 installs the version on go.mod's `go`
#   line (src/installer.ts parseGoVersionFile matches /^go (\d+(\.\d+)*)/m and
#   v5 does not read the `toolchain` directive). This script reads that same
#   line and sets GOTOOLCHAIN to it, so a newer Go installed locally cannot
#   hide a standard-library advisory CI would find. The `+auto` suffix keeps
#   CI's own behaviour: go may still switch UP to a newer `toolchain` directive
#   or a dependency that needs one, exactly as the go command does on the runner.
#
#   golangci-lint. The version lives in scripts/golangci-lint-version and
#   nowhere else; this script installs the official release binary of that
#   version into bin/ (git-ignored) and the workflow never names a version.
#
# govulncheck and gosec are installed @latest, as CI does. That means results
# drift with advisory-database and tool releases. That is intended: a new
# advisory is meant to fail the build rather than wait to be noticed.
#
# Side effects of running this script on a developer machine:
#
#   - `go install ...@latest` OVERWRITES govulncheck and gosec in the shared Go
#     bin dir (GOBIN, else GOPATH/bin), the same binaries `npm run security:*`
#     uses. It does not install into the repository.
#   - Toolchain downloads triggered by GOTOOLCHAIN go into GOMODCACHE.
#   - The golangci-lint release binary goes into bin/ (git-ignored).
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo="$(dirname "$here")"
cd "$repo"

usage() {
  echo "usage: bash scripts/ci.sh <lint|test|security|screens|all>" >&2
  exit 2
}

if [ "$#" -ne 1 ]; then
  usage
fi
job="$1"

# Mirror setup-go@v5's go-version-file handling: the first `go X.Y.Z` line of
# go.mod, and nothing else. A version without a patch number is refused rather
# than guessed, because setup-go would resolve it to "latest patch" and a
# GOTOOLCHAIN of go1.25 names no real toolchain.
use_ci_toolchain() {
  local v
  v="$(sed -n 's/^go \([0-9][0-9]*\(\.[0-9][0-9]*\)*\).*$/\1/p' go.mod | head -1)"
  case "$v" in
    [0-9]*.[0-9]*.[0-9]*) ;;
    *)
      echo "ci.sh: cannot read a full go X.Y.Z version from the go directive in go.mod (got: '$v')" >&2
      exit 1
      ;;
  esac
  export GOTOOLCHAIN="go${v}+auto"
  echo "ci.sh: GOTOOLCHAIN=$GOTOOLCHAIN"
  go version
}

# Install the golangci-lint release binary named in scripts/golangci-lint-version
# with the project's own install script, pinned to the same tag so the script
# and the binary cannot be from different releases. Prints the binary's path, or
# prints nothing and returns 1 on failure; the caller checks both, because bash
# clears errexit inside a command substitution.
golangci_lint_bin() {
  local ver dir
  ver="$(tr -d '[:space:]' <"$here/golangci-lint-version")"
  if ! printf '%s\n' "$ver" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
    echo "ci.sh: scripts/golangci-lint-version must hold a version like v2.13.2 (got: '$ver')" >&2
    return 1
  fi
  dir="$repo/bin/golangci-lint-$ver"
  if [ ! -x "$dir/golangci-lint" ]; then
    echo "ci.sh: installing golangci-lint $ver into $dir" >&2
    curl --proto '=https' --tlsv1.2 -sSfL \
      "https://raw.githubusercontent.com/golangci/golangci-lint/$ver/install.sh" |
      sh -s -- -b "$dir" "$ver" >&2 || return 1
  fi
  printf '%s\n' "$dir/golangci-lint"
}

# `go install <pkg>@latest`, then print the path of the binary it wrote. The
# path is built from the Go bin dir rather than trusting PATH, which may hold
# an older copy of the tool. A failed install prints nothing and returns 1, so a
# stale copy already in the bin dir is never run in its place.
install_go_tool() {
  local name="$1" pkg="$2" bin
  go install "$pkg" >&2 || return 1
  bin="$(bash "$here/go-bin-dir.sh")" || return 1
  printf '%s\n' "$bin/$name"
}

# Run `<tool path> args...` where the path comes from a helper above. The helper
# is called in a command substitution, where errexit is off, so its status and
# its output are both checked here: a failed install must fail the job.
#   run_tool <installer function> <installer args...> -- <tool args...>
run_tool() {
  local -a inst=()
  while [ "$1" != "--" ]; do
    inst+=("$1")
    shift
  done
  shift
  local bin
  bin="$("${inst[@]}")" || { echo "ci.sh: installing ${inst[*]} failed" >&2; return 1; }
  if [ -z "$bin" ] || [ ! -x "$bin" ]; then
    echo "ci.sh: no executable at '$bin' after installing ${inst[*]}" >&2
    return 1
  fi
  "$bin" "$@"
}

job_lint() {
  use_ci_toolchain

  # gofmt does not follow GOTOOLCHAIN, so take it from the selected toolchain's
  # GOROOT; the gofmt first on PATH could belong to a different Go.
  #
  # The file list is git's: tracked files plus untracked ones that are not
  # ignored (new files not yet `git add`ed), which is what a fresh CI checkout
  # of the same change would hold. Files git ignores or excludes are skipped:
  # .claude/ is in .git/info/exclude and its workspace archives hold
  # deliberately malformed Go that `gofmt -l .` would report. A tracked .go file
  # deleted in the working tree and not yet committed makes gofmt error out;
  # commit or restore it first.
  #
  # An empty list is an error. gofmt run with no arguments reads stdin, finds
  # nothing and prints nothing, which would look exactly like a clean tree.
  local gofmt unformatted nfiles
  gofmt="$(go env GOROOT)/bin/gofmt"
  nfiles="$(git ls-files -z --cached --others --exclude-standard -- '*.go' | tr -cd '\0' | wc -c | tr -d ' ')"
  if [ "$nfiles" -eq 0 ]; then
    echo "ci.sh: no .go files found by git ls-files; refusing to report a clean gofmt" >&2
    exit 1
  fi
  unformatted="$(git ls-files -z --cached --others --exclude-standard -- '*.go' | xargs -0 "$gofmt" -l)"
  if [ -n "$unformatted" ]; then
    echo "These files are not gofmt-clean:"
    echo "$unformatted"
    exit 1
  fi

  go vet ./...

  # .golangci.yml is a v2 config, so the binary has to be v2. The pin is concrete
  # so a linter release cannot turn a green branch red on its own; bump it in
  # scripts/golangci-lint-version, the only place that names it.
  run_tool golangci_lint_bin -- run
}

job_test() {
  use_ci_toolchain

  go test -race -timeout 5m ./...

  # Plan §2's dependency rules are enforced by a test, not by review. Run
  # explicitly as well as via ./... so a failure names itself in the log.
  go test -run TestImportRules ./internal/arch/

  # A test that writes golden files as a side effect is the only way CI could
  # silently accept a rendering change. Fail if the tree moved. Locally this
  # also fails on golden changes you have not staged yet (`git diff` compares
  # the working tree with the index), which is the same question CI asks of the
  # pushed tree.
  git diff --exit-code -- internal/tui/testdata
}

job_security() {
  use_ci_toolchain

  # Known vulnerabilities in dependencies and in the standard library, with
  # reachability analysis. A new advisory fails the build.
  run_tool install_go_tool govulncheck golang.org/x/vuln/cmd/govulncheck@latest -- ./...

  # Static analysis. CI's policy is that every finding is either fixed or
  # carries a #nosec with a written reason; an unannotated finding fails the
  # job. -exclude-dir=.claude skips agent scratch, which is untracked and so
  # absent from a CI checkout (the flag changes nothing there) but present in a
  # local working tree. It matches `npm run security:sast`.
  run_tool install_go_tool gosec github.com/securego/gosec/v2/cmd/gosec@latest -- -quiet -exclude-generated -exclude-dir=.claude ./...

  bash scripts/secret-scan.sh

  # The seed corpus only. Real fuzzing is `npm run fuzz`.
  go test -run 'Fuzz' ./...
}

job_screens() {
  python3 scripts/lint-screens.py
}

# Each job runs in its own bash process, not a subshell: a subshell used as an
# `if` condition silently turns `set -e` off, and a failing step would then be
# skipped over instead of failing its job.
job_all() {
  local failed="" j
  for j in lint test security screens; do
    echo
    echo "=== ci.sh $j ==="
    if ! bash "$here/ci.sh" "$j"; then
      failed="$failed $j"
    fi
  done
  echo
  if [ -n "$failed" ]; then
    echo "ci.sh: FAILED jobs:$failed" >&2
    exit 1
  fi
  echo "ci.sh: all jobs passed"
}

case "$job" in
  lint)     job_lint ;;
  test)     job_test ;;
  security) job_security ;;
  screens)  job_screens ;;
  all)      job_all ;;
  *)        usage ;;
esac
