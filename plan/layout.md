# Kirsch — Basic Layout (Draft)

> **Status: Planning phase draft.** This describes the intended structure of
> the codebase. Nothing here exists yet; Milestone 0 will bootstrap it.

## Package Layout

```
cmd/kirsch/main.go          entry point; wires everything together
internal/app                coordinator: wires agent <-> TUI, owns root context
internal/tui                Bubble Tea program; renders events, sends decisions
internal/agent              state machine, budget, compaction, prompts
internal/provider           Provider interface + anthropic adapter + fake for tests
internal/tool               tool registry + implementations
internal/policy             approval decisions, command allowlist, path denylist
internal/workspace          root detection, path containment, ignore, project detect
internal/session            JSONL event store, resume, repair
internal/config             global + project config, secrets from env only
internal/telemetry          structured debug log, token/cost tracking
testdata/                   fixture repos for tests (added in Milestone 1)
scripts/                    smoke-test and release tooling (Milestones 4–5)
plan/                       build instructions: spec, ADRs, UI spec, milestones
doc/                        end-user documentation (written in Milestone 5)
```

## Dependency Rules (hard, non-negotiable)

These rules are the backbone of the architecture. They keep the agent core
testable with a fake provider and fake tools, and keep the TUI dumb.

1. **`internal/agent` imports no implementations.** It must not import
   `internal/tui`, `internal/provider`, or `internal/tool` directly. It
   depends on interfaces only.
2. **`internal/tui` must not import `internal/provider`.** All communication
   between them flows through `internal/app` via typed messages.
3. **One goroutine owns all JSONL writes** (the session store's writer).
4. **TUI updates from other goroutines go through `program.Send(msg)` only** —
   never direct model access from agent/tool code.

```
            ┌────────────┐
            │  main.go   │
            └─────┬──────┘
                  │
            ┌─────▼──────┐   typed messages
            │ internal/  │◄──────────────┐
            │    app     │               │
            └─┬───────┬──┘               │
              │       │                  │
   ┌──────────▼─┐  ┌──▼────────┐         │
   │internal/tui│  │internal/  │         │
   │(Bubble Tea)│  │  agent    │         │
   └────────────┘  └──┬────────┘         │
                     │ interfaces only  │
        ┌────────┬────┴─────┬────────────┴──┐
        ▼        ▼          ▼               ▼
   provider    tool      policy         session
   (anthropic/ (registry, allowlist,    (JSONL,
    fake)       impls)    denylist)      resume)
```

## TUI Layout (v0.1 sketch)

```
┌─ Kirsch ─ my-project ─ main ──────────────────┐
│ [conversation viewport: user msgs, assistant  │
│  streaming text, tool cards, approval cards]  │
├───────────────────────────────────────────────┤
│ status: model, busy/spinner, tokens, errors   │
├───────────────────────────────────────────────┤
│ > composer (multiline input)                  │
└───────────────────────────────────────────────┘
```

Key bindings: `Enter` send · `Shift+Enter` newline · `Esc`/`Ctrl+C` cancel
turn · `y`/`n` approve/reject pending action · `d` view diff · `?` help ·
`Ctrl+C` twice fast = force quit (idle `q` quits).

Slash commands: `/help` `/status` `/diff` `/files` `/new` `/compact` `/quit`.

The full TUI spec is drafted in [`ui-spec-v0.1.md`](ui-spec-v0.1.md) and
will be validated by the Milestone 0 prototype.

## What Lives Where (cheat sheet)

| Concern | Package |
|---|---|
| Turn state machine, prompt building, compaction triggers | `internal/agent` |
| Streaming model calls, retries, delta accumulation | `internal/provider` |
| Tool contracts, envelopes, uniform errors | `internal/tool` |
| Allowlist/denylist matching, approval gating | `internal/policy` |
| Path containment, symlink-escape blocking, `.gitignore` | `internal/workspace` |
| JSONL append/load/resume/repair | `internal/session` |
| Config precedence, env-only secrets | `internal/config` |
| Token/cost tracking, debug logging | `internal/telemetry` |
| Rendering, input handling, modals | `internal/tui` |
| Wiring, root context, message routing | `internal/app` |

## Resolved Planning Decisions

- [x] ADR set confirmed: seven ADRs are drafted in [`adr/`](adr/) — Go+Bubble
      Tea, JSONL session store, provider abstraction, workspace confinement,
      patch-only editing, session-scoped approvals, untrusted tool output.
- [x] `testdata/` fixture list confirmed: five fixtures — `repo-small`,
      `repo-symlink-escape`, `repo-wordpress-plugin`, `repo-go-module`,
      `repo-node` — committed as plain directories (real symlinks checked in,
      no submodules/generation). Theme detection via loose headers in a
      `detect/` testdata dir.
- [x] Release tooling: **goreleaser** (locked) + a small `scripts/build.sh`
      wrapper for goreleaser-less local builds.
- [x] Module path: `github.com/djm56/kirsch` (from git remote).
- [x] Key precedence when both env vars are set: `KIRSCH_ANTHROPIC_API_KEY`
      wins silently.
