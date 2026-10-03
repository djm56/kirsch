# Milestone 3 — Instruction Set

> **Status: Refined 2026-10-02** against the code at `3c7ce2c` and ADR 0008. Ready to execute: the live probe has run, and the `opencode` default model, `minimax-m3`, is recorded in plan §5 (amendment 80).

## Ground Rules (read first)

1. **Milestone 2 is complete.** CI is green on `3c7ce2c`. The operator accepted it on 2026-10-02 with the deferred items listed under "Carried from Milestone 2" in `plan/PROGRESS.md`. Milestone 3 takes three of its eleven bullets, in whole or in part, into m3-d6: the unwired settings and `ProgressSink` (bullet 4), the ignored command timeout (bullet 5), and the stale comments (bullet 11, except the grace-poll comment). The rest stay carried.
2. **This is the first milestone that spends money.** Every test in it runs
   against `provider.Fake` unless explicitly marked as a live smoke test. A test
   suite that needs an API key is a test suite that stops being run. Live tests run against the `opencode` endpoint, which is flat-rate. They are opt-in, and never part of `go test ./...` or CI. The subscription's 5-hour and weekly usage limits mean a careless live loop can lock testing out for hours or days, so live tests are few and deliberate.
3. **Locked decisions** (do not re-litigate):
   - One Messages-format adapter, two built-in endpoints (`opencode` as default, `anthropic`), and the trust boundary per ADR 0008. An endpoint is configuration, not code: each one sets `base_url`, `auth` (`bearer` or `x-api-key`), `api_key_env`, `model`, `prompt_caching`, and `thinking`. The whole `[provider]` section is honoured only from global config and flags (ADR 0008, ADR 0003).
   - **The project config layer is an allowlist** (operator ruling, 2026-10-02). A project file (`<workspace>/.kirsch/config.toml`) may set only `[context].project_files`, and each entry must resolve inside the workspace. Every other key in a project file — in `[provider]`, `[policy]`, `[context]`, `[session]`, `[telemetry]` or any later table — is ignored with a warning naming it, with one exception: a credential-shaped key such as `api_key` is refused outright by `checkSecrets`, which runs first (M2 behaviour, kept). A setting added later is global-only unless it is deliberately added to the allowlist. The reason: today `mergeFile` (`internal/config/config.go:186`) decodes a project file over the whole `Config`, so a repository can set anything. `run_command` already reads `env_passthrough` (`internal/tool/run_command.go:368`), and M3 starts reading approval settings, the project-context path and cap, and a debug log that records request bodies. This also closes an exposure that shipped in M2.
   - API keys come from the environment only, never from a config file. The lookup order is `KIRSCH_` + `api_key_env`, then unprefixed. The refusal is already implemented in `internal/config`.
   - `internal/agent` imports no implementation package. The import-rule check
     from M1 Task 10 already guards this and will start failing the moment it is
     violated — which is the point.
   - Thinking blocks are preserved and round-tripped at every `thinking` setting, including `off` (ADR 0008). This replaced the "Default `off`" wording in plan §6.5 and §8 (amendment 75); ADR 0008 governs.
4. Every task has a checkable result.

---

## When Kirsch first talks to a model

The first live call is the m3-d2 smoke test against `opencode`. The live probe, run before M3 starts, calls the endpoint directly and does not exercise Kirsch's adapter. The first conversation in the TUI is at m3-d6, when `submit()` hands the composer text to the app instead of to `fakeDriver`.

---

## Deliverables

