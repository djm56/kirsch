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
| 3 | m4-d3 Event Recording Wiring | Not yet testable |
| 4 | m4-d4 Resume Reconstruction | Not yet testable |
| 5 | m4-d5 Resume State + CLI | Not yet testable |
| 6 | m4-d6 Single-Instance Guard | Not yet testable |
| 7 | m4-d7 Compaction | Not yet testable |
| 8 | m4-d8 Slash Commands for Real | Not yet testable |
| 9 | m4-d9 Token and Cost Tracking | Not yet testable |
| 10 | m4-d10 Final-Task Report | Not yet testable |
| 11 | m4-d11 Smoke Test | Not yet testable |
| 12 | m4-d12 Final Acceptance | Not yet testable |

Sections marked not yet testable are kept as placeholders and are filled in with
runnable steps as each deliverable lands.

---

## Model for this run

No live model has been used for this walkthrough yet — §1 runs only unit and race
tests against the checkout, and §2–§12 are placeholders. The first live model use
is the m4-d11 smoke test. When that section is filled in, the default model
`minimax-m2.7` applies there (plan amendment 83), and `KIRSCH_LIVE_MODEL` overrides
only the live test, not the TUI. No model choices have been made for this run.

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
m4-d1 acceptance, grouped to match the commands below. Tick each box as you run it.

**Event mapping — every RecordKind maps to the expected plan §7 event type, every line carries `"v":1`:**

```bash
go test -run 'TestAppendMapping|TestAppendToolEvents' -v ./internal/session/
```

- [✅] `--- PASS: TestAppendMapping`
- [✅] `--- PASS: TestAppendToolEvents`

**Durability, cadence, close, and the error hook:**

```bash
go test -run 'TestFlushDurability|TestCadenceFlush|TestCloseSemantics|TestOnErrorHook' -v ./internal/session/
```

- [✅] `--- PASS: TestFlushDurability`
- [✅] `--- PASS: TestCadenceFlush`
- [✅] `--- PASS: TestCloseSemantics`
- [✅] `--- PASS: TestOnErrorHook`

**Concurrent append — many goroutines record without interleaving or loss:**

```bash
go test -run TestConcurrentAppend -v ./internal/session/
```

- [✅] `--- PASS: TestConcurrentAppend`

**Optional benchmark — the cost of queuing an event:**

```bash
go test -bench=BenchmarkRecord -benchtime=1s ./internal/session/
```

- [✅] `BenchmarkRecord` runs (optional)

**Gates:**

```bash
npm run check
```

- [✅] `npm run check` exits 0 (fmt, vet, lint, test, screens all green)

```bash
npm run test:race
```

- [✅] `npm run test:race` exits 0 (race-enabled test run clean)

```bash
$(go env GOPATH)/bin/govulncheck ./internal/session/...
```

- [ ] `govulncheck ./internal/session/...` reports no vulnerabilities in Kirsch code (the full path is used so no PATH setup is needed; the binary installs to `$(go env GOPATH)/bin/govulncheck`, and `npm run tools` installs it and prints the directory)

Findings
=== Symbol Results ===

No vulnerabilities found.

Your code is affected by 0 vulnerabilities.
This scan also found 1 vulnerability in packages you import and 12
vulnerabilities in modules you require, but your code doesn't appear to call
these vulnerabilities.
Use '-show verbose' for more details.

But we can ignore this for now we can address this later

This section is the operator's independent verification of the m4-d1 acceptance
(the boxes in milestone-4.md are already ticked with mission evidence —
mission-20261014-01).

---

## 2 · Storage Layout + Repair (m4-d2)

`internal/session` gained the plan §7 storage layout and repair-on-load in mission-20261014-05. The corrected layout (post fix round 2a) is: state root `~/.local/share/kirsch/` (XDG-adjacent); sessions live under `<root>/sessions/<workspace-hash>/`; each session file is `<timestamp>-<ulid>.jsonl`; and `index.json` (canonical workspace path → `{last_session_id, updated_at}`) and `usage.json` (cumulative token totals) sit at the root beside `sessions/` — matching plan §7's tree (kirsch-plan.md:366–374). The ULID is a stdlib-only Crockford base32 encoding (48-bit millisecond timestamp + 80 bits `crypto/rand`), so no new module was added. This section runs the unit, race, and fuzz tests that back the m4-d2 acceptance, grouped to match the commands below. Tick each box as you run it.

