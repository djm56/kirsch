# Milestone 4 — Instruction Set

> **Status: Detailed instruction set — rewritten 2026-10-13 (mission-20261013-06, plan §11 amendment 86).** The outline drafted 2026-09-12 (amendment 50) is superseded. Milestone 3 is complete and operator-accepted (2026-10-13), so the agent-loop shape the old document assumed is now known from the code rather than guessed. Per the operator's directive that Milestone 4 coding be broken into much smaller junior-developer chunks, this rewrite expands the scope statement into twelve deliverables (m4-d1…m4-d12), each sized for one small junior mission (one @hicks code step + one @apone review, at most one fix round). This rewrite executes the old "Note for whoever executes this" (re-read against the code and rewrite what no longer fits): the three questions it flagged are answered from the M3 code in *The three questions, answered*, and the old Tasks 1–9 are re-cut into the twelve chunks with the Task→chunk mapping recorded for mechanical coverage. The tree lands uncommitted for operator review.

## Ground Rules (read first)

1. **Milestone 3 must be complete.** Every box ticked, CI green.
2. **This is the milestone where Kirsch becomes useful and dangerous at the same
   time.** It completes real tasks against real repositories. The smoke test is
   not a formality; it is the only thing that exercises the whole system with a
   real model.
3. **Locked decisions** (do not re-litigate):
   - The reconstruction rule (plan §7): resume rebuilds the provider message
     array from `user.message`, `assistant.message`, `assistant.thinking` and
     `tool.completed` — **never** from `assistant.delta`. Deltas exist to replay
     the *look* of streaming and are not the record of what was sent.
   - One goroutine owns all JSONL writes (architecture.md §3 rule 3).
   - Compaction rewrites only the *conversation*. The transcript and the log
     keep everything, which is why `/compact` does not make scrollback vanish.
4. Every task has a checkable result.

---

## The three questions, answered

The 2026-09-12 outline deferred three questions to "Milestone 3's implementation". M3 is landed, so they are now answered from the code (research doc `m4-research.md`, @apone, mission-20261013-06 Step 1).

### Q1: How does state flow from one turn to the next in the actual agent loop?

State flows through a single persistent `*agent.Conversation` held by `App.conv`.

- `internal/app/app.go:372` — `App` struct field `conv *agent.Conversation`.
- `internal/app/app.go:394` — `App.New()` initialises it: `conv: &agent.Conversation{}`.
- `internal/app/app.go:658` — `Submit()` passes the same `conv` pointer to `RunTurn`: `a.RunTurn(ag, a.conv, userText)`.
- `internal/app/app.go:832` — `RunTurn` signature: `func (a *App) RunTurn(ag *agent.Agent, conv *agent.Conversation, userText string)`.
- `internal/agent/agent.go:406` — `Agent.Turn` mutates `conv` in place: `func (a *Agent) Turn(ctx context.Context, conv *Conversation, userText string, emit func(Event)) error`.
- `internal/agent/agent.go:482-484` — `appendMessage` appends to `conv.Messages`: `conv.Messages = append(conv.Messages, m)` then `a.record(RecordAppend, m)`.

**Flow per turn:** user text appended as `user.message` → `conv.Messages` (agent.go:482); a deep copy of `conv.Messages` is sent to the model (agent.go:418); the stream accumulates the assistant message (agent.go:497-502); on `MessageDone` the assistant message is appended to `conv` (agent.go:482); tool calls run sequentially and results append as `user.message` (agent.go:482); the loop continues until no more tool calls or an error / max-turns.

**Key point for M4:** `conv` is the in-memory session state. There is no persistence layer yet. The `Recorder` interface exists (`internal/agent/agent.go:325-328`) but is currently nil in production (`internal/app/app.go:628-632` — `Agent` construction does not set `Recorder`).

### Q2: What form does the pending turn take in the session file?

There is no session file yet. `internal/session` does not exist. The pending turn is the in-memory `*agent.Conversation` plus the session-scoped approval grants held by `*policy.Policy`.

- `internal/agent/agent.go:330-334` — `Conversation` struct: `Messages []Message`.
- `internal/policy/policy.go:53-69` — `Policy` holds `grants [][]string`.
- `internal/policy/policy.go:211` — `Grant(argv)` appends to `grants`.
- `internal/policy/policy.go:240` — `ClearGrants()` resets `grants = nil`.

**What M4 must build** (plan §7): a JSONL session file with the event schema; a single writer goroutine (architecture.md §3 rule 3); `index.json` for auto-resume; resume reconstruction from `user.message`, `assistant.message`, `assistant.thinking`, `tool.completed` — **never** `assistant.delta`; and restoration of session-scoped grants from `approval.scope_granted` events.

**The Recorder seam** (`internal/agent/agent.go:325-328`):

```go
type Recorder interface {
    // Record records one change to the conversation.
    Record(kind RecordKind, m Message)
}
```

This is the m3-d3 carried item: "Recorder has no error return." The store is built around this interface; the decision is recorded in m4-d1.

### Q3: Where does thinking get stored, and what gets compacted out?

Thinking blocks are stored in `agent.Thinking` with a `DropOnSummary bool` marker. They are persisted to the conversation (and will be persisted to JSONL as `assistant.thinking` events). Compaction drops them, does not summarise them.

- `internal/agent/agent.go:94-107` — `Thinking` struct, including `DropOnSummary bool` ("the M4-compaction marker. When true, compactors must drop this block rather than summarise it").
- `internal/agent/agent.go:758` — marker set on every `EventThinkingDone`: `s.open.DropOnSummary = true`.
- `plan/spec/kirsch-plan.md:330` — persistence rule: "persisted to JSONL as `assistant.thinking` (with its signature) so resume rebuilds a valid message array".
- `plan/spec/kirsch-plan.md:331` — compaction rule: "excluded from compaction summaries — dropped, not summarised."

**Key point for M4:** the `DropOnSummary` marker is already in place. Compaction must check this field and drop the block rather than include it in the summary. The JSONL event `assistant.thinking` must carry the signature so resume can echo it back.

---

## Deliverables