| ID | Title | Tasks | Depends on | Parallel | Owns |
|---|---|---|---|---|---|
| m3-d1 | Provider interface and fake | 1 | — | Yes | `internal/provider/` (interface, StreamEvent, ToolCall, fake implementation) |
| m3-d2 | Messages adapter and endpoint config | 2 | m3-d1 | Yes | `internal/provider/anthropic/` (Messages wire format, streaming client, retry, caching, error mapping), `internal/config/` project-layer allowlist (only `[context].project_files` honoured from project files) and the endpoint map; `plan/testing/security.md` (config and URL cases) |
| m3-d3 | Agent state machine | 3 | m3-d1 | Yes | `internal/agent/` (turn loop, Request/Response, interfaces declared, no implementation imports) |
| m3-d4 | System prompt and thinking | 4–5 | m3-d2, m3-d3 | Yes | `internal/agent/prompt/system.md`, agent streaming event handling for thinking blocks |
| m3-d5 | Onboarding screens | 6 | m3-d1, m3-d2, m3-d3 | Yes | `kirsch-ui-screens.md` (state 14), `internal/tui/` (rendering), `internal/app/` (error state wiring, endpoint key variables per endpoint) |
| m3-d6 | Wiring and integration | 7–8 | all | No | `internal/app/` (adapters, main wiring), `internal/tui/` (real tokens, slash commands removed), M2 carry-over items (policy settings, command timeout, ProgressSink, stale comments); `internal/tool/run_command.go`, `internal/tool/apply_patch.go`, `internal/policy/`, `internal/config/` comments, `plan/testing/security.md` |

### Acceptance criteria per deliverable

**m3-d1 (Provider interface and fake):** Done when `provider.Fake` can script text-only turns, single and multiple tool calls, thinking blocks, mid-stream errors, and a turn that blocks until cancelled; no Anthropic-specific identifiers appear in exported types; and `go test ./internal/provider/...` is green.

**m3-d2 (Messages adapter and endpoint config):** Done when endpoint config decodes `base_url`, `auth`, `api_key_env`, `model`, `prompt_caching`, and `thinking` per endpoint; two built-in endpoints work (`opencode` and `anthropic`); every project-file key except `[context].project_files` is ignored with a warning, and a credential-shaped key is refused outright; `base_url` validation rejects non-loopback `http://`, userinfo-disguised loopback and IPv4-mapped loopback; a 3xx response is an error and is never followed; retry logic works correctly for 5xx, lockout and transient 429 (ADR 0008), 401/403, and 400; prompt caching shows non-zero `cache_read` on turn two, live only, and only on an endpoint whose probe result shows it reports cache usage; partial tool-call JSON accumulates correctly; and unit tests replay recorded SSE streams rather than hand-written fixtures.

**m3-d3 (Agent state machine):** Done when multiple tool calls execute sequentially; rejections short-circuit correctly but all requested calls return results; hallucinated tool names self-correct within two retries; max-turn guard trips at 25; cancellation returns within 1s and the next request holds `tool_result` for every cancelled `tool_use` id; and all tests run against fakes, not real providers.

**m3-d4 (System prompt and thinking):** Done when system.md is embedded and assembled in the correct order; project context injects correctly (first file from config, entries confined to the workspace, capped 32KB, read once per session); untrusted-input rule surfaces injected project context to the user rather than obeying it; a response containing thinking blocks round-trips them verbatim, signature included, on the second request at every `thinking` setting (`off`, `low`, `medium`, `high`); and the injection test passes.

**m3-d5 (Onboarding screens):** Done when screen 14 is drawn and lint-clean, the golden test covers all fourteen states including no API key, not a Git repo, and unknown model, error states render as onboarding (not error cards), and the "not yet drawn" note in ui-spec §13 is removed.

**m3-d6 (Wiring and integration):** Done when tool-call-then-answer works end to end; the provider is integrated and wired through the app; token counts in the status bar come from real Usage events; budget estimation surfaces `context_overflow` when oversized; debug slash commands are deleted; `go test -race ./...` is green; all fourteen golden screens match or the reference is correctly updated; the M2 carry-over items are wired (policy settings, command timeout, ProgressSink, stale comments removed); `/status` shows what Task 7 item 6 lists; approval ids are distinct; and `plan/testing/security.md` carries the Task 7 item 9 cases.

### Pinned file constraint