**Layout + index — path generation, the §7 tree, and index.json read/write:**

```bash
go test -run 'TestDefaultStateRoot|TestCanonicalize|TestWorkspaceHash|TestSessionID|TestSessionPath|TestNewLayout|TestLayoutCreateSession|TestLayoutCreateSessionSortable|TestLayoutLastSession|TestIndexPathAndUsagePath|TestLayoutIndexJSONSchema|TestReadIndexMissing|TestWriteAndReadIndex|TestIndexSetUpdatesEntry|TestIndexMultipleWorkspaces' -v ./internal/session/
```

- [✅] `--- PASS: TestDefaultStateRoot`
- [✅] `--- PASS: TestCanonicalize`
- [✅] `--- PASS: TestWorkspaceHash`
- [✅] `--- PASS: TestSessionID`
- [✅] `--- PASS: TestSessionPath`
- [✅] `--- PASS: TestNewLayout`
- [✅] `--- PASS: TestLayoutCreateSession`
- [✅] `--- PASS: TestLayoutCreateSessionSortable`
- [✅] `--- PASS: TestLayoutLastSession`
- [✅] `--- PASS: TestIndexPathAndUsagePath`
- [✅] `--- PASS: TestLayoutIndexJSONSchema`
- [✅] `--- PASS: TestReadIndexMissing`
- [✅] `--- PASS: TestWriteAndReadIndex`
- [✅] `--- PASS: TestIndexSetUpdatesEntry`
- [✅] `--- PASS: TestIndexMultipleWorkspaces`

**Usage — usage.json schema and the update path:**

```bash
go test -run 'TestReadUsageMissing|TestWriteAndReadUsage|TestUsageAddUpdatesTotals|TestAddUsageRootUpdatesFile' -v ./internal/session/
```

- [✅] `--- PASS: TestReadUsageMissing`
- [✅] `--- PASS: TestWriteAndReadUsage`
- [✅] `--- PASS: TestUsageAddUpdatesTotals`
- [✅] `--- PASS: TestAddUsageRootUpdatesFile`

**Repair-on-load — a torn final line truncates and marks the session `recovered`, never blocking startup:**

```bash
go test -run 'TestRepairTruncatesTornFinalLine|TestRepairNoCorruption|TestRepairEmptyFile' -v ./internal/session/
```

- [✅] `--- PASS: TestRepairTruncatesTornFinalLine`
- [✅] `--- PASS: TestRepairNoCorruption`
- [✅] `--- PASS: TestRepairEmptyFile`

**Gates:**

```bash
npm run check
```

- [✅] `npm run check` exits 0 (fmt, vet, lint, test, screens all green)

```bash
npm run test:race
```

- [✅] `npm run test:race` exits 0 (race-enabled test run clean)

```bash
$(go env GOPATH)/bin/govulncheck ./internal/session/...
```

- [✅] `govulncheck ./internal/session/...` reports no vulnerabilities in Kirsch code (the full path is used so no PATH setup is needed; the binary installs to `$(go env GOPATH)/bin/govulncheck`, and `npm run tools` installs it and prints the directory)

```bash
FUZZTIME=10s npm run fuzz
```

- [✅] `FUZZTIME=10s npm run fuzz` — `FuzzSessionPathContained` survives 10s (all fuzz targets survive 10s each)

This section is the operator's independent verification of the m4-d2 acceptance (the boxes in milestone-4.md are already ticked with mission evidence — mission-20261014-05). It is the record for the five m4-d2 acceptance boxes: the §7 storage layout, `index.json` mapping, `usage.json` existence, repair-on-load → `recovered`, and the unit tests.

---

## 3 · Event Recording Wiring (m4-d3)

**Not yet testable.** This section is filled in with runnable steps when m4-d3 lands.

