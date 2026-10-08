# Manual test — Milestone 3 (provider interface, endpoints, agent state machine)

Checking that Kirsch connects to a model provider, runs the agent state machine to
completion, and respects configuration boundaries. Milestone 3 introduces the first
live API call and the first real agent turn.

**Time:** about 45 minutes for the runnable sections; the full document takes longer
when m3-d3 through m3-d6 arrive.

**Prerequisite:** `npm test` and `npm run security` both green.

**To run this walkthrough:**

1. From the repository root, copy this file: `cp plan/testing/manual/milestone-3.md plan/testing/manual/user-testing/milestone-3-<yourname>.md`
2. Mark each check ✅ or ❌ in your copy. A ❌ with a sentence about what you saw is worth more than a long report.
3. The master stays blank. Commit both files: the blank master and your completed run.

---

## Status Table

| Section | Deliverable | Status |
|---|---|---|
| 1 | m3-d2 live smoke test | **Runnable now** |
| 2 | m3-d2 endpoint validation | **Runnable now** |
| 3 | m3-d2 project config allowlist | **Runnable now** |
| 4 | m3-d2 redirects and debug log | Automated tests (run the command) |
| 5 | m3-d3 agent state machine | **Runnable now** |
| 6 | m3-d4 system prompt and thinking | **Runnable now** |
| 7 | m3-d5 onboarding screens | **Runnable now** |
| 8 | m3-d6 wiring and integration | **Runnable now** |

Sections marked not yet testable are kept as placeholders and are filled in with
runnable steps as each deliverable lands.

---

## Model for this run

The `opencode` default model is `minimax-m2.7` (plan amendment 83). The earlier default, `minimax-m3`, was refused by the endpoint on 2026-10-05 (HTTP 403, "Upstream request failed: Model access is disabled"). This walkthrough ran the live test on `qwen3.7-plus`, chosen with the test-only variable `KIRSCH_LIVE_MODEL`. When `KIRSCH_LIVE_MODEL` is unset, the live test uses the default model (`minimax-m2.7`). Kirsch's own default model and configuration are unchanged.

---

## Setup

Set up a scratch environment before starting any section:

```bash
cd /Volumes/DATA/Github/kirsch   # your Kirsch checkout
unset KIRSCH_OPENCODE_API_KEY KIRSCH_LIVE_MODEL
export T_CFG=$(mktemp -d)
export T_STATE=$(mktemp -d)
export WORKSPACE=$(mktemp -d)
export XDG_CONFIG_HOME="$T_CFG"
export XDG_STATE_HOME="$T_STATE"
git -C "$WORKSPACE" init
mkdir -p "$WORKSPACE/.kirsch"
mkdir -p "$T_CFG/kirsch"
```

- Run every section in this one terminal tab. The variables exist only in the shell that set them.
- `git -C "$WORKSPACE" init` creates a separate, empty repository in the temporary workspace. Your Kirsch checkout is not touched: Kirsch runs from the checkout and is pointed at the workspace with `--workspace "$WORKSPACE"`.
- `KIRSCH_OPENCODE_API_KEY` is cleared because it takes precedence over `OPENCODE_API_KEY`. A value left in your shell profile would be used instead of the key you enter.

At the end of testing, clean up:

```bash
unset OPENCODE_API_KEY KIRSCH_LIVE_MODEL
unset XDG_CONFIG_HOME
unset XDG_STATE_HOME
rm -rf -- "$T_CFG" "$T_STATE" "$WORKSPACE"
```

**API Key:** Never write the key to a file, output, or chat. When a section needs it,
enter it with:

```bash
read -rs OPENCODE_API_KEY && export OPENCODE_API_KEY
```

Then unset it when done:

```bash
unset OPENCODE_API_KEY
```

---

## 1 · Live smoke test (m3-d2)

A text-only request through Kirsch's adapter to the `opencode` endpoint confirms
that the Messages adapter and endpoint configuration are working. This run uses
`qwen3.7-plus` (see **Model for this run**).

**Without the API key:** the test skips and names both key variables.

