# Progress — Kirsch

**Where we are:** Milestone 2 is complete, accepted by the operator on 2026-10-02 with deferred items (see *Carried from Milestone 2*). Milestone 3, the provider and agent loop, is complete: m3-d1 through m3-d6 are done (m3-d6 = wiring and integration, mission-20261007-02). Remaining before the milestone is operator-accepted: the operator's live run (Task 8 box 321, `npm run test:live`), a manual walkthrough, and the commit.

**What's next:** operator items for Milestone 3 close-out — run the live test (`npm run test:live`, Task 8 box 321), perform a manual walkthrough of the testing docs, and commit the accumulated m3-d6 delta (currently uncommitted on top of `04e30cc`). Carried items are listed under *Carried from m3-d6*, *Carried from m3-d5*, *Carried from m3-d4*, *Carried from m3-d3* and *Carried from m3-d2*.

This file is the single source of truth for live state — what is in progress and what you can start next. Milestone instruction documents carry static definition only (IDs, tasks, dependencies, ownership of files) and never carry status or owner information.

## Milestones

| # | Milestone | Status |
|---|---|---|
| 0 | Repo skeleton + static TUI prototype | ☑ Complete |
| 1 | Workspace engine + read-only tools | ☑ Complete |
| 2 | Patches, commands, approvals | ☑ Complete (operator-accepted 2026-10-02, deferred items below) |
| 3 | Provider + agent loop | ☑ Complete |
| 4 | Real task loop + sessions | ☐ Not started |
| 5 | Polish + release (v0.1.0) | ☐ Not started |

## Milestone 2 — deliverables

| ID | Title | Status | Owner | Waits on |
|---|---|---|---|---|
| m2-d1 | Patch infrastructure | done | — | — |
| m2-d2 | Policy | done | — | — |
| m2-d3 | Approval flow | done | — | m2-d2 |
| m2-d4 | Tool implementations | done | — | m2-d1, m2-d2 |
| m2-d5 | TUI integration | done | — | m2-d3, m2-d4 |
| m2-d6 | Testing & acceptance | done | — | all |

## Milestone 3 — deliverables

Definitions are in `plan/milestones/milestone-3.md`. The live probe ran on 2026-10-02, and the `opencode` default was `minimax-m3` (plan §11 amendment 80). On 2026-10-05 the endpoint refused `minimax-m3`; the live smoke test ran on `qwen3.7-plus` (amendment 82), and the default became `minimax-m2.7` (amendment 83).

| ID | Title | Status | Owner | Waits on |
|---|---|---|---|---|
| m3-d1 | Provider interface and fake | done | — | — |
| m3-d2 | Messages adapter and endpoint config | done | — | m3-d1 |
| m3-d3 | Agent state machine | done | — | m3-d1 |
| m3-d4 | System prompt and thinking | done | — | m3-d2, m3-d3 |
| m3-d5 | Onboarding screens | done | — | m3-d1, m3-d2, m3-d3 |
| m3-d6 | Wiring and integration | done | — | all |

## Carried from m3-d5

The onboarding screens landed on 2026-10-07 (mission-20261007-01). Open items that remain:

- **`/status` data-flow notice and `/status` display (Task 7 item 6):** when the active endpoint is `opencode`, `/status` states that prompts and file contents go to OpenCode's gateway and the model host, not Anthropic (ADR 0008 lines 217–219) — the onboarding-screen half landed at m3-d5; Task 8 box 323 ticks at m3-d6 when both halves exist. **Resolved at m3-d6:** both halves exist and box 323 is ticked (step 7 centralised the wording in `internal/tui/messages.go`).
- **main.go wiring:** `App.CheckOnboarding` is built but not yet called — m3-d6's main wiring calls it at startup and hands the TUI the onboarding state path. **Resolved at m3-d6:** `CheckOnboarding` is called at startup via `startupPostAttach`/`onboardingCheck` (step 3).
- **Centralise the ADR 0008 data-flow wording** in one place before the `/status` half lands, so the onboarding screen and `/status` cannot drift (also ledgered in FINDINGS.md this close). **Resolved at m3-d6:** the wording is centralised in `internal/tui/messages.go` (step 7).
- **Advisory:** absence-check tests use fixed sleeps (200ms in `TestCheckOnboardingKeyPresentSuppressesMessage`); if such tests accumulate, extract a shared `drainOrTimeout` helper.

