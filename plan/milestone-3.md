# Milestone 3 — Instruction Set

> **Status: Drafted ahead of schedule — refine before executing.** Written
> 2026-09-12 alongside M2 and M4 at the owner's request, rather than after
> Milestone 2 lands as the builder's rules prescribe. Plan §11 amendment 50
> records why that is a trade and not a free win: M1's execution produced seven
> corrections to instructions written one milestone ahead. Treat the *shape* of
> this document as settled and the *detail* as provisional. Re-read it against
> the code when Milestone 2 is done, and amend before starting.

## Ground Rules (read first)

1. **Milestone 2 must be complete.** Every box in [`milestone-2.md`](milestone-2.md)
   ticked, CI green.
2. **This is the first milestone that spends money.** Every test in it runs
   against `provider.Fake` unless explicitly marked as a live smoke test. A test
   suite that needs an API key is a test suite that stops being run.
3. **Locked decisions** (do not re-litigate):
   - One provider for v0.1: Anthropic. The interface exists to make the fake
     possible and the code testable, not to support a second vendor (ADR 0003).
   - API keys come from the environment only, never from a config file. The
     refusal is already implemented in `internal/config`.
   - `internal/agent` imports no implementation package. The import-rule check
     from M1 Task 10 already guards this and will start failing the moment it is
     violated — which is the point.
   - Extended thinking ships **plumbed but off** (plan §6.5). Bolting it on
     later means retrofitting the message-echo rule, which breaks the second
     turn of every thinking-enabled request.
4. Every task has a checkable result.

---

## What makes this milestone hard

Two things, and neither is the HTTP call.

**Multiple tool calls in one assistant message.** The API rejects a message that
answers only some of the tool calls it was given. So every requested call must
produce a result, even when an earlier one was rejected or the turn was
cancelled — the short-circuited ones return `policy_denied` or `cancelled`
rather than being omitted. Plan §8 M3 states this; it is easy to read past and
expensive to discover.

**Thinking blocks must round-trip verbatim.** Once thinking is interleaved with
tool use, the next request in the same turn has to echo the thinking block back
with its signature intact. Dropping it produces an API-shape error on the second
turn only, which is exactly the kind of bug that survives a demo.

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

## Task 2 — `internal/provider/anthropic`

1. Streaming client over the Messages API, using the current model IDs from the
   plan §5 model table.
2. **Retry:** 3× exponential backoff on 5xx and network errors; honour
   `Retry-After` on 429; fail immediately and clearly on 401/403 with the
   onboarding message; on 400, log the full request to the debug log and fail —
   a 400 is a bug in Kirsch, and the request body is the evidence.
3. **Prompt caching:** a cache breakpoint after the system prompt and tool
   definitions. They are stable for the life of a session, so every turn after
   the first should read them from cache. The proof is a non-zero `cache_read`
   on turn two, and it belongs in the live smoke test rather than a unit test.
4. Accumulate partial tool-call JSON across deltas. A tool call arrives in
   fragments and is not valid JSON until the last one.
5. Map HTTP and stream errors to `provider_error`.

**Check:** unit tests against recorded fixtures — capture a handful of real SSE
streams once, commit them, and replay. Do not hand-write them: hand-written
fixtures encode what you *think* the API sends.

## Task 3 — `internal/agent`: the turn state machine

```
user task → build request → stream → [tool calls?] → policy → approval →
execute → results back to model → … → final answer
```

with four exits: completion, cancellation, error, and the max-turn guard
(default 25 tool rounds).

1. **Interfaces, declared here and supplied by `app`.** `agent.Model`,
   `agent.Tools`, `agent.Approver`, `agent.Recorder`. This package imports no
   implementation, and the M1 import check enforces it.
2. **Multiple tool calls execute sequentially, in the order returned.** A
   rejection or error short-circuits the remainder, and **every** requested call
   still returns a result — `policy_denied` or `cancelled` for the ones that did
   not run. See the note near the top.
3. Max-turn guard trips to `max_turns_exceeded` and surfaces as an error card.
4. A hallucinated tool name returns `tool_input_invalid` listing the real tools
   (already implemented in the M1 registry) and the model gets **two** retries
   before it becomes an error card.