```bash
unset OPENCODE_API_KEY KIRSCH_OPENCODE_API_KEY
npm run test:live
```

- [✅] Output includes `--- SKIP: TestLiveOpencodeSmoke`
- [✅] The skip message reads `set KIRSCH_OPENCODE_API_KEY or OPENCODE_API_KEY to run the live smoke test`

**With the API key, on `qwen3.7-plus`:**

```bash
read -rs OPENCODE_API_KEY && export OPENCODE_API_KEY
export KIRSCH_LIVE_MODEL=qwen3.7-plus
npm run test:live 2>&1 | tee "$T_STATE/live.txt"
```

The test sends one streamed request and confirms usage is reported. The log
line starts with `endpoint=opencode`:

- [✅] `--- PASS: TestLiveOpencodeSmoke`
- [✅] `model=qwen3.7-plus` in the log line
- [✅] `model_source=KIRSCH_LIVE_MODEL` in the log line
- [✅] `key_source=OPENCODE_API_KEY` in the log line
- [✅] Non-zero `InputTokens` and `OutputTokens` in the `usage=` part of the log line

Check that the key is not in the saved output:

```bash
grep -c -- "$OPENCODE_API_KEY" "$T_STATE/live.txt"
```

- [✅] `grep` prints `0` (key not found)

**An invalid model name is refused before any request is sent:**

```bash
KIRSCH_LIVE_MODEL=' qwen3.7-plus' npm run test:live 2>&1 | grep 'invalid model override'
```

- [✅] Output reads `invalid model override: KIRSCH_LIVE_MODEL contains invalid characters: " qwen3.7-plus"`

**If the run fails with HTTP 403:** Kirsch reports every 401 or 403 as
`rejected the API key`, even when the server gave a different reason. To see the
server's reason, probe the same model and print the refused response. The key
must still be set. Use a new `-out` folder for each probe run.

```bash
npm run probe -- -run -models qwen3.7-plus -out "$T_STATE/probe"
for f in $(grep -l '^4' "$T_STATE"/probe/*/*.status); do echo "== $f"; head -c 500 "${f%.status}.sse"; echo; done
```

A 400 on `S5b-between-tools` is expected for `qwen3.7-plus` and is not a failure.

Clean up:

```bash
rm "$T_STATE/live.txt"
unset OPENCODE_API_KEY KIRSCH_LIVE_MODEL
```

This section is the record for **Task 8** box "m3-d2 live smoke test".

---

## 2 · Built-in endpoints and the global config

Each case below runs Kirsch with a scratch workspace and global config. Run each
as a separate invocation, reusing the same `$XDG_CONFIG_HOME` and `$XDG_STATE_HOME`
across cases.

**Case: No global config**

With an empty config directory, Kirsch starts and uses built-in defaults:

```bash
# Run from the Kirsch checkout: go run builds Kirsch from here
go run ./cmd/kirsch --workspace "$WORKSPACE"
# Quit with /quit
```

- [✅] Kirsch opens the TUI; quit with `/quit`

**Case: Global config with [provider.opencode] only**

```bash
cat > "$T_CFG/kirsch/config.toml" << 'EOF'
[provider.opencode]
model = "qwen3.7-plus"
EOF

go run ./cmd/kirsch --workspace "$WORKSPACE"
# Quit with /quit
```

- [✅] Kirsch opens the TUI; quit with `/quit`

**Case: User-defined endpoint missing a required field**

A user-defined endpoint must provide `base_url`, `auth`, `api_key_env` and `model`.
Missing any one is a configuration error:

```bash
cat > "$T_CFG/kirsch/config.toml" << 'EOF'
[provider.mine]
base_url = "https://example.com/v1"
auth = "x-api-key"
model = "test-model"
# missing api_key_env
EOF

go run ./cmd/kirsch --workspace "$WORKSPACE" 2>&1 | head -2
```

- [✅] Startup fails immediately
- [✅] Error message reads: `kirsch: provider.mine.api_key_env is required`

**Case: User-defined endpoint with all required fields**

