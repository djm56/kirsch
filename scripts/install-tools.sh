#!/usr/bin/env bash
# Install the development and security tooling Kirsch's checks depend on.
#
# Everything here is a Go program installed into $(go env GOPATH)/bin. Nothing
# is vendored and nothing is required to build or run Kirsch itself — these are
# the tools that check it.
set -euo pipefail

bin="$(go env GOPATH)/bin"
echo "Installing into $bin"
echo

install() {
  local name="$1" pkg="$2" why="$3"
  printf '  %-14s %s\n' "$name" "$why"
  go install "$pkg" >/dev/null 2>&1 || {
    echo "    FAILED: go install $pkg" >&2
    return 1
  }
}

install govulncheck    "golang.org/x/vuln/cmd/govulncheck@latest" \
  "known CVEs in dependencies, with reachability analysis"
install gosec          "github.com/securego/gosec/v2/cmd/gosec@latest" \
  "static security analysis (SAST)"
install staticcheck    "honnef.co/go/tools/cmd/staticcheck@latest" \
  "correctness and simplification analysis"
install golangci-lint  "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest" \
  "linter aggregator, configured by .golangci.yml"
install osv-scanner    "github.com/google/osv-scanner/v2/cmd/osv-scanner@latest" \
  "dependency scanning against the OSV database"

echo
echo "Done. Ensure $bin is on your PATH, then run:"
echo "    npm run security"
