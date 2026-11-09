#!/usr/bin/env bash
# Collect the licence texts of everything linked into ./cmd/kirsch, for the
# release archive.
#
#   bash scripts/third-party-licenses.sh
#
# The script takes no arguments and refuses any. It always writes to
# build/licenses under the repository the script itself sits in, so there is no
# caller-supplied path to point it at the wrong directory.
#
# For every module other than this one in `go list -deps ./cmd/kirsch`, every
# top-level file in the module's directory whose name starts with LICENSE,
# LICENCE, COPYING or NOTICE (any case) is copied to
# build/licenses/<module path>/. Go's own LICENSE goes to
# build/licenses/go/LICENSE, because the binary carries the Go runtime. The
# texts come from the module cache at run time; none are kept in the
# repository.
#
# The module list is the union of `go list` over every goos x goarch pair in
# .goreleaser.yaml with CGO_ENABLED=0, the way goreleaser builds, so a module
# imported under a build tag on one target only is still covered. The pairs are
# read from .goreleaser.yaml, not listed here.
#
# build/licenses is removed first, so a module dropped from the build leaves no
# stale licence behind. Nothing else is deleted: the path is built from the
# script's own location, and the script exits 2 if the repository root (the
# parent of the directory the script sits in) has no go.mod, or if build or
# build/licenses is a symlink. A symlinked build would be followed: rm -rf
# would delete the licenses directory inside its target. A symlinked
# build/licenses would not be followed: rm -rf would remove only the link and
# leave its target alone. It is refused as a precaution all the same, because
# a link there means the output has been pointed somewhere unexpected.
#
# A module with no such file is an error: the script names it and exits 1, so a
# missing licence stops a release instead of shipping without one. This script
# does not set GOTOOLCHAIN; go list runs with the toolchain the build uses.
# .goreleaser.yaml runs it as a before hook and the archive ships the output
# under licenses/.
set -euo pipefail

if [ "$#" -ne 0 ]; then
  echo "usage: bash scripts/third-party-licenses.sh   (takes no arguments; writes build/licenses)" >&2
  exit 2
fi

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo="$(dirname "$here")"
out="$repo/build/licenses"

# Refuse to run from a copy that is not inside a checkout of this module.
if [ ! -f "$repo/go.mod" ]; then
  echo "third-party-licenses: no go.mod in '$repo'; run the script from its place in the repository" >&2
  exit 2
fi

# rm -rf -- "$out" follows a symlinked build, deleting the licenses directory
# in its target. It does not follow a symlinked build/licenses (it removes only
# the link); that case is refused as a precaution, not to prevent a deletion.
for p in "$repo/build" "$out"; do
  if [ -L "$p" ]; then
    echo "third-party-licenses: '$p' is a symlink; refusing to run" >&2
    exit 2
  fi
done

cd "$repo"

# Print the entries of a one-line flow list in .goreleaser.yaml, such as
# `goos: [darwin, linux]`, space separated. Twin of goreleaser_list in
# scripts/ci.sh, with the same sed pattern; this one only checks that the key
# gives one line with no empty entry, ci.sh also checks the shape of the file.
goreleaser_list() {
  local key="$1" lines
  lines="$(sed -n "s/^[[:space:]]*$key:[[:space:]]*\[\(.*\)\][[:space:]]*\$/\1/p" .goreleaser.yaml)"
  if [ -z "$lines" ] || [ "$(printf '%s\n' "$lines" | wc -l | tr -d ' ')" -ne 1 ]; then
    echo "third-party-licenses: .goreleaser.yaml must hold exactly one line like '$key: [a, b]' (found: '$lines')" >&2
    return 1
  fi
  printf '%s\n' "$lines" | awk -F, '{
    for (i = 1; i <= NF; i++) {
      t = $i; gsub(/^[ \t]+|[ \t]+$/, "", t)
      if (t == "" || t ~ /[ \t]/) bad = 1
      printf "%s%s", (i > 1 ? " " : ""), t
    }
    print ""
    exit bad
  }' || {
    echo "third-party-licenses: .goreleaser.yaml '$key' list has an empty or malformed entry" >&2
    return 1
  }
}

oses="$(goreleaser_list goos)"
arches="$(goreleaser_list goarch)"
if [ -z "$oses" ] || [ -z "$arches" ]; then
  echo "third-party-licenses: goos or goarch list in .goreleaser.yaml is empty" >&2
  exit 1
fi

# One line per module: "<path>@<version> <dir>". Module paths and cache
# directories contain no whitespace. Each go list is captured on its own so that
# a failing one stops the script, which a process substitution would not.
mods=""
for os in $oses; do
  for arch in $arches; do
    one="$(GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 go list -deps -f '{{with .Module}}{{if not .Main}}{{.Path}}@{{.Version}} {{.Dir}}{{end}}{{end}}' ./cmd/kirsch)"
    if [ -z "$one" ]; then
      echo "third-party-licenses: go list found no modules linked into ./cmd/kirsch for $os/$arch" >&2
      exit 1
    fi
    mods="$mods$one"$'\n'
  done
done
mods="$(printf '%s' "$mods" | sort -u)"

# The one deletion in this script, on the fixed path built above.
rm -rf -- "$out"
mkdir -p "$out"

count=0
while read -r modver dir; do
  path="${modver%@*}"
  dest="$out/$path"
  found=0
  while IFS= read -r f; do
    mkdir -p "$dest"
    cp "$f" "$dest/"
    found=$((found + 1))
  done < <(find "$dir" -maxdepth 1 -type f \( -iname 'LICENSE*' -o -iname 'LICENCE*' -o -iname 'COPYING*' -o -iname 'NOTICE*' \))
  if [ "$found" -eq 0 ]; then
    echo "third-party-licenses: no licence file in $modver ($dir)" >&2
    exit 1
  fi
  count=$((count + 1))
done <<<"$mods"

goroot="$(go env GOROOT)"
if [ ! -f "$goroot/LICENSE" ]; then
  echo "third-party-licenses: no LICENSE in GOROOT ($goroot)" >&2
  exit 1
fi
mkdir -p "$out/go"
cp "$goroot/LICENSE" "$out/go/LICENSE"

echo "third-party-licenses: $count modules + Go → ${out#"$repo"/}"