## Carried from m3-d6

The wiring and integration deliverable landed on 2026-10-08 (mission-20261007-02). Open items that remain:

- **Operator live run (Task 8 box 321):** `npm run test:live` = `go test -tags live -run TestLive -count=1 -v ./internal/provider/anthropic/` with `OPENCODE_API_KEY` set; the default model is `minimax-m2.7`, `KIRSCH_LIVE_MODEL` overrides. It runs `TestLiveOpencodeSmoke` + `TestLiveThinkingRoundTrip` + `TestLiveModel`. Box 321's live opencode answer citing a file/line is operator-observed in the TUI, not automated. If the live run returns a 400 on a thinking-effort mapping, revisit the mapping against the live Anthropic docs (carried from m3-d4).
- **`scripts/__pycache__/` cleanup:** the directory is untracked and not in `.gitignore`; remove it at commit time (flagged at Step 10, deferred per the operator efficiency directive).

## Carried from m3-d4

The system prompt, project-context loader and thinking round-trip landed on 2026-10-06 (mission-20261006-03). Open items that remain:

- **Live round-trip test (operator, by hand):** `TestLiveThinkingRoundTrip` (`internal/provider/anthropic/live_test.go`, build tag `live`) is built, not run. Run `npm run test:live` (= `go test -tags live -run TestLive -count=1 -v ./internal/provider/anthropic/`) with `OPENCODE_API_KEY` set; the default model is `minimax-m2.7`, `KIRSCH_LIVE_MODEL` overrides. It checks thinking-block echo on the second request at every setting including `off`, plus the replayed assistant-message-made-only-of-thinking check carried from m3-d3.
- **(routed to m3-d6) Prompt wiring:** `internal/app` wires `Agent.System` (from `prompt.Assemble` with `Env` + `LoadProjectContext` output) and the `Engine` adapter (`workspace.Workspace` → `prompt.Engine`); `/status` shows the chosen project-context file and size (Task 7); TUI thinking-card rendering (collapsed dimmed card) and the max-turn error card are already routed to m3-d6.
- **Thinking effort mapping decision:** `off` → `thinking.type "between_tools"` + `output_config.effort "high"`; `low`/`medium`/`high` → `"adaptive"` + `"low"`/`"medium"`/`"high"` — decided against the live Anthropic docs (cited in `internal/provider/anthropic/encode.go`); if the operator's live run returns 400 on a mapping, revisit against the docs.

## Carried from m3-d3

The core turn loop landed on 2026-10-05 (mission-20261005-02, plan amendment 84) and the four guards landed on 2026-10-06 (mission-20261006-02); m3-d3 is done. Open items that remain:

- **Recorder has no error return:** a session store cannot report a failed write. Decide this before the store is built.
- **Turn drops Usage and StopReason,** so truncation at `max_tokens` is invisible to its caller.
- **Other edges:** an empty user text becomes an empty text block; a nil Model or Tools panics; `emit` runs synchronously inside the provider callback; a session resumed on an assistant tool_use would receive a plain user message.
- **Calibration warnings:** there is no isolating mutant for an EventError after MessageDone, and nothing mechanically proves a predictions file was not edited after its run.
- **(routed to m3-d6) `max_turns_exceeded` error-card rendering:** the guard is verified at agent level (TestMaxTurnGuard, internal/agent/agent_test.go); the card itself lands at m3-d6.
- **(routed to m3-d6) Unjoined cancellation goroutine:** the goroutine running a cancelled tool is not joined — the agent cannot force a misbehaving tool to stop, only stop waiting for it; revisit if a tool can outlive its context (@apone, step 2 review).

