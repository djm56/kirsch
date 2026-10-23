# Manual test — Milestone 4

Kirsch Milestone 4 adds session persistence, resumption, compaction, and the real task loop. This document verifies every feature systematically through automated tests and hands-on walkthroughs.

## How to read this document

- **Automated tests** run directly with `go test` or `npm` scripts and produce pass/fail output. Look for `--- PASS:` in test output.
- **Hands-on tests** require running Kirsch in a terminal and checking behaviour manually. These are marked "operator-observed" or "(hands-on)".
- **Checkboxes** (`- [ ]`) track each check. Mark `- [✅]` when it passes, `- [❌]` if it fails.
- **If a check fails**, capture the exact command, terminal size, error message, and relevant logs (see Reporting at the end).
- **Prerequisites**: `npm run check` and `npm run test:race` must pass before you start. Govulncheck shows pre-existing standard-library vulnerabilities in this codebase that the operator is managing separately — see Known limitations.

## Order of work

§11 starts with two automated offline test runs (steps 1–2), then hands-on and live steps (3–5) that cost tokens.

| Section | Deliverable | Type |
|---------|-------------|------|
| 1 | m4-d1 Session Store Core | Automated |
| 2 | m4-d2 Storage Layout + Repair | Automated |
| 3 | m4-d3 Event Recording Wiring | Automated |
| 4 | m4-d4 Resume Reconstruction | Automated |
| 5 | m4-d5 Resume State + CLI | Automated + hands-on |
| 6 | m4-d6 Single-Instance Guard | Automated + hands-on |
| 7 | m4-d7 Compaction | Automated |
| 8 | m4-d8 Slash Commands for Real | Automated + hands-on |
| 9 | m4-d9 Token and Cost Tracking | Automated + hands-on |
| 10 | m4-d10 Final-Task Report | Hands-on |
| 11 | m4-d11 Smoke Test | Hands-on + live |
| 12 | m4-d12 Final Acceptance | Sign-off |

## Setup

For §1: no special setup needed. Run tests from the checkout root.

For §2–§12: set up a scratch environment with temporary config, state, and workspace directories:

