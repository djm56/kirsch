#!/usr/bin/env bash
# Scan the working tree for committed credentials.
#
# Kirsch refuses to read secrets from config files (internal/config) and redacts
# them from the debug log (internal/telemetry). This checks the other direction:
# that none have been committed to the repository itself.
#
# Deliberately dependency-free. gitleaks is better and worth installing, but a
# check that needs an install is a check that gets skipped.
set -euo pipefail

# testdata/repo-small/.env is a fixture: it exists precisely so the denylist has
# something real to refuse, and its "secret" is the string hunter2.
exclude='^(testdata/|plan/|.*_test\.go$|scripts/secret-scan\.sh$)'

patterns=(
  'sk-ant-[A-Za-z0-9_-]{20,}'
  'ghp_[A-Za-z0-9]{36}'
  'AKIA[0-9A-Z]{16}'
  '-----BEGIN [A-Z ]*PRIVATE KEY-----'
  'api[_-]?key["'"'"']?\s*[:=]\s*["'"'"'][A-Za-z0-9_-]{16,}'
)

found=0
files=$(git ls-files | grep -Ev "$exclude" || true)
[ -z "$files" ] && { echo "No files to scan."; exit 0; }

for pat in "${patterns[@]}"; do
  if hits=$(echo "$files" | xargs grep -nEI "$pat" 2>/dev/null); then
    echo "Possible credential matching /$pat/:"
    echo "$hits" | sed 's/^/  /'
    found=1
  fi
done

if [ "$found" -ne 0 ]; then
  echo
  echo "If one of these is a fixture rather than a real credential, add its path"
  echo "to the exclude pattern in this script and say why."
  exit 1
fi
echo "No committed credentials found in $(echo "$files" | wc -l | tr -d ' ') tracked files."
