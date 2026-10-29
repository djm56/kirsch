#!/usr/bin/env bash
# Build bin/kirsch, optionally stamping the version the binary reports.
#
#   bash scripts/build.sh            stamp only when this script's own repository
#                                    has HEAD exactly on a v<digit>... tag whose
#                                    version has the shape below, and `git status
#                                    --porcelain` succeeds and reports a clean
#                                    tree; otherwise build unstamped
#   bash scripts/build.sh 1.2.3      stamp 1.2.3 (a leading "v" is removed)
#
# Version shape, after one leading "v" is removed:
#   <digits>.<digits>.<digits>, optionally followed by -<suffix> or +<suffix>
#   where the suffix is made of letters, digits, ".", "+" and "-".
# An argument that fails the shape is an error (exit 2). A tag-derived value
# that fails it builds unstamped, so a tag such as v1-notes is never stamped.
#
# Git is used only when `git rev-parse --show-toplevel` is this script's own
# repository root, so a source tree unpacked inside some other repository, or
# outside any repository, builds unstamped. If `git status` itself fails, one
# line goes to stderr and the build is unstamped; the tree state is unknown.
#
# The stamp is the bare version: `kirsch --version` prints "kirsch <version>"
# and the TUI adds its own "v". An unstamped build reports devVersion from
# cmd/kirsch/main.go, unless Go's own build info records a clean release tag
# (see resolveVersion there). The repository's milestone tags (m3, m4, ...)
# and `git describe` strings with a -N-g<hash> suffix are never stamped.
#
# Silent on success, apart from the git-status-failure line; errors go to
# stderr with a non-zero exit.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."
root="$(pwd -P)"

version_re='^[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.+-]+)?$'

# own_repo succeeds when git's top-level directory is this script's repository
# root, compared as physical paths.
own_repo() {
  local top
  top="$(git rev-parse --show-toplevel 2>/dev/null)" || return 1
  [ "$(cd "$top" && pwd -P)" = "$root" ]
}

version=""
if [ "$#" -gt 1 ]; then
  echo "build.sh: expected at most one argument (the version), got $#" >&2
  exit 2
elif [ "$#" -eq 1 ]; then
  version="${1#v}"
  if ! [[ "$version" =~ $version_re ]]; then
    echo "build.sh: version argument '$1' is not X.Y.Z[-suffix|+suffix]" >&2
    exit 2
  fi
elif command -v git >/dev/null 2>&1 && own_repo; then
  # stderr is deliberately not discarded here: a failing `git status` must be
  # visible, and it means the tree state is unknown, so nothing is stamped.
  if ! dirty="$(git status --porcelain)"; then
    echo "build.sh: git status failed; building unstamped" >&2
  elif [ -z "$dirty" ]; then
    # --exact-match fails unless HEAD itself carries a matching tag; that miss
    # is the normal case, so its error text is dropped and tag stays empty.
    tag="$(git describe --tags --exact-match --match 'v[0-9]*' HEAD 2>/dev/null || true)"
    candidate="${tag#v}"
    if [[ "$candidate" =~ $version_re ]]; then
      version="$candidate"
    fi
  fi
fi

mkdir -p bin
if [ -n "$version" ]; then
  go build -ldflags "-X main.version=${version}" -o bin/kirsch ./cmd/kirsch
else
  go build -o bin/kirsch ./cmd/kirsch
fi