## Carried from m3-d2

Recorded done on 2026-10-05 at the operator's decision. Details are in plan amendment 82.

- **Live turn-two `cache_read` check:** not built; carried to m3-d6, where the first real conversation has a second turn.
- **401/403 message:** every 401 or 403 is reported as "rejected the API key", even when the server's reason differs.
- **400 debug log:** a 400 writes the full request body to the debug log, as the plan specifies. Clipping it is the operator's decision.
- **Config review items:** the project-warning dedupe keys on the key only, uses an unstable sort, and has no test forcing a duplicate; `[[context]]` gets the list-of-strings message; `UnmarshalTOML`'s doc comment is thin; the endpoint field-key sort is uncalibrated; a `nolint:misspell` sits on a verbatim test line.
- **Wire-format review items:** usage outside int64 fails instead of clamping; a start event without `content_block`; a duplicated object check in `encode.go`; test and doc gaps; thin calibration for the cancel guard, the messageDone bypass and parseIndex's leading-zero check.
- **Live-test review items:** two unrecorded calibration mutants in `livemodel_test.go`; `live_test.go`'s comment does not say the override is checked only once a key is present.
- **Default-model review items:** no calibration mutant targets `minimax-m2.7`'s context or output figures; `config_test.go:57` overrides the model to the new default, so its model check cannot fail; two comments over-reach (`models_test.go:47` names the opencode default, and `models.go:36` says the figures are sourced from documentation).

## Carried from Milestone 2

Deferred by the operator on 2026-10-02, when Milestone 2 was accepted. Details are in `plan/milestones/milestone-2.md` (the status paragraph after Task 9, and *Open defects*) and in plan amendments 61–73.

- **Two unticked Task 9 boxes:** the 500ms timeout kill, and cancelled-process death. To close the 500ms box, either fix its test and tighten the bound (1–2s today) or relax the requirement.
- **Tests that cannot detect a regression:** `TestRunCommandTimeoutProcessGroupDead` and `TestRunCommandCancellationProcessGroupDead` cannot fail as written, and `TestRunCommandStdinIsDevNull` passes without the code it names.
- **Suspected trailing-newline defect** in `internal/patch` (Task 3.4), in two shapes. Not yet reproduced.
- **Unwired settings and hooks:** `allow_session_scoped_grants`, `require_approval_for_patches` and `require_approval_for_commands` are never read (amendment 72), and `run_command` output is not streamed (amendment 73). Taken into m3-d6 (2026-10-02).
- **Configured command timeout ignored:** `run_command` never reads `default_command_timeout_seconds`. An omitted timeout falls back to 3600s with no ceiling, and the tool's schema says "Maximum 3600" where plan §3 says 300 (amendment 70). Taken into m3-d6 (2026-10-02).
- **Stdin wording:** the Task 9 box says a stdin reader "fails immediately". In fact it sees EOF and exits, normally with success (amendment 71). Rewording the box is the operator's decision.
- **Untested secret patterns:** `*_SECRET`, `AWS_*`, and `*_KEY` outside the walkthrough.
- **Unverified blank approval card:** an approval request with an unspecified operation sends empty `Subject` and `Detail`. Whether the card renders blank is unchecked.
- **No `-update` flag** in `golden_test.go`, so screen grids are regenerated by hand.
- **Walkthrough checks not retested:** in §10, `/run` in `/help` (fixed in code) and the two `/patch` checks; in §1, the expand check.
- **Stale code comments:** `internal/policy/policy.go:57` and `internal/config/config.go:44` imply the policy settings are wired, which they are not. The `config.go:44` comment is taken into m3-d6 (2026-10-02); re-check `policy.go:57` once the settings are wired. `internal/tool/run_command.go:~316–321` and `~336–337` say the grace poll always waits 2s, which is probably wrong; confirm and fix.

## What has landed

