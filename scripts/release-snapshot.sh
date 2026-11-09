#!/usr/bin/env bash
# Build the release archives locally without publishing anything.
#
#   bash scripts/release-snapshot.sh     (or: npm run release:snapshot)
#
# Runs goreleaser in snapshot mode with --clean: it builds every target in
# .goreleaser.yaml, writes the archives and checksums file to dist/, and
# creates no release and pushes nothing. The goreleaser version is the one in
# scripts/goreleaser-version, the same pin the release workflow reads. The
# tool is run with `go run`, so nothing is installed; the first run downloads
# goreleaser and, if go.mod's toolchain is older than goreleaser needs, the
# toolchain it asks for. The version is the same as the workflow's, but the
# binary differs: this builds goreleaser from source through `go run`, while the
# workflow runs the release binary fetched by the action.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo="$(dirname "$here")"

ver="$(tr -d '[:space:]' <"$here/goreleaser-version")"
if ! printf '%s\n' "$ver" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
  echo "release-snapshot: scripts/goreleaser-version must hold a version like v2.19.0 (got: '$ver')" >&2
  exit 1
fi

cd "$repo"
exec go run "github.com/goreleaser/goreleaser/v2@$ver" release --snapshot --clean