This milestone breaks into twelve independently claimable deliverables, each sized for one small junior mission (one @hicks code step + one @apone review, at most one fix round). Each deliverable specifies what it covers, what it owns in the codebase, and which other deliverables must be complete before work can start. Two developers can work in parallel on deliverables with no shared files and no dependency relationship (see the parallel notes per chunk).

| ID | Deliverable | Depends on | File ownership | Parallel | Acceptance summary |
|---|---|---|---|---|---|
| m4-d1 | Session Store Core | — | `internal/session/` | No (foundation) | `Store` implements `agent.Recorder`; JSONL `"v":1`; single writer goroutine; buffered `fsync` ~500ms + on flush; unit tests for append/flush/concurrent append |
| m4-d2 | Storage Layout + Repair | m4-d1 | `internal/session/` | No | Plan §7 path layout; `index.json`; `usage.json`; repair-on-load truncates torn final line → `recovered`; unit tests |
| m4-d3 | Event Recording Wiring | m4-d1, m4-d2 | `internal/session/`, `internal/app/` | No | `Submit()` sets `ag.Recorder`; every M3 event type recorded with correct schema; `usage` + `approval.scope_granted` + `turn.started/completed` recorded; integration test |
| m4-d4 | Resume Reconstruction | m4-d3 | `internal/session/`, `internal/app/` | No | `session.Resume` rebuilds `*agent.Conversation` + transcript from `user/assistant.message/thinking/tool.completed` only; `compaction.applied` applied; tool ids correlate; unit tests |
| m4-d5 | Resume State + CLI | m4-d4 | `internal/session/`, `internal/app/`, `cmd/kirsch/` | No | `kirsch resume` / `--new` / auto-resume via `index.json`; grants restored; pending state restored; integration test |
| m4-d6 | Single-Instance Guard | m4-d2 | `internal/session/`, `internal/app/`, `internal/tui/` | Yes (with m4-d3/4/5) | Advisory `flock` on `<workspace-hash>/.lock`; second instance degrades (fresh start, `⚠ another Kirsch is running here`, grants disabled); unit + integration tests |
| m4-d7 | Compaction | m4-d3 | `internal/session/`, `internal/app/`, `internal/agent/` | Yes (with m4-d4/5/6) | Model call using session provider/model; keeps task + last 4 turns + structured summary; cancellable; failure → `context_overflow`; thinking dropped; `compaction.applied` recorded; unit tests |
| m4-d8 | Slash Commands for Real | m4-d3, m4-d5, m4-d7 | `internal/tui/`, `internal/app/` | No | `/status` shows model/branch/dirty/session id/tokens(in/out/cache)/compaction%/context file/grants; `/diff` `/files` `/approvals` `/new` `/compact` `/quit` `/exit` on real state; unknown command → dim hint; cache tokens resolve m3-d2 |
| m4-d9 | Token and Cost Tracking | m4-d3 | `internal/telemetry/`, `internal/tui/`, `internal/session/` | Yes (with m4-d4/5/6) | `usage` events → `usage.json` cumulative totals; status bar real token counts (in/out/cache read/write); unknown model degrades with warning + conservative budget; unit tests |
| m4-d10 | Final-Task Report | m4-d7 | `internal/agent/prompt/`, `internal/app/` | Yes (with m4-d8/9) | System prompt enforces final-task report (summary, files changed, commands run pass/fail, limitations); survives compaction; unit test |
| m4-d11 | Smoke Test | m4-d1…m4-d10 | `scripts/` | No | `scripts/smoke-test.sh` executable; runs on `testdata/repo-go-module` with real model; prompt "Add input validation to the Divide function and cover it with a test"; produces patch, applies after approval, runs `go test ./...`, ends with report naming expected files; integration test |
| m4-d12 | Final Acceptance | m4-d11 | `plan/PROGRESS.md`, `CHANGELOG.md`, `plan/spec/kirsch-plan.md` | No | All Task 9 boxes checked; CHANGELOG updated; `../PROGRESS.md` ☑ Complete; commit tagged `m4`; `go test -race ./...`, `golangci-lint fmt --diff`, `go vet`, `golangci-lint run`, CI clean |

### Task→chunk coverage (DIR-014 — mechanically checkable)

Every old Task 1–9 item lands in exactly one chunk (research doc coverage check, PASS):

| Old Task | Chunk(s) |
|---|---|
| Task 1 (session store) | m4-d1, m4-d2 |
| Task 2 (resume) | m4-d4, m4-d5 |
| Task 3 (single-instance guard) | m4-d6 |
| Task 4 (compaction) | m4-d7 |
| Task 5 (slash commands) | m4-d8 |
| Task 6 (token/cost tracking) | m4-d9 |
| Task 7 (final-task report) | m4-d10 |
| Task 8 (smoke test) | m4-d11 |
| Task 9 (final acceptance) | m4-d12 |

Chunks m4-d3 (event recording wiring) and m4-d12 (final acceptance) are not themselves old tasks: m4-d3 is the enabling wiring that the old Task 1's "store" implicitly required and that every later chunk depends on; m4-d12 consolidates the old Task 9 acceptance. Nothing in *Explicitly Out of Scope* appears in any chunk.

---

## What makes this milestone hard

**Three representations of one conversation, and they are not the same thing.**
architecture.md §6 names them: the transcript (what the user saw, untruncated),
the conversation (what the model sees, truncated and compacted), and the session
log (every event, append-only). Conflating any two is a bug, and this is the
milestone where all three exist simultaneously for the first time. Most of the
subtle failures here will be one of them standing in for another.

**Resume is not "load the file".** It has to reconstruct a *valid provider
message array* — every tool call answered, thinking blocks in place with their
signatures, compaction applied rather than replayed. A session that loads and
renders beautifully but produces an API-shape error on the next turn has failed.

**Durability has a cost and a deadline.** `fsync` every ~500ms and on turn
completion. A `kill -9` mid-write must leave a file that loads as `recovered`
rather than one that refuses to load at all.

---

## m4-d1 — Session Store Core

**Scope:** The Recorder-seam decision applied (see below); JSONL envelope `"v":1` (ADR 0002); single writer goroutine (architecture.md §3 rule 3); append API; buffered flush with `fsync` every ~500ms and on an explicit flush call (turn completion).

