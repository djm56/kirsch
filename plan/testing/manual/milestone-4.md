# Manual test — Milestone 4 (session store, resume, compaction, the real task loop)

Checking that Kirsch persists its session, resumes it, compacts it, and runs the
real task loop end to end. Milestone 4 introduces the session store, resume and
compaction, and the first real task against a real repository.

**Time:** about 10–15 minutes for §1 now; the full document grows as the later
deliverables land.

**Prerequisite:** `npm run check` and `npm run test:race` are green. `npm run security` is RED pre-existing: govulncheck reports 10 vulnerabilities in `golang.org/x/net` v0.56.0 (fixed in v0.60.0) and `net/http` at `go1.25.13` (fixed in `go1.26.9`). They appeared between 2026-10-08 and 2026-10-14 with zero repository change; none are in Kirsch code. The scoped `govulncheck ./internal/session/...` is clean. This is a carried item — the operator's dependency decision is pending.

**To run this walkthrough:**

1. From the repository root, copy this file: `cp plan/testing/manual/milestone-4.md plan/testing/manual/user-testing/milestone-4-<yourname>.md`
2. Mark each check ✅ or ❌ in your copy. A ❌ with a sentence about what you saw is worth more than a long report.
3. The master stays blank. Commit both files: the blank master and your completed run.

---

## Status Table

| Section | Deliverable | Status |
|---|---|---|
| 1 | m4-d1 Session Store Core | **Runnable now** |
| 2 | m4-d2 Storage Layout + Repair | **Runnable now** |
| 3 | m4-d3 Event Recording Wiring | **Runnable now** |
| 4 | m4-d4 Resume Reconstruction | **Runnable now** |
| 5 | m4-d5 Resume State + CLI | **Runnable now** |
| 6 | m4-d6 Single-Instance Guard | **Runnable now** |
| 7 | m4-d7 Compaction | **Runnable now** |
| 8 | m4-d8 Slash Commands for Real | **Runnable now** |
| 9 | m4-d9 Token and Cost Tracking | **Runnable now** |
| 10 | m4-d10 Final-Task Report | **Testable** |
| 11 | m4-d11 Smoke Test | Not yet testable |
| 12 | m4-d12 Final Acceptance | Not yet testable |

---

## Model for this run

No live model is needed for this walkthrough yet. The first live model use is the
m4-d11 smoke test, where the default model `minimax-m2.7` applies (plan amendment
83). `KIRSCH_LIVE_MODEL` overrides only the live test, not the TUI — the M3
precedent.

---

## Setup

§1 runs from the checkout alone and needs none of this scratch environment. The
scratch environment below is needed from m4-d2 onward (storage layout writes
under `XDG_STATE_HOME`, and the TUI runs from m4-d5/d6/d8).

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

**API Key:** Never write the key to a file, output, or chat. When a section needs it, enter it with:

```bash
read -rs OPENCODE_API_KEY && export OPENCODE_API_KEY
```

Then unset it when done:

```bash
unset OPENCODE_API_KEY
```

---

## 1 · Session store core (m4-d1)

`internal/session` landed in mission-20261014-01. The `Store` type implements
`agent.Recorder` (no-error `Record(kind RecordKind, m Message)` signature, per
the operator-confirmed Recorder-seam decision); every line is JSONL with `"v":1`
(ADR 0002); a single writer goroutine owns all writes (architecture.md §3 rule
3); writes are buffered and `fsync`'d every ~500ms and on an explicit `Flush()`
call; and a `WithOnError` hook receives mid-turn write/`fsync` failures so they
are never silently lost. This section runs the unit and race tests that back the
m4-d1 acceptance, grouped to match the commands below.

**Event mapping — every RecordKind maps to the expected plan §7 event type, every line carries `"v":1`:**

```bash
go test -run 'TestAppendMapping|TestAppendToolEvents' -v ./internal/session/
```

- [ ] `--- PASS: TestAppendMapping`
- [ ] `--- PASS: TestAppendToolEvents`

**Durability, cadence, close, and the error hook:**

```bash
go test -run 'TestFlushDurability|TestCadenceFlush|TestCloseSemantics|TestOnErrorHook' -v ./internal/session/
```

- [ ] `--- PASS: TestFlushDurability`
- [ ] `--- PASS: TestCadenceFlush`
- [ ] `--- PASS: TestCloseSemantics`
- [ ] `--- PASS: TestOnErrorHook`

**Concurrent append — many goroutines record without interleaving or loss:**

```bash
go test -run TestConcurrentAppend -v ./internal/session/
```

- [ ] `--- PASS: TestConcurrentAppend`

**Optional benchmark — the cost of queuing an event:**

```bash
go test -bench=BenchmarkRecord -benchtime=1s ./internal/session/
```

- [ ] `BenchmarkRecord` runs (optional)

**Gates:**

```bash
npm run check
```

- [ ] `npm run check` exits 0 (fmt, vet, lint, test, screens all green)

```bash
npm run test:race
```

- [ ] `npm run test:race` exits 0 (race-enabled test run clean)

```bash
$(go env GOPATH)/bin/govulncheck ./internal/session/...
```

- [ ] `govulncheck ./internal/session/...` reports no vulnerabilities in Kirsch code (the full path is used so no PATH setup is needed; the binary installs to `$(go env GOPATH)/bin/govulncheck`, and `npm run tools` installs it and prints the directory)

This section is the operator's independent verification of the m4-d1 acceptance
(the boxes in milestone-4.md are already ticked with mission evidence —
mission-20261014-01).

---

## 2 · Storage Layout + Repair (m4-d2)

`internal/session` gained the plan §7 storage layout and repair-on-load in mission-20261014-05. The corrected layout (post fix round 2a) is: state root `~/.local/share/kirsch/` (XDG-adjacent); sessions live under `<root>/sessions/<workspace-hash>/`; each session file is `<timestamp>-<ulid>.jsonl`; and `index.json` (canonical workspace path → `{last_session_id, updated_at}`) and `usage.json` (cumulative token totals) sit at the root beside `sessions/` — matching plan §7's tree (kirsch-plan.md:366–374). The ULID is a stdlib-only Crockford base32 encoding (48-bit millisecond timestamp + 80 bits `crypto/rand`), so no new module was added. This section runs the unit, race, and fuzz tests that back the m4-d2 acceptance, grouped to match the commands below.

**Layout + index — path generation, the §7 tree, and index.json read/write:**

```bash
go test -run 'TestDefaultStateRoot|TestCanonicalize|TestWorkspaceHash|TestSessionID|TestSessionPath|TestNewLayout|TestLayoutCreateSession|TestLayoutCreateSessionSortable|TestLayoutLastSession|TestIndexPathAndUsagePath|TestLayoutIndexJSONSchema|TestReadIndexMissing|TestWriteAndReadIndex|TestIndexSetUpdatesEntry|TestIndexMultipleWorkspaces' -v ./internal/session/
```

- [ ] `--- PASS: TestDefaultStateRoot`
- [ ] `--- PASS: TestCanonicalize`
- [ ] `--- PASS: TestWorkspaceHash`
- [ ] `--- PASS: TestSessionID`
- [ ] `--- PASS: TestSessionPath`
- [ ] `--- PASS: TestNewLayout`
- [ ] `--- PASS: TestLayoutCreateSession`
- [ ] `--- PASS: TestLayoutCreateSessionSortable`
- [ ] `--- PASS: TestLayoutLastSession`
- [ ] `--- PASS: TestIndexPathAndUsagePath`
- [ ] `--- PASS: TestLayoutIndexJSONSchema`
- [ ] `--- PASS: TestReadIndexMissing`
- [ ] `--- PASS: TestWriteAndReadIndex`
- [ ] `--- PASS: TestIndexSetUpdatesEntry`
- [ ] `--- PASS: TestIndexMultipleWorkspaces`

**Usage — usage.json schema and the update path:**

```bash
go test -run 'TestReadUsageMissing|TestWriteAndReadUsage|TestUsageAddUpdatesTotals|TestAddUsageRootUpdatesFile' -v ./internal/session/
```

