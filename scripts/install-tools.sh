#!/usr/bin/env bash
# Install the development and security tooling Kirsch's checks depend on.
#
# Everything here is a Go program installed into the directory `go install`
# writes to. Nothing is vendored and nothing is required to build or run Kirsch
# itself — these are the tools that check it.
set -euo pipefail

# Where `go install` writes is GOBIN when set and GOPATH/bin otherwise, and
# scripts/go-tool.sh has to look in the same place this writes to. The rule
# lives in one script so the two cannot drift apart; resolving it relative to
# this file keeps this script runnable from any directory.
here="$(dirname "${BASH_SOURCE[0]}")"
bin="$(bash "$here/go-bin-dir.sh")"

echo "Installing into $bin"
echo

# Names accumulate here rather than in an array: this script has to run under
# the bash 3.2 that ships with macOS, where `set -u` and an empty array do not
# get along. Tool names contain no spaces, so splitting on whitespace is safe.
installed=""

install() {
  local name="$1" pkg="$2" why="$3"
  printf '  %-14s %s\n' "$name" "$why"
  go install "$pkg" >/dev/null 2>&1 || {
    echo "    FAILED: go install $pkg" >&2
    return 1
  }
  installed="$installed $name"
}

install govulncheck    "golang.org/x/vuln/cmd/govulncheck@latest" \
  "known CVEs in dependencies, with reachability analysis"
install gosec          "github.com/securego/gosec/v2/cmd/gosec@latest" \
  "static security analysis (SAST)"
install staticcheck    "honnef.co/go/tools/cmd/staticcheck@latest" \
  "correctness and simplification analysis"
# v2 bundles the formatters (gofmt, gofumpt, goimports, gci, golines) inside the
# same binary, so `golangci-lint fmt` needs nothing else installed.
install golangci-lint  "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest" \
  "linter aggregator and formatter, configured by .golangci.yml"
install osv-scanner    "github.com/google/osv-scanner/v2/cmd/osv-scanner@latest" \
  "dependency scanning against the OSV database"

echo
echo "Done. The npm scripts find these via scripts/go-tool.sh, so no PATH"
echo "change is needed to run:"
echo "    npm run check"
echo "    npm run security"
echo
echo "Add $bin to your PATH to call them directly from a shell."

# scripts/go-tool.sh deliberately prefers a copy already on PATH — an operator
# who has chosen their own build should keep it. The cost is that a different
# binary earlier on PATH silently wins over everything installed above, and the
# install appears to have done nothing. This is the one moment the two can be
# compared, so say it here rather than leaving it to be discovered.
shadowed=""
for name in $installed; do
  onpath="$(command -v "$name" 2>/dev/null || true)"
  # Compared by inode rather than by string. A PATH entry with a trailing
  # slash, or one reaching $bin through a symlinked directory, spells the very
  # file just installed a different way, and a string compare reads that
  # spelling as a foreign copy.
  if [ -n "$onpath" ] && [ ! "$onpath" -ef "$bin/$name" ]; then
    shadowed="$shadowed
    $name -> $onpath"
  fi
done

if [ -n "$shadowed" ]; then
  cat >&2 <<EOF

Note: a different copy of these is earlier on your PATH, and that is the one
the npm scripts will run:$shadowed

That is fine if it is deliberate. If it is not — or if a check later complains
about a version — remove the copy above, or put $bin ahead of it.
EOF
fi