m3-d5 edits [`../spec/kirsch-ui-screens.md`](../spec/kirsch-ui-screens.md), which is parsed by the test suite (`internal/tui/golden_test.go` and `scripts/lint-screens.py`). The deliverable handover requires `npm run check` to pass, and the file must not be moved from its path at `plan/spec/`.

### How two developers work this milestone

The parallel path follows this sequence of dependencies and unblocking:

1. **Stage 1.** Dev A claims m3-d1 (provider interface and fake). This is a gate for all downstream work.
2. **Once m3-d1 is done,** Dev A moves to m3-d2 (anthropic adapter) while Dev B claims m3-d3 (agent state machine). These can run in parallel: m3-d2 is tested with recorded fixtures (from Task 2), m3-d3 is tested against the fake from m3-d1 (Task 3).
3. **Once m3-d2 and m3-d3 are done,** both developers are unblocked. Dev A continues through m3-d4 (system prompt and thinking), while Dev B claims m3-d5 (onboarding screens). Both depend on m3-d2 and m3-d3 and can run in parallel since they own different files; m3-d4 modifies `internal/agent/prompt/`, while m3-d5 modifies `internal/tui/` and `internal/app/`.
4. **Once all five deliverables are done,** either developer can claim m3-d6 (wiring and integration), which integrates everything and runs acceptance tests.

This sequence describes task ordering only and makes no estimate of how long any deliverable takes.

### Note on parallelization

The constraint that `internal/agent` imports no implementation package actually *enables* parallelization rather than limiting it: m3-d3 is tested exclusively against fakes, so m3-d2 can be built and tested independently in parallel. The interface from m3-d1 and the fake from m3-d1 are separable in practice — the fake is tested as part of d1's acceptance and can be used in d3's tests before d2's adapter is complete.

---

## What makes this milestone hard

Two things, and neither is the HTTP call.

**Multiple tool calls in one assistant message.** The API rejects a message that
answers only some of the tool calls it was given. So every requested call must
produce a result, even when an earlier one was rejected or the turn was
cancelled — the short-circuited ones return `policy_denied` or `cancelled`
rather than being omitted. Plan §8 M3 states this; it is easy to read past and
expensive to discover.

**Thinking blocks must round-trip verbatim at every setting.** The adapter must preserve and echo back every thinking block the response returns, with its signature intact, at every `thinking` setting including `off`. Dropping a block produces an API-shape error on the second turn only, which is exactly the kind of bug that survives a demo.

---

## Task 1 — `internal/provider`: the interface and the fake

Build the fake **first**. Everything downstream is tested against it, and
writing it first forces the interface to be honest about what a caller needs.

```go
type Provider interface {
    Stream(ctx context.Context, req Request, onEvent func(StreamEvent)) error
}

type StreamEvent struct {
    Type      EventType   // TextDelta, ThinkingDelta, ThinkingDone,
                          // ToolCallStart, ToolCallDelta, ToolCallEnd,
                          // MessageDone, Error
    Text      string
    ToolCall  *ToolCall   // carries the provider's own id
    Usage     *Usage      // on MessageDone: input, output, cache read, cache write
    Err       error
}
```

1. Events are **provider-neutral**. Every scrap of Anthropic-specific delta
   accumulation lives inside the adapter; if a field name from the Anthropic
   API appears in this package's exported surface, it is in the wrong place.
2. `ToolCall` carries the provider's `id`. Requests, approvals and results
   correlate on it, and it is what makes a multi-tool-call turn resumable in M4.
3. `Usage` carries cache read and write counts separately — `/status` shows
   them, and they are how anyone can tell whether prompt caching is actually
   working.
4. **`provider.Fake`** replays scripted turns:

   ```go
   fake := provider.NewFake(
       provider.Turn{Text: "Looking now", ToolCalls: []provider.ToolCall{...}},
       provider.Turn{Text: "Done. Summary follows…"},
   )
   ```
   It must be able to script: text-only turns, single and multiple tool calls,
   thinking blocks, mid-stream errors, and a turn that blocks until its context
   is cancelled. The last one is not optional — it is how cancellation gets
   tested without a network.