- [ ] `--- PASS: TestReadUsageMissing`
- [ ] `--- PASS: TestWriteAndReadUsage`
- [ ] `--- PASS: TestUsageAddUpdatesTotals`
- [ ] `--- PASS: TestAddUsageRootUpdatesFile`

**Repair-on-load — a torn final line truncates and marks the session `recovered`, never blocking startup:**

```bash
go test -run 'TestRepairTruncatesTornFinalLine|TestRepairNoCorruption|TestRepairEmptyFile' -v ./internal/session/
```

- [ ] `--- PASS: TestRepairTruncatesTornFinalLine`
- [ ] `--- PASS: TestRepairNoCorruption`
- [ ] `--- PASS: TestRepairEmptyFile`

**Gates:**

```bash
npm run check
```

- [ ] `npm run check` exits 0 (fmt, vet, lint, test, screens all green)

```bash
npm run test:race
```

- [ ] `npm run test:race` exits 0 (race-enabled test run clean)

```bash
$(go env GOPATH)/bin/govulncheck ./internal/session/...
```

- [ ] `govulncheck ./internal/session/...` reports no vulnerabilities in Kirsch code (the full path is used so no PATH setup is needed; the binary installs to `$(go env GOPATH)/bin/govulncheck`, and `npm run tools` installs it and prints the directory)

```bash
FUZZTIME=10s npm run fuzz
```

- [ ] `FUZZTIME=10s npm run fuzz` — `FuzzSessionPathContained` survives 10s (all fuzz targets survive 10s each)

This section is the operator's independent verification of the m4-d2 acceptance (the boxes in milestone-4.md are already ticked with mission evidence — mission-20261014-05). It is the record for the five m4-d2 acceptance boxes: the §7 storage layout, `index.json` mapping, `usage.json` existence, repair-on-load → `recovered`, and the unit tests.

---

## 3 · Event Recording Wiring (m4-d3)

`internal/app` wired the real M3 event stream into the session store in mission-20261014-09. `Submit()` sets `ag.Recorder` to the session store (app.go:737, store built via `ensureStore` at app.go:548/706); every plan §7 event type is recorded to JSONL — `turn.started`/`turn.completed` (app.go:959/988), `user.message`/`assistant.message`/`assistant.thinking`/`tool.completed` (store.go `Record` → event.go §7 mapping), `usage` (app.go:1069 `RecordUsage` + usage.json via `AddUsage` at app.go:1072), and `approval.scope_granted` (app.go:1318–1323 after `pol.Grant`); `assistant.delta` is excluded per the reconstruction rule (§7:360); the `user.message` `role` field was removed to match the §7 schema (fix round 2a, re-verified 2b); and the store's `OnError` hook is wired to the telemetry debug log (app.go:572–574). This section runs the integration and store tests that back the m4-d3 acceptance, grouped to match the commands below.

**Integration — a full turn records the expected event sequence, and OnError reaches telemetry:**

```bash
go test -run 'TestSubmitRecordsFullEventSequence|TestSessionStoreOnErrorLogsToTelemetry' -v ./internal/app/
```

- [ ] `--- PASS: TestSubmitRecordsFullEventSequence`
- [ ] `--- PASS: TestSessionStoreOnErrorLogsToTelemetry`

**Session store — the event mapping and error hook (the m4-d3 delta removed the `role` assertion from `TestAppendMapping`):**

```bash
go test -run 'TestAppendMapping|TestAppendToolEvents|TestFlushDurability|TestConcurrentAppend|TestCadenceFlush|TestCloseSemantics|TestOnErrorHook' -v ./internal/session/
```

- [ ] `--- PASS: TestAppendMapping`
- [ ] `--- PASS: TestAppendToolEvents`
- [ ] `--- PASS: TestFlushDurability`
- [ ] `--- PASS: TestConcurrentAppend`
- [ ] `--- PASS: TestCadenceFlush`
- [ ] `--- PASS: TestCloseSemantics`
- [ ] `--- PASS: TestOnErrorHook`

**Gates:**

```bash
npm run check
```

- [ ] `npm run check` exits 0 (fmt, vet, lint, test, screens all green)

```bash
npm run test:race
```

