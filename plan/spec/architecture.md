# Kirsch — Architecture

> **Status: Settled for v0.1.** This document explains *why* the codebase is
> shaped the way it is, and states the rules that keep it that way. The
> [build spec](kirsch-plan.md) says what to build and in what order; this says
> how the pieces fit and which seams are load-bearing. Where they disagree, the
> plan wins on scope and this wins on structure.

## 1. What the architecture has to survive

Kirsch is small, but four forces pull on it hard enough to dictate its shape.
Every structural rule below traces back to one of them:

1. **Deterministic testability.** An agent loop tested only against a live
   model is untested. Approval gating, cancellation, hallucinated tool names,
   and the max-turn guard must be provable offline, for free, in milliseconds.
2. **Cancellation everywhere.** A stuck tool or a hung HTTP stream must never
   freeze the UI, and `Esc` must return control in under a second — including
   mid-tool, mid-stream, and mid-approval.
3. **No silent side effects.** Every write and every execution passes a policy
   decision the user can see. That is a property of the *call graph*, not of
   programmer discipline, so the graph has to make bypassing it awkward.
4. **Durability across a crash.** The session record must survive `kill -9`
   with at most one torn line, and a resumed session must reconstruct both what
   the user saw and what the model saw.

A fifth force is negative but just as real: **this is a solo-maintained v0.1.**
Indirection that does not pay for itself in one of the four above is a cost, not
an investment. See §9, Anti-goals.

## 2. Package layout

```
cmd/kirsch/main.go          entry point: flags, subcommands, builds the App
internal/app                coordinator: wiring, adapters, root context
internal/tui                Bubble Tea program: renders state, emits intents
internal/agent              turn state machine, prompts, budget, compaction
internal/provider           Provider interface + anthropic adapter + Fake
internal/tool               tool registry + implementations + envelope
internal/policy             allowlist, session grants, approval decisions
internal/workspace          root detection, containment, denylist, ignore, detect
internal/session            JSONL event store: append, load, resume, repair
internal/config             global + project config, env-only secrets
internal/telemetry          structured debug log; token/cost tracking (M4)
plan/                       build instructions (this file, spec, ADRs, UI spec)
doc/                        end-user documentation (Milestone 5)
testdata/                   fixture repositories (Milestone 1)
scripts/                    smoke test and release tooling (Milestones 4–5)
```

Package creation is milestone-ordered; see the build order table in
[plan §2](kirsch-plan.md). No package is created before the milestone that needs
it, and none is skipped.

## 3. Dependency rules

```
                    ┌──────────────┐
                    │   main.go    │
                    └──────┬───────┘
                           │ builds
                    ┌──────▼───────┐
      ┌─────────────┤ internal/app ├─────────────┐
      │             └──────┬───────┘             │
      │ typed msgs         │ adapters            │ owns
      │ program.Send       │                     │
┌─────▼──────┐      ┌──────▼───────┐      ┌──────▼──────┐
│internal/tui│      │internal/agent│      │  root ctx   │
└────────────┘      └──────┬───────┘      └─────────────┘
                           │ consumer-defined interfaces only
      ┌──────────┬─────────┼──────────┬──────────┐
      ▼          ▼         ▼          ▼          ▼
  provider     tool     policy    session   workspace
  (anthropic,  (registry (allowlist (JSONL,   (containment,
   Fake)        impls)    grants)   resume)    ignore)
```

**The rules, and why each exists:**