```bash
cd /Volumes/DATA/Github/kirsch
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

- Run every section in the same terminal tab. The variables exist only in that shell.
- `WORKSPACE` is a separate git repository — your Kirsch checkout is never touched.
- `KIRSCH_OPENCODE_API_KEY` is cleared to avoid interfering with testing.

Cleanup when done:

```bash
unset OPENCODE_API_KEY KIRSCH_LIVE_MODEL
unset XDG_CONFIG_HOME XDG_STATE_HOME
rm -rf -- "$T_CFG" "$T_STATE" "$WORKSPACE"
```

**API Key:** When a test needs one, enter it securely (never paste into a file or chat):

```bash
read -rs OPENCODE_API_KEY && export OPENCODE_API_KEY
```

Unset it when done:

```bash
unset OPENCODE_API_KEY
```

---

## 1 · Session store core (m4-d1)

**What this tests:** Kirsch stores every conversation event to disk in JSONL format with durability guarantees, fast cadence (flushed every ~500ms and on demand), and error handling that never silently loses data.

**Before you start:** Working directory is the Kirsch checkout. No scratch environment needed for this section.

**Steps:**

1. Event mapping — every event type maps to the correct JSONL field and carries schema version `"v":1`:

```bash
go test -run 'TestAppendMapping|TestAppendToolEvents' -v ./internal/session/
```

**Expect:** Both tests pass:
```
--- PASS: TestAppendMapping
--- PASS: TestAppendToolEvents
```

- [✅] TestAppendMapping
- [✅] TestAppendToolEvents

2. Durability and cadence — writes are buffered, flushed on schedule and on demand, and close semantics are safe:

```bash
go test -run 'TestFlushDurability|TestCadenceFlush|TestCloseSemantics|TestOnErrorHook' -v ./internal/session/
```

**Expect:** All four tests pass.

- [✅] TestFlushDurability
- [✅] TestCadenceFlush
- [✅] TestCloseSemantics
- [✅] TestOnErrorHook

3. Concurrent append — many goroutines write without data loss or interleaving:

```bash
go test -run TestConcurrentAppend -v ./internal/session/
```

**Expect:** Test passes with no race detector warnings.

- [✅] TestConcurrentAppend

4. Benchmark (optional) — measure the cost of recording an event:

```bash
go test -bench=BenchmarkRecord -benchtime=1s ./internal/session/
```

**Expect:** Benchmark completes and reports ns/op and B/op.

- [✅] BenchmarkRecord runs

5. Code quality gates:

```bash
npm run check
```

**Expect:** All checks pass (formatting, vet, lint, test, screens).

- [✅] npm run check exits 0

```bash
npm run test:race
```

**Expect:** Race detector finds no issues.

- [✅] npm run test:race exits 0

```bash
$(go env GOPATH)/bin/govulncheck ./internal/session/...
```

**Expect:** No vulnerabilities in Kirsch code. (Standard library vulnerabilities are pre-existing; see Known limitations.)

- [ ] govulncheck ./internal/session/... reports no vulnerabilities in Kirsch code

> **Operator notes:**
> ```
> === Symbol Results ===
>
> No vulnerabilities found.
>
> Your code is affected by 0 vulnerabilities.
> This scan also found 1 vulnerability in packages you import and 12
> vulnerabilities in modules you require, but your code doesn't appear to call
> these vulnerabilities.
> Use '-show verbose' for more details.
> ```
>
> But we can ignore this for now we can address this later

**If it fails:** Capture the test output (command + full stderr/stdout) and your Go version (`go version`).

---

## 2 · Storage Layout + Repair (m4-d2)

**What this tests:** Session files are stored in `~/.local/share/kirsch/sessions/<workspace-hash>/` as JSONL files named `<timestamp>-<ulid>.jsonl`. Index and usage metadata files live at the root. Torn final lines are repaired on load.

**Before you start:** Run the Setup section above to create `$T_CFG`, `$T_STATE`, `$WORKSPACE`, and the environment variables.

**Steps:**

1. Path and layout generation:

```bash
go test -run 'TestDefaultStateRoot|TestCanonicalize|TestWorkspaceHash|TestSessionID|TestSessionPath|TestNewLayout|TestLayoutCreateSession|TestLayoutCreateSessionSortable|TestLayoutLastSession|TestIndexPathAndUsagePath|TestLayoutIndexJSONSchema|TestReadIndexMissing|TestWriteAndReadIndex|TestIndexSetUpdatesEntry|TestIndexMultipleWorkspaces' -v ./internal/session/
```

**Expect:** All 15 tests pass.

- [✅] TestDefaultStateRoot
- [✅] TestCanonicalize
- [✅] TestWorkspaceHash
- [✅] TestSessionID
- [✅] TestSessionPath
- [✅] TestNewLayout
- [✅] TestLayoutCreateSession
- [✅] TestLayoutCreateSessionSortable
- [✅] TestLayoutLastSession
- [✅] TestIndexPathAndUsagePath
- [✅] TestLayoutIndexJSONSchema
- [✅] TestReadIndexMissing
- [✅] TestWriteAndReadIndex
- [✅] TestIndexSetUpdatesEntry
- [✅] TestIndexMultipleWorkspaces

2. Usage metadata — `usage.json` schema and updates:

```bash
go test -run 'TestReadUsageMissing|TestWriteAndReadUsage|TestUsageAddUpdatesTotals|TestAddUsageRootUpdatesFile' -v ./internal/session/
```

**Expect:** All 4 tests pass.

- [✅] TestReadUsageMissing
- [✅] TestWriteAndReadUsage
- [✅] TestUsageAddUpdatesTotals
- [✅] TestAddUsageRootUpdatesFile

3. Repair on load — torn final lines are truncated without blocking startup:

```bash
go test -run 'TestRepairTruncatesTornFinalLine|TestRepairNoCorruption|TestRepairEmptyFile' -v ./internal/session/
```

**Expect:** All 3 tests pass.

- [✅] TestRepairTruncatesTornFinalLine
- [✅] TestRepairNoCorruption
- [✅] TestRepairEmptyFile

4. Code quality gates:

```bash
npm run check
```

- [✅] npm run check exits 0

```bash
npm run test:race
```

- [✅] npm run test:race exits 0

```bash
$(go env GOPATH)/bin/govulncheck ./internal/session/...
```

- [✅] govulncheck ./internal/session/... reports no vulnerabilities in Kirsch code

5. Fuzz tests (10 seconds each):

```bash
FUZZTIME=10s npm run fuzz
```

**Expect:** All fuzz targets survive 10 seconds. `FuzzSessionPathContained` is one target you'll see.

- [✅] Fuzz targets pass (10s each)

**If it fails:** Capture the test output and any seed corpus files if the fuzz test reports a failure case.

---

## 3 · Event Recording Wiring (m4-d3)

**What this tests:** The session store is wired into the real event stream. Every user message, assistant reply, tool execution, usage update, and approval grant is recorded to JSONL.

**Before you start:** Scratch environment from Setup (§2).

**Steps:**

1. Integration — a full turn records all expected event types, and errors reach telemetry:

```bash
go test -run 'TestSubmitRecordsFullEventSequence|TestSessionStoreOnErrorLogsToTelemetry' -v ./internal/app/
```

**Expect:** Both tests pass.

- [✅] TestSubmitRecordsFullEventSequence
- [✅] TestSessionStoreOnErrorLogsToTelemetry

2. Session store — event mapping and error hook (reusing m4-d1 tests):

```bash
go test -run 'TestAppendMapping|TestAppendToolEvents|TestFlushDurability|TestConcurrentAppend|TestCadenceFlush|TestCloseSemantics|TestOnErrorHook' -v ./internal/session/
```

**Expect:** All 7 tests pass.

- [✅] TestAppendMapping
- [✅] TestAppendToolEvents
- [✅] TestFlushDurability
- [✅] TestConcurrentAppend
- [✅] TestCadenceFlush
- [✅] TestCloseSemantics
- [✅] TestOnErrorHook

3. Code quality gates:

```bash
npm run check
```

- [✅] npm run check exits 0

```bash
npm run test:race
```

**Expect:** Test passes. Note: `TestRunTurnApprovalToolUsesApprovalPath` in `internal/app/turn_adapter_test.go:348` is a known pre-existing flake (fails ~2/10 runs under `-race`). If it fails, retry; if it persists across multiple runs, note it in your report.

- [✅] npm run test:race exits 0

```bash
$(go env GOPATH)/bin/govulncheck ./internal/session/... ./internal/app/...
```

- [✅] govulncheck reports no vulnerabilities in Kirsch code (standard library vulnerabilities are pre-existing)

**If it fails:** Capture the test output. If `TestRunTurnApprovalToolUsesApprovalPath` fails, note how many attempts it took to pass.

---

## 4 · Resume Reconstruction (m4-d4)

**What this tests:** Kirsch rebuilds the conversation and transcript from a session file by replaying the correct event types in the correct order. Tool call IDs are validated, and torn final lines are handled.

**Before you start:** Scratch environment from Setup.

**Steps:**

1. Reconstruction + round-trip — events are replayed into conversation and transcript; full write/read cycles are byte-identical:

```bash
go test -run 'TestResumeEmptyFile|TestResumeMergesConsecutiveUserMessages|TestResumeReconstructsAssistantText|TestResumeReconstructsAssistantThinking|TestResumeReconstructsToolCallAndResult|TestResumeMergesConsecutiveToolResults|TestResumeRoundTripAllEventTypes|TestResumeRoundTripMerge' -v ./internal/session/
```

**Expect:** All 8 tests pass.

- [✅] TestResumeEmptyFile
- [✅] TestResumeMergesConsecutiveUserMessages
- [✅] TestResumeReconstructsAssistantText
- [✅] TestResumeReconstructsAssistantThinking
- [✅] TestResumeReconstructsToolCallAndResult
- [✅] TestResumeMergesConsecutiveToolResults
- [✅] TestResumeRoundTripAllEventTypes
- [✅] TestResumeRoundTripMerge

2. Delta exclusion, compaction, and torn-line repair:

```bash
go test -run 'TestResumeIgnoresAssistantDelta|TestResumeAppliesCompactionToConversationNotTranscript|TestResumeTornLineTruncates' -v ./internal/session/
```

**Expect:** All 3 tests pass.

- [✅] TestResumeIgnoresAssistantDelta
- [✅] TestResumeAppliesCompactionToConversationNotTranscript
- [✅] TestResumeTornLineTruncates

3. Malformed session detection — tool ID mismatches, duplicates, orphans, empty messages, and compaction range errors:

```bash
go test -run 'TestResumeReturnsMalformedSessionOnToolMismatch|TestResumeReturnsMalformedSessionOnDuplicateToolResult|TestResumeReturnsMalformedSessionOnOrphanToolResult|TestResumeReturnsMalformedSessionOnEmptyAssistantMessage|TestResumeReturnsMalformedSessionOnCompactionRangeOutOfBounds' -v ./internal/session/
```

**Expect:** All 5 tests pass.

- [✅] TestResumeReturnsMalformedSessionOnToolMismatch
- [✅] TestResumeReturnsMalformedSessionOnDuplicateToolResult
- [✅] TestResumeReturnsMalformedSessionOnOrphanToolResult
- [✅] TestResumeReturnsMalformedSessionOnEmptyAssistantMessage
- [✅] TestResumeReturnsMalformedSessionOnCompactionRangeOutOfBounds

4. Code quality gates:

```bash
npm run check
```

- [✅] npm run check exits 0

```bash
npm run test:race
```

- [✅] npm run test:race exits 0

```bash
$(go env GOPATH)/bin/govulncheck ./internal/session/...
```

- [✅] govulncheck ./internal/session/... reports no vulnerabilities in Kirsch code

5. Fuzz tests:

```bash
FUZZTIME=10s npm run fuzz
```

- [✅] Fuzz targets pass (`FuzzResume` is one; all survive 10s)

**If it fails:** Capture the test output and any fuzz seed corpus that triggered the failure.

---

## 5 · Resume State + CLI (m4-d5)

**What this tests:** The CLI `kirsch resume` command, `--new` flag, and auto-resume (default) load the last session, restore grants, and show prior transcript in the TUI. Errors fall back to a fresh session without blocking startup.

**Before you start:** Scratch environment from Setup.

**Steps:**

1. Grants restoration — `approval.scope_granted` events are replayed through the grant API:

```bash
go test -run 'TestLoadGrantsReadsScopeGrantedEvents|TestLoadGrantsStopsAtTornLine|TestLoadGrantsEmptyFile|TestLoadGrantsMissingFile' -v ./internal/session/
```

**Expect:** All 4 tests pass.

- [✅] TestLoadGrantsReadsScopeGrantedEvents
- [✅] TestLoadGrantsStopsAtTornLine
- [✅] TestLoadGrantsEmptyFile
- [✅] TestLoadGrantsMissingFile

2. CLI routing — `resume`, `--new`, and auto-resume (with error fallback):

```bash
go test -run 'TestChooseSessionInitResume|TestChooseSessionInitNewFlag|TestChooseSessionInitAutoResumeNoSession|TestChooseSessionInitAutoResumeResumeError|TestChooseSessionInitAutoResumeDisabled' -v ./cmd/kirsch/
```

**Expect:** All 5 tests pass.

- [✅] TestChooseSessionInitResume
- [✅] TestChooseSessionInitNewFlag
- [✅] TestChooseSessionInitAutoResumeNoSession
- [✅] TestChooseSessionInitAutoResumeResumeError
- [✅] TestChooseSessionInitAutoResumeDisabled

3. Integration — quit mid-task, resume, and complete: transcript and grants are identical:

```bash
go test -run 'TestResumeCompletesPartialTurn' -v ./internal/app/
```

**Expect:** Test passes.

- [✅] TestResumeCompletesPartialTurn

4. Code quality gates:

```bash
npm run check
```

- [✅] npm run check exits 0

```bash
npm run test:race
```

- [✅] npm run test:race exits 0

```bash
$(go env GOPATH)/bin/govulncheck ./internal/session/... ./internal/app/... ./cmd/kirsch/...
```

- [✅] govulncheck reports no vulnerabilities in Kirsch code

5. Hands-on — manual TUI resume (operator-observed):

Set up a test by starting Kirsch and quitting mid-task, then restarting:

```bash
cd /Volumes/DATA/Github/kirsch
export XDG_STATE_HOME="$T_STATE" XDG_CONFIG_HOME="$T_CFG"
# Start Kirsch, begin a task, and quit mid-turn (Ctrl+C twice or /quit before the turn finishes)
go run ./cmd/kirsch --workspace "$WORKSPACE"
# Restart with no flags — Kirsch auto-resumes the last session
go run ./cmd/kirsch --workspace "$WORKSPACE"
```

**Expect:** The resumed TUI shows prior transcript (with a recovered-session warning if the session was repaired) and active grants from before.

- [✅] Resumed TUI shows prior transcript with any recovery notice
- [✅] Active grants are restored from the prior session
- [✅] Completing the turn from the resumed state matches a non-interrupted run

**If it fails:** Capture the transcript, any warning messages, and the command you used to start Kirsch.

---

## 6 · Single-Instance Guard (m4-d6)

**What this tests:** Only one Kirsch instance can run in the same workspace at a time. A second instance shows a warning, doesn't auto-resume, and disables session grants.

**Before you start:** Scratch environment from Setup.

**Steps:**

1. Lock acquisition — first instance locks the workspace, second instance contends:

```bash
go test -run 'TestAcquireSucceeds|TestSecondAcquireContends' -v ./internal/session/ && go test -run 'TestAcquireLockNonHeldError' -v ./internal/app/
```

**Expect:** All 3 tests pass.

- [✅] TestAcquireSucceeds
- [✅] TestSecondAcquireContends
- [✅] TestAcquireLockNonHeldError

2. Second-instance degradation — session grants are disabled:

```bash
go test -run 'TestLockSecondInstanceDegrades' -v ./internal/session/
```

- [✅] TestLockSecondInstanceDegrades

3. Integration — two instances in same workspace:

```bash
go test -run 'TestLockSecondInstanceDegrades' -v ./internal/app/
```

- [✅] TestLockSecondInstanceDegrades

4. CLI routing — `kirsch resume` is refused when locked, auto-resume is skipped:

```bash
go test -run 'TestChooseSessionInitExplicitResumeConcurrentInstance|TestChooseSessionInitAutoResumeConcurrentInstance' -v ./cmd/kirsch/
```

- [✅] TestChooseSessionInitExplicitResumeConcurrentInstance
- [✅] TestChooseSessionInitAutoResumeConcurrentInstance

5. Status bar — the warning glyph appears:

```bash
go test -run 'TestStatusWarningRendersWithGlyph' -v ./internal/tui/
```

- [✅] TestStatusWarningRendersWithGlyph

6. Code quality gates:

```bash
npm run check
```

- [✅] npm run check exits 0

```bash
npm run test:race
```

- [✅] npm run test:race exits 0

```bash
$(go env GOPATH)/bin/govulncheck ./internal/session/... ./internal/app/... ./cmd/kirsch/... ./internal/tui/...
```

- [✅] govulncheck reports no vulnerabilities in Kirsch code

7. Hands-on — two terminal windows (operator-observed):

Terminal A (holds the lock):

```bash
cd /Volumes/DATA/Github/kirsch
export XDG_STATE_HOME="$T_STATE" XDG_CONFIG_HOME="$T_CFG"
go run ./cmd/kirsch --workspace "$WORKSPACE"
```

Terminal B (tries to start in the same workspace):

```bash
cd /Volumes/DATA/Github/kirsch
export XDG_STATE_HOME="$T_STATE" XDG_CONFIG_HOME="$T_CFG"
go run ./cmd/kirsch --workspace "$WORKSPACE"
```

**Expect:**
- Terminal A acquires the lock and runs normally
- Terminal B shows `⚠ another Kirsch is running here` in the status bar
- Terminal B does not auto-resume (shows a fresh workspace)
- Terminal B offers no session-grant approval (no `[a]` key shown)
- Terminal B's `kirsch resume` command is refused
- Quit Terminal A, and Terminal B now acquires the lock and proceeds normally

- [✅] Terminal A acquires the lock and runs normally
- [✅] Terminal B shows the warning glyph in the status bar
- [✅] Terminal B does not auto-resume
- [✅] Terminal B offers no session-grant approval option
- [✅] Terminal B's `kirsch resume` is refused
- [✅] Quit Terminal A, Terminal B acquires the lock and proceeds normally

**If it fails:** Note which terminal showed unexpected behaviour and capture the status bar text.

---

## 7 · Compaction (m4-d7)

**What this tests:** Kirsch compacts long conversation histories by calling the model with a summary prompt, keeping the original task and last 4 turns verbatim, and replacing the middle. Thinking blocks are never summarised. Cancellation leaves the session unchanged. Errors surface `context_overflow`.

**Before you start:** Scratch environment from Setup.

**Steps:**

1. Planning — when to compact and what to keep:

```bash
go test -run 'TestPlanCompactionNotEnoughTurns|TestPlanCompactionEmptySlice|TestPlanCompactionSevenTurns|TestPlanCompactionFirstMessageKept|TestPlanCompactionToolResultOnlyDoesNotStartTurn|TestThinkingBlocksNotInSummaryRequest' -v ./internal/session/
```

**Expect:** All 6 tests pass.

- [✅] TestPlanCompactionNotEnoughTurns
- [✅] TestPlanCompactionEmptySlice
- [✅] TestPlanCompactionSevenTurns
- [✅] TestPlanCompactionFirstMessageKept
- [✅] TestPlanCompactionToolResultOnlyDoesNotStartTurn
- [✅] TestThinkingBlocksNotInSummaryRequest

2. Compaction correctness — immutability, thinking preservation, cancellation, error handling:

```bash
go test -run 'TestCompactionSuccessPathImmutability|TestCompactionRequestShape|TestThinkingBlocksPreservedInResult|TestCompactionCancellation|TestCompactionModelError|TestCompactionEmptySummary|TestCompactionWhitespaceOnlySummary|TestCompactionCancelAndEmit|TestCompactionEventError|TestCompactionWithFiveTurns|TestCompactionRoundTrip|TestCompactionRoundTripWithToolCalls|TestCompactionOriginalTaskAndLastFourPreserved' -v ./internal/session/
```

**Expect:** All 13 tests pass.

- [✅] TestCompactionSuccessPathImmutability
- [✅] TestCompactionRequestShape
- [✅] TestThinkingBlocksPreservedInResult
- [✅] TestCompactionCancellation
- [✅] TestCompactionModelError
- [✅] TestCompactionEmptySummary
- [✅] TestCompactionWhitespaceOnlySummary
- [✅] TestCompactionCancelAndEmit
- [✅] TestCompactionEventError
- [✅] TestCompactionWithFiveTurns
- [✅] TestCompactionRoundTrip
- [✅] TestCompactionRoundTripWithToolCalls
- [✅] TestCompactionOriginalTaskAndLastFourPreserved

3. App integration — threshold, request flow, cancellation, failure:

```bash
go test -run 'TestSubmitBelowThresholdSendsOneRequestAndNoCompaction|TestSubmitAboveThresholdCompactsThenRunsTheTurn|TestSubmitCancelDuringCompactionLeavesSessionUntouched|TestSubmitCancelAfterSummaryBeforeSwapNeverHalfApplies|TestSubmitCompactionProviderErrorSurfacesContextOverflow|TestSubmitFewerThanSixTurnsOverBudgetSurfacesContextOverflow|TestSubmitOverBudgetCompactsInsteadOfRefusing|TestSubmitStillOverBudgetAfterCompactionSurfacesContextOverflow|TestSubmitCompactionSurvivesResume' -v ./internal/app/
```

**Expect:** All 9 tests pass.

- [✅] TestSubmitBelowThresholdSendsOneRequestAndNoCompaction
- [✅] TestSubmitAboveThresholdCompactsThenRunsTheTurn
- [✅] TestSubmitCancelDuringCompactionLeavesSessionUntouched
- [✅] TestSubmitCancelAfterSummaryBeforeSwapNeverHalfApplies
- [✅] TestSubmitCompactionProviderErrorSurfacesContextOverflow
- [✅] TestSubmitFewerThanSixTurnsOverBudgetSurfacesContextOverflow
- [✅] TestSubmitOverBudgetCompactsInsteadOfRefusing
- [✅] TestSubmitStillOverBudgetAfterCompactionSurfacesContextOverflow
- [✅] TestSubmitCompactionSurvivesResume

4. TUI messages — status bar and header updates:

```bash
go test -run 'TestCompactingMsgSetsBusyVerbCompacting|TestSessionCompactedMsgMarksHeaderCompactedAndKeepsBusy|TestCompactingBusyIsClearedByTurnErrorMsg|TestCompactingBusyIsClearedByTurnCancelledMsg' -v ./internal/tui/
```

**Expect:** All 4 tests pass.

- [✅] TestCompactingMsgSetsBusyVerbCompacting
- [✅] TestSessionCompactedMsgMarksHeaderCompactedAndKeepsBusy
- [✅] TestCompactingBusyIsClearedByTurnErrorMsg
- [✅] TestCompactingBusyIsClearedByTurnCancelledMsg

5. Code quality gates:

```bash
npm run check
```

- [✅] npm run check exits 0

```bash
npm run test:race
```

- [✅] npm run test:race exits 0

```bash
$(go env GOPATH)/bin/govulncheck ./internal/session/... ./internal/app/...
```

- [✅] govulncheck reports no vulnerabilities in Kirsch code

**If it fails:** Capture the test output and any error messages about context budget calculations.

---

## 8 · Slash Commands for Real (m4-d8)

**What this tests:** Commands like `/status`, `/diff`, `/files`, `/approvals`, `/new`, `/compact`, `/quit` operate on real session state.

**Before you start:** Scratch environment from Setup.

**Steps:**

1. `/status` fields — model, session ID, token counts, compaction %, grants:

```bash
go test -run 'TestStatusInfoShowsAccumulatedTokens|TestStatusInfoSessionID|TestStatusInfoCompactionPercent|TestStatusInfoActiveGrants|TestStatusInfoDirtyCount' -v ./internal/app/
```

**Expect:** All 5 tests pass.

- [✅] TestStatusInfoShowsAccumulatedTokens
- [✅] TestStatusInfoSessionID
- [✅] TestStatusInfoCompactionPercent
- [✅] TestStatusInfoActiveGrants
- [✅] TestStatusInfoDirtyCount

2. `/diff` — shows working tree diff (background, non-blocking, 200 KiB cap):

```bash
go test -run 'TestDiffCommandDoesNotBlockUpdate|TestDiffCommandOpensModalWithCallbackLines|TestDiffCommandCleanTreeShowsNotice|TestDiffCommandErrorShowsNotice|TestDiffCommandWithoutCallbackShowsNotice|TestDiffResultTruncationMarkerDerivesFromCap|TestDiffResultDoesNotReplaceAnOpenModal|TestDiffModalLinesAreSanitised|TestDiffResultIgnoresStaleSequence|TestDiffResultDroppedAfterNewSession|TestDiffResultBlockedByConfirmPrompt|TestGetDiffCountsExactlyAndExcludesUntracked|TestGetDiffCleanTreeIsEmptyWithNoError|TestGetDiffErrorPaths|TestGetDiffHeaderExclusionStopsAtFirstHunk|TestGetDiffSanitisesLines|TestGetDiffAppliesTheByteCap|TestCountDiffLinesStatesTheHeaderRule|TestDiffCommandThroughRealWiring' -v ./internal/tui/ ./internal/app/ ./cmd/kirsch/
```

**Expect:** All 19 tests pass.

- [✅] TestDiffCommandDoesNotBlockUpdate
- [✅] TestDiffCommandOpensModalWithCallbackLines
- [✅] TestDiffCommandCleanTreeShowsNotice
- [✅] TestDiffCommandErrorShowsNotice
- [✅] TestDiffCommandWithoutCallbackShowsNotice
- [✅] TestDiffResultTruncationMarkerDerivesFromCap
- [✅] TestDiffResultDoesNotReplaceAnOpenModal
- [✅] TestDiffModalLinesAreSanitised
- [✅] TestDiffResultIgnoresStaleSequence
- [✅] TestDiffResultDroppedAfterNewSession
- [✅] TestDiffResultBlockedByConfirmPrompt
- [✅] TestGetDiffCountsExactlyAndExcludesUntracked
- [✅] TestGetDiffCleanTreeIsEmptyWithNoError
- [✅] TestGetDiffErrorPaths
- [✅] TestGetDiffHeaderExclusionStopsAtFirstHunk
- [✅] TestGetDiffSanitisesLines
- [✅] TestGetDiffAppliesTheByteCap
- [✅] TestCountDiffLinesStatesTheHeaderRule
- [✅] TestDiffCommandThroughRealWiring

3. `/files` — lists files touched by successful patches (cleaned, de-duplicated, reset on resume):

```bash
go test -run 'TestFilesCommandListsPaths|TestFilesCommandEmptyNotice|TestFilesCommandWithoutCallbackShowsNotice|TestFilesCommandSanitisesPaths|TestToolExecutorRecordsTouchedFilesOnlyForSuccessfulPatches|TestAddTouchedFileNormalisesAndDeduplicates' -v ./internal/tui/ ./internal/app/
```

**Expect:** All 6 tests pass.

- [✅] TestFilesCommandListsPaths
- [✅] TestFilesCommandEmptyNotice
- [✅] TestFilesCommandWithoutCallbackShowsNotice
- [✅] TestFilesCommandSanitisesPaths
- [✅] TestToolExecutorRecordsTouchedFilesOnlyForSuccessfulPatches
- [✅] TestAddTouchedFileNormalisesAndDeduplicates

4. `/approvals` — lists grants and confirms clearing:

```bash
go test -run 'TestApprovalsNoticeSanitisesGrants' -v ./internal/tui/
```

- [✅] TestApprovalsNoticeSanitisesGrants

5. `/new` — starts new session with confirm if turn running:

```bash
go test -run 'TestSlashNewRunsCallbackInCmdAndResetsOnDone|TestSlashNewDoesNotBlockUpdate|TestSlashNewFailureKeepsTranscript|TestNewSessionIsolatesTheOldTurnFromTheNewSession|TestNewSessionFailureLeavesTheSessionIntact|TestSubmitRefusesWhileNewSessionWaitsForATurn|TestSlashNewThroughRealWiringStartsANewSession' -v ./internal/tui/ ./internal/app/ ./cmd/kirsch/
```

**Expect:** All 7 tests pass.

- [✅] TestSlashNewRunsCallbackInCmdAndResetsOnDone
- [✅] TestSlashNewDoesNotBlockUpdate
- [✅] TestSlashNewFailureKeepsTranscript
- [✅] TestNewSessionIsolatesTheOldTurnFromTheNewSession
- [✅] TestNewSessionFailureLeavesTheSessionIntact
- [✅] TestSubmitRefusesWhileNewSessionWaitsForATurn
- [✅] TestSlashNewThroughRealWiringStartsANewSession

6. `/compact` — manual compaction on demand (no turn start):

```bash
go test -run 'TestCompactCommandHoldsBusyUntilRealOutcome|TestCompactCommandRefusesSubmitAndKeepsBusyOnEsc|TestManualCompactMsgOutcomesShowDistinctNotices|TestCompactCommandWithoutCallback|TestManualCompactAppliesWithoutStartingATurn|TestManualCompactWithTooFewTurnsReportsNothing|TestManualCompactProviderErrorIsAFailureNotAnOverflow|TestManualCompactCancelDuringModelCallReportsCancelled|TestManualCompactCancelAfterSummaryBeforeSwapNeverHalfApplies|TestManualCompactRefusesWhileATurnIsLive|TestSubmitRefusesWhileManualCompactIsRunning' -v ./internal/tui/ ./internal/app/
```

**Expect:** All 11 tests pass.

- [✅] TestCompactCommandHoldsBusyUntilRealOutcome
- [✅] TestCompactCommandRefusesSubmitAndKeepsBusyOnEsc
- [✅] TestManualCompactMsgOutcomesShowDistinctNotices
- [✅] TestCompactCommandWithoutCallback
- [✅] TestManualCompactAppliesWithoutStartingATurn
- [✅] TestManualCompactWithTooFewTurnsReportsNothing
- [✅] TestManualCompactProviderErrorIsAFailureNotAnOverflow
- [✅] TestManualCompactCancelDuringModelCallReportsCancelled
- [✅] TestManualCompactCancelAfterSummaryBeforeSwapNeverHalfApplies
- [✅] TestManualCompactRefusesWhileATurnIsLive
- [✅] TestSubmitRefusesWhileManualCompactIsRunning

7. Dirty count and diff driver blocking:

```bash
go test -run 'TestDirtyCount|TestDirtyCountNonGitAndErrorBranches' -v ./internal/workspace/
```

- [✅] TestDirtyCount
- [✅] TestDirtyCountNonGitAndErrorBranches

```bash
go test -run 'TestDiffArgsShape|TestGitDiffDoesNotRunConfiguredDiffDrivers' -v ./internal/tool/
```

- [✅] TestDiffArgsShape
- [✅] TestGitDiffDoesNotRunConfiguredDiffDrivers

8. Hands-on — run the commands manually (operator-observed):

```bash
cd /Volumes/DATA/Github/kirsch
export XDG_STATE_HOME="$T_STATE" XDG_CONFIG_HOME="$T_CFG"
go run ./cmd/kirsch --workspace "$WORKSPACE"
```

Then in Kirsch, try each command:

- [✅] `/status` shows model, session ID, tokens, compaction %, grants
- [✅] `/diff` shows the working tree (clean or with changes), scrollable
- [✅] `/files` after patches shows touched files (empty before any patches)
- [✅] `/approvals` lists grants (empty or with active grants)
- [✅] `/new` creates a new session and changes the session ID in `/status`
- [✅] `/compact` (if ≥6 turns run) compacts; otherwise shows "nothing to compact yet"
- [✅] `/quit` and `/exit` exit Kirsch
- [✅] An unknown command like `/hello` shows a dim hint, never an error

9. Code quality gates:

```bash
npm run check
```

- [✅] npm run check exits 0

```bash
npm run test:race
```

- [✅] npm run test:race exits 0

**If it fails:** Capture the command you ran, the output you saw, and any error messages in the status bar or modal.

---

## 9 · Token and Cost Tracking (m4-d9)

**What this tests:** Kirsch tracks token usage (input, output, cache read/write) and cost per call, persists totals to `usage.json`, shows them in the status bar, and handles unknown models gracefully.

**Before you start:** Scratch environment from Setup.

**Steps - Automated tests:**

1. Provider — cost calculation for known and unknown models:

```bash
go test -run 'TestUnknownModelFallback|TestCostPerTokenModel|TestCostFlatModel|TestCostUnknownModel|TestCostAllPerTokenModels|TestCostZeroTokens|TestConservativeBudgetEqualsTableMinimum|TestConservativeBudgetFollowsTheTable|TestCostStateStrings' -v ./internal/provider/
```

**Expect:** All 9 tests pass.

- [✅] TestUnknownModelFallback
- [✅] TestCostPerTokenModel
- [✅] TestCostFlatModel
- [✅] TestCostUnknownModel
- [✅] TestCostAllPerTokenModels
- [✅] TestCostZeroTokens
- [✅] TestConservativeBudgetEqualsTableMinimum
- [✅] TestConservativeBudgetFollowsTheTable
- [✅] TestCostStateStrings

2. Session usage — file I/O, corruption recovery, concurrency, locking:

```bash
go test -run 'TestUsageAddUpdatesTotals|TestAddUsageRootUpdatesFile|TestUsageAccumulationWithUnknownCost|TestUsageBackwardCompatibility|TestAddUsageConcurrentGoroutines|TestAddUsageCorruptFileRecovery|TestAddUsageRecoveriesInOneSecondKeepBothFiles|TestAddUsageUnreadableFileLeftUntouched|TestWriteJSONFileRemovesTempWhenRenameFails|TestWriteJSONFileLeavesNoTempOnSuccess|TestWriteJSONFileConcurrentWritersNeverCollide|TestAddUsageWaitsForHeldLock|TestAddUsageReturnsErrLockHeldWhenLockStaysHeld|TestTryLockInterruptHandling' -v ./internal/session/
```

**Expect:** All 14 tests pass.

- [✅] TestUsageAddUpdatesTotals
- [✅] TestAddUsageRootUpdatesFile
- [✅] TestUsageAccumulationWithUnknownCost
- [✅] TestUsageBackwardCompatibility
- [✅] TestAddUsageConcurrentGoroutines
- [✅] TestAddUsageCorruptFileRecovery
- [✅] TestAddUsageRecoveriesInOneSecondKeepBothFiles
- [✅] TestAddUsageUnreadableFileLeftUntouched
- [✅] TestWriteJSONFileRemovesTempWhenRenameFails
- [✅] TestWriteJSONFileLeavesNoTempOnSuccess
- [✅] TestWriteJSONFileConcurrentWritersNeverCollide
- [✅] TestAddUsageWaitsForHeldLock
- [✅] TestAddUsageReturnsErrLockHeldWhenLockStaysHeld
- [✅] TestTryLockInterruptHandling

3. Compaction — usage recorded before, during, and after:

```bash
go test -run 'TestCompactionUsageCapture|TestCompactionUsageReturnedOnError|TestCompactionUsageReturnedOnEveryErrorPath' -v ./internal/session/
```

- [✅] TestCompactionUsageCapture
- [✅] TestCompactionUsageReturnedOnError
- [✅] TestCompactionUsageReturnedOnEveryErrorPath

4. App — session totals and per-call pricing:

```bash
go test -run 'TestSubmitSendsExactSessionTotalsPerTurn|TestSessionCostPricesEachCallAtItsOwnModel|TestSubmitCompactionSendsSessionTotals|TestManualCompactSendsSessionTotals|TestSessionTotalsRestartAfterNewSession|TestUnknownModelNoticeOnceAcrossTwoTurns|TestKnownModelShowsNoUnknownNotice|TestUnknownModelNoticeAgainAfterNewSession|TestSubmitCompactionRecordsSummaryAndTurnUsage|TestManualCompactRecordsSummaryUsage|TestSubmitCompactionFailureRecordsUsageAndLeavesSessionUntouched|TestManualCompactStreamErrorRecordsUsage|TestRecordUsageReportsCorruptUsageFileThroughLogger' -v ./internal/app/
```

**Expect:** All 13 tests pass.

- [✅] TestSubmitSendsExactSessionTotalsPerTurn
- [✅] TestSessionCostPricesEachCallAtItsOwnModel
- [✅] TestSubmitCompactionSendsSessionTotals
- [✅] TestManualCompactSendsSessionTotals
- [✅] TestSessionTotalsRestartAfterNewSession
- [✅] TestUnknownModelNoticeOnceAcrossTwoTurns
- [✅] TestKnownModelShowsNoUnknownNotice
- [✅] TestUnknownModelNoticeAgainAfterNewSession
- [✅] TestSubmitCompactionRecordsSummaryAndTurnUsage
- [✅] TestManualCompactRecordsSummaryUsage
- [✅] TestSubmitCompactionFailureRecordsUsageAndLeavesSessionUntouched
- [✅] TestManualCompactStreamErrorRecordsUsage
- [✅] TestRecordUsageReportsCorruptUsageFileThroughLogger

5. Wiring — usage event reaches the status bar:

```bash
go test -run 'TestProviderEventToAgentMapsUsage|TestUsageNonzero|TestUsageEventThroughRealWiringReachesTheStatusBar' -v ./internal/app/ ./internal/agent/ ./cmd/kirsch/
```

- [✅] TestProviderEventToAgentMapsUsage
- [✅] TestUsageNonzero
- [✅] TestUsageEventThroughRealWiringReachesTheStatusBar

6. TUI — status bar layout and rendering:

```bash
go test -run 'TestSessionUsageWideRender|TestSessionUsageFlatNoCost|TestSessionUsageSmallCostFormat|TestSessionUsageUnknownPricingShowsQuestionMark|TestSessionUsageCostBoundary|TestSessionUsageASCIIGlyphs|TestSessionUsageNarrowLayoutFallsBackToSessionTotal|TestStatusLadderDropsTokensBeforeGrants|TestUnknownModelNotice' -v ./internal/tui/
```

- [✅] TestSessionUsageWideRender
- [✅] TestSessionUsageFlatNoCost
- [✅] TestSessionUsageSmallCostFormat
- [✅] TestSessionUsageUnknownPricingShowsQuestionMark
- [✅] TestSessionUsageCostBoundary
- [✅] TestSessionUsageASCIIGlyphs
- [✅] TestSessionUsageNarrowLayoutFallsBackToSessionTotal
- [✅] TestStatusLadderDropsTokensBeforeGrants
- [✅] TestUnknownModelNotice

**Steps - Hands-on tests (requires live model or config):**

7. Inspect `usage.json` — where it lives and its structure:

```bash
grep -r "DefaultStateRoot\|share/kirsch" /Volumes/DATA/Github/kirsch/internal/session/layout.go
```

**Expect:** Shows `~/.local/share/kirsch/` as the default state root. In this walkthrough, usage.json is at `$XDG_STATE_HOME/kirsch/usage.json` because `$XDG_STATE_HOME` is set.

- [✅] Default state root is `~/.local/share/kirsch/`

8. Run a turn with a live model (optional — requires API key):

```bash
cd /Volumes/DATA/Github/kirsch
export XDG_STATE_HOME="$T_STATE" XDG_CONFIG_HOME="$T_CFG"
go run ./cmd/kirsch --workspace "$WORKSPACE"
# (in Kirsch) read -rs OPENCODE_API_KEY && export OPENCODE_API_KEY
# (in Kirsch) type a short message and press Enter
```

**Expect:** Status bar shows `in N out N cache R/W · $cost`. Token count is > 0. Cost is > $0.

- [✅] Status bar shows tokens and cost after a live turn
- [✅] Cost is non-zero
- [✅] `/status` shows the same token counts
- [✅] `usage.json` contains cumulative totals (prompt, completion, cache read/write, total, cost)
- [✅] A second turn shows accumulated totals (`in 2N out 2M`)

9. Unknown model (no key needed):

```bash
cat > "$T_CFG/kirsch/config.toml" << 'EOF'
[provider.opencode]
model = "unknown-model-xyz"
EOF
# Restart Kirsch
go run ./cmd/kirsch --workspace "$WORKSPACE"
# (in Kirsch) type a message
```

**Expect:** Status bar shows a once-per-session notice and uses `?` for cost.

- [✅] Status bar shows `cost display unavailable, conservative budget in use`
- [✅] `/status` shows `$?` instead of cost
- [✅] After `/new`, the notice appears again

10. Compaction usage:

Run ≥6 turns (fast with fake provider) and then `/compact`.

**Expect:** Status bar shows `compacting` during compaction. After, `/status` shows total tokens including the compaction call.

- [✅] Status bar shows `compacting` during compaction
- [✅] After compaction, `/status` shows total tokens including the summary call
- [✅] `usage.json` includes usage for the compaction call itself

11. Narrow terminal (status bar fallback):

Resize your terminal to fewer than ~40 columns and observe the status bar:

- [❌] Below the fallback width, the status bar shows only `N tok` (the total token count), dropping tokens first, then grants, then model name.

> **Operator notes:**
> No this goes from this: minimax-m2.7 · idle · in 8.5k out 977 cache 23.7k/3.3k to this:  minimax-m2.7 · idle

12. Code quality gates:

```bash
npm run check
```

- [✅] npm run check exits 0

```bash
npm run test:race
```

- [✅] npm run test:race exits 0

**If it fails:** Capture the test output, the status bar text, and the contents of `usage.json` (without revealing the API key).

---

## 10 · Final-Task Report (m4-d10)

**What this tests:** Every task that makes file changes must end with a structured report (Summary, Files changed, Commands run with PASS/FAIL, Limitations). The report format is preserved across compaction.

**Before you start:** Scratch environment from Setup (§2). Optional: API key for live testing.

**Steps - Automated:**

1. System prompt enforcement:

```bash
go test -count=1 -run 'TestAssembleFinalReportEnforcement' ./internal/agent/prompt/
```

- [✅] TestAssembleFinalReportEnforcement

2. Compaction preservation:

```bash
go test -count=1 -run 'TestCompactionSystemPromptCommandOutcomes|TestCompactionSystemPromptPreservesFinalReport|TestCompactionRequestShape' ./internal/session/
```

- [✅] TestCompactionSystemPromptCommandOutcomes
- [✅] TestCompactionSystemPromptPreservesFinalReport
- [✅] TestCompactionRequestShape

3. System prompt survives compaction:

```bash
go test -count=1 -run 'TestSystemPromptSurvivesCompaction' ./internal/app/
```

- [✅] TestSystemPromptSurvivesCompaction

**Steps - Hands-on (operator-observed):**

4. Give Kirsch a small task (e.g., edit a file and run `go test`):

```bash
cd /Volumes/DATA/Github/kirsch
export XDG_STATE_HOME="$T_STATE" XDG_CONFIG_HOME="$T_CFG"
go run ./cmd/kirsch --workspace "$WORKSPACE"
```

**Expect:** Reply ends with four labelled sections: Summary, Files changed, Commands run (with PASS/FAIL per command), Limitations.

- [✅] Final report is present with all four sections
- [✅] PASS/FAIL appears for each command

5. Ask a pure question (e.g., "What is the Fibonacci sequence?"):

**Expect:** No report required (answer only).

- [✅] Pure question does not require a report

6. Long session or compaction test — run ≥6 turns or execute `/compact`, then finish a task:

**Expect:** Report still appears after compaction.

- [✅] Final report appears after compaction

7. Code quality gates:

```bash
npm run check
```

- [✅] npm run check exits 0

**If it fails:** Capture the task you gave, the reply, and whether the report sections are present.

---

## 11 · Smoke Test (m4-d11)

**What this tests:** A complete end-to-end workflow. Kirsch is given a real coding task ("Add input validation to the Divide function and cover it with a test"), auto-approves patches, and produces a valid report.

**Before you start:** Scratch environment from Setup. For live test (steps 4–5): API key and a clean config.

**Steps - Automated:**

1. Offline smoke test with fake provider:

```bash
cd /Volumes/DATA/Github/kirsch
go test -count=1 -run 'TestSmokeOffline' ./internal/app/
```

**Expect:** Test passes (one positive control plus the negative controls, each a `t.Run` subtest; add `-v` to list them).

- [✅] TestSmokeOffline (all controls pass)

2. Report parsing:

```bash
go test -count=1 -run 'TestParseReportSummary|TestParseListLines' ./internal/app/
```

- [✅] TestParseReportSummary
- [✅] TestParseListLines

**Steps - Hands-on (operator-run, costs tokens):**

3. Verify the script exits 2 when no API key is set:

```bash
env -u ANTHROPIC_API_KEY -u KIRSCH_ANTHROPIC_API_KEY -u OPENCODE_API_KEY -u KIRSCH_OPENCODE_API_KEY bash scripts/smoke-test.sh
echo rc=$?
```

**Expect:** Script exits with rc=2.

- [✅] rc=2 (no key error)

4. Clean config and run the live smoke test (costs tokens on a real model):

Before running steps 4–5, start with a clean config:

```bash
rm -f "$XDG_CONFIG_HOME/kirsch/config.toml"
```

Then, from the checkout root, enter the key (it is not echoed) and run the script. The script does not prompt; with no key exported it exits 2.

```bash
cd /Volumes/DATA/Github/kirsch
read -rs OPENCODE_API_KEY && export OPENCODE_API_KEY
bash scripts/smoke-test.sh
echo rc=$?
```

(Keep OPENCODE_API_KEY exported for step 5; unset it at the end of step 5.)

**Expect:** Script exits 0. The test runs `go test -tags smoke -count=1 -timeout 10m -run '^TestSmokeLive$' -v ./internal/app/` and checks for `--- PASS: TestSmokeLive`. The smoke test verifies: patch applies after auto-approval, `go test ./...` passes in the modified workspace, report labels are present (Summary, Files changed, Commands run, Limitations), report lists exactly {calc/divide.go, calc/divide_test.go}, git status in the temp workspace shows exactly those two files touched, the agent itself ran a `go test` command, a test passing a zero divisor exists (source scan) and `go test ./...` passes.

- [✅] Script exits 0 and reports `--- PASS: TestSmokeLive`

5. Manual run (optional):

If you skipped step 4, first run `rm -f "$XDG_CONFIG_HOME/kirsch/config.toml"` and enter the key as in step 4.

Create a temp workspace from the tracked fixture files:

```bash
SMOKE_WS=$(mktemp -d)
cp /Volumes/DATA/Github/kirsch/testdata/repo-go-module/go.mod "$SMOKE_WS/"
cp -r /Volumes/DATA/Github/kirsch/testdata/repo-go-module/calc "$SMOKE_WS/"
git -C "$SMOKE_WS" init
git -C "$SMOKE_WS" add .
git -C "$SMOKE_WS" -c user.name=kirsch -c user.email=kirsch@example.invalid commit -m "Initial commit"
mkdir -p "$XDG_CONFIG_HOME/kirsch"
printf '[session]\nstorage_dir = "%s"\n' "$T_STATE/sessions" > "$XDG_CONFIG_HOME/kirsch/config.toml"
cd /Volumes/DATA/Github/kirsch
# The key from step 4 must still be exported.
go run ./cmd/kirsch --workspace "$SMOKE_WS"
# (in Kirsch) type: "Add input validation to the Divide function and cover it with a test"
# (approve every prompt: the patch and, if asked, the agent's go test command)
# (when the report appears, type /quit)
```

After the run, check the workspace diff, then confirm the tracked fixture was not touched:

```bash
git -C "$SMOKE_WS" diff --stat
# (expected: calc/divide.go and calc/divide_test.go)
rm -f "$XDG_CONFIG_HOME/kirsch/config.toml"
rm -rf -- "$SMOKE_WS"; unset SMOKE_WS OPENCODE_API_KEY
git status testdata/repo-go-module/
# (should show nothing modified)
```

If a stray `.git` exists in `testdata/repo-go-module/` from an earlier run, remove it with `rm -rf testdata/repo-go-module/.git`.

**Expect:** Kirsch asks to approve a patch to calc/divide.go and calc/divide_test.go (and possibly the agent's go test command); after approval the final reply has the four report sections (Summary, Files changed, Commands run with PASS/FAIL, Limitations); `git -C "$SMOKE_WS" diff --stat` (run before cleanup) shows those two files. Session files are written under `$T_STATE/sessions/` because of the scratch config above (Kirsch's default is `~/.local/share/kirsch/sessions/`; `XDG_STATE_HOME` only moves `debug.log`). Check with `ls "$T_STATE/sessions"`.

- [✅] Manual run produces a valid patch and report

**If it fails:** Capture the error output, the shell rc, whether the temp workspace was cleaned up, and whether `testdata/repo-go-module/` has uncommitted changes.

---

## 12 · Final Acceptance (m4-d12)

**Testable (operator-gated).** This section completes Milestone 4. Sections §1–§11 cover all acceptance criteria. Run the final gate checks and sign off.

**Steps - Gate checks (run locally):**

1. Race detector on full test suite:

```bash
cd /Volumes/DATA/Github/kirsch
go test -race -count=1 ./...
```

**Expect:** All tests pass under race detector.

- [✅] ok (race-enabled full test suite passes)

2. Code format (wrapped by npm script):

```bash
npm run fmt:check
```

Equivalent direct command (works whether or not golangci-lint is on PATH):

```bash
bash scripts/go-tool.sh golangci-lint fmt --diff
```

**Expect:** No changes needed (no formatting drift). Note: bare `golangci-lint` is usually not on PATH. Always use the npm script or `bash scripts/go-tool.sh golangci-lint ...`; the wrapper finds it in `$(go env GOPATH)/bin`.

- [✅] No changes needed

3. Static analysis:

```bash
go vet ./...
```

**Expect:** All packages clean.

- [✅] All packages clean

4. Linting (wrapped by npm script):

```bash
npm run lint
```

Equivalent direct command (works whether or not golangci-lint is on PATH):

```bash
bash scripts/go-tool.sh golangci-lint run
```

**Expect:** 0 issues. Note: bare `golangci-lint` is usually not on PATH. Always use the npm script or `bash scripts/go-tool.sh golangci-lint ...`; the wrapper finds it in `$(go env GOPATH)/bin`.

- [✅] 0 issues

5. Screens (UI layout):

```bash
npm run screens
```

**Expect:** `kirsch-ui-screens.md` grids and content OK.

- [✅] `kirsch-ui-screens.md: grids OK, content OK`

6. Code quality (all checks combined):

```bash
npm run check
```

**Expect:** All of the above pass (format, vet, lint, screens, and test).

- [✅] npm run check exits 0

**Steps - Operator sign-off:**

7. Live smoke test from §11:

- [✅] `bash scripts/smoke-test.sh` passes (costs tokens; re-enter the key first as in §11 step 4 if you unset it, and make sure `$XDG_CONFIG_HOME/kirsch/config.toml` is absent)

I am skipping this as you just cant fix this at all

8. Sections §1–§11 complete:

- [✅] All sections tested and checked

9. Milestone 4 acceptance boxes:

- [✅] All boxes ticked in `plan/milestones/milestone-4.md`

10. CI green:

- [ ] GitHub Actions passes

11. Milestone 4 complete (create tag and update progress):

- [ ] `git tag m4`
- [ ] `plan/PROGRESS.md` Milestones table row 4 set to complete

---

## Known limitations

- **Govulncheck:** `npm run security` reports 10 pre-existing vulnerabilities in `golang.org/x/net` v0.56.0 (fixed in v0.60.0) and Go runtime `net/http` at v1.25.13 (fixed in v1.26.9). None are in Kirsch code. The operator's dependency strategy is pending.
- **m4-d1 JSONL gaps:** `tool.completed` omits `tool` and `duration_ms`, and `assistant.message` omits `stop_reason`. These fields are populated at m4-d3 wiring (real event stream). This is expected.
- **Live model:** Default model is `minimax-m2.7` (plan amendment 83). The smoke test reads the config from `$XDG_CONFIG_HOME/kirsch/config.toml` and the project config; start with a clean config before §11 steps 4–5. §9 step 9 writes `model = "unknown-model-xyz"` to `$T_CFG/kirsch/config.toml`; remove that file before §11 steps 4–5 or the live smoke fails at config validation.

---

## Reporting

For any check marked ❌:

1. The exact command you ran
2. Terminal size (`echo $COLUMNS $LINES`) and whether you are in tmux
3. Section and check name
4. Config file content if relevant (Kirsch `config.toml`)
5. Debug log if relevant: `cat "$XDG_STATE_HOME/kirsch/debug.log"`

Never paste API keys in reports.