- [ ] `npm run test:race` exits 0 (race-enabled test run clean) — NOTE: `TestRunTurnApprovalToolUsesApprovalPath` (`internal/app/turn_adapter_test.go:348`) is a known PRE-EXISTING flake (2/10 under `-race`, proven pre-existing by the Step 2b stash protocol on the clean pre-delta tree; the m4-d2 close's green race run was a lucky pass). A full green run can require a retry; widen the collector timeout or count terminal messages if it trips. See *Carried from m4-d3* in plan/PROGRESS.md.

```bash
$(go env GOPATH)/bin/govulncheck ./internal/session/... ./internal/app/...
```

- [ ] `govulncheck ./internal/session/... ./internal/app/...` — 10 pre-existing operator-deferred vulnerabilities in the Go standard library (`net/http` at `go1.25.13`, fixed in `go1.26.9`); none are in the changed code. The operator's dependency decision is pending.

This section is the operator's independent verification of the m4-d3 acceptance (the boxes in milestone-4.md are already ticked with mission evidence — mission-20261014-09). It is the record for the seven m4-d3 acceptance boxes: `app.Submit()` sets `ag.Recorder`; every §7 event type recorded (user.message, assistant.message, assistant.thinking, tool.completed, usage, approval.scope_granted, turn.started/turn.completed; assistant.delta excluded); usage feeds usage.json; OnError wired to telemetry; the user.message role-field deviation removed; and the integration test.

---

## 4 · Resume Reconstruction (m4-d4)

`internal/session` gained `Resume(path)` in mission-20261015-01. It rebuilds the
`*agent.Conversation` and a plain `Transcript` from the JSONL session log per the
locked reconstruction rule (plan §7): only `user.message`/`assistant.message`/
`assistant.thinking`/`tool.completed` contribute; `assistant.delta` is explicitly
skipped (UI replay only); `compaction.applied` replaces the conversation range
with the summary, leaving the transcript intact; tool-call ids correlate via a
pending map with `ErrMalformedSession` on mismatch/duplicate/orphan; and
consecutive `user.message`/`tool.completed` merge (the `RecordMerge` semantics
the store is documented to provide). This section runs the unit and race tests
that back the m4-d4 acceptance, grouped to match the commands below. Tick each
box as you run it.

**Reconstruction + round-trip — the four event types rebuild the conversation and transcript, and full store-write/resume cycles are byte-identical:**

```bash
go test -run 'TestResumeEmptyFile|TestResumeMergesConsecutiveUserMessages|TestResumeReconstructsAssistantText|TestResumeReconstructsAssistantThinking|TestResumeReconstructsToolCallAndResult|TestResumeMergesConsecutiveToolResults|TestResumeRoundTripAllEventTypes|TestResumeRoundTripMerge' -v ./internal/session/
```

- [ ] `--- PASS: TestResumeEmptyFile`
- [ ] `--- PASS: TestResumeMergesConsecutiveUserMessages`
- [ ] `--- PASS: TestResumeReconstructsAssistantText`
- [ ] `--- PASS: TestResumeReconstructsAssistantThinking`
- [ ] `--- PASS: TestResumeReconstructsToolCallAndResult`
- [ ] `--- PASS: TestResumeMergesConsecutiveToolResults`
- [ ] `--- PASS: TestResumeRoundTripAllEventTypes`
- [ ] `--- PASS: TestResumeRoundTripMerge`

**Delta-ignoring + compaction + torn-line repair — `assistant.delta` is skipped, `compaction.applied` replaces the conversation range only, and an invalid final line truncates like repair-on-load:**

```bash
go test -run 'TestResumeIgnoresAssistantDelta|TestResumeAppliesCompactionToConversationNotTranscript|TestResumeTornLineTruncates' -v ./internal/session/
```

- [ ] `--- PASS: TestResumeIgnoresAssistantDelta`
- [ ] `--- PASS: TestResumeAppliesCompactionToConversationNotTranscript`
- [ ] `--- PASS: TestResumeTornLineTruncates`

**Malformed sessions — a tool.id mismatch, duplicate, orphan, empty assistant.message, or out-of-bounds compaction range returns `ErrMalformedSession` (distinct from torn-line repair):**

```bash
go test -run 'TestResumeReturnsMalformedSessionOnToolMismatch|TestResumeReturnsMalformedSessionOnDuplicateToolResult|TestResumeReturnsMalformedSessionOnOrphanToolResult|TestResumeReturnsMalformedSessionOnEmptyAssistantMessage|TestResumeReturnsMalformedSessionOnCompactionRangeOutOfBounds' -v ./internal/session/
```

- [ ] `--- PASS: TestResumeReturnsMalformedSessionOnToolMismatch`
- [ ] `--- PASS: TestResumeReturnsMalformedSessionOnDuplicateToolResult`
- [ ] `--- PASS: TestResumeReturnsMalformedSessionOnOrphanToolResult`
- [ ] `--- PASS: TestResumeReturnsMalformedSessionOnEmptyAssistantMessage`
- [ ] `--- PASS: TestResumeReturnsMalformedSessionOnCompactionRangeOutOfBounds`

**Gates:**

```bash
npm run check
```

- [ ] `npm run check` exits 0 (fmt, vet, lint, test, screens all green)

```bash
npm run test:race
```

- [ ] `npm run test:race` exits 0 (race-enabled test run clean) — NOTE: `TestRunTurnApprovalToolUsesApprovalPath` (`internal/app/turn_adapter_test.go:348`) is a known PRE-EXISTING flake (2/10 under `-race`, proven pre-existing by the Step 2b stash protocol on the clean pre-delta tree; the m4-d2 close's green race run was a lucky pass). A full green run can require a retry; widen the collector timeout or count terminal messages if it trips. See *Carried from m4-d3* in plan/PROGRESS.md.

```bash
$(go env GOPATH)/bin/govulncheck ./internal/session/...
```

- [ ] `govulncheck ./internal/session/...` reports no vulnerabilities in Kirsch code (the full path is used so no PATH setup is needed; the binary installs to `$(go env GOPATH)/bin/govulncheck`, and `npm run tools` installs it and prints the directory)

```bash
FUZZTIME=10s npm run fuzz
```

- [ ] `FUZZTIME=10s npm run fuzz` — `FuzzResume` survives 10s (all fuzz targets survive 10s each)

This section is the operator's independent verification of the m4-d4 acceptance (the boxes in milestone-4.md are already ticked with mission evidence — mission-20261015-01). It is the record for the six m4-d4 acceptance boxes: `session.Resume(path)` returns the reconstructed `*agent.Conversation` and transcript; reconstruction uses only `user.message`/`assistant.message`/`assistant.thinking`/`tool.completed` (never `assistant.delta`); `assistant.delta` is ignored; `compaction.applied` replaces the compacted range with the summary; tool call ids correlate; and the unit tests reconstruct from JSONL with all event types and verify the conversation matches the original.

---

## 5 · Resume State + CLI (m4-d5)

`cmd/kirsch` gained the `resume` subcommand and `--new` flag plus no-flags
auto-resume via `index.json` in mission-20261015-04. `chooseSessionInit`
(main.go:151) routes the `resume` subcommand to `App.ResumeLastSession`
(main.go:152–155), the `--new` flag to `App.StartNewSession` (main.go:157–160),
and the no-flags path to auto-resume when `cfg.Session.AutoResume` is true
(config.go:74/:114, default true) — falling back to a fresh session with a
distinguishing notice on any resume error, so startup never blocks (main.go:162–175,
fix round 2a, re-verified 2b). `internal/app` wires `ResumeLastSession`
(app.go:599) → `Layout.LastSession` (app.go:613) + `session.Resume`, restores
session-scoped grants from `approval.scope_granted` events via `session.LoadGrants`
→ the existing `policy.Grant` API (app.go:661–675; apply_patch grants correctly
skipped, `ErrPatchGrantForbidden`, policy.go:165–166), and restores pending state
(`a.conv` = the reconstructed `*agent.Conversation`, app.go:647–648). The resumed
transcript is rendered in the TUI via `tui.TranscriptLoadedMsg` (messages.go:171),
including the ui-spec ⚠ recovered-session warning and the truncation-point notice
(update.go:292–297). This section runs the unit, CLI-routing, and integration tests
that back the m4-d5 acceptance, grouped to match the commands below. Tick each box
as you run it.

**Grants restoration — `approval.scope_granted` events replay through `policy.Grant`; patch grants refused:**

```bash
go test -run 'TestLoadGrantsReadsScopeGrantedEvents|TestLoadGrantsStopsAtTornLine|TestLoadGrantsEmptyFile|TestLoadGrantsMissingFile' -v ./internal/session/
```

- [ ] `--- PASS: TestLoadGrantsReadsScopeGrantedEvents`
- [ ] `--- PASS: TestLoadGrantsStopsAtTornLine`
- [ ] `--- PASS: TestLoadGrantsEmptyFile`
- [ ] `--- PASS: TestLoadGrantsMissingFile`

**CLI session-init routing — `resume` / `--new` / auto-resume (incl. the error-fallback paths):**

```bash
go test -run 'TestChooseSessionInitResume|TestChooseSessionInitNewFlag|TestChooseSessionInitAutoResumeNoSession|TestChooseSessionInitAutoResumeResumeError|TestChooseSessionInitAutoResumeDisabled' -v ./cmd/kirsch/
```

- [ ] `--- PASS: TestChooseSessionInitResume`
- [ ] `--- PASS: TestChooseSessionInitNewFlag`
- [ ] `--- PASS: TestChooseSessionInitAutoResumeNoSession`
- [ ] `--- PASS: TestChooseSessionInitAutoResumeResumeError`
- [ ] `--- PASS: TestChooseSessionInitAutoResumeDisabled`

**Integration — quit mid-task, resume, complete the turn: identical transcript, grants, valid message array:**

```bash
go test -run 'TestResumeCompletesPartialTurn' -v ./internal/app/
```

- [ ] `--- PASS: TestResumeCompletesPartialTurn`

**Gates:**

```bash
npm run check
```

- [ ] `npm run check` exits 0 (fmt, vet, lint, test, screens all green)

```bash
npm run test:race
```

- [ ] `npm run test:race` exits 0 (race-enabled test run clean) — NOTE: `TestRunTurnApprovalToolUsesApprovalPath` (`internal/app/turn_adapter_test.go:348`) is a known PRE-EXISTING flake (2/10 under `-race`, proven pre-existing by the Step 2b stash protocol on the clean pre-delta tree). A full green run can require a retry; widen the collector timeout or count terminal messages if it trips. See *Carried from m4-d3* in plan/PROGRESS.md. This m4-d5 close-out run was green.

```bash
$(go env GOPATH)/bin/govulncheck ./internal/session/... ./internal/app/... ./cmd/kirsch/...
```

- [ ] `govulncheck ./internal/session/... ./internal/app/... ./cmd/kirsch/...` — 10 pre-existing operator-deferred vulnerabilities in the Go standard library (`net/http` at `go1.25.13`, fixed in `go1.26.9`); none are in the changed code. The operator's dependency decision is pending.

**Operator-observed (optional, not automated) — the manual TUI resume path:**

This step is operator-observed only; it is modelled on the M3 §8 pattern and is not covered by an automated test. Set up the scratch environment from the Setup section, then:

```bash
cd /Volumes/DATA/Github/kirsch
export XDG_STATE_HOME="$T_STATE" XDG_CONFIG_HOME="$T_CFG"
# Start Kirsch, begin a task, and quit mid-turn (Ctrl+C twice / /quit before the turn completes)
go run ./cmd/kirsch --workspace "$WORKSPACE"
# Restart with no flags — Kirsch auto-resumes the last session
go run ./cmd/kirsch --workspace "$WORKSPACE"
```

- [ ] (operator-observed) the resumed TUI shows the prior transcript (the ⚠ recovered-session warning if the session was repaired) and the active grants restored from the prior session
- [ ] (operator-observed) continuing the turn from the resumed state completes it and the transcript is identical to a non-interrupted run

This section is the operator's independent verification of the m4-d5 acceptance (the boxes in milestone-4.md are already ticked with mission evidence — mission-20261015-04). It is the record for the six m4-d5 acceptance boxes: `kirsch resume` loads the last session via `index.json`; `kirsch --new` starts fresh even if a last session exists; with no flags Kirsch auto-resumes the last session; session-scoped grants are restored from `approval.scope_granted` events; pending state (the in-progress turn) is restored; and the integration test — quit mid-task, resume, complete the turn — yields an identical transcript, grants, and a valid message array.

---

## 6 · Single-Instance Guard (m4-d6)

`internal/session` gained advisory `flock` support in mission-20261016-02. `Acquire(root, workspace)` takes a non-blocking advisory `flock` on `<workspace-hash>/.lock` (Unix only; non-unix stub at lock_other.go:28, not supported in v0.1); `Release` at lock.go:63, `Close` at :79; `ErrLockHeld` at lock_error.go:8. `cmd/kirsch/main.go:202` calls `(*App).AcquireLock` at startup before the resume decision; on lock held, app.go:522 sends `another Kirsch is running here`, app.go:528–529 logs and guard off; app.go:1469 `canApproveForSession` withholds approval; status bar internal/tui/statusbar.go:39–40 shows `⚠ another Kirsch is running here`. This section runs the unit, integration, and CLI-routing tests that back the m4-d6 acceptance, grouped to match the commands below.

**Lock acquisition and second-instance detection — `Acquire` on `<workspace-hash>/.lock`, `ConcurrentInstance()` on lock failure:**

```bash
go test -run 'TestAcquireSucceeds|TestSecondAcquireContends' -v ./internal/session/ && go test -run 'TestAcquireLockNonHeldError' -v ./internal/app/
```

- [ ] `--- PASS: TestAcquireSucceeds`
- [ ] `--- PASS: TestSecondAcquireContends`
- [ ] `--- PASS: TestAcquireLockNonHeldError`

**Second-instance degradation — sessionGrantsDisabled flag, auto-resume skipped:**

```bash
go test -run 'TestLockSecondInstanceDegrades' -v ./internal/session/
```

- [ ] `--- PASS: TestLockSecondInstanceDegrades`

**Integration — two instances in same workspace, second degrades:**

```bash
go test -run 'TestLockSecondInstanceDegrades' -v ./internal/app/
```

- [ ] `--- PASS: TestLockSecondInstanceDegrades`

**CLI routing — explicit `kirsch resume` refused, auto-resume skipped when concurrent instance:**

```bash
go test -run 'TestChooseSessionInitExplicitResumeConcurrentInstance|TestChooseSessionInitAutoResumeConcurrentInstance' -v ./cmd/kirsch/
```

- [ ] `--- PASS: TestChooseSessionInitExplicitResumeConcurrentInstance`
- [ ] `--- PASS: TestChooseSessionInitAutoResumeConcurrentInstance`

**Status bar — the `⚠ another Kirsch is running here` glyph:**

```bash
go test -run 'TestStatusWarningRendersWithGlyph' -v ./internal/tui/
```

- [ ] `--- PASS: TestStatusWarningRendersWithGlyph`

**Gates:**

```bash
npm run check
```

- [ ] `npm run check` exits 0 (fmt, vet, lint, test, screens all green)

```bash
npm run test:race
```

- [ ] `npm run test:race` exits 0 (race-enabled test run clean) — NOTE: `TestRunTurnApprovalToolUsesApprovalPath` (`internal/app/turn_adapter_test.go:348`) is a known PRE-EXISTING flake (2/10 under `-race`, proven pre-existing by the Step 2b stash protocol on the clean pre-delta tree). A full green run can require a retry; widen the collector timeout or count terminal messages if it trips. See *Carried from m4-d3* in plan/PROGRESS.md.

```bash
$(go env GOPATH)/bin/govulncheck ./internal/session/... ./internal/app/... ./cmd/kirsch/... ./internal/tui/...
```

- [ ] `govulncheck ./internal/session/... ./internal/app/... ./cmd/kirsch/... ./internal/tui/...` — 10 pre-existing operator-deferred vulnerabilities in the Go standard library (`net/http` at `go1.25.13`, fixed in `go1.26.9`); none are in the changed code. The operator's dependency decision is pending.

**Operator-observed (optional, not automated) — the manual guard in a running TUI:**

This step is operator-observed only; it is modelled on the M3 §8 pattern and is not covered by an automated test. Set up the scratch environment from the Setup section, with two terminal windows open to the same `$WORKSPACE`:

Terminal A (hold the lock):
```bash
cd /Volumes/DATA/Github/kirsch
export XDG_STATE_HOME="$T_STATE" XDG_CONFIG_HOME="$T_CFG"
go run ./cmd/kirsch --workspace "$WORKSPACE"
```

Terminal B (attempt to start while A holds the lock):
```bash
cd /Volumes/DATA/Github/kirsch
export XDG_STATE_HOME="$T_STATE" XDG_CONFIG_HOME="$T_CFG"
go run ./cmd/kirsch --workspace "$WORKSPACE"
```

- [ ] (operator-observed) Terminal A acquires the lock and runs normally
- [ ] (operator-observed) Terminal B shows `⚠ another Kirsch is running here` in the status bar
- [ ] (operator-observed) Terminal B does not auto-resume (shows a fresh workspace, no prior session transcript)
- [ ] (operator-observed) Terminal B offers no session-grant approval option (if a session grant check is available, no `[a]` approval-for-session key is shown)
- [ ] (operator-observed) In Terminal B, `kirsch resume` is refused (no `resume` subcommand available when locked out)
- [ ] (operator-observed) Quit Terminal A (Ctrl+C twice or `/quit`), then Terminal B continues — it now acquires the lock and proceeds normally

**Optional: Lock-unavailable case — state directory unwritable (tests the `lock unavailable — single-instance guard off` fallback):**

This is optional testing of the fallback path when the lock file cannot be created. For example, make `$T_STATE/kirsch` unwritable and verify:

```bash
chmod 000 "$T_STATE"/kirsch/sessions/$(cd "$WORKSPACE" && stat -f '%A' . | xargs -I {} python3 -c "import hashlib; print(hashlib.sha256(b'$(pwd)').hexdigest()[:16])")
go run ./cmd/kirsch --workspace "$WORKSPACE"
chmod 755 "$T_STATE"/kirsch/sessions/$(cd "$WORKSPACE" && stat -f '%A' . | xargs -I {} python3 -c "import hashlib; print(hashlib.sha256(b'$(pwd)').hexdigest()[:16])")
```

- [ ] (optional) Lock unavailable warning appears; app still starts and session grants are disabled (as tested in integration tests)
- [ ] (optional) Note: this uses a 200ms settle window; testing may need retry tolerance for heavy load

This section is the operator's independent verification of the m4-d6 acceptance (the boxes in milestone-4.md are already ticked with mission evidence — mission-20261016-02). It is the record for the four m4-d6 acceptance boxes: on start Kirsch takes advisory `flock` on `<workspace-hash>/.lock`; second instance in same workspace does not auto-resume (starts fresh), shows `⚠ another Kirsch is running here`, and disables session-scoped grants; unit tests for lock acquisition and second-instance degradation; and integration test verifying two instances.

---

## 7 · Compaction (m4-d7)

`internal/session` gained `type Compaction` in mission-20261017-02. The `PlanCompaction` and `Compact` functions estimate the request budget (if estimated > 75% of context window − 4096 output reserve), build a model request keeping the original task and last 4 turns verbatim with a structured summary of turns in between (Objective, Files inspected, Key findings, Files changed, Commands run, Open questions). Cancellation before or after the summary returns leaves the session unchanged and never half-applies. Failure (provider error, empty summary, timeout) surfaces `context_overflow` and directs the user to `/new`. ALL thinking blocks are dropped, never summarised (every assembled block carries the `DropOnSummary` flag anyway). The `compaction.applied` event records the replaced range and summary. This section runs the unit, integration, and TUI tests that back the m4-d7 acceptance.

**Session compaction — planning, structure, thinking handling:**

```bash
go test -run 'TestPlanCompactionNotEnoughTurns|TestPlanCompactionEmptySlice|TestPlanCompactionSevenTurns|TestPlanCompactionFirstMessageKept|TestPlanCompactionToolResultOnlyDoesNotStartTurn|TestThinkingBlocksNotInSummaryRequest' -v ./internal/session/
```

- [ ] `--- PASS: TestPlanCompactionNotEnoughTurns`
- [ ] `--- PASS: TestPlanCompactionEmptySlice`
- [ ] `--- PASS: TestPlanCompactionSevenTurns`
- [ ] `--- PASS: TestPlanCompactionFirstMessageKept`
- [ ] `--- PASS: TestPlanCompactionToolResultOnlyDoesNotStartTurn`
- [ ] `--- PASS: TestThinkingBlocksNotInSummaryRequest`

**Compaction correctness — immutability, request shape, thinking preservation, cancellation, error handling:**

```bash
go test -run 'TestCompactionSuccessPathImmutability|TestCompactionRequestShape|TestThinkingBlocksPreservedInResult|TestCompactionCancellation|TestCompactionModelError|TestCompactionEmptySummary|TestCompactionWhitespaceOnlySummary|TestCompactionCancelAndEmit|TestCompactionEventError|TestCompactionWithFiveTurns|TestCompactionRoundTrip|TestCompactionRoundTripWithToolCalls|TestCompactionOriginalTaskAndLastFourPreserved' -v ./internal/session/
```

- [ ] `--- PASS: TestCompactionSuccessPathImmutability`
- [ ] `--- PASS: TestCompactionRequestShape`
- [ ] `--- PASS: TestThinkingBlocksPreservedInResult`
- [ ] `--- PASS: TestCompactionCancellation`
- [ ] `--- PASS: TestCompactionModelError`
- [ ] `--- PASS: TestCompactionEmptySummary`
- [ ] `--- PASS: TestCompactionWhitespaceOnlySummary`
- [ ] `--- PASS: TestCompactionCancelAndEmit`
- [ ] `--- PASS: TestCompactionEventError`
- [ ] `--- PASS: TestCompactionWithFiveTurns`
- [ ] `--- PASS: TestCompactionRoundTrip`
- [ ] `--- PASS: TestCompactionRoundTripWithToolCalls`
- [ ] `--- PASS: TestCompactionOriginalTaskAndLastFourPreserved`

**App compaction integration — threshold logic, request flow, cancellation, failure:**

```bash
go test -run 'TestSubmitBelowThresholdSendsOneRequestAndNoCompaction|TestSubmitAboveThresholdCompactsThenRunsTheTurn|TestSubmitCancelDuringCompactionLeavesSessionUntouched|TestSubmitCancelAfterSummaryBeforeSwapNeverHalfApplies|TestSubmitCompactionProviderErrorSurfacesContextOverflow|TestSubmitFewerThanSixTurnsOverBudgetSurfacesContextOverflow|TestSubmitOverBudgetCompactsInsteadOfRefusing|TestSubmitStillOverBudgetAfterCompactionSurfacesContextOverflow|TestSubmitCompactionSurvivesResume' -v ./internal/app/
```

- [ ] `--- PASS: TestSubmitBelowThresholdSendsOneRequestAndNoCompaction`
- [ ] `--- PASS: TestSubmitAboveThresholdCompactsThenRunsTheTurn`
- [ ] `--- PASS: TestSubmitCancelDuringCompactionLeavesSessionUntouched`
- [ ] `--- PASS: TestSubmitCancelAfterSummaryBeforeSwapNeverHalfApplies`
- [ ] `--- PASS: TestSubmitCompactionProviderErrorSurfacesContextOverflow`
- [ ] `--- PASS: TestSubmitFewerThanSixTurnsOverBudgetSurfacesContextOverflow`
- [ ] `--- PASS: TestSubmitOverBudgetCompactsInsteadOfRefusing`
- [ ] `--- PASS: TestSubmitStillOverBudgetAfterCompactionSurfacesContextOverflow`
- [ ] `--- PASS: TestSubmitCompactionSurvivesResume`

**TUI compaction messages — status bar glyph, header update:**

```bash
go test -run 'TestCompactingMsgSetsBusyVerbCompacting|TestSessionCompactedMsgMarksHeaderCompactedAndKeepsBusy|TestCompactingBusyIsClearedByTurnErrorMsg|TestCompactingBusyIsClearedByTurnCancelledMsg' -v ./internal/tui/
```

- [ ] `--- PASS: TestCompactingMsgSetsBusyVerbCompacting`
- [ ] `--- PASS: TestSessionCompactedMsgMarksHeaderCompactedAndKeepsBusy`
- [ ] `--- PASS: TestCompactingBusyIsClearedByTurnErrorMsg`
- [ ] `--- PASS: TestCompactingBusyIsClearedByTurnCancelledMsg`

**Gates:**

```bash
npm run check
```

- [ ] `npm run check` exits 0 (fmt, vet, lint, test, screens all green)

```bash
npm run test:race
```

- [ ] `npm run test:race` exits 0 (race-enabled test run clean) — NOTE: `TestRunTurnApprovalToolUsesApprovalPath` (`internal/app/turn_adapter_test.go:348`) is a known PRE-EXISTING flake (2/10 under `-race`, proven pre-existing by the Step 2b stash protocol on the clean pre-delta tree; the m4-d2 close's green race run was a lucky pass). A full green run can require a retry; widen the collector timeout or count terminal messages if it trips. See *Carried from m4-d3* in plan/PROGRESS.md.

```bash
$(go env GOPATH)/bin/govulncheck ./internal/session/... ./internal/app/...
```

- [ ] `govulncheck ./internal/session/... ./internal/app/...` — 10 pre-existing operator-deferred vulnerabilities in the Go standard library (`net/http` at `go1.25.13`, fixed in `go1.26.9`); none are in the changed code. The operator's dependency decision is pending.

This section is the operator's independent verification of the m4-d7 acceptance (the boxes in milestone-4.md are already ticked with mission evidence — mission-20261017-02). It is the record for the seven m4-d7 acceptance boxes: compaction is a model call using the session's provider and model; it keeps the original task and last 4 turns verbatim with a structured summary of the middle; it is cancellable (cancelled compaction leaves session uncompacted and turn unstarted); on failure it surfaces `context_overflow` telling the user to `/new`; thinking blocks are dropped (never summarised, all assembled blocks carry the DropOnSummary flag); `compaction.applied` event recorded with range and summary; and unit tests covering planning, thinking preservation, cancellation, and failure handling.

---

## 8 · Slash Commands for Real (m4-d8)

`internal/tui` and `internal/app` gained the real slash commands in mission-20261017-03. The commands operate on real session state: `/status` shows model, endpoint, base_url, key source, proxy, context file + size, branch, `dirty N`, `session <id|none>`, `tokens in N out N cache N` and `compaction N%`, plus `grants N|none`; `/diff` shows the working tree diff (background run, no UI block, 10s timeout, 200 KiB cap); `/files` lists workspace paths touched by successful `apply_patch` calls this session; `/approvals` lists active grants and confirms clearing; `/new` starts a new session (confirms if a turn is in flight); `/compact` triggers manual compaction on demand; `/quit` and `/exit` quit; and an unknown `/command` shows a dim inline hint, never an error card. This section runs the unit, app, and integration tests that back the m4-d8 acceptance, grouped to match the commands below.

**`/status` fields — model, branch, dirty count, session id, tokens (in/out/cache), compaction %, context file, grants:**

```bash
go test -run 'TestStatusInfoShowsAccumulatedTokens|TestStatusInfoSessionID|TestStatusInfoCompactionPercent|TestStatusInfoActiveGrants|TestStatusInfoDirtyCount' -v ./internal/app/
```

- [ ] `--- PASS: TestStatusInfoShowsAccumulatedTokens`
- [ ] `--- PASS: TestStatusInfoSessionID`
- [ ] `--- PASS: TestStatusInfoCompactionPercent`
- [ ] `--- PASS: TestStatusInfoActiveGrants`
- [ ] `--- PASS: TestStatusInfoDirtyCount`

**`/diff` shows working tree diff (background, 10s timeout, 200 KiB cap, unstaged only):**

```bash
go test -run 'TestDiffCommandDoesNotBlockUpdate|TestDiffCommandOpensModalWithCallbackLines|TestDiffCommandCleanTreeShowsNotice|TestDiffCommandErrorShowsNotice|TestDiffCommandWithoutCallbackShowsNotice|TestDiffResultTruncationMarkerDerivesFromCap|TestDiffResultDoesNotReplaceAnOpenModal|TestDiffModalLinesAreSanitised|TestDiffResultIgnoresStaleSequence|TestDiffResultDroppedAfterNewSession|TestDiffResultBlockedByConfirmPrompt|TestGetDiffCountsExactlyAndExcludesUntracked|TestGetDiffCleanTreeIsEmptyWithNoError|TestGetDiffErrorPaths|TestGetDiffHeaderExclusionStopsAtFirstHunk|TestGetDiffSanitisesLines|TestGetDiffAppliesTheByteCap|TestCountDiffLinesStatesTheHeaderRule|TestDiffCommandThroughRealWiring' -v ./internal/tui/ ./internal/app/ ./cmd/kirsch/
```

- [ ] `--- PASS: TestDiffCommandDoesNotBlockUpdate`
- [ ] `--- PASS: TestDiffCommandOpensModalWithCallbackLines`
- [ ] `--- PASS: TestDiffCommandCleanTreeShowsNotice`
- [ ] `--- PASS: TestDiffCommandErrorShowsNotice`
- [ ] `--- PASS: TestDiffCommandWithoutCallbackShowsNotice`
- [ ] `--- PASS: TestDiffResultTruncationMarkerDerivesFromCap`
- [ ] `--- PASS: TestDiffResultDoesNotReplaceAnOpenModal`
- [ ] `--- PASS: TestDiffModalLinesAreSanitised`
- [ ] `--- PASS: TestDiffResultIgnoresStaleSequence`
- [ ] `--- PASS: TestDiffResultDroppedAfterNewSession`
- [ ] `--- PASS: TestDiffResultBlockedByConfirmPrompt`
- [ ] `--- PASS: TestGetDiffCountsExactlyAndExcludesUntracked`
- [ ] `--- PASS: TestGetDiffCleanTreeIsEmptyWithNoError`
- [ ] `--- PASS: TestGetDiffErrorPaths`
- [ ] `--- PASS: TestGetDiffHeaderExclusionStopsAtFirstHunk`
- [ ] `--- PASS: TestGetDiffSanitisesLines`
- [ ] `--- PASS: TestGetDiffAppliesTheByteCap`
- [ ] `--- PASS: TestCountDiffLinesStatesTheHeaderRule`
- [ ] `--- PASS: TestDiffCommandThroughRealWiring`

**`/files` lists files touched by `apply_patch` this session (cleaned, de-duplicated, reset on resume):**

```bash
go test -run 'TestFilesCommandListsPaths|TestFilesCommandEmptyNotice|TestFilesCommandWithoutCallbackShowsNotice|TestFilesCommandSanitisesPaths|TestToolExecutorRecordsTouchedFilesOnlyForSuccessfulPatches|TestAddTouchedFileNormalisesAndDeduplicates' -v ./internal/tui/ ./internal/app/
```

- [ ] `--- PASS: TestFilesCommandListsPaths`
- [ ] `--- PASS: TestFilesCommandEmptyNotice`
- [ ] `--- PASS: TestFilesCommandWithoutCallbackShowsNotice`
- [ ] `--- PASS: TestFilesCommandSanitisesPaths`
- [ ] `--- PASS: TestToolExecutorRecordsTouchedFilesOnlyForSuccessfulPatches`
- [ ] `--- PASS: TestAddTouchedFileNormalisesAndDeduplicates`

**`/approvals` lists active grants and confirms clearing (y/n/Esc):**

```bash
go test -run 'TestApprovalsNoticeSanitisesGrants' -v ./internal/tui/
```

- [ ] `--- PASS: TestApprovalsNoticeSanitisesGrants`

**`/new` starts new session with confirm if turn in flight (resets conversation, tokens, compaction, touched, grants):**

```bash
go test -run 'TestSlashNewRunsCallbackInCmdAndResetsOnDone|TestSlashNewDoesNotBlockUpdate|TestSlashNewFailureKeepsTranscript|TestNewSessionIsolatesTheOldTurnFromTheNewSession|TestNewSessionFailureLeavesTheSessionIntact|TestSubmitRefusesWhileNewSessionWaitsForATurn|TestSlashNewThroughRealWiringStartsANewSession' -v ./internal/tui/ ./internal/app/ ./cmd/kirsch/
```

- [ ] `--- PASS: TestSlashNewRunsCallbackInCmdAndResetsOnDone`
- [ ] `--- PASS: TestSlashNewDoesNotBlockUpdate`
- [ ] `--- PASS: TestSlashNewFailureKeepsTranscript`
- [ ] `--- PASS: TestNewSessionIsolatesTheOldTurnFromTheNewSession`
- [ ] `--- PASS: TestNewSessionFailureLeavesTheSessionIntact`
- [ ] `--- PASS: TestSubmitRefusesWhileNewSessionWaitsForATurn`
- [ ] `--- PASS: TestSlashNewThroughRealWiringStartsANewSession`

**`/compact` triggers manual compaction on demand (status bar shows `compacting`, no turn start):**

```bash
go test -run 'TestCompactCommandHoldsBusyUntilRealOutcome|TestCompactCommandRefusesSubmitAndKeepsBusyOnEsc|TestManualCompactMsgOutcomesShowDistinctNotices|TestCompactCommandWithoutCallback|TestManualCompactAppliesWithoutStartingATurn|TestManualCompactWithTooFewTurnsReportsNothing|TestManualCompactProviderErrorIsAFailureNotAnOverflow|TestManualCompactCancelDuringModelCallReportsCancelled|TestManualCompactCancelAfterSummaryBeforeSwapNeverHalfApplies|TestManualCompactRefusesWhileATurnIsLive|TestSubmitRefusesWhileManualCompactIsRunning' -v ./internal/tui/ ./internal/app/
```

- [ ] `--- PASS: TestCompactCommandHoldsBusyUntilRealOutcome`
- [ ] `--- PASS: TestCompactCommandRefusesSubmitAndKeepsBusyOnEsc`
- [ ] `--- PASS: TestManualCompactMsgOutcomesShowDistinctNotices`
- [ ] `--- PASS: TestCompactCommandWithoutCallback`
- [ ] `--- PASS: TestManualCompactAppliesWithoutStartingATurn`
- [ ] `--- PASS: TestManualCompactWithTooFewTurnsReportsNothing`
- [ ] `--- PASS: TestManualCompactProviderErrorIsAFailureNotAnOverflow`
- [ ] `--- PASS: TestManualCompactCancelDuringModelCallReportsCancelled`
- [ ] `--- PASS: TestManualCompactCancelAfterSummaryBeforeSwapNeverHalfApplies`
- [ ] `--- PASS: TestManualCompactRefusesWhileATurnIsLive`
- [ ] `--- PASS: TestSubmitRefusesWhileManualCompactIsRunning`

**Dirty count on real workspace state:**

```bash
go test -run 'TestDirtyCount|TestDirtyCountNonGitAndErrorBranches' -v ./internal/workspace/
```

- [ ] `--- PASS: TestDirtyCount`
- [ ] `--- PASS: TestDirtyCountNonGitAndErrorBranches`

**Diff driver blocking — `--no-ext-diff --no-textconv`:**

```bash
go test -run 'TestDiffArgsShape|TestGitDiffDoesNotRunConfiguredDiffDrivers' -v ./internal/tool/
```

- [ ] `--- PASS: TestDiffArgsShape`
- [ ] `--- PASS: TestGitDiffDoesNotRunConfiguredDiffDrivers`

**Gates:**

```bash
npm run check
```

- [ ] `npm run check` exits 0 (fmt, vet, lint, test, screens all green)

```bash
npm run test:race
```

- [ ] `npm run test:race` exits 0 (race-enabled test run clean) — NOTE: `TestRunTurnApprovalToolUsesApprovalPath` (`internal/app/turn_adapter_test.go:348`) is a known PRE-EXISTING flake (2/10 under `-race`, proven pre-existing by the Step 2b stash protocol on the clean pre-delta tree). A full green run can require a retry; widen the collector timeout or count terminal messages if it trips. See *Carried from m4-d3* in plan/PROGRESS.md.

This section is the operator's independent verification of the m4-d8 acceptance (the boxes in milestone-4.md are already ticked with mission evidence — mission-20261017-03). It is the record for the nine m4-d8 acceptance boxes: `/status` fields; `/diff` on dirty/clean tree, background run, block UI block, 200 KiB cap, unstaged only, late result dropped, over modal/confirm; `/files` on touched paths, cleaned, de-duplicated, reset on resume; `/approvals` with/without grants confirming clear (y/n/Esc); `/new` with confirm if turn running, isolation, failure handling; `/compact` on demand without turn start, status glyph, notices, message refusal, failure handling; dirty count; and diff driver blocking.

---

## 9 · Token and Cost Tracking (m4-d9)

`internal/provider`, `internal/session`, `internal/app`, `internal/agent`, and `internal/tui` gained token and cost tracking in mission-20261017-05. `usage` events are recorded to `usage.json` as cumulative totals (prompt, completion, cache-read, cache-write, total, cost_usd). The status bar shows session totals `in N out N cache R/W · $cost` (4 decimals for costs < $0.01, 2 otherwise), with fallback `N tok` on narrow terminals. Cost is calculated per call at that call's model price; cache-write price is 1.25× input and unverified. An unknown model degrades gracefully with a once-per-session notice and uses the smallest context window in the model table (currently 128000) as the conservative budget. Protected by an advisory flock on `usage.json.lock` (~1s retry, then drops the increment and logs a warning). A JSON-corrupt file is moved aside to `usage.json.corrupt-<UTC nanosecond stamp>` and counting restarts. This section runs the unit, integration, and TUI tests that back the m4-d9 acceptance.

**Provider — cost calculation and unknown-model handling:**

```bash
go test -run 'TestUnknownModelFallback|TestCostPerTokenModel|TestCostFlatModel|TestCostUnknownModel|TestCostAllPerTokenModels|TestCostZeroTokens|TestConservativeBudgetEqualsTableMinimum|TestConservativeBudgetFollowsTheTable|TestCostStateStrings' -v ./internal/provider/
```

- [ ] `--- PASS: TestUnknownModelFallback`
- [ ] `--- PASS: TestCostPerTokenModel`
- [ ] `--- PASS: TestCostFlatModel`
- [ ] `--- PASS: TestCostUnknownModel`
- [ ] `--- PASS: TestCostAllPerTokenModels`
- [ ] `--- PASS: TestCostZeroTokens`
- [ ] `--- PASS: TestConservativeBudgetEqualsTableMinimum`
- [ ] `--- PASS: TestConservativeBudgetFollowsTheTable`
- [ ] `--- PASS: TestCostStateStrings`

**Session usage — file I/O, corruption recovery, concurrency, locking:**

```bash
go test -run 'TestUsageAddUpdatesTotals|TestAddUsageRootUpdatesFile|TestUsageAccumulationWithUnknownCost|TestUsageBackwardCompatibility|TestAddUsageConcurrentGoroutines|TestAddUsageCorruptFileRecovery|TestAddUsageRecoveriesInOneSecondKeepBothFiles|TestAddUsageUnreadableFileLeftUntouched|TestWriteJSONFileRemovesTempWhenRenameFails|TestWriteJSONFileLeavesNoTempOnSuccess|TestWriteJSONFileConcurrentWritersNeverCollide' -v ./internal/session/
```

- [ ] `--- PASS: TestUsageAddUpdatesTotals`
- [ ] `--- PASS: TestAddUsageRootUpdatesFile`
- [ ] `--- PASS: TestUsageAccumulationWithUnknownCost`
- [ ] `--- PASS: TestUsageBackwardCompatibility`
- [ ] `--- PASS: TestAddUsageConcurrentGoroutines`
- [ ] `--- PASS: TestAddUsageCorruptFileRecovery`
- [ ] `--- PASS: TestAddUsageRecoveriesInOneSecondKeepBothFiles`
- [ ] `--- PASS: TestAddUsageUnreadableFileLeftUntouched`
- [ ] `--- PASS: TestWriteJSONFileRemovesTempWhenRenameFails`
- [ ] `--- PASS: TestWriteJSONFileLeavesNoTempOnSuccess`
- [ ] `--- PASS: TestWriteJSONFileConcurrentWritersNeverCollide`

**Lock — advisory flock on usage.json.lock:**

```bash
go test -run 'TestAddUsageWaitsForHeldLock|TestAddUsageReturnsErrLockHeldWhenLockStaysHeld|TestTryLockInterruptHandling' -v ./internal/session/
```

- [ ] `--- PASS: TestAddUsageWaitsForHeldLock`
- [ ] `--- PASS: TestAddUsageReturnsErrLockHeldWhenLockStaysHeld`
- [ ] `--- PASS: TestTryLockInterruptHandling`

**Compaction — usage recorded before, during and after compaction:**

```bash
go test -run 'TestCompactionUsageCapture|TestCompactionUsageReturnedOnError|TestCompactionUsageReturnedOnEveryErrorPath' -v ./internal/session/
```

- [ ] `--- PASS: TestCompactionUsageCapture`
- [ ] `--- PASS: TestCompactionUsageReturnedOnError`
- [ ] `--- PASS: TestCompactionUsageReturnedOnEveryErrorPath`

**App — session totals accumulation and per-call pricing:**

```bash
go test -run 'TestSubmitSendsExactSessionTotalsPerTurn|TestSessionCostPricesEachCallAtItsOwnModel|TestSubmitCompactionSendsSessionTotals|TestManualCompactSendsSessionTotals|TestSessionTotalsRestartAfterNewSession|TestUnknownModelNoticeOnceAcrossTwoTurns|TestKnownModelShowsNoUnknownNotice|TestUnknownModelNoticeAgainAfterNewSession' -v ./internal/app/
```

- [ ] `--- PASS: TestSubmitSendsExactSessionTotalsPerTurn`
- [ ] `--- PASS: TestSessionCostPricesEachCallAtItsOwnModel`
- [ ] `--- PASS: TestSubmitCompactionSendsSessionTotals`
- [ ] `--- PASS: TestManualCompactSendsSessionTotals`
- [ ] `--- PASS: TestSessionTotalsRestartAfterNewSession`
- [ ] `--- PASS: TestUnknownModelNoticeOnceAcrossTwoTurns`
- [ ] `--- PASS: TestKnownModelShowsNoUnknownNotice`
- [ ] `--- PASS: TestUnknownModelNoticeAgainAfterNewSession`

**Compaction error paths — usage recorded on all failure paths:**

```bash
go test -run 'TestSubmitCompactionRecordsSummaryAndTurnUsage|TestManualCompactRecordsSummaryUsage|TestSubmitCompactionFailureRecordsUsageAndLeavesSessionUntouched|TestManualCompactStreamErrorRecordsUsage|TestRecordUsageReportsCorruptUsageFileThroughLogger' -v ./internal/app/
```

- [ ] `--- PASS: TestSubmitCompactionRecordsSummaryAndTurnUsage`
- [ ] `--- PASS: TestManualCompactRecordsSummaryUsage`
- [ ] `--- PASS: TestSubmitCompactionFailureRecordsUsageAndLeavesSessionUntouched`
- [ ] `--- PASS: TestManualCompactStreamErrorRecordsUsage`
- [ ] `--- PASS: TestRecordUsageReportsCorruptUsageFileThroughLogger`

**Wiring — usage event reaches the status bar through the real pipeline:**

```bash
go test -run 'TestProviderEventToAgentMapsUsage|TestUsageNonzero|TestUsageEventThroughRealWiringReachesTheStatusBar' -v ./internal/app/ ./internal/agent/ ./cmd/kirsch/
```

- [ ] `--- PASS: TestProviderEventToAgentMapsUsage`
- [ ] `--- PASS: TestUsageNonzero`
- [ ] `--- PASS: TestUsageEventThroughRealWiringReachesTheStatusBar`

**TUI rendering — status bar layout, cost format, unknown-model notice:**

```bash
go test -run 'TestSessionUsageWideRender|TestSessionUsageFlatNoCost|TestSessionUsageSmallCostFormat|TestSessionUsageUnknownPricingShowsQuestionMark|TestSessionUsageCostBoundary|TestSessionUsageASCIIGlyphs|TestSessionUsageNarrowLayoutFallsBackToSessionTotal|TestStatusLadderDropsTokensBeforeGrants|TestUnknownModelNotice' -v ./internal/tui/
```

- [ ] `--- PASS: TestSessionUsageWideRender`
- [ ] `--- PASS: TestSessionUsageFlatNoCost`
- [ ] `--- PASS: TestSessionUsageSmallCostFormat`
- [ ] `--- PASS: TestSessionUsageUnknownPricingShowsQuestionMark`
- [ ] `--- PASS: TestSessionUsageCostBoundary`
- [ ] `--- PASS: TestSessionUsageASCIIGlyphs`
- [ ] `--- PASS: TestSessionUsageNarrowLayoutFallsBackToSessionTotal`
- [ ] `--- PASS: TestStatusLadderDropsTokensBeforeGrants`
- [ ] `--- PASS: TestUnknownModelNotice`

**Gates:**

```bash
npm run check
```

- [ ] `npm run check` exits 0 (fmt, vet, lint, test, screens all green)

```bash
npm run test:race
```

- [ ] `npm run test:race` exits 0 (race-enabled test run clean) — NOTE: `TestRunTurnApprovalToolUsesApprovalPath` (`internal/app/turn_adapter_test.go:348`) is a known PRE-EXISTING flake (2/10 under `-race`). A full green run can require a retry.

This section is the operator's independent verification of the m4-d9 acceptance (the boxes in milestone-4.md are already ticked with mission evidence — mission-20261017-05). It is the record for the five m4-d9 acceptance boxes: `usage` events recorded to `usage.json`; status bar shows real token counts; unknown models degrade with warning and conservative budget; cost calculated from model pricing; and 38 unit tests covering usage accumulation, unknown-model degradation, file I/O, corruption recovery, concurrency, locking, compaction, session totals, per-call pricing, error paths, wiring and TUI rendering.

---

## 10 · Final-Task Report (m4-d10)

**Testable.** The system prompt (internal/agent/prompt/system.md §6) enforces the final-task report format: every task that took actions must end with a structured report (Summary, Files changed, Commands run with PASS/FAIL, Limitations). A pure question may be answered without a report. The report structure is preserved across compaction: the compaction prompt preserves the most recent report verbatim. This section runs the unit tests that back the m4-d10 acceptance.

**Automated — unit tests verifying system prompt and compaction:**

(a) System prompt enforcement:

```bash
go test -count=1 -run 'TestAssembleFinalReportEnforcement' ./internal/agent/prompt/
```

- [ ] `--- PASS: TestAssembleFinalReportEnforcement`

(b) Compaction prompt preservation and structure:

```bash
go test -count=1 -run 'TestCompactionSystemPromptCommandOutcomes|TestCompactionSystemPromptPreservesFinalReport|TestCompactionRequestShape' ./internal/session/
```

- [ ] `--- PASS: TestCompactionSystemPromptCommandOutcomes`
- [ ] `--- PASS: TestCompactionSystemPromptPreservesFinalReport`
- [ ] `--- PASS: TestCompactionRequestShape`

(c) System prompt survives compaction:

```bash
go test -count=1 -run 'TestSystemPromptSurvivesCompaction' ./internal/app/
```

- [ ] `--- PASS: TestSystemPromptSurvivesCompaction`

**Live — report production and compaction survival:**

(d) Give Kirsch a small task that edits a file and runs `go test ./...`, then check that the reply ends with the four labelled report sections (Summary, Files changed, Commands run, Limitations) and that PASS/FAIL appears for each command:

- [ ] Final report is present, properly formatted, and shows PASS/FAIL per command

(e) Ask a pure question (e.g., "What is the Fibonacci sequence?") and check that no report is required:

- [ ] Pure question does not require a final report

(f) Run a long session, or execute `/compact` to trigger compaction, then finish a task and check that the report is still produced (this is the only real check of survival):

- [ ] Final report still appears after compaction

**Gates:**

```bash
npm run check
```

- [ ] `npm run check` exits 0 (fmt, vet, lint, test, screens all green)

---

## 11 · Smoke Test (m4-d11)

**Not yet testable.** This section is filled in with runnable steps when m4-d11 lands.

What will be checked (from milestone-4.md m4-d11 acceptance): `scripts/smoke-test.sh`
exists and is executable; it runs against `testdata/repo-go-module` with a real
model; the prompt is "Add input validation to the Divide function and cover it with
a test"; the run produces a shown patch, applies it after approval, runs
`go test ./...`, and ends with a report that names exactly the expected files; and
the full end-to-end integration test passes.

---

## 12 · Final Acceptance (m4-d12)

**Not yet testable.** This section is filled in with runnable steps when m4-d12 lands.

What will be checked (from milestone-4.md m4-d12 acceptance — the old Task 9 boxes):
the smoke test passes on a real model end to end; quit mid-task, resume, and
complete the turn — identical transcript, pending state and grants, valid message
array; a session truncated by `kill -9` loads as `recovered` and is resumable; a
second instance warns and degrades per §7; compaction is cancellable and never
half-applies, and failure surfaces `context_overflow`; all slash commands operate
on real state; token and cost tracking appears in the status bar and `usage.json`;
and `go test -race ./...`, `golangci-lint fmt --diff`, `go vet`, `golangci-lint run`,
and CI are all clean.

---

## Known limitations

- `npm run security` is RED pre-existing: govulncheck reports 10 vulnerabilities in `golang.org/x/net` v0.56.0 (fixed in v0.60.0) and `net/http` at `go1.25.13` (fixed in `go1.26.9`), which appeared between 2026-10-08 and 2026-10-14 with zero repository change. None are in Kirsch code; the scoped `govulncheck ./internal/session/...` is clean. The operator's dependency decision is pending.
- m4-d1's JSONL has `omitempty` gaps: `tool.completed` omits `tool` and `duration_ms`, and `assistant.message` omits `stop_reason`. These are populated at the m4-d3 wiring, which records the real event stream.
- No live model is needed until m4-d11. The default model `minimax-m2.7` applies there; `KIRSCH_LIVE_MODEL` overrides only the live test, not the TUI.

---

## Reporting

For anything marked ❌:

1. The exact command and what happened
2. Your terminal size (`echo $COLUMNS $LINES`) and whether you are in tmux
3. The section number and case name
4. The config file content if the case involves one
5. The debug log (`cat "$XDG_STATE_HOME/kirsch/debug.log"`) if relevant

Never paste the API key into a report.