**Check:** `go test ./internal/provider/...` green. The fake can express every
event type. No Anthropic identifier appears outside the adapter file.

## Task 2 — Messages adapter and endpoint config

1. **Endpoint configuration per ADR 0008:** Each endpoint sets `base_url`, `auth` (`bearer` | `x-api-key`), `api_key_env`, `model`, `prompt_caching`, and `thinking`. Two built-in endpoints: `opencode` (default, flat-rate, model `minimax-m3`) with `x-api-key` auth and key variable `OPENCODE_API_KEY`, and `anthropic` with `x-api-key` auth and key variable `ANTHROPIC_API_KEY`. User-defined endpoints in global config only. A user-defined endpoint missing any of `base_url`, `auth`, `api_key_env` or `model` is a config error.

2. **Trust boundary:** A project file (`<workspace>/.kirsch/config.toml`) is decoded into a scratch struct first. Only `[context].project_files` is taken from it. Every other key is discarded, with a warning naming the ignored keys that is distinct from the unknown-key warning. `checkSecrets` still runs over the whole project file first, so a credential in it is refused, not merely ignored. HTTPS only, except a loopback host written literally as `localhost`, an address in `127.0.0.0/8`, or `::1`, taken from the host `net/url` parses after userinfo is removed. No hostname is resolved to decide this. IPv4-mapped forms such as `::ffff:127.0.0.1` are refused. Redirects (3xx) are errors.

3. **Fix the defaults:** `Defaults().Provider.Default` (`config.go:75`, today `"anthropic"`) becomes `"opencode"`. The `opencode` endpoint's model is `minimax-m3` (plan §5, amendment 80). The `anthropic` model changes from `claude-sonnet-5` (`config.go:77`) to `claude-sonnet-5-5`. `ProviderConfig` holds a map of named endpoints instead of the fixed `Anthropic` field, and the "only provider in v0.1" comment (`config.go:36`) is removed. `checkSecrets` refuses a value-bearing key in any endpoint table.

4. **HTTP adapter:** Streaming client over the Messages API. Always sends the `anthropic-version` header. Key variable lookup: `KIRSCH_` + `api_key_env` first, then unprefixed. On `opencode`, every request also sends `User-Agent: kirsch/<version>` and `x-opencode-session` with one random ID per Kirsch session, reused when the session is resumed (ADR 0008, client identity); a 400 with error type `MissingSessionID` is surfaced as a configuration error, not retried.

5. **Retry:** 3 retries (4 requests in all) with exponential back-off on 5xx and network errors; a 429 is a lockout when it carries the probe-recorded usage-limit signal (surfaced as such without retry); otherwise `Retry-After` back-off (exponential back-off when the header is absent), within the same 3-retry budget; 401/403 fails immediately with the onboarding message; 400 logs the full request to the debug log and fails.
   The debug log is enabled only by `--debug`, `KIRSCH_DEBUG=1` or the global config (`cmd/kirsch/main.go:71` today also accepts a project value; the allowlist removes it).

6. **Prompt caching:** per endpoint. The non-zero `cache_read` on turn two is checked live only on an endpoint whose probe result shows it reports cache usage.

7. Accumulate partial tool-call JSON across deltas. A tool call arrives in fragments and is not valid JSON until the last one.

8. Map HTTP and stream errors to `provider_error`.

**Check:** replay recorded SSE fixtures captured from each endpoint actually used. The live probe's `opencode` captures (`testdata/probe/opencode/2026-10-02T1421Z/`, `minimax-m3` and `minimax-m2.7`) are the first. `anthropic` fixtures are captured only if and when an Anthropic key is available. Until then, the `anthropic` endpoint is covered by request-shape tests (URL, auth header, version header).

## Task 3 — `internal/agent`: the turn state machine

```
user task → build request → stream → [tool calls?] → policy → approval →
execute → results back to model → … → final answer
```