| Completed | Deliverable | What landed |
|---|---|---|
| 2026-10-07 | m3-d5 — onboarding screens | Screen 12 (state 14) drawn and golden-tested; TUI onboarding rendering (dim, never error cards); app CheckOnboarding wiring with endpoint key variables; ui-spec §13 gap closed. (mission-20261007-01) |
| 2026-10-08 | m3-d6 — wiring and integration | `mission-20261007-02`. TUI turn-message pipeline (7 plain-field messages, thinking collapsed dimmed card); debug slash commands + help block deleted, screen 06 shrinks 80×34→80×28; submit() hands composer text to a real agent turn; status bar real Usage tokens + context budget indicator, hard-coded model removed; `/status` display (endpoint, base_url host, key source variable, proxy host, context file, data-flow notice) + prompt wiring at session start; approval IDs distinct via tool invocation ID; policy.New reads three settings from global config; run_command timeout default + 300 ceiling (amendment 85) + ProgressSink streaming; six config boxes covered; provider surface (Fake completeness, retry matrix, redirect refusal, request-shape, session header); security.md cases 7–11 + run_command-writes-config known limit; testing docs sections 5–8 runnable. Reviews: steps 2b, 4, 6b, 8, 10, 12, 16, 18b APPROVED; steps 6, 14, 18 NOT APPROVED then resolved. **Uncommitted** on top of `04e30cc`. |
| 2026-10-06 | m3-d4 system prompt and thinking | `mission-20261006-03`. `internal/agent/prompt`: embedded six-section system prompt (plan §6.1 order, untrusted-input rule verbatim, project context fenced and labelled untrusted), `Env` + `Assemble`, and the project-context loader — consumer-declared `Engine` interface, first-existing FIFO, engine-refused entries skipped with named warnings, regular-file check before any read, bounded reader (never `Stat`-sized), cap `min(supplied, 32768)` with line-boundary truncation marker, chosen file/size/warnings exposed. `agent.Request.System`/`Agent.System` carried on every request. Injection test: instruction surfaced in the fenced untrusted section, exactly once in the system prompt, zero times in messages, rule sections byte-identical. Adapter thinking request field: `off` → `between_tools` + high effort, `low`/`medium`/`high` → `adaptive`, decided against the live Anthropic docs. Thinking blocks round-trip byte-identical (signature included) on the second request at every setting; `Thinking.DropOnSummary` marks blocks for M4 compaction drop. Opt-in live round-trip test built for the operator. Reviews: steps 2, 4b, 6b APPROVED; step 4 NOT APPROVED (DIR-016 process violation, resolved at 4a/4b); step 6 APPROVED with conditions (gosec G304) resolved by vasquez escalation 6a (one-line `#nosec` annotation). **Uncommitted** on top of `8e37449`. |
| 2026-10-05 | m3-d3 turn loop (core) | `mission-20261005-02`. `internal/agent`: provider-neutral types and the `Model`, `Tools` and `Recorder` interfaces, with no implementation imports. `Agent.Turn` runs tool calls one after another in the order returned and completes a message only on `MessageDone`. Every error is wrapped (`ErrStreamProtocol`, `ErrStreamFailed`), everything stored or handed out is copied, and thinking blocks stay separate. A failed turn keeps what happened, and the next text merges into a trailing user message; `Recorder` records appends and merges so the conversation can be rebuilt exactly (amendment 84). Tested only against fakes. The guards come next. |
| 2026-10-06 | m3-d3 guards | `mission-20261006-02`. `internal/agent`: the four m3-d3 guards — sequential tool calls short-circuit on the first failure with every requested call still returning a result (`policy_denied` propagates from a denial, otherwise `cancelled`); hallucinated tool names return `tool_input_invalid` listing the real tools, two retries before `ErrToolNameHallucination`, known tools with invalid input pass through and do not consume the budget; max-turn guard trips at 25 tool rounds (`ErrMaxTurnsExceeded`); cancellation honoured before and during each call, returns within 1s, every `tool_use` id gets a `cancelled` result, follow-up turn succeeds. Plus `validateCallIDs` (empty/duplicate `tool_use` ids rejected as `ErrStreamProtocol`) and the unknown-tool/invalid-input split. 12 mirrored mutants killed in scratch; review APPROVED (0 CRITICAL, 0 WARNINGS); re-review APPROVED. |
| 2026-10-05 | opencode default model | `mission-20261005-01`. The `opencode` default is `minimax-m2.7` (amendment 83): `config.Defaults()`, a known flat-rate `minimax-m2.7` row in the model table, and a 256-token limit on the live smoke test. The `minimax-m3` row stays. |
| 2026-10-05 | m3-d2 Messages adapter and endpoint config | `mission-20261003-03`. `internal/provider/anthropic`: Messages request encoding, SSE decoding, and a streaming HTTP client with retry, lockout, redirect refusal and error mapping, tested against the recorded probe streams. `internal/config`: the endpoint map (`opencode` default, `anthropic`), `base_url` validation and the project-file allowlist. An opt-in live smoke test (`npm run test:live`), which the operator passed on `qwen3.7-plus` through `KIRSCH_LIVE_MODEL` because the endpoint refused `minimax-m3` (amendment 82). The live turn-two `cache_read` check is carried to m3-d6. |
| 2026-10-02 | M2 close-out, from the operator's review of milestone-2.md | `mission-20261002-01`. `/run` is visible in `/help` (it shares a row with `/patch`), and a clipped help footer names the scroll keys. `npm run lint` and `npm run fmt:check` are clean: 13 findings fixed, none suppressed. A rename with edits keeps the source file's permission bits. A real `git diff` of a CRLF file now applies, and a patch that would change line endings is rejected before the approval prompt. `plan/milestones/milestone-2.md` now records evidence for every Task 9 box (15 of 18 ticked), the departures from the instruction set (plan amendments 61–73) and the open defects. **Uncommitted** on top of `7c0133d`. |
| 2026-10-01 | M2 UX pass, from the operator's request | `mission-20261001-02`. Tool cards open as a 10-line preview: Enter collapses, `d` opens the full output. In the prompt, ↑/↓ recall history, and Shift+↑ or Tab move to the cards. An approved command shows one card with `· approved` on its head. A stray key releases a pending approval card instead of answering it. Terminals 24 or more rows tall get a key-hint row. Clipped modals show a `↓ N more` / `↑ N above` marker. `read_file` heads no longer repeat the path, and trailing empty output rows are trimmed. A refused session grant now confirms as approved, because the command runs. `npm run security` is clean. Walkthrough §12 has been added for retest. **Uncommitted:** all changes sit in the working tree on top of `7c0133d`. |
| 2026-10-01 | M2 walkthrough defects, from the operator's manual run | Three missions answered the operator's run of `plan/testing/manual/milestone-2.md`. `mission-20260928-01` fixed a deadlock that froze Kirsch when a command was approved for the session with `a`, with regression tests driving a real `tea.Program` in `internal/app/deadlock_test.go`. `mission-20260930-01` added the `c to clear (confirm)` instruction to the grants modal footer. `mission-20261001-01` replaced the hardcoded `2.4s` on approval cards with the real time from request to decision, and made command result cards name their command: `describeInput` now reads `argv`, sanitised. A test confirms `/run` is reachable by scrolling `/help`. Each mission also corrected the walkthrough against the code. **Uncommitted:** all three missions' changes, including the untracked `internal/app/deadlock_test.go`, are in the working tree only. **Open for the operator:** the retest of the items left blank for it, and the open questions in the walkthrough's Findings Summary. |
| 2026-09-26 | M2 manual testing walkthrough | `plan/testing/manual/milestone-2.md` — a blank master walkthrough for Milestone 2 with 73 unticked checkboxes across eleven sections, indexed in `plan/testing/README.md`. Written from source-reading of the code and review found **twenty-one graded findings across three fix rounds**—fifteen of them false claims about what appears on screen—before approval. Evidence artefact at `.claude/memory/workspace/walkthrough-evidence/`, which verifies the document's `/patch` and `/run` claims against the real `policy` package including post-grant decisions; this artefact lives in mission scratch and will be archived. **One open warning:** §9's opening paragraph describes shells and quote-parsing behavior above a checklist that tests neither—confusing rather than false, deferred to the operator. |
| 2026-09-25 | m2-d5 TUI integration | `internal/tui` and `internal/app`: the approval surface now shows what it is asking about. Cards render the real request — a patch's file list, a command's argv, a rename named the same way on the card and in the modal title — and every text field is sanitised where the message is received, because the one screen that asks for consent was the only one in the transcript rendering untrusted content unchanged. The `[a]` key appears only where policy permits a session grant, and a refused grant corrects the card through a confirmed-outcome message correlated by approval id. The diff modal renders the parsed patch with per-file counts, and a change carrying no hunks says what it is rather than showing an empty body. `/approvals` lists the grants the policy actually holds and clears them through it, replacing a counter that production wrote and only tests read. `/patch` and `/run` are wired, labelled, and registered — the two tools that change the world were reachable from nowhere until this deliverable. Golden screen 06 is reconciled at 33 rows, the minimum the layout arithmetic allows. **Four items are recorded open, none of them regressions:** the help overlay clamps its body at common terminal heights with nothing on screen to say content is below the fold; a tool card's name and target reach the renderer unsanitised, the same defect this deliverable closed one card along; the box builder does not bound a row against its own width; and several tests carry names that outrun what their assertions can discriminate. See findings dated 2026-09-25. |
| 2026-09-25 | m2-d4 Tool implementations | `internal/tool`: `apply_patch` and `run_command`, the first tools that change the world and so the first that must ask first. `apply_patch` parses, resolves every path including rename sources, stages into temp files and only then requests approval, committing the bytes it validated rather than re-applying — so an invalid patch never produces a prompt. `run_command` takes an argv slice and never a shell, builds the child's environment from scratch and strips secrets last, reads stdin from the null device, confines the working directory, caps output at 200KB head-and-tail, and bounds cancellation with a process-group kill plus a `WaitDelay` backstop for descendants that leave the group. Supporting changes: a mutex and an operation-typed grant path in `internal/policy`, and a `Stage`/`Commit`/`Discard` seam in `internal/patch`. **Adversarial coverage is partial** — path escapes are solid, environment filtering has a known case-sensitivity gap, and the renamed-shell evasion is documented rather than tested. See findings dated 2026-09-25. |
| 2026-09-23 | m2-d3 Approval flow | `internal/app`: `Approver.Request` blocks on a per-approval buffered channel from the tool goroutine and selects on the decision, the turn context and the root context, deleting its map entry on every path; `Resolve` never blocks, taking the lock only for the lookup and sending under `select`/`default`. Session approvals call `policy.Grant`. The TUI resolves through `wireCallbacks` in `cmd/kirsch`, with a `Cancelled` outcome distinct from rejection so an abandoned turn is not recorded as a refusal. Approve, reject, approve-for-session and escape-to-cancel are proven through real `Model.Update` dispatch; cancellation returns inside a second with no parked goroutines. |
| 2026-09-23 | m2-d2 Policy | `internal/policy`: the command allowlist and session-grant engine. Argv-slice matching so a semicolon-bearing element cannot be split; shells and runners refused at any install location by basename; `ForPatch` never auto-allows; grants refuse wildcards, empty prefixes and shells, and are disabled wholesale by config. Shipped allowlist entries are exact-match, so `go build -toolexec=` and `git diff --output=` require approval. |
| 2026-09-22 | m2-d1 Patch infrastructure | `internal/patch`: unified-diff parser, renderer, and atomic applier (standard library only, 44 tests). Fixtures at `testdata/repo-patch/` and diff corpus at `testdata/patches/`. |

---

**Outstanding verification:** see [testing/README.md](testing/README.md#outstanding-verification).