**Dependencies:** None — this is the foundation.

**File ownership:** `internal/session/` (new package).

**Parallel execution:** None — every later chunk depends on it.

**Recorder-seam decision (Bishop; operator-confirmed 2026-10-14).** Keep the no-error `Recorder` interface as landed and reviewed in M3 (`internal/agent/agent.go:313–328`): `Record(kind RecordKind, m Message)`, called synchronously from `Agent.Turn` (`agent.go:488–492`), handing `Record` a deep copy it owns. Durability and error-surfacing live in the store layer, not the interface:

- The single writer goroutine (architecture.md §3 rule 3) owns all writes and `fsync`s.
- A write/`fsync` failure **at turn completion** surfaces to the user — the turn cannot be considered durable.
- A mid-turn write/`fsync` failure goes to the telemetry debug log — the turn continues, and repair-on-load (m4-d2) recovers a torn final line.

This is a Bishop decision, confirmed by the operator on 2026-10-14 (mission-20261014-01, "happy with your suggestion") and applied as the instruction for m4-d1: the no-error `Recorder` interface is kept, with durability and error-surfacing in the store layer. The tree lands uncommitted; the decision is no longer pending operator override.

**Instruction items:**

1. Create `internal/session` with a `Store` type that implements `agent.Recorder` (interface at `agent.go:313–328`).
2. Apply the Recorder-seam decision above: keep the no-error `Record(kind RecordKind, m Message)` signature; durability and error-surfacing live in the store layer.
3. Write JSONL with `"v":1` on every line (ADR 0002; plan §7 schema).
4. Run a single writer goroutine that owns all writes (architecture.md §3 rule 3); every other goroutine posts events to it.
5. Buffer writes and `fsync` every ~500ms and on an explicit flush call (turn completion).
6. Unit-test append, flush, and concurrent-append safety.

**Acceptance:**

- [x] `internal/session` package exists with `Store` type
  - (store.go:46 `type Store struct`; mission-20261014-01, review APPROVED 0 CRITICAL / 0 WARNINGS)
- [x] `Store` implements `agent.Recorder` interface
  - (store_test.go:433 `var _ agent.Recorder = (*Store)(nil)`; no-error signature per the operator-confirmed seam decision)
- [x] JSONL file created with `"v":1` on every line
  - (event.go:18 `eventVersion = 1`, the `Version int \`json:"v"\`` field on every event constructor — event.go:123, 154, 169, 205, 228)
- [x] Single writer goroutine owns all writes
  - (store.go:110 `go s.writer()`, writer loop store.go:187–217; Record posts to a buffered channel, cap 256)
- [x] Buffered flush with `fsync` every ~500ms and on explicit flush call
  - (store.go:92 default cadence 500ms, store.go:190 ticker, store.go:203–206 fsync, store.go:151–171 `Flush() error`)
- [x] Unit tests: append, flush, concurrent append safety
  - (store_test.go: TestAppendMapping, TestAppendToolEvents, TestFlushDurability, TestConcurrentAppend, TestCadenceFlush, TestCloseSemantics, TestOnErrorHook; `npm run check` and `npm run test:race` exit 0)

**Must-know-before:** The Recorder-seam decision above — this chunk applies it.

**Maps from old Task(s):** Task 1 (session store), with m4-d2.

---

## m4-d2 — Storage Layout + Repair

**Scope:** `<workspace-hash>/<timestamp>-<ulid>.jsonl` path structure (plan §7); `index.json` (canonical path → last session); `usage.json`; repair-on-load (a torn final line truncates and marks the session `recovered`).

**Dependencies:** m4-d1 (store core).

**File ownership:** `internal/session/`.

**Parallel execution:** None — depends on m4-d1.

**Instruction items:**

1. Implement the plan §7 storage layout: `~/.local/share/kirsch/sessions/<workspace-hash>/<timestamp>-<ulid>.jsonl`, plus `index.json` and `usage.json`.
2. `index.json` maps the canonical workspace path → `{last_session_id, updated_at}` (plan §7 storage layout).
3. `usage.json` exists with its schema defined, even if empty.
4. Repair-on-load: a truncated final line truncates the session there and marks it `recovered`; startup never blocks on one bad file (plan §7 rules).
5. Unit-test path generation, `index.json` read/write, and repair on a truncated file (truncate a real session mid-line, not hand-crafted damage).

**Acceptance:**