with four exits: completion, cancellation, error, and the max-turn guard
(default 25 tool rounds).

1. **Interfaces, declared here and supplied by `app`.** `agent.Model`,
   `agent.Tools`, and `agent.Recorder`. Approval happens inside the tools themselves (see `tool.Approver` at `internal/tool/apply_patch.go:22`, called at `:125` and `internal/tool/run_command.go:128`, implemented by `approvalAdapter` (`internal/app/app.go:236`), which calls `RequestToolApproval` (`:722`)). A rejection comes back to the agent as a `policy_denied` result, not as a separate interface. `plan/spec/architecture.md` and plan §11 amendment 78 record the same shape. This package imports no implementation, and the M1 import check enforces it.
2. **Multiple tool calls execute sequentially, in the order returned.** A
   rejection or error short-circuits the remainder, and **every** requested call
   still returns a result — `policy_denied` or `cancelled` for the ones that did
   not run. See the note near the top.
3. Max-turn guard trips to `max_turns_exceeded` and surfaces as an error card.
4. A hallucinated tool name returns `tool_input_invalid` listing the real tools
   (already implemented in the M1 registry) and the model gets **two** retries
   before it becomes an error card.
5. Cancellation at any point returns within 1s. Afterwards, the next request's
   message array holds a `tool_result` for every `tool_use` id in the cancelled
   turn (`cancelled` for the ones that did not run), and a follow-up turn succeeds.

**Check:** fake-provider tests for: tool-call-then-answer; two tool calls where
the first is rejected and the second still returns a result; a hallucinated tool
name self-correcting; the max-turn guard; cancellation mid-tool.

## Task 4 — System prompt and project context

1. `internal/agent/prompt/system.md`, embedded with `go:embed`. A file, not a Go
   string literal, so changes to agent behaviour show up in a diff and get
   reviewed.
2. Assembled in the fixed plan §6.1 order: role and constraints; environment
   block; tool-use guidance; **the untrusted-input rule**; project context;
   final-report format.
3. **Project context injection** per §6.2: the first existing file from
   `[context].project_files`. Each entry is resolved against the workspace root
   and refused if it is absolute, contains `..`, or resolves (after symlinks)
   outside the workspace root. The read goes through the workspace engine, so its
   denylist applies. The cap is 32KB; global config may lower it, and a global value above 32768 is clamped to 32768. The
   target must be a regular file, read through a bounded reader at the cap,
   not sized by `Stat`. A refused entry is skipped and a warning names it. A
   wrong-typed `project_files` in a project file is a warning and the key is
   ignored, not a startup failure. The built-in default `.kirsch/context.md`
   (`config.go:94`) is removed from `Defaults()`: `.kirsch` is on the workspace
   denylist, so that entry could never load. The default list becomes
   `AGENTS.md`, `CLAUDE.md`.
   The content is fenced, labelled untrusted, read once per session and re-read
   only on `/new`.
4. The untrusted-input rule is not decoration. `testdata/repo-prompt-injection`
   has existed since M1 for this moment: a fake-provider test asserts the
   instruction in `src/helper.php` is **surfaced to the user, not obeyed**
   (plan §9.7, ADR 0007).

**Check:** the injection test passes; injected project context appears in the
request exactly once; the chosen project-context file is exposed to the app; its `/status` display is checked in Task 7.

## Task 5 — Thinking round-trip

Per ADR 0008, the adapter preserves every thinking block and passes it back unchanged, signature included, at every `thinking` setting.