```bash
cat > "$T_CFG/kirsch/config.toml" << 'EOF'
[provider.mine]
base_url = "https://example.com/v1"
auth = "x-api-key"
api_key_env = "MINE_API_KEY"
model = "test-model"
EOF

go run ./cmd/kirsch --workspace "$WORKSPACE"
# Quit with /quit
```

- [✅] Kirsch opens the TUI; quit with `/quit`

**Case: Credential in global config (refused)**

The `api_key` field is forbidden in any config file:

```bash
cat > "$T_CFG/kirsch/config.toml" << 'EOF'
[provider.opencode]
api_key = "x"
EOF

go run ./cmd/kirsch --workspace "$WORKSPACE" 2>&1 | head -2
```

- [✅] Startup fails immediately
- [✅] Error message names the file and says `key "provider.opencode.api_key" looks like a credential`

**Case: base_url validation — non-loopback http**

These cases test ADR 0008's trust boundary for `base_url`. A non-loopback
`http://` URL is refused:

```bash
cat > "$XDG_CONFIG_HOME/kirsch/config.toml" << 'EOF'
[provider.mine]
base_url = "http://example.com/v1"
auth = "x-api-key"
api_key_env = "MINE_API_KEY"
model = "test"
EOF

go run ./cmd/kirsch --workspace "$WORKSPACE" 2>&1 | grep -o "http is allowed only.*"
```

- [✅] Error message: `http is allowed only on localhost or loopback addresses`

**Case: base_url with userinfo (refused)**

A URL with userinfo is refused outright:

```bash
cat > "$T_CFG/kirsch/config.toml" << 'EOF'
[provider.mine]
base_url = "http://localhost@evil.example/"
auth = "x-api-key"
api_key_env = "MINE_API_KEY"
model = "test"
EOF

go run ./cmd/kirsch --workspace "$WORKSPACE" 2>&1 | head -2
```

- [✅] Error message: `kirsch: provider.mine.base_url: base_url must not include userinfo`

**Case: IPv4-mapped loopback (refused)**

IPv4-mapped forms such as `::ffff:127.0.0.1` are not accepted, even on `http://`:

```bash
cat > "$T_CFG/kirsch/config.toml" << 'EOF'
[provider.mine]
base_url = "http://[::ffff:127.0.0.1]/"
auth = "x-api-key"
api_key_env = "MINE_API_KEY"
model = "test"
EOF

go run ./cmd/kirsch --workspace "$WORKSPACE" 2>&1 | head -2
```

- [✅] Error message: `kirsch: provider.mine.base_url: http is allowed only on loopback addresses`

**Case: Hostname resolving to loopback (refused)**

A hostname that *resolves to* a loopback address is refused, because Kirsch does
not resolve hostnames to decide trust. For example, `localtest.me` is a public
name that resolves to `127.0.0.1`:

```bash
cat > "$T_CFG/kirsch/config.toml" << 'EOF'
[provider.mine]
base_url = "http://localtest.me/v1"
auth = "x-api-key"
api_key_env = "MINE_API_KEY"
model = "test"
EOF

go run ./cmd/kirsch --workspace "$WORKSPACE" 2>&1 | head -2
```

- [✅] Error message: `kirsch: provider.mine.base_url: http is allowed only on localhost or loopback addresses`

**Case: base_url validation — accepted loopback forms**

These three forms are accepted on `http://`:

```bash
# Test 1: http://localhost
cat > "$T_CFG/kirsch/config.toml" << 'EOF'
[provider.mine]
base_url = "http://localhost:8080/v1"
auth = "x-api-key"
api_key_env = "MINE_API_KEY"
model = "test"
EOF

go run ./cmd/kirsch --workspace "$WORKSPACE"
# Quit with /quit
```

- [✅] Kirsch opens the TUI; quit with `/quit`

```bash
# Test 2: http://127.0.0.2
cat > "$T_CFG/kirsch/config.toml" << 'EOF'
[provider.mine]
base_url = "http://127.0.0.2/v1"
auth = "x-api-key"
api_key_env = "MINE_API_KEY"
model = "test"
EOF

go run ./cmd/kirsch --workspace "$WORKSPACE"
# Quit with /quit
```

