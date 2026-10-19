#!/usr/bin/env bash
# Run the smoke test against a real configured provider.
#
# The test requires a valid API key to be set in one of:
#   ANTHROPIC_API_KEY or KIRSCH_ANTHROPIC_API_KEY (for Anthropic)
#   OPENCODE_API_KEY or KIRSCH_OPENCODE_API_KEY (for OpenCode)
#
# The test has a 5-minute timeout and requires tests to pass.
# Exit code: 2 if no API key found; 1 if tests fail; 0 if all pass.
#
# Run this script only when ready to spend tokens on the API:
#   bash scripts/smoke-test.sh
set -euo pipefail

# Check for at least one API key.
if [[ -z "${ANTHROPIC_API_KEY:-}" && -z "${KIRSCH_ANTHROPIC_API_KEY:-}" && \
      -z "${OPENCODE_API_KEY:-}" && -z "${KIRSCH_OPENCODE_API_KEY:-}" ]]; then
  echo "No API key found. Set one of:" >&2
  echo "  ANTHROPIC_API_KEY or KIRSCH_ANTHROPIC_API_KEY" >&2
  echo "  OPENCODE_API_KEY or KIRSCH_OPENCODE_API_KEY" >&2
  exit 2
fi

# Run the live smoke test with -v and capture output to a temp file with cleanup.
output_file=$(mktemp)
trap "rm -f '$output_file'" EXIT

go test -tags smoke -count=1 -timeout 10m -run '^TestSmokeLive$' -v ./internal/app/ 2>&1 | tee "$output_file" || {
  exit_code=$?
  exit "$exit_code"
}

# Guard: fail if the test did not run.
# Use grep -qF -- to avoid parsing '--- PASS: TestSmokeLive' as a grep option.
if ! grep -qF -- '--- PASS: TestSmokeLive' "$output_file"; then
  echo "error: test did not pass (TestSmokeLive result not found)" >&2
  exit 1
fi