1. `ThinkingDelta` / `ThinkingDone` events flow at every setting, including `off`.
2. `off` maps to `"between_tools"` on the `anthropic` endpoint's default model (`claude-sonnet-5-5`), with effort held at `high` or below. How effort is set, and how `low | medium | high` map onto the request, is decided here against the documentation.
3. Every thinking block is **preserved verbatim and echoed back** with its signature in later requests within the same turn. Rendered as a collapsed dimmed card; kept in the in-memory transcript in a form M4 can persist, and marked so that M4's compaction drops it rather than summarising it.
4. On `minimax-m3` the live probe saw thinking blocks only with `enabled` or `between_tools`, and none with `disabled` or with no `thinking` field. Whether m3 accepts an echoed thinking block was not probed, because its S2 returned no thinking; `minimax-m2.7` did accept one. The live round-trip check therefore runs on m3 with `thinking` set to `enabled` or `between_tools`, never `off`.

**Check:** a scripted turn that returns a thinking block, run at each `thinking` setting including `off`, round-trips its block without an API-shape error on the *second* request (tested on the fake always and live wherever the probe showed the model returns thinking blocks). One request proves nothing here.

## Task 6 — Onboarding, and golden state 14

ui-spec §13 state 14 — no API key, not a Git repo — has been deferred since
Milestone 0 because there was no provider to be missing a key for (amendment
24). This is the milestone where it becomes reachable, so **draw its screen in
`kirsch-ui-screens.md` before capturing its golden file**, exactly as every
other screen was.

1. No API key: name the active endpoint's two key variables in order (for example `KIRSCH_OPENCODE_API_KEY` → `OPENCODE_API_KEY`), and state that keys are never read from config files.
2. Not a Git repo: the `--workspace` message, already implemented in M1.
3. Unknown model: a dim notice that cost display is unavailable and a conservative budget is in use — never a hard failure.
4. When the active endpoint is `opencode`, both the onboarding screen and `/status` state that prompts and file contents go to OpenCode's gateway and the model host, not Anthropic (ADR 0008).
5. These are onboarding screens, not error cards. No red border. §7.5.

**Check:** screen 14 drawn and lint-clean; the golden test covers all fourteen
states; the §13 table's "not yet drawn" note is removed.

## Task 7 — Wiring, and removing the scaffolding

1. The model now drives tools. **Delete the debug slash commands** from M1 and
   M2 (`/read`, `/ls`, `/search`, `/gitstatus`, `/gitdiff`, `/patch`, `/run`)
   and the `debug (M1–M2)` block from the help overlay. They were labelled as
   scaffolding precisely so this step is a deletion rather than a negotiation.
   Screen 06 gets six rows back — the blank spacer, the heading, and the four
   command rows — and shrinks from 80×34 to 80×28. (Body height is the taller of
   the two columns: `max(left 15, right 17)` = 17, a 21-row box, plus the seven
   rows the frame spends around it — a two-row header, two separator rules, the
   status bar, the composer and the key-hint row.)
2. Token counts in the status bar come from real `Usage` events, and the model name comes from the active endpoint; the hard-coded `claude-sonnet-5` in `internal/tui/model.go:173` and `internal/tui/statusbar.go:11` is removed.
3. Budget estimation per §6.3: chars/4, 4096-token output reserve, auto-compact
   above 75%. Compaction itself is M4; here, exceeding the budget surfaces
   `context_overflow` rather than silently sending an oversized request.