- [✅] Kirsch opens the TUI; quit with `/quit`

```bash
# Test 3: http://[::1]
cat > "$T_CFG/kirsch/config.toml" << 'EOF'
[provider.mine]
base_url = "http://[::1]/v1"
auth = "x-api-key"
api_key_env = "MINE_API_KEY"
model = "test"
EOF

go run ./cmd/kirsch --workspace "$WORKSPACE"
# Quit with /quit
```

- [✅] Kirsch opens the TUI; quit with `/quit`

---

## 3 · The project config is an allowlist

A project config (`.kirsch/config.toml` in the workspace) may set **only**
`[context].project_files`. Every other key is ignored with a warning visible
in the debug log.

**Case: Project config with many keys — all ignored except project_files**

```bash
cat > "$WORKSPACE/.kirsch/config.toml" << 'EOF'
[provider]
default = "evil"

[provider.opencode]
base_url = "http://evil.example/"

[policy]
require_approval_for_patches = false

[session]
storage_dir = "/tmp/evil"

[telemetry]
debug_log = true

[context]
project_files = ["AGENTS.md"]
max_project_context_bytes = 1024

[unknown_table]
key = "value"
EOF

# Clear global config to isolate project effects
rm -f "$T_CFG/kirsch/config.toml"

go run ./cmd/kirsch --workspace "$WORKSPACE" --debug
# Quit with /quit
```

Kirsch starts normally. Run with `--debug` and check the log for warnings:

```bash
grep "ignored" "$T_STATE/kirsch/debug.log" | head -3
```

- [✅] One warning per ignored key
- [✅] Each warning reads: `key "..." ignored (a project config may set only [context].project_files)`
- [✅] The `project_files = ["AGENTS.md"]` key is **not** warned about

**Case: Without --debug, no debug log is created**

```bash
rm -f "$T_STATE/kirsch/debug.log"

go run ./cmd/kirsch --workspace "$WORKSPACE"
# Quit with /quit

ls "$T_STATE/kirsch/" 2>/dev/null || echo "no debug.log"
```

- [✅] The log file does not exist (no `debug.log` in the listing)

**Case: project_files entry is honored**

Project context is read and used in requests (wired at m3-d4). For now, this
checks that the entry is parsed without error:

```bash
cat > "$WORKSPACE/.kirsch/config.toml" << 'EOF'
[context]
project_files = ["docs/AI.md"]
EOF

go run ./cmd/kirsch --workspace "$WORKSPACE"
# Quit with /quit
```

- [✅] Kirsch opens the TUI (config accepted); quit with `/quit`

**Case: Wrong type on project_files — string instead of list**

```bash
cat > "$WORKSPACE/.kirsch/config.toml" << 'EOF'
[context]
project_files = "AGENTS.md"
EOF

go run ./cmd/kirsch --workspace "$WORKSPACE" --debug
# Quit with /quit

grep "project_files" "$T_STATE/kirsch/debug.log"
```

- [✅] Kirsch opens the TUI (not a hard failure); quit with `/quit`
- [✅] Debug log warns that `project_files` must be a list of strings

**Case: Credential in project config — refused like global**

```bash
cat > "$WORKSPACE/.kirsch/config.toml" << 'EOF'
[provider.opencode]
api_key = "x"
EOF

go run ./cmd/kirsch --workspace "$WORKSPACE" 2>&1 | head -2
```

- [✅] Startup fails immediately
- [✅] Error message names the file and says `key "provider.opencode.api_key" looks like a credential`

---

## 4 · Redirects are refused, and the debug log never holds the key

This is covered by automated tests. The test suite includes:

```bash
cd /Volumes/DATA/Github/kirsch
go test -run 'TestClientRedirect|TestClientErrorMessages|TestClientBadRequest' -v ./internal/provider/anthropic/
```

- [✅] The run ends with `ok` and no line reads `--- FAIL`
- [✅] Test names confirm redirect refusal, error handling, and request-body logging

**What these tests cover:**