| # | Rule | Force it serves |
|---|---|---|
| 1 | `internal/agent` imports **no implementation package** — not `tui`, `provider`, `tool`, `policy`, or `session`. It declares the interfaces it needs and `app` supplies them. | Testability (§1.1). The agent loop under test is the real one, wired to fakes. |
| 2 | `internal/tui` imports **no** `provider`, `tool`, `workspace`, or `agent`. It receives typed messages and emits typed intents. | Testability + cancellation. The TUI is drivable by synthetic `tea.KeyMsg` with zero real infrastructure behind it. |
| 3 | **One goroutine owns all JSONL writes.** | Durability (§1.4). Concurrent appends produce interleaved partial lines, which is exactly the corruption the format is meant to survive. |
| 4 | **TUI state is mutated only inside `Update`.** Other goroutines reach it through `program.Send` and nothing else. | Bubble Tea's model is single-threaded by construction; a direct write from a tool goroutine is a data race that manifests as corrupted rendering, not a panic. |
| 5 | **Every filesystem path crosses `workspace.Resolve`.** No tool opens a path it built itself. | No silent side effects (§1.3). Containment enforced at one chokepoint cannot be forgotten at seven call sites. |

Rules 1 and 2 are **enforced by a test**, not by review — see
[milestone-1.md](../milestones/milestone-1.md) Task 10. The test exists before the packages it
guards, because the day `internal/provider` first appears is precisely the day
nobody is thinking about import rules.

### Why `internal/app` exists

It is tempting to delete it and let the TUI call tools directly. The reason not
to: `app` is the only place that knows about *both* sides, so it is the only
place adapters can live. It owns four jobs and nothing else:

1. **Wiring** — construct workspace, config, telemetry, registry, policy,
   provider, session store, agent, and the Bubble Tea program.
2. **Adapting** — turn `tool.Registry` into `agent.Tools`, `provider.Provider`
   into `agent.Model`, `session.Store` into `agent.Recorder`, and itself into
   `agent.Approver`.
3. **Routing** — TUI intents in, typed messages out via `program.Send`.
4. **Context ownership** — the hierarchy in §5.

It contains no business logic. If a rule about *what Kirsch may do* ends up in
`app`, it is in the wrong package.

## 4. The interfaces that matter

Four abstractions earn their keep. Signatures are indicative, not final.

**`agent.Model`** — the agent's view of a provider. Declared in `agent`,
implemented by an adapter in `app` over `internal/provider`:

```go
type Model interface {
    Stream(ctx context.Context, req Request, onEvent func(Event)) error
    Info() ModelInfo   // context window, max output, pricing (plan §5)
}
```

Events are provider-neutral: `TextDelta`, `ThinkingDelta`, `ThinkingDone`,
`ToolCallStart`, `ToolCallDelta`, `ToolCallEnd`, `MessageDone` (carrying usage),
`Error`. All Anthropic-specific accumulation stays inside the adapter — if an
Anthropic concept leaks into this event set, the abstraction has failed and
`provider.Fake` stops being a faithful stand-in (ADR 0003).

**`agent.Tools`** — the agent never sees `tool.Result`:

```go
type ToolCall   struct { ID, Name string; Input json.RawMessage }
type ToolResult struct { OK bool; Content string; Truncated bool; ErrorKind string }

type Tools interface {
    Describe() []ToolSpec                            // names + JSON schemas
    Invoke(ctx context.Context, c ToolCall) ToolResult
}
```

**`agent.Approver`** — the seam that makes approval testable, and the least
obvious piece of the design:

```go
type Approver interface {
    Request(ctx context.Context, req ApprovalRequest) (Decision, error)
}
```

The agent blocks on this call. `app` implements it by sending an approval
message to the TUI and waiting on a reply channel; `ctx` cancellation unblocks
it when the user hits `Esc`. In tests, a fake `Approver` answers instantly with
a scripted decision. Without this interface, approval flow can only be tested
through the UI, which is how approval bugs reach users.

**`tool.Tool`** — one envelope for every tool (plan §3):

```go
type Tool interface {
    Name() string
    Schema() json.RawMessage
    Invoke(ctx context.Context, raw json.RawMessage) Result
}
```

`Invoke` returns no `error` and never panics: the registry wrapper recovers
panics into a `Result`. A tool that can crash the process is a tool that can
lose a session.

## 5. Concurrency and cancellation

### Goroutine inventory

There are exactly five kinds of goroutine, and each owns its state exclusively:

| Goroutine | Owns | Never does |
|---|---|---|
| **TUI loop** (Bubble Tea) | All view state: transcript, selection, focus, modal, composer | Block. On anything. Ever. |
| **App router** | Wiring, pending-approval channels | Hold a lock across a `program.Send` |
| **Turn runner** (one per turn) | Agent conversation state for that turn | Touch TUI state |
| **Tool runner** (one per tool call) | Subprocess handles, file reads | Outlive its `toolCtx` |
| **Session writer** (one, for the process) | The JSONL file handle and buffer | Exist twice |

**The deadlock this design is avoiding:** the turn runner blocks on
`Approver.Request`, which waits on the TUI to answer. If the TUI could block
waiting on the turn runner, the two would wedge permanently and the only
recovery would be `kill`. Hence rule: *the TUI never blocks.* It sends on
buffered channels or not at all, and every wait lives on a non-TUI goroutine.

### Context hierarchy

```
rootCtx      process lifetime; cancelled by quit / SIGINT twice
 └─ sessionCtx   one session; cancelled by /new and on quit
     └─ turnCtx     one user message; cancelled by Esc or Ctrl+C
         └─ toolCtx    one tool call; timeout ∧ turn cancellation
```

Cancellation obligations, in order of how easily they are gotten wrong:

- `run_command` kills the **process group** — `SIGTERM`, 2s grace, `SIGKILL`.
  Closing pipes leaves orphans holding the terminal (plan §3.2).
- The provider stream is abandoned on `turnCtx`, and partial assistant text
  already rendered **stays** in the transcript, marked cancelled. Erasing what
  the user watched arrive is worse than leaving it.
- A pending approval resolves as rejected-by-cancellation, and the model is
  told, so a resumed session is not left with a dangling tool call.
- Target: `Esc` → composer usable in **under 1 second**, verified by test, not
  by feel.

Cancellation wired in later is cancellation that does not work, which is why
`turnCtx` exists from Milestone 1 — exercised with a deliberately slow tool
long before there is a model call to cancel.

## 6. Three representations of one conversation

This is the piece most likely to be mishandled, so it is stated plainly: there
are **three** representations of "the conversation", they are not the same, and
conflating any two of them is a bug.

| | Lives in | Contains | Purpose |
|---|---|---|---|
| **Transcript** | `tui` | Everything the user saw: full tool output, cards, expand state, selection | Human reading |
| **Conversation** | `agent` | What the model sees: truncated tool results, compacted history, thinking blocks | The next request |
| **Session log** | `session` | Every event, untruncated, append-only, versioned | Durability and resume |

Consequences that follow directly:

- A tool result truncated to ~4000 tokens for the model is **not** truncated in
  the transcript or the log. The user sees what happened; the model sees what
  fits.
- Compaction rewrites the *conversation* only. The transcript and log are
  untouched (ADR 0002), which is why `/compact` does not make scrollback vanish.
- Resume rebuilds transcript **and** conversation from the log — which is the
  entire reason the event schema carries `assistant.message` with tool-call ids
  and `tool.completed` with content, rather than only deltas (plan §7).
- `assistant.delta` events exist for replaying the *look* of streaming. They are
  never the source of truth for what was sent to the model.

## 7. A turn, end to end

The canonical flow. Every arrow crosses a package boundary listed in §3.

1. **TUI** — user presses `Enter`. The composer emits a `SubmitMsg` intent; the
   composer is disabled and the status bar goes busy.
2. **App** — creates `turnCtx`, records `turn.started` and `user.message`,
   starts a turn runner goroutine.
3. **Agent** — assembles the request: system prompt (plan §6.1), project
   context (§6.2), conversation, tool schemas from `Tools.Describe()`. Checks
   the budget; compacts first if over (§6.3).
4. **Provider** — streams. `TextDelta`s flow back through app to the TUI as
   coalesced repaints. Thinking deltas render as a dimmed card.
5. **Agent** — on `ToolCallEnd`, asks policy via app: allowed, needs approval,
   or denied.