4. `submit()` hands the composer text to the app, which starts a real agent turn. `fakeDriver` remains only as a test fixture.
5. New TUI-owned message types, with no field of a type from `provider`, `agent`, `tool` or `workspace`, for: assistant text deltas, thinking deltas, usage, tool-call start and result, command output chunks, turn complete, turn error (including `max_turns_exceeded` and `context_overflow`), and cancelled. `internal/arch/imports_test.go` passed vacuously for `internal/agent` and `internal/provider` until now; it must still pass.
6. `/status` shows: the endpoint name and `base_url` host; the key's source variable name, never the value, a prefix, or its length; the proxy host when a proxy is in effect, never its userinfo; and the context file loaded.
7. **Approval IDs.** The two `ID: 1` placeholders (`internal/tool/apply_patch.go:126`, `internal/tool/run_command.go:129`) are replaced. Approval requests carry the tool call's provider id, or an app-allocated unique id mapped to it.
8. **M2 carry-over:** `policy.New` receives `allow_session_scoped_grants`, `require_approval_for_patches` and `require_approval_for_commands` from global config only (amendment 72, and the Ground Rule 3 ruling). `run_command` reads `default_command_timeout_seconds` for an omitted timeout. The schema text, the code's ceiling and plan §3 (default 60, maximum 300) are made to agree on one maximum, and the outcome is recorded as a new plan §11 amendment (the defect is amendment 70). `RunCommand.ProgressSink` is wired so that output streams to the card (amendment 73). The comments describing these as unwired are removed, including `config.go:44` ("consumed from M2"). The grace-poll comment in `run_command.go` stays carried.
9. **Security cases.** `plan/testing/security.md` gains one case per Task 8 trust-boundary box
(project allowlist, `[context]` confinement, `debug_log`, credential refusal, `base_url`, redirect) and the `run_command`-writes-config known limit from ADR 0008. m3-d2 writes the config and URL cases; m3-d6 writes the rest.

## Task 8 — Final acceptance

- [ ] `provider.Fake` expresses every event type, including a turn that blocks
      until cancelled