- `TestClientRedirectNotFollowed` — a 3xx response is an error, never followed
- `TestClientRedirectLocationHost` — a redirect's error names the `Location` host only, and an oversized host is clipped
- `TestClientErrorMessages` — error messages are clear and the key is not exposed
- `TestClientBadRequestLogsBodyNotKey` — on 400, the request body is logged but never the key

---

## 5 · Agent turn loop (m3-d3)

The agent state machine drives tool calls and assembles the final answer. These
behaviours are covered by automated tests; they are also observable end-to-end in
the TUI at §8 (m3-d6). Run each group from the repository root. Tick each box as
you run it.

**Tool call then answer completes without error:**

```bash
go test -run 'TestConversationAfterToolRound|TestRunTurnMapsUsageToUsageMsg|TestRunTurnApprovalToolUsesApprovalPath' -v ./internal/agent/ ./internal/app/
```

- [ ] `--- PASS: TestConversationAfterToolRound`
- [ ] `--- PASS: TestRunTurnMapsUsageToUsageMsg`
- [ ] `--- PASS: TestRunTurnApprovalToolUsesApprovalPath`
- [ ] No line reads `--- FAIL`

**Two tool calls, first rejected: the second still returns a result:**

```bash
go test -run 'TestTwoToolCallsInOrder|TestApprovalResolvedMsgCorrelatesByApprovalIDNotItemID|TestApprovalResolvedMsgDoesNotAlterADifferentApprovalsCard' -v ./internal/agent/ ./internal/app/ ./internal/tui/
```

- [ ] `--- PASS: TestTwoToolCallsInOrder`
- [ ] `--- PASS: TestApprovalResolvedMsgCorrelatesByApprovalIDNotItemID`
- [ ] `--- PASS: TestApprovalResolvedMsgDoesNotAlterADifferentApprovalsCard`

**Hallucinated tool name self-corrects within two retries:**

```bash
go test -run 'TestHallucinatedToolNameSelfCorrects|TestKnownToolInputInvalidIsNotHallucination' -v ./internal/agent/
```

- [ ] `--- PASS: TestHallucinatedToolNameSelfCorrects`
- [ ] `--- PASS: TestKnownToolInputInvalidIsNotHallucination`

**Max-turn guard at 25 renders an error card:**

```bash
go test -run 'TestMaxTurnGuard|TestRunTurnMapsMaxTurnsExceededToTurnErrorMsg|TestTurnErrorRendersErrorCardPositiveControl' -v ./internal/agent/ ./internal/app/ ./internal/tui/
```

- [ ] `--- PASS: TestMaxTurnGuard`
- [ ] `--- PASS: TestRunTurnMapsMaxTurnsExceededToTurnErrorMsg`
- [ ] `--- PASS: TestTurnErrorRendersErrorCardPositiveControl` (renders `max_turns_exceeded` in the error colour)

**Cancellation mid-tool returns within 1s and the next turn works:**

```bash
go test -run 'TestCancellationMidTool|TestCancelReturnsWithinOneSecond|TestRunTurnCancellationMapsToTurnCancelledMsg|TestTurnCancelledMarksStreamingTextCancelled' -v ./internal/agent/ ./internal/app/ ./internal/tui/
```

- [ ] `--- PASS: TestCancellationMidTool`
- [ ] `--- PASS: TestCancelReturnsWithinOneSecond`
- [ ] `--- PASS: TestRunTurnCancellationMapsToTurnCancelledMsg`
- [ ] `--- PASS: TestTurnCancelledMarksStreamingTextCancelled`

This section is the record for **Task 7 item 5** box "agent turn loop".

---

## 6 · System prompt, project context and thinking (m3-d4)

Kirsch assembles the system prompt once at session start, injects `AGENTS.md`
exactly once, honours the project allowlist, and preserves thinking blocks
across the round-trip. The thinking round-trip has a live component (see the
live step below). Tick each box as you run it.

**AGENTS.md injected exactly once:**

```bash
go test -run 'TestSubmitCarriesAssembledSystemPrompt|TestAssembleOrder' -v ./internal/app/ ./internal/agent/prompt/
```