6. **App** — if approval is needed, sends an approval card to the TUI and blocks
   the turn runner on `Approver.Request`. The TUI captures input exclusively
   until the user answers `y` / `a` / `n` / `d`.
7. **App** — on approval, runs the tool on a tool-runner goroutine under
   `toolCtx`. Incremental output streams into the card.
8. **Agent** — receives the `ToolResult`, appends it to the conversation,
   returns to step 4. Multiple tool calls in one message run **sequentially**,
   and every requested call gets a result even if an earlier one was rejected —
   the API rejects a message that answers only some of them (plan §8 M3).
9. **Agent** — on `MessageDone` with no tool calls, the turn ends. Usage is
   recorded; the final report format is enforced by the system prompt.
10. **App** — records `turn.completed`, flushes the session log with `fsync`,
    re-enables the composer.

At any point between 3 and 9, `Esc` cancels `turnCtx` and the flow unwinds
through §5's obligations.

## 8. Error model

Errors split into two classes that must never be confused:

**Tool errors** — `workspace_violation`, `patch_conflict`, `command_failed`,
`tool_input_invalid`, and the rest of plan §3.3. These are *expected outcomes*.
They go back to the model as tool results so it can adapt, and appear in the
transcript as error cards. A failing command is information, not a crash.

**Infrastructure errors** — provider unreachable after retries, session
directory unwritable, config malformed, context overflow after compaction.
These are *not* the model's business. They abort the turn and surface to the
user with something actionable. Feeding "your API key is invalid" to the model
as a tool result produces an agent that apologises and retries forever.

Two rules follow: never surface a tool error to the user as a failure of
Kirsch, and never hand an infrastructure error to the model dressed as a tool
result. A third, from ADR 0007: tool result *content* is untrusted data in both
directions — text inside it that issues instructions is reported, not obeyed.

## 9. Anti-goals

Deliberately absent, and to stay absent through v0.1:

- **No event bus or pub/sub.** Message flow is explicit and traceable through
  `app`. A bus makes every flow greppable only by accident.
- **No dependency-injection container.** `main.go` constructs everything in one
  readable function. When that function becomes hard to read, *that* is the
  signal to reconsider — not before.
- **No plugin or middleware layer.** Tools are a fixed set registered in code.
  MCP is explicitly out of scope (plan §1).
- **No abstraction over the filesystem or `git`.** `workspace` is the seam; a
  second one underneath it buys nothing and hides the chokepoint that §3 rule 5
  depends on.
- **No generic "service" interfaces** for packages with one implementation.
  Four interfaces are abstracted (§4) because each is a testing seam. Nothing
  else qualifies.

## 10. Testing seams

The architecture exists to make this table short and cheap:

| Seam | Fake | Buys |
|---|---|---|
| `agent.Model` | `provider.Fake` with scripted turns | Agent loop tested offline, deterministically, free (ADR 0003) |
| `agent.Approver` | Scripted decisions | Approval gating and rejection paths without driving the UI |
| `agent.Tools` | In-memory fake tools | Tool-call handling without touching disk |
| `tool.Tool` | Panicking / slow / huge-output fakes | Registry robustness, truncation, cancellation |
| Bubble Tea `Update` | Synthetic `tea.KeyMsg` + golden `View()` | UI behaviour and rendering without a terminal |
| `workspace` | `testdata/` fixtures | Containment proven against real symlinks |

## 11. Glossary

- **Workspace** — the Git root (or `--workspace` override) and the hard
  boundary of all file access.
- **Session** — one conversation with its own JSONL log; survives restarts.
- **Turn** — everything that happens from one user message to the next final
  assistant answer, including any number of tool rounds.
- **Tool round** — one model call plus the tool calls it requested. The
  max-turn guard (default 25) counts these, not turns.
- **Compaction** — replacing the middle of the *conversation* with a summary.
  Never touches the transcript or the log.
- **Session grant** — a session-scoped approval for a command prefix (ADR 0006).
- **Envelope** — the uniform `tool.Result` every tool returns (plan §3).