5. Cancellation at any point returns within 1s and leaves the session
   resumable.

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
3. **Project context injection** per §6.2: first existing file from
   `[context].project_files`, capped at 32KB, fenced and labelled untrusted,
   read once per session and re-read only on `/new`.
4. The untrusted-input rule is not decoration. `testdata/repo-prompt-injection`
   has existed since M1 for this moment: a fake-provider test asserts the
   instruction in `src/helper.php` is **surfaced to the user, not obeyed**
   (plan §9.7, ADR 0007).

**Check:** the injection test passes; injected project context appears in the
request exactly once; `/status` reports which file was loaded.

## Task 5 — Extended thinking plumbing

Default `off`. Build it anyway, per §6.5.

1. `ThinkingDelta` / `ThinkingDone` events flow whether or not thinking is on.
2. When enabled: rendered as a collapsed dimmed card; **preserved verbatim and
   echoed back** with its signature in later requests within the same turn;
   persisted for M4's resume; excluded from compaction summaries — dropped, not
   summarised.

**Check:** a thinking-enabled scripted turn round-trips its block without an
API-shape error on the *second* request. One request proves nothing here.

## Task 6 — Onboarding, and golden state 14

ui-spec §13 state 14 — no API key, not a Git repo — has been deferred since
Milestone 0 because there was no provider to be missing a key for (amendment
24). This is the milestone where it becomes reachable, so **draw its screen in
`kirsch-ui-screens.md` before capturing its golden file**, exactly as every
other screen was.

1. No API key: name the environment variables read, in precedence order
   (`KIRSCH_ANTHROPIC_API_KEY` → `ANTHROPIC_API_KEY`), and state that keys are
   never read from config files.
2. Not a Git repo: the `--workspace` message, already implemented in M1.
3. Unknown model: a dim notice that cost display is unavailable and a
   conservative budget is in use — never a hard failure.
4. These are onboarding screens, not error cards. No red border. §7.5.

**Check:** screen 14 drawn and lint-clean; the golden test covers all fourteen
states; the §13 table's "not yet drawn" note is removed.

## Task 7 — Wiring, and removing the scaffolding

1. The model now drives tools. **Delete the debug slash commands** from M1 and
   M2 (`/read`, `/ls`, `/search`, `/gitstatus`, `/gitdiff`, `/patch`, `/run`)
   and the `debug (M1 only)` block from the help overlay. They were labelled as
   scaffolding precisely so this step is a deletion rather than a negotiation.
   Screen 06 gets three rows back and shrinks from 80×30.
2. Token counts in the status bar come from real `Usage` events.
3. Budget estimation per §6.3: chars/4, 4096-token output reserve, auto-compact
   above 75%. Compaction itself is M4; here, exceeding the budget surfaces
   `context_overflow` rather than silently sending an oversized request.

## Task 8 — Final acceptance

- [ ] `provider.Fake` expresses every event type, including a turn that blocks
      until cancelled
- [ ] No Anthropic identifier appears outside the adapter
- [ ] Retry behaviour correct for 5xx, 429 with `Retry-After`, 401/403 and 400
- [ ] Tool-call-then-answer works end to end on the fake
- [ ] Two tool calls, first rejected: the second **still returns a result**
- [ ] Hallucinated tool name self-corrects within two retries
- [ ] Max-turn guard trips at 25 and renders an error card
- [ ] Cancellation mid-tool returns within 1s and leaves the session resumable
- [ ] Prompt-injection fixture: the instruction is reported, not obeyed
- [ ] Project context appears in the request exactly once
- [ ] A thinking-enabled turn round-trips its block on the **second** request
- [ ] Screen 14 drawn, lint-clean, and all fourteen golden states captured
- [ ] Debug slash commands and their help block are **gone**
- [ ] Live: Kirsch answers a read-only question about a real repo with evidence,
      and `/status` shows a non-zero cache read on the second turn
- [ ] `go test -race ./...`, `gofmt`, `go vet`, `golangci-lint`, CI all clean
- [ ] **No session store or compaction code exists yet**

**Definition of done:** every box checked, CHANGELOG updated, progress table set
to `☑ Complete`, commit tagged.

---

## Explicitly Out of Scope (reminder)

No session store, no resume, no compaction, no cost tracking, no second
provider. The agent can hold a conversation and use tools; it cannot yet
remember one after the process exits.