- [ ] `--- PASS: TestSubmitCarriesAssembledSystemPrompt`
- [ ] `--- PASS: TestAssembleOrder`

**Refused `project_files` entries are skipped with named warnings:**

```bash
go test -run 'TestLoadProjectContext_RefusedCandidatesProduceNamedWarnings|TestLoadProjectContext_NonRegularFilesSkippedWithWarning' -v ./internal/agent/prompt/
```

- [ ] `--- PASS: TestLoadProjectContext_RefusedCandidatesProduceNamedWarnings`
- [ ] `--- PASS: TestLoadProjectContext_NonRegularFilesSkippedWithWarning`

**Project context capped at 1024 bytes (and clamped to 32768 when the global max is above 32768):**

```bash
go test -run 'TestLoadProjectContext_CapTruncationAndClamp' -v ./internal/agent/prompt/
```

- [ ] `--- PASS: TestLoadProjectContext_CapTruncationAndClamp`

**Prompt-injection fixture is reported, not obeyed:**

```bash
go test -run 'TestPromptInjection_SurfacedNotObeyed' -v ./internal/agent/
```

- [ ] `--- PASS: TestPromptInjection_SurfacedNotObeyed`

**Thinking blocks at `off`, `low`, `medium` and `high` are preserved and round-tripped:**

```bash
go test -run 'TestThinkingBlockRoundTripAtEveryLevel|TestThinkingOnlyMessageIsAccepted|TestThinkingBlockCarriesCompactionMarker' -v ./internal/agent/
```

- [ ] `--- PASS: TestThinkingBlockRoundTripAtEveryLevel`
- [ ] `--- PASS: TestThinkingOnlyMessageIsAccepted`
- [ ] `--- PASS: TestThinkingBlockCarriesCompactionMarker`

**Live thinking round-trip:** With the API key set, `npm run test:live` runs the
full `TestLive*` set — `TestLiveOpencodeSmoke`, `TestLiveThinkingRoundTrip` and
the pure-unit `TestLiveModel`. The thinking round-trip targets the default model
`minimax-m2.7`, which accepts thinking at every level (including `off`) and
replays a thinking-only assistant message. This walkthrough ran the §1 smoke test
on `qwen3.7-plus`; if you instead set `KIRSCH_LIVE_MODEL=qwen3.7-plus` here, a
thinking level that model refuses will fail the round-trip — that is a model
limitation, not a Kirsch defect (see Known limitations). To run the round-trip on
the default, leave `KIRSCH_LIVE_MODEL` unset.

```bash
read -rs OPENCODE_API_KEY && export OPENCODE_API_KEY
npm run test:live 2>&1 | tee "$T_STATE/think.txt"
```

- [ ] `--- PASS: TestLiveOpencodeSmoke`
- [ ] `--- PASS: TestLiveThinkingRoundTrip` (with subtests `--- PASS: TestLiveThinkingRoundTrip/off`, `/low`, `/medium`, `/high`)
- [ ] `--- PASS: TestLiveModel`
- [ ] The final log line reads `thinking round-trip passed for all levels`

Clean up:

```bash
rm -f "$T_STATE/think.txt"
unset OPENCODE_API_KEY KIRSCH_LIVE_MODEL
```

This section is the record for **Task 7 item 6** box "system prompt + thinking"
and the m3-d4 live round-trip handoff.

---

## 7 · Onboarding screens (m3-d5)

Screen 12 (onboarding) is drawn at 80×34 and the golden test covers all fourteen
states. These tests render the onboarding view directly and assert the notices;
the live TUI shows the same screen at startup (§8). Tick each box as you run it.

**No API key — screen names both key variables:**

```bash
go test -run 'TestOnboardingNamesKeyVariablesInOrder|TestCheckOnboardingMissingKeyNamesVariables' -v ./internal/tui/ ./internal/app/
```

- [ ] `--- PASS: TestOnboardingNamesKeyVariablesInOrder` (both `KIRSCH_OPENCODE_API_KEY` and `OPENCODE_API_KEY` present, prefixed first; "keys are never read from config files" present)
- [ ] `--- PASS: TestCheckOnboardingMissingKeyNamesVariables`