What will be checked (from milestone-4.md m4-d3 acceptance): `app.Submit()` sets
`ag.Recorder` to the session store; every M3 event type is recorded to JSONL with
the correct plan §7 schema (`user.message`, `assistant.message`,
`assistant.thinking`, `tool.completed`, `usage`, `approval.scope_granted`,
`turn.started`/`turn.completed`); and the store's `OnError` hook is wired to the
telemetry debug log so mid-turn write/`fsync` failures surface, never silently lost.

---

## 4 · Resume Reconstruction (m4-d4)

**Not yet testable.** This section is filled in with runnable steps when m4-d4 lands.

What will be checked (from milestone-4.md m4-d4 acceptance): `session.Resume(path)`
rebuilds the `*agent.Conversation` and transcript from `user.message`,
`assistant.message`, `assistant.thinking` and `tool.completed` only — never
`assistant.delta`; `compaction.applied` events replace the compacted range with
the summary; tool call ids correlate (every `tool.completed` matches its
`assistant.message` tool_call); and the result is a valid provider message array.

---

## 5 · Resume State + CLI (m4-d5)

**Not yet testable.** This section is filled in with runnable steps when m4-d5 lands.

What will be checked (from milestone-4.md m4-d5 acceptance): `kirsch resume` loads
the last session for the workspace via `index.json`; `kirsch --new` starts fresh
even if a last session exists; with no flags Kirsch auto-resumes the last session;
session-scoped grants are restored from `approval.scope_granted` events; pending
state (the in-progress turn) is restored; and an integration test — quit mid-task,
resume, complete the turn — yields an identical transcript, grants, and a valid
message array.

---

## 6 · Single-Instance Guard (m4-d6)

**Not yet testable.** This section is filled in with runnable steps when m4-d6 lands.

What will be checked (from milestone-4.md m4-d6 acceptance): on start Kirsch takes
an advisory `flock` on `<workspace-hash>/.lock`; a second instance in the same
workspace does **not** auto-resume (starts fresh), shows `⚠ another Kirsch is
running here` in the status bar, and disables session-scoped grants.

---

## 7 · Compaction (m4-d7)

**Not yet testable.** This section is filled in with runnable steps when m4-d7 lands.

What will be checked (from milestone-4.md m4-d7 acceptance): compaction is a model
call using the session's provider and model; it keeps the original task verbatim,
the last 4 turns verbatim, and a structured summary of the middle; it is
cancellable (a cancelled compaction leaves the session uncompacted and the pending
turn unstarted); on failure it surfaces `context_overflow` and tells the user to
`/new` rather than sending a silent oversized request; thinking blocks are dropped
(not summarised) by checking `Thinking.DropOnSummary`; and a `compaction.applied`
event is recorded with the replaced range and summary.

---

## 8 · Slash Commands for Real (m4-d8)

**Not yet testable.** This section is filled in with runnable steps when m4-d8 lands.

What will be checked (from milestone-4.md m4-d8 acceptance): `/status` shows model,
branch, dirty count, session id, tokens (in/out/cache), compaction %,
project-context file, and active grants; `/diff` shows the working tree diff;
`/files` lists files touched this session; `/approvals` lists active grants and
offers to clear; `/new` starts a new session (confirming if a turn is in flight);
`/compact` triggers manual compaction; `/quit` and `/exit` quit (alias); and an
unknown `/command` shows a dim inline hint, never an error card, and is never sent
to the model.

---

## 9 · Token and Cost Tracking (m4-d9)

**Not yet testable.** This section is filled in with runnable steps when m4-d9 lands.

What will be checked (from milestone-4.md m4-d9 acceptance): `usage` events are
recorded to `usage.json` as cumulative token/cost totals; the status bar shows real
token counts (input, output, cache read, cache write); and an unknown model
degrades with a warning and a conservative budget (not a failure), suggesting the
smallest context window in the §5 model table as the fallback budget.

---

## 10 · Final-Task Report (m4-d10)

**Not yet testable.** This section is filled in with runnable steps when m4-d10 lands.

What will be checked (from milestone-4.md m4-d10 acceptance): the system prompt
enforces the final-task report — summary, files changed, commands run with
pass/fail, and limitations — and the report survives compaction (it is included in
the summary).

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