- [x] Storage layout matches plan §7: `~/.local/share/kirsch/sessions/<workspace-hash>/<timestamp>-<ulid>.jsonl`
  - (layout.go:38–44 `DefaultStateRoot` → `~/.local/share/kirsch`; layout.go:73–75 `SessionDir` → `<root>/sessions/<workspace-hash>`; layout.go:81–83 session path; layout.go:226–232 `IndexPath`/`UsagePath` → `<root>/index.json` + `<root>/usage.json`, matching §7's tree kirsch-plan.md:366–374; corrected in fix round 2a after Step 2 C1, re-verified 2b APPROVED 0/0/0)
- [x] `index.json` maps canonical workspace path → `{last_session_id, updated_at}`
  - (index.go:10–22 `IndexEntry`/`Index` schema; index.go:28–61 `ReadIndex`/`WriteIndex`/`Set`; layout_test.go:286 `TestLayoutIndexJSONSchema`)
- [x] `usage.json` exists (schema defined, even if empty)
  - (usage.go:12–21 `Usage` struct — `PromptTokens`/`CompletionTokens`/`TotalTokens`/`UpdatedAt`; usage_test.go:11 `TestReadUsageMissing`, :26 `TestWriteAndReadUsage`, :62 `TestUsageAddUpdatesTotals`, :87 `TestAddUsageRootUpdatesFile`)
- [x] Repair-on-load: truncated final line → session loads as `recovered`, never blocks startup
  - (repair.go:20 `repairSession` truncates to last good line; store.go:104 `New` calls `openAndRepair`; store.go:132–152 `openAndRepair` opens `O_RDWR|O_APPEND` and continues on repair error; store.go:122 `Store.Recovered()`; repair_test.go:18 `TestRepairTruncatesTornFinalLine` truncates a REAL Store-written session mid-line; loading never blocks startup)
- [x] Unit tests: path generation, index.json read/write, repair on truncated file
  - (29 test functions across layout_test.go / index_test.go / usage_test.go / repair_test.go / store_test.go + `FuzzSessionPathContained` in fuzz_test.go + `BenchmarkRecord`, all passing; gates green — `npm run check`, `npm run test:race`, scoped `govulncheck ./internal/session/...` clean, `FUZZTIME=10s npm run fuzz`)

**Must-know-before:** m4-d1 must be complete.

**Maps from old Task(s):** Task 1 (session store), with m4-d1.

---

## m4-d3 — Event Recording Wiring

**Scope:** Record the real M3 event stream into the store: `user.message`, `assistant.message`, `assistant.thinking`, `tool.completed`, `usage`, `approval.scope_granted`, `turn.started` (plan §7 schema).

**Dependencies:** m4-d1, m4-d2 (store + layout).

**File ownership:** `internal/session/`, `internal/app/` (wiring).

**Parallel execution:** None — depends on m4-d1, m4-d2.

**Instruction items:**

1. `app.Submit()` sets `ag.Recorder` to the session store (currently nil at `app.go:628–632`).
2. Record every event type from `agent.go` to JSONL with the correct plan §7 schema.
3. Record `usage` events (`agent.go:512–514` `EventUsage`) — currently emitted but not persisted.
4. Record `approval.scope_granted` events when policy grants session scope.
5. Record `turn.started` and `turn.completed` events.
6. Integration test: run a turn, verify the JSONL contains the expected events.

**Acceptance:**

- [x] `app.Submit()` sets `ag.Recorder` to the session store
  - (app.go:737 `Recorder: store` in the `agent.Agent` built by `Submit`; the store is created via `ensureStore` at app.go:706/548, which builds the m4-d2 layout; mission-20261014-09, review APPROVED after one fix round)
- [x] Every event type from `agent.go` is recorded to JSONL with correct schema (plan §7)
  - (event.go:21–35 event-type constants; store.go `Record` maps `RecordKind`→§7 type; the §7 mapping pasted and verified in Step 1 — turn.started, user.message, assistant.thinking, assistant.message, tool.completed, usage, turn.completed; assistant.delta excluded per the reconstruction rule §7:360; the `role`-field deviation removed in fix round 2a, re-verified 2b; mission-20261014-09)
- [x] `usage` events recorded (`agent.go:512–514` `EventUsage`)
  - (app.go:1069 `store.RecordUsage(*ev.Usage)` on `agent.EventUsage`; usage.json fed via `session.AddUsage` at app.go:1072; mission-20261014-09)
- [x] `approval.scope_granted` events recorded when policy grants session scope
  - (app.go:1318–1323 `store.RecordScopeGranted(action, strings.Join(app.argv, " "))` after `pol.Grant` succeeds in `Resolve`; mission-20261014-09)
- [x] `turn.started` and `turn.completed` events recorded
  - (app.go:959 `store.RecordTurnStarted(turn)` and app.go:988 `store.RecordTurnCompleted(status)` in `RunTurn`; mission-20261014-09)
- [x] Integration test: run a turn, verify JSONL contains expected events
  - (internal/app/session_recording_test.go:69 `TestSubmitRecordsFullEventSequence` — fake provider + real `Store` on a temp dir; full sequence asserted; mission-20261014-09)
- [x] The store's `OnError` hook is wired to the telemetry debug log — mid-turn write/`fsync` failures surface, never silently lost
  - (app.go:572–574 `session.WithOnError(func(err error){ a.log.Debug("session store error", "error", err) })` in `ensureStore`; verified by internal/app/session_recording_test.go:252 `TestSessionStoreOnErrorLogsToTelemetry`; mission-20261014-09)
  - (added 2026-10-14, mission-20261014-01, from the m4-d1 Step 1 improvement note — the default no-op hook is fine for tests but must be wired to telemetry in m4-d3 so turn-continuation mid-turn failures are never silently lost)

**Must-know-before:** m4-d1, m4-d2 must be complete. The Recorder-seam decision (m4-d1) is applied.

**Maps from old Task(s):** No old Task number of its own — this is the event-recording wiring the old Task 1's "store" implicitly required and that m4-d4–m4-d12 depend on.

---

## m4-d4 — Resume Reconstruction

**Scope:** Rebuild the transcript **and** the conversation from JSONL per the locked reconstruction rule (from `user.message` / `assistant.message` / `assistant.thinking` / `tool.completed` — **never** `assistant.delta`); apply `compaction.applied` rather than replaying pre-compaction history.

**Dependencies:** m4-d3 (event recording).

**File ownership:** `internal/session/` (resume logic), `internal/app/` (wiring).

**Parallel execution:** None — depends on m4-d3.

**Instruction items:**

1. `session.Resume(path)` returns the reconstructed `*agent.Conversation` and transcript.
2. Reconstruction uses only `user.message`, `assistant.message`, `assistant.thinking`, `tool.completed` (plan §7 reconstruction rule).
3. `assistant.delta` events are ignored for reconstruction (UI replay only).
4. `compaction.applied` events replace the compacted range with the summary.
5. Correlate tool call ids: every `tool.completed` matches its `assistant.message` tool_call (plan §7 reconstruction rule).
6. Unit-test reconstruction from JSONL with all event types; verify the conversation matches the original.

**Acceptance:**

- [x] `session.Resume(path)` returns reconstructed `*agent.Conversation` and transcript
  - (resume.go:49 `func Resume(path string) (*agent.Conversation, Transcript, error)`; Transcript/TranscriptMessage plain structs (resume.go:27–35, agent.Role/agent.Block, no TUI import))
- [x] Reconstruction uses only `user.message`, `assistant.message`, `assistant.thinking`, `tool.completed` (plan §7 reconstruction rule)
  - (resume.go:82–164 the switch handles exactly user.message/assistant.message/assistant.thinking/tool.completed + compaction.applied; assistant.delta explicitly skipped at resume.go:157–159)
- [x] `assistant.delta` events are ignored for reconstruction (used only for UI replay)
  - (resume.go:157–159 explicit skip; TestResumeIgnoresAssistantDelta proves it)
- [x] `compaction.applied` events replace the compacted range with the summary
  - (resume.go:147–155 + applyCompaction :256–269 (conversation only); TestResumeAppliesCompactionToConversationNotTranscript proves BOTH sides (conversation compacted, transcript intact))
- [x] Tool call ids correlate: every `tool.completed` matches its `assistant.message` tool_call
  - (resume.go:119–137 pending map; ErrMalformedSession on mismatch/duplicate/orphan; five malformed tests each fail on genuinely bad input)
- [x] Unit tests: reconstruct from JSONL with all event types, verify conversation matches original
  - (TestResumeRoundTripAllEventTypes + TestResumeRoundTripMerge (deep equality; RecordMerge semantics matching agent.go:468–479); 16 unit tests + FuzzResume (21,931 execs/10s) all pass; gates green)

**Must-know-before:** m4-d3 must record all event types correctly.

**Maps from old Task(s):** Task 2 (resume), with m4-d5.

---

## m4-d5 — Resume State + CLI

**Scope:** Pending state restored; session-scoped grants restored from `approval.scope_granted` events; `kirsch resume`, `--new`, auto-resume via `index.json`.

**Dependencies:** m4-d4 (resume reconstruction).

**File ownership:** `internal/session/`, `internal/app/`, `cmd/kirsch/` (CLI flags).

**Parallel execution:** None — depends on m4-d4.

**Instruction items:**

1. `kirsch resume` loads the last session for the workspace via `index.json`.
2. `kirsch --new` starts a fresh session even if a last session exists.
3. Auto-resume: if no flags, resume the last session (plan §7).
4. Restore session-scoped grants from `approval.scope_granted` events.
5. Restore pending state (the in-progress turn).
6. Integration test: quit mid-task, resume, complete the turn — identical transcript, grants, valid message array.

**Acceptance:**

- [x] `kirsch resume` loads last session for workspace (via `index.json`)
  - (cmd/kirsch/main.go:149–153 `chooseSessionInit` routes the `resume` subcommand to `App.ResumeLastSession` → `Layout.LastSession` at app.go:613 + `session.Resume`)
- [x] `kirsch --new` starts fresh session even if last session exists
  - (main.go:154–157 — the `--new` flag routes to `App.StartNewSession`, independent of any last session)
- [x] Auto-resume: if no flags, resume last session (plan §7)
  - (main.go:158–175 — `cfg.Session.AutoResume` defaults true (config.go:74, :114); post-fix-2a ANY resume error falls back to a fresh session with a distinguishing notice, so startup never blocks; only `StartNewSession`'s own failure surfaces)
- [x] Session-scoped grants restored from `approval.scope_granted` events
  - (app.go:661–675 — `session.LoadGrants` (grants.go) returns `(action, pattern)` pairs replayed through the existing `policy.Grant` API; `apply_patch` grants are correctly skipped — policy refuses patch grants with `ErrPatchGrantForbidden`, policy.go:165–166)
- [x] Pending state (in-progress turn) restored
  - (app.go:647–648 — `a.conv = conv`, the reconstructed `*agent.Conversation`; the next `Submit` continues from it)
- [x] Integration test: quit mid-task, resume, complete the turn — identical transcript, grants, valid message array
  - (resume_integration_test.go:24–172 `TestResumeCompletesPartialTurn` — quit mid-task, resume, complete the turn: identical transcript (3 items), grants restored (1 grant), valid message array (3 messages, tool call/result correlation verified by the fake provider))

**Must-know-before:** m4-d4 must reconstruct correctly.

**Maps from old Task(s):** Task 2 (resume), with m4-d4.

---

## m4-d6 — Single-Instance Guard

**Scope:** Advisory `flock` on `<workspace-hash>/.lock`; degrade-not-block (fresh start, `⚠ another Kirsch is running here`, grants disabled).

**Dependencies:** m4-d2 (storage layout — needs the `<workspace-hash>` directory).

**File ownership:** `internal/session/` (lock logic), `internal/app/` (wiring), `internal/tui/` (status bar warning).

**Parallel execution:** Can run alongside m4-d3, m4-d4, m4-d5 (independent of event recording / resume logic).

**Instruction items:**

1. On start, take an advisory `flock` on `<workspace-hash>/.lock` (plan §7 single-instance guard).
2. A second instance in the same workspace does **not** auto-resume (starts fresh), shows `⚠ another Kirsch is running here` in the status bar, and disables session-scoped grants (plan §7).
3. Unit-test lock acquisition and second-instance degradation.
4. Integration test: two instances, verify the second degrades.

**Acceptance:**

- [x] On start, Kirsch takes advisory `flock` on `<workspace-hash>/.lock`
  - (internal/session/lock.go:29 `Acquire` acquires flock on `<workspace-hash>/.lock` (Unix only); cmd/kirsch/main.go:202 calls `(*App).AcquireLock` at startup before the resume decision; mission-20261016-02, review APPROVED, 0 CRITICAL, 2 warnings carried)
- [x] Second instance in same workspace: does not auto-resume (starts fresh), shows `⚠ another Kirsch is running here` in status bar, disables session-scoped grants
  - (app.go:522 sends `another Kirsch is running here` when lock held; app.go:528–529 logs lock error and guard off; main.go:174 skips auto-resume via `ConcurrentInstance()` check; app.go:1469 `canApproveForSession` withholds approval; internal/tui/statusbar.go:39–40 status bar shows `⚠ another Kirsch is running here`; mission-20261016-02)
- [x] Unit tests: lock acquisition, second-instance degradation
  - (internal/session/lock_test.go: TestAcquireSucceeds, TestSecondAcquireContends; internal/app/lock_integration_test.go: TestAcquireLockNonHeldError, TestLockSecondInstanceDegrades; cmd/kirsch/main_test.go: TestChooseSessionInitExplicitResumeConcurrentInstance, TestChooseSessionInitAutoResumeConcurrentInstance; internal/tui/status_test.go: TestStatusWarningRendersWithGlyph; mission-20261016-02)
- [x] Integration test: two instances, verify second degrades
  - (internal/app/lock_integration_test.go: TestLockSecondInstanceDegrades — two App instances on same workspace, second locked out; mission-20261016-02)

**Must-know-before:** m4-d2 must create the `<workspace-hash>` directory.

**Maps from old Task(s):** Task 3 (single-instance guard).

---

## m4-d7 — Compaction

**Scope:** The model call per plan §6.4 (original task verbatim + last four turns + structured summary; cancellable, never half-applied; failure surfaces `context_overflow`, no silent oversized request; thinking dropped not summarised).

**Dependencies:** m4-d3 (event recording — needs the `compaction.applied` event type).

**File ownership:** `internal/session/` (compaction logic), `internal/app/` (wiring), `internal/agent/` (may need compaction-aware `Turn`).

**Parallel execution:** Can run alongside m4-d4, m4-d5, m4-d6 (independent of resume / lock logic).

**Instruction items:**

1. Compaction is a model call using the session's provider and model (plan §6.4).
2. Keep: the original task verbatim, the last 4 turns verbatim, a structured summary of the middle (plan §6.4).
3. Cancellable: a cancelled compaction leaves the session uncompacted and the pending turn unstarted (plan §6.4).
4. On failure: surface `context_overflow` and tell the user to `/new` — no silent oversized request (plan §6.4).
5. Drop thinking blocks (check `Thinking.DropOnSummary` — `agent.go:106`, set at `agent.go:758`), never summarise them (plan §6.5).
6. Record a `compaction.applied` event with the replaced range + summary (plan §6.4, §7 schema).
7. Unit-test compaction with thinking blocks, cancellation, and failure handling.

**Acceptance:**

- [x] Compaction is a model call using the session's provider and model
  - (app.go:1842 `runCompaction` passes session provider/model; compact.go:114 `Compact` makes the request; mission-20261017-02)
- [x] Keeps: original task verbatim, last 4 turns verbatim, structured summary of middle
  - (compact.go:114–200 keeps message 0 and last 4 turns, structures summary with Objective/Files inspected/Key findings/Files changed/Commands run/Open questions; mission-20261017-02)
- [x] Cancellable: cancelled compaction leaves session uncompacted, pending turn unstarted
  - (compact.go:114 Compact returns on context cancel; app.go:1842 runCompaction never applies on cancel; TestSubmitCancelDuringCompactionLeavesSessionUntouched and TestSubmitCancelAfterSummaryBeforeSwapNeverHalfApplies; mission-20261017-02)
- [x] On failure: surfaces `context_overflow`, tells user to `/new` (no silent oversized request)
  - (app.go:1842 RunTurn maps any compaction error other than cancellation to a `context_overflow` TurnErrorMsg telling the user to /new; TestSubmitCompactionProviderErrorSurfacesContextOverflow and TestCompactionModelError; mission-20261017-02)
- [x] Thinking blocks dropped (check `Thinking.DropOnSummary` — `agent.go:106`)
  - (renderMessagesForSummary at compact.go:208 drops every thinking block without checking the flag, and the flag is set on every assembled block anyway; TestThinkingBlocksNotInSummaryRequest, TestThinkingBlocksPreservedInResult; mission-20261017-02)
- [x] `compaction.applied` event recorded with replaced range + summary
  - (store.go:239 RecordCompaction; app.go:1842 calls store.RecordCompaction after successful Compact; TestSubmitAboveThresholdCompactsThenRunsTheTurn, TestCompactionRoundTrip; mission-20261017-02)
- [x] Unit tests: compaction with thinking blocks, cancellation, failure handling
  - (internal/session/compact_test.go: 19 tests covering structure, thinking, cancellation, errors, round-trip; internal/app/compaction_test.go: 9 tests; internal/tui/turn_messages_test.go: 4 tests; mission-20261017-02)

**Must-know-before:** m4-d3 must record events; the `Thinking.DropOnSummary` marker exists (`agent.go:106`, `agent.go:758`).

**Maps from old Task(s):** Task 4 (compaction).

---

## m4-d8 — Slash Commands for Real

**Scope:** `/status` `/diff` `/files` `/approvals` `/new` `/compact` `/quit` `/exit` on real state; `/status` fields per ui-spec line 525 incl. tokens (in/out/cache) — this resolves the m3-d2 cache_read carried item.

**Dependencies:** m4-d3 (event recording — needs real session state), m4-d5 (resume — `/new` needs session management), m4-d7 (compaction — `/compact` needs compaction logic).

**File ownership:** `internal/tui/` (slash command handlers), `internal/app/` (wiring).

**Parallel execution:** None — depends on m4-d3, m4-d5, m4-d7.

**Instruction items:**

1. `/status` shows: model, branch, dirty count, session id, tokens (in/out/cache), compaction %, project-context file, active grants (ui-spec line 525).
2. `/diff` shows the working tree diff.
3. `/files` lists files touched this session.
4. `/approvals` lists active grants, offers to clear.
5. `/new` starts a new session (confirm if a turn is in flight).
6. `/compact` triggers manual compaction.
7. `/quit` and `/exit` quit (alias).
8. Unknown `/command` → dim inline hint, not an error card, never sent to the model (ui-spec line 534).
9. Display cache tokens in `/status` (resolves the m3-d2 cache_read carried item — the status bar currently sums cache tokens into the total at `internal/tui/update.go:339`; `/status` omits them per ui-spec line 525).

**Acceptance:**

- [x] `/status` shows: model, branch, dirty count, session id, tokens (in/out/cache), compaction %, project-context file, active grants (ui-spec line 525) — mission-20261017-03 Step 6, TestStatusInfoSessionID + TestStatusInfoCompactionPercent + TestStatusInfoActiveGrants + TestStatusInfoDirtyCount + TestStatusInfoShowsAccumulatedTokens
- [x] `/diff` shows working tree diff — mission-20261017-03 Step 6, TestDiffCommandDoesNotBlockUpdate + TestDiffCommandOpensModalWithCallbackLines + TestDiffCommandCleanTreeShowsNotice + TestDiffCommandErrorShowsNotice + TestDiffCommandWithoutCallbackShowsNotice + TestDiffResultTruncationMarkerDerivesFromCap + TestDiffResultDoesNotReplaceAnOpenModal + TestDiffModalLinesAreSanitised + TestDiffResultIgnoresStaleSequence + TestDiffResultDroppedAfterNewSession + TestDiffResultBlockedByConfirmPrompt
- [x] `/files` lists files touched this session — mission-20261017-03 Step 6, TestFilesCommandListsPaths + TestFilesCommandEmptyNotice + TestFilesCommandWithoutCallbackShowsNotice + TestFilesCommandSanitisesPaths
- [x] `/approvals` lists active grants, offers to clear — mission-20261017-03 Step 6, TestApprovalsNoticeSanitisesGrants
- [x] `/new` starts new session (confirm if turn in flight) — mission-20261017-03 Step 6, TestSlashNewRunsCallbackInCmdAndResetsOnDone + TestSlashNewDoesNotBlockUpdate + TestSlashNewFailureKeepsTranscript + TestNewSessionIsolatesTheOldTurnFromTheNewSession + TestNewSessionFailureLeavesTheSessionIntact + TestSubmitRefusesWhileNewSessionWaitsForATurn
- [x] `/compact` triggers manual compaction — mission-20261017-03 Step 6, TestCompactCommandHoldsBusyUntilRealOutcome + TestCompactCommandRefusesSubmitAndKeepsBusyOnEsc + TestManualCompactMsgOutcomesShowDistinctNotices + TestCompactCommandWithoutCallback + TestManualCompactAppliesWithoutStartingATurn + TestManualCompactWithTooFewTurnsReportsNothing + TestManualCompactProviderErrorIsAFailureNotAnOverflow + TestManualCompactCancelDuringModelCallReportsCancelled + TestManualCompactCancelAfterSummaryBeforeSwapNeverHalfApplies + TestManualCompactRefusesWhileATurnIsLive + TestSubmitRefusesWhileManualCompactIsRunning
- [x] `/quit` and `/exit` quit (alias) — mission-20261017-03 Step 6, unchanged from baseline
- [x] Unknown `/command` → dim inline hint, not error card, never sent to model (ui-spec line 534) — mission-20261017-03 Step 6, baseline behavior
- [x] Cache tokens displayed in `/status` (resolves m3-d2 carried item) — mission-20261017-03 Step 6, TestStatusInfoShowsAccumulatedTokens + TestGetDiffCountsExactlyAndExcludesUntracked + TestStatusInfoDirtyCount

**Must-know-before:** m4-d3, m4-d5, m4-d7 must be complete.

**Maps from old Task(s):** Task 5 (slash commands). **Carried item resolved:** m3-d2 live turn-two `cache_read` check → `/status` cache tokens (ui-spec line 525).

---

## m4-d9 — Token and Cost Tracking

**Scope:** `internal/telemetry` (M1 built for this); status bar; `usage.json`; unknown-model degradation with warning + conservative budget.

**Dependencies:** m4-d3 (event recording — needs `usage` events).

**File ownership:** `internal/telemetry/`, `internal/tui/` (status bar), `internal/session/` (`usage.json`).

**Parallel execution:** Can run alongside m4-d4, m4-d5, m4-d6 (independent of resume / lock logic).

**Instruction items:**

1. Record `usage` events to `usage.json` (cumulative token/cost totals) — resolves the m3-d3 "Turn drops Usage and StopReason" carried item (`EventUsage` at `agent.go:512–514` is emitted but not persisted).
2. Status bar shows real token counts (input, output, cache read, cache write).
3. Unknown models degrade with a warning + conservative budget (not a failure) — suggest the smallest context window in the §5 model table as the fallback budget.
4. Cost calculated from model pricing (plan §5 model table).
5. Unit-test usage accumulation and unknown-model degradation.

**Acceptance:**

- [x] `usage` events recorded to `usage.json` (cumulative token/cost totals) — mission-20261017-05: TestUsageAddUpdatesTotals, TestAddUsageRootUpdatesFile, TestUsageAccumulationWithUnknownCost, TestUsageBackwardCompatibility, TestSubmitSendsExactSessionTotalsPerTurn
- [x] Status bar shows real token counts (input, output, cache read, cache write) — mission-20261017-05: TestSessionUsageWideRender, TestSessionUsageFlatNoCost, TestSessionUsageSmallCostFormat, TestStatusLadderDropsTokensBeforeGrants
- [x] Unknown models degrade with warning + conservative budget (not failure) — mission-20261017-05: TestUnknownModelFallback, TestConservativeBudgetEqualsTableMinimum, TestConservativeBudgetFollowsTheTable, TestUnknownModelNoticeOnceAcrossTwoTurns
- [x] Cost calculated from model pricing (plan §5 model table) — mission-20261017-05: TestCostPerTokenModel, TestCostFlatModel, TestCostUnknownModel, TestCostAllPerTokenModels, TestSessionCostPricesEachCallAtItsOwnModel
- [x] Unit tests: usage accumulation, unknown-model degradation — mission-20261017-05: 38 passing tests across internal/provider/models_test.go, internal/session/usage_test.go, internal/session/lock_test.go, internal/session/compact_test.go, internal/app/session_usage_test.go, internal/app/compaction_test.go, internal/agent/agent_test.go, internal/tui/turn_messages_test.go, cmd/kirsch/usage_wiring_test.go

**Must-know-before:** m4-d3 must record `usage` events.

**Maps from old Task(s):** Task 6 (token/cost tracking). **Carried item resolved:** m3-d3 "Turn drops Usage and StopReason" → `usage` events recorded to JSONL and `usage.json`.

---

## m4-d10 — Final-Task Report

**Scope:** Enforced in the system prompt: summary, files changed, commands run with pass/fail, limitations.

**Dependencies:** m4-d7 (compaction — the report must survive compaction).

**File ownership:** `internal/agent/prompt/` (system prompt), `internal/app/` (wiring).

**Parallel execution:** Can run alongside m4-d8, m4-d9 (independent of slash commands / token tracking).

**Instruction items:**

1. The system prompt enforces the final-task report (plan §6.1 item 6).
2. The report includes: summary, files changed, commands run with pass/fail, limitations.
3. The report survives compaction (included in the summary).
4. Unit-test that the system prompt includes the report instruction.

**Acceptance:**

- [x] System prompt enforces final-task report (plan §6.1 item 6)
- [x] Report includes: summary, files changed, commands run with pass/fail, limitations
- [x] Report survives compaction (included in summary) — (instruction-level: compaction prompt preserves the most recent report; system prompt unchanged across compaction — TestSystemPromptSurvivesCompaction; live check in testing walkthrough §10)
- [x] Unit tests: system prompt includes report instruction

**Must-know-before:** m4-d7 must handle compaction correctly.

**Maps from old Task(s):** Task 7 (final-task report).

---

## m4-d11 — Smoke Test

**Scope:** `scripts/smoke-test.sh` against `testdata/repo-go-module` with a real model ("Add input validation to the Divide function and cover it with a test").

**Dependencies:** m4-d1 through m4-d10 (all prior chunks).

**File ownership:** `scripts/` (smoke test script).

**Parallel execution:** None — depends on all prior chunks.

**Instruction items:**

1. Create `scripts/smoke-test.sh`, executable.
2. Run it against `testdata/repo-go-module` with a real model.
3. Prompt: "Add input validation to the Divide function and cover it with a test".
4. The run produces a shown patch, applies it after approval, runs `go test ./...`, and ends with a report.
5. The report names exactly the expected files.
6. Integration test: full end-to-end run passes.

**Acceptance:**

- [ ] `scripts/smoke-test.sh` exists and is executable
- [ ] Runs against `testdata/repo-go-module` with real model
- [ ] Prompt: "Add input validation to the Divide function and cover it with a test"
- [ ] Produces shown patch, applies after approval, runs `go test ./...`, ends with report
- [ ] Report names exactly the expected files
- [ ] Integration test: full end-to-end run passes

**Must-know-before:** All prior chunks must be complete.

**Maps from old Task(s):** Task 8 (smoke test).

---

## m4-d12 — Final Acceptance

**Scope:** The old Task 9 boxes (milestone-4.md lines 142–152); the definition of done (every box checked, CHANGELOG updated, `../PROGRESS.md` ☑ Complete, commit tagged).

**Dependencies:** m4-d11 (smoke test).

**File ownership:** `plan/PROGRESS.md`, `CHANGELOG.md`, `plan/spec/kirsch-plan.md` (amendment log if needed).

**Parallel execution:** None — final step.

**Instruction items:**

1. Tick every box below (the old Task 9 acceptance).
2. Update `CHANGELOG.md` under `[Unreleased]` → `Added` with the Milestone 4 entries.
3. Set `plan/PROGRESS.md` Milestone 4 to `☑ Complete`.
4. Tag the commit `m4` (the operator's tag; the tree lands uncommitted from this planning mission, so the tag is applied at close-out).
5. Run `go test -race ./...`, `golangci-lint fmt --diff`, `go vet ./...`, `golangci-lint run`, and CI — all clean.

**Acceptance (old Task 9 boxes):**

- [ ] Smoke test passes on a real model, end to end
- [ ] Quit mid-task, resume, and **complete the turn** — identical transcript, pending state and grants, and a valid message array
- [ ] A session truncated by `kill -9` loads as `recovered` and is resumable
- [ ] A second instance warns and degrades per §7
- [ ] Compaction is cancellable and never half-applies; failure surfaces `context_overflow` rather than an oversized request
- [ ] All slash commands operate on real state
- [ ] Token and cost tracking in the status bar and `usage.json`
- [ ] `go test -race ./...`, `golangci-lint fmt --diff`, `go vet`, `golangci-lint run`, CI all clean

**Definition of done:** every box checked, CHANGELOG updated, `../PROGRESS.md` set to `☑ Complete`, and a commit tagged so Milestone 5 starts from a known point.

**Must-know-before:** m4-d11 must pass.

**Maps from old Task(s):** Task 9 (final acceptance).

---

## Carried items disposition (M4-relevant)

From `plan/PROGRESS.md` *Carried from m3-d3* / *Carried from m3-d2*:

| Source | Item | Disposition in M4 |
|--------|------|-------------------|
| m3-d3 | "Recorder has no error return" | **Resolved by the m4-d1 Recorder-seam decision** — no-error interface kept; durability and error-surfacing in the store layer. |
| m3-d3 | "Turn drops Usage and StopReason" | **Resolved in m4-d9** — `usage` events recorded to JSONL and `usage.json` (emitted at `agent.go:512–514`, not persisted before M4). |
| m3-d2 | "Live turn-two cache_read check" | **Resolved in m4-d8** — `/status` shows cache tokens (ui-spec line 525); the status bar already sums cache tokens (`internal/tui/update.go:339`) but `/status` omitted them. |

Not M4-relevant carried items (resolved or deferred to M5) are unchanged and listed under their original headings in `plan/PROGRESS.md`.

---

## Explicitly Out of Scope (reminder)

No `kirsch doctor`, no release tooling, no user documentation in `doc/`, no
session branching or search. Those are Milestone 5.

---

## Executed record (replaces the old "Note for whoever executes this")

The 2026-09-12 outline carried a note: "Before starting, re-read this document against the code and rewrite what no longer fits." This rewrite does exactly that, against the landed Milestone 3 (operator-accepted 2026-10-13):

- The three questions the note said would come from M3's implementation are answered in *The three questions, answered*, each traced to `internal/agent/agent.go` and `internal/app/app.go` with file:line citations (research doc `m4-research.md`, @apone, mission-20261013-06 Step 1).
- The old Tasks 1–9 are re-cut into twelve chunks (m4-d1…m4-d12). The mapping is recorded per chunk and consolidated in *Task→chunk coverage* (DIR-014) so the coverage count is mechanically checkable: every old task lands in exactly one chunk, and nothing in *Explicitly Out of Scope* appears in any chunk.
- The Recorder-seam carried item (m3-d3) is resolved by a recorded Bishop decision in m4-d1, applied as the no-error `Recorder` interface with durability and error-surfacing in the store layer.
- The plan was amended, not worked around: amendment 86 records this expansion.