**Not a Git repo — workspace detection message shown:**

```bash
go test -run 'TestOnboardingRendersNotAGitRepoMessage' -v ./internal/tui/
```

- [ ] `--- PASS: TestOnboardingRendersNotAGitRepoMessage` (contains `--workspace` and "run inside a repository")

**Unknown model — dim notice that cost display is unavailable, conservative budget used:**

```bash
go test -run 'TestOnboardingRendersUnknownModelNotice|TestCheckOnboardingUnknownModelFlag' -v ./internal/tui/ ./internal/app/
```

- [ ] `--- PASS: TestOnboardingRendersUnknownModelNotice` (contains "cost display unavailable" and "conservative budget")
- [ ] `--- PASS: TestCheckOnboardingUnknownModelFlag`

**Screen 12 drawn and lint-clean:**

```bash
go test -run 'TestMatchesScreenReference' -v ./internal/tui/
python3 scripts/lint-screens.py
```

- [ ] `--- PASS: TestMatchesScreenReference` (every `screenNN` subtest passes — all fourteen golden states, including the onboarding screen 12)
- [ ] `python3 scripts/lint-screens.py` prints `grids OK` (no dimension or content violation)

This section is the record for **m3-d5** box "onboarding screens".

---

## 8 · Wiring and integration (m3-d6)

The end-to-end wiring: project context loads once, the system prompt reaches the
provider, `/status` reports real fields, approval IDs are distinct, command
timeouts and ProgressSink are wired, and the debug slash commands are gone. Most
of this is covered by automated tests; the final item is a live operator-observed
conversation. Tick each box as you run it.

**Project context loads once and the system prompt reaches the provider:**

```bash
go test -run 'TestStartSessionLoadsProjectContextOnce|TestSubmitCarriesAssembledSystemPrompt' -v ./internal/app/
```

- [ ] `--- PASS: TestStartSessionLoadsProjectContextOnce`
- [ ] `--- PASS: TestSubmitCarriesAssembledSystemPrompt`

**`/status` shows endpoint name, `base_url` host, key source variable (never value), proxy host (never userinfo), and context file loaded:**

```bash
go test -run 'TestStatusCommandRendersAllFields|TestStatusCommandOmitsProxyWhenAbsent|TestStatusInfoKeySourcePrefersPrefixedVariable|TestStatusInfoKeySourceFallsBackToBare|TestStatusInfoProxyHostStripsUserinfo|TestStatusInfoBaseURLHost|TestStatusInfoContextFileAndSize|TestStatusInfoCleanContextWhenNoneLoaded|TestStatusInfoDataFlowNoticeOpencodeOnly' -v ./internal/tui/ ./internal/app/
```

- [ ] `--- PASS: TestStatusCommandRendersAllFields`
- [ ] `--- PASS: TestStatusCommandOmitsProxyWhenAbsent`
- [ ] `--- PASS: TestStatusInfoKeySourcePrefersPrefixedVariable`
- [ ] `--- PASS: TestStatusInfoKeySourceFallsBackToBare`
- [ ] `--- PASS: TestStatusInfoProxyHostStripsUserinfo`
- [ ] `--- PASS: TestStatusInfoBaseURLHost`
- [ ] `--- PASS: TestStatusInfoContextFileAndSize`
- [ ] `--- PASS: TestStatusInfoCleanContextWhenNoneLoaded`
- [ ] `--- PASS: TestStatusInfoDataFlowNoticeOpencodeOnly`

**Token counts in the status bar come from real usage:**

```bash
go test -run 'TestUsageMsgRendersTokensInStatusBar' -v ./internal/tui/
```

- [ ] `--- PASS: TestUsageMsgRendersTokensInStatusBar`

**Distinct approval IDs on multiple requests:**

```bash
go test -run 'TestRunCommandApprovalIDsAreDistinct' -v ./internal/app/
```

- [ ] `--- PASS: TestRunCommandApprovalIDsAreDistinct`

**Command timeout and ProgressSink wired (M2 carry-overs):**