- [ ] `grep -ri anthropic internal/provider internal/agent --exclude-dir=anthropic` finds nothing
- [ ] Retry, against a test server, asserting the request count and error kind per case: 5xx and network error — 4 requests, then `provider_error`; lockout 429 — 1 request, surfaced as a lockout; transient 429 then 200 — 2 requests, the second after `Retry-After`; 401/403 — 1 request, onboarding message; 400 — 1 request, request body in the debug log; 3xx — 1 request, error naming the status and `Location` host
- [ ] m3-d2 live smoke test, on `opencode`: one streamed text-only request through Kirsch's adapter completes and reports usage; run by hand, never in `go test ./...` or CI
- [ ] Tool-call-then-answer works end to end on the fake
- [ ] Two tool calls, first rejected: the second **still returns a result**
- [ ] Hallucinated tool name self-corrects within two retries
- [ ] Max-turn guard trips at 25 and renders an error card
- [ ] Cancellation mid-tool returns within 1s; the next request holds a `tool_result` for every `tool_use` id of the cancelled turn, and a follow-up fake turn succeeds
- [ ] Prompt-injection fixture: the instruction is reported, not obeyed
- [ ] Project context appears in the request exactly once
- [ ] A scripted turn whose response contains a thinking block is replayed at each of `off`, `low`, `medium` and `high`; each time the second request carries the block byte-identical, signature included
- [ ] A thinking block in the in-memory transcript carries the marker that tells M4's compaction to drop it; a unit test asserts the marker
- [ ] Screen 14 drawn, lint-clean, and all fourteen golden states captured
- [ ] Debug slash commands and their help block are **gone**
- [ ] A project file is an allowlist: a table-driven test sets, from a project file, every key in `[provider]` (including `default` and a new endpoint), every key in `[policy]`, `[context].max_project_context_bytes`, every key in `[session]` and `[telemetry]`, and an unknown table; each leaves the effective config unchanged and produces a warning naming the key, distinct from the unknown-key warning
- [ ] A project file's `[context].project_files` is honoured: an entry `AGENTS.md` inside the workspace is injected, and the request carries its content exactly once
- [ ] Refused `project_files` entries — `/etc/hosts`, `../outside.md`, a symlink leading outside the workspace, and a denylisted path (`.env`) — inject nothing from that path, are skipped, and each is named in a warning
- [ ] A project file with `[context] project_files = "AGENTS.md"` (a string, not a list) starts normally, leaves the effective config unchanged, and produces a warning naming the key
- [ ] A `project_files` entry that is a directory or a FIFO is skipped with a warning naming it; the read does not block and does not rely on `Stat` size
- [ ] `Defaults().Context.ProjectFiles` is `["AGENTS.md", "CLAUDE.md"]`
- [ ] Global `max_project_context_bytes = 1024` truncates project context at 1024 bytes; a global value of 40000 with a 40000-byte `AGENTS.md` injects exactly 32768 bytes
- [ ] A project file with `[telemetry] debug_log = true` does not enable the debug log
- [ ] A credential-shaped key (`api_key = "x"`) in a project file is refused by `checkSecrets`, not merely ignored
- [ ] `base_url` tests: refused — non-loopback `http://`, `http://localhost@evil.example/`, `http://[::ffff:127.0.0.1]/`, and a hostname that merely resolves to loopback; accepted — `http://localhost`, `http://127.0.0.2`, `http://[::1]`
- [ ] A test server returning a 3xx makes the adapter return an error naming the status and `Location` host; no second request is sent, and `x-api-key` never reaches the redirect target
- [ ] `plan/testing/security.md` has a case for each trust-boundary box above, plus the `run_command`-writes-config known limit (ADR 0008)
- [ ] Key value never appears in output, log, or recorded fixtures; verified by grepping fixtures and debug log
- [ ] `/status` names the key's source variable (e.g. `KIRSCH_OPENCODE_API_KEY`), never the value, a prefix or its length; with `HTTPS_PROXY=http://alice:s3cret@proxy.example:3128` it shows `proxy.example` and contains neither `alice`, `s3cret` nor `@`; it also shows the endpoint name, the `base_url` host and the context file loaded
- [ ] `Defaults().Provider.Default == "opencode"`; the `opencode` endpoint's model is `minimax-m3`; the `anthropic` endpoint's model is `claude-sonnet-5-5`, not the legacy `claude-sonnet-5`
- [ ] A global endpoint missing any of `base_url`, `auth`, `api_key_env` or `model` is a config error naming the key; a global table naming a built-in endpoint and setting only `model` leaves that endpoint's other keys at their built-in values
- [ ] A value-bearing key inside any global endpoint table is refused by `checkSecrets`
- [ ] `grep -n 'consumed from M2' internal/config/config.go` and `grep -n 'claude-sonnet-5"' internal/tui/*.go` find nothing
- [ ] `policy.New` honours each of `allow_session_scoped_grants`, `require_approval_for_patches` and `require_approval_for_commands` from global config; one test per setting flipped from its default
- [ ] An omitted `timeout_seconds` uses `default_command_timeout_seconds`, and an over-ceiling value is refused or clamped; one test each; schema text, code and plan §3 state the same maximum
- [ ] `RunCommand.ProgressSink` is wired; command output streams to the card
- [ ] Live, on `opencode`: Kirsch's answer to a read-only question about this repo cites a file and line that the transcript's `read_file` or `search_code` results contain; the cache-read check applies only if the probe showed cache reporting
- [ ] `anthropic` endpoint: request-shape tests verify URL, auth header, and version header; live fixture coverage depends on an Anthropic key being available
- [ ] With endpoint `opencode`, a test finds the data-flow notice on both the onboarding screen and `/status`
- [ ] Two approval requests in one turn carry distinct ids; `grep -n 'ID: *1,' --exclude='*_test.go' internal/tool/*.go` finds nothing
- [ ] `internal/arch/imports_test.go` passes with `internal/agent` and `internal/provider` populated
- [ ] On `opencode`, every request carries `User-Agent: kirsch/<version>` and the same `x-opencode-session` for the whole session; a test asserts both, and a 400 `MissingSessionID` surfaces as a configuration error
- [ ] `go test -race ./...`, `golangci-lint fmt --diff`, `go vet`,
      `golangci-lint run`, CI all clean
- [ ] **No session store or compaction code exists yet**

**Definition of done:** every box checked, CHANGELOG updated, `../PROGRESS.md`
set to `☑ Complete`, commit tagged.

---

## Explicitly Out of Scope (reminder)

No session store, no resume, no compaction, no cost tracking, no second wire format (OpenAI-format endpoints, ADR 0008 option 3). The agent can hold a conversation and use tools; it cannot yet remember one after the process exits.