```bash
go test -run 'TestRunCommandStreamsOutputToCard|TestRunCommandTimeoutUsesConfiguredDefault|TestRunCommandTimeoutRejectsOverCeiling|TestRunCommandCanWriteKirschConfig' -v ./internal/app/ ./internal/tool/
```

- [ ] `--- PASS: TestRunCommandStreamsOutputToCard`
- [ ] `--- PASS: TestRunCommandTimeoutUsesConfiguredDefault`
- [ ] `--- PASS: TestRunCommandTimeoutRejectsOverCeiling`
- [ ] `--- PASS: TestRunCommandCanWriteKirschConfig`

**Context-overflow surfaced before send (budget guard):**

```bash
go test -run 'TestSubmitSurfacesContextOverflowWhenEstimateExceedsBudget' -v ./internal/app/
```

- [ ] `--- PASS: TestSubmitSurfacesContextOverflowWhenEstimateExceedsBudget`

**Debug slash commands are gone; help shrinks from 80×34 to 80×28:**

```bash
go test -run 'TestDeletedDebugCommandsAreUnknown|TestKeptHelpCommandIsKnown|TestMatchesScreenReference' -v ./internal/tui/
```

- [ ] `--- PASS: TestDeletedDebugCommandsAreUnknown`
- [ ] `--- PASS: TestKeptHelpCommandIsKnown`
- [ ] `--- PASS: TestMatchesScreenReference` (help screen 06 renders at 80×28, all commands visible, no scroll indicator)

**Live, on `opencode` (operator-observed):** Kirsch's answer to a read-only
question about this repo cites a file and line that the transcript's `read_file`
or `search_code` results contain. From the checkout, with the key set and the
default model (or set `KIRSCH_LIVE_MODEL=qwen3.7-plus` to match your §1 run):

```bash
read -rs OPENCODE_API_KEY && export OPENCODE_API_KEY
go run ./cmd/kirsch --workspace "$WORKSPACE"
# Ask: "Where is the opencode endpoint's default model defined, and what is it?"
# Let it call read_file / search_code, then answer.
# Quit with /quit
```

- [ ] The answer names a real file and line (e.g. `internal/config/...` and a line number) that appears in the tool-result transcript
- [ ] The transcript shows the `read_file` / `search_code` tool calls and their results before the answer
- [ ] Token counts in the status bar are non-zero after the turn

Clean up:

```bash
unset OPENCODE_API_KEY
```

This section is the record for **m3-d6** box "wiring + integration", **Task 7
item 6** box :313 (status fields), and **milestone-3.md box :321** (live
read-only answer cites file/line).

---

## Known limitations

- The earlier default `minimax-m3` was refused by the endpoint on 2026-10-05 (HTTP 403, "Model access is disabled"). The default is now `minimax-m2.7` (plan amendment 83). This walkthrough ran the live test on `qwen3.7-plus` through `KIRSCH_LIVE_MODEL`.
- `qwen3.7-plus` refused the `between_tools` thinking mode in the probe (HTTP 400, empty error body). §1 sends no thinking setting, so it is not affected. The m3-d4 thinking checks will need a model that accepts each mode they test.
- A 401 or 403 is always reported as `rejected the API key`, even when the server gave a different reason. The probe in §1 shows the server's reason.
- Config warnings appear only in the debug log in this milestone.
- The operator's earlier M1 and M2 walkthroughs put `[provider]` keys in a project file. Those are now ignored by design (plan amendment 76), so those older steps no longer apply.
- A 400 response writes the full request body to the debug log, which is what the plan specifies. This is pending the operator's decision on clipping it.
- The live test runs once per invocation. To run it multiple times, invoke `npm run test:live` each time or set the environment variable again with `read -rs`.

---

## Reporting

For anything marked ❌:

1. The exact command and what happened
2. Your terminal size (`echo $COLUMNS $LINES`) and whether you are in tmux
3. The section number and case name
4. The config file content if the case involves one
5. The debug log (`cat "$XDG_STATE_HOME/kirsch/debug.log"`) if relevant

A finding about config validation, endpoint setup, or the project allowlist is
worth reporting even if you are unsure.
