# Kirsch

> **Status: Milestones 0 and 1 complete.** Kirsch reads and searches a real
> repository from inside the TUI, safely: every path crosses a containment
> check, the path denylist is enforced in one place, and there is no code
> anywhere that writes a file, runs an arbitrary command, or calls a model. The
> authoritative build spec is [`plan/kirsch-plan.md`](plan/kirsch-plan.md);
> progress is tracked in [`plan/README.md`](plan/README.md).
>
> ```
> go run ./cmd/kirsch
> ```

Kirsch is an open-source, **terminal-native coding agent written in Go**. Its
primary interface is a simple full-screen Bubble Tea TUI, in the spirit of
Claude Code / OpenCode / Pi.

It works inside a Git repository, where it can:

- Investigate code (read, list, search files)
- Propose and apply small, reviewable patches
- Run controlled verification commands (tests, builds, linters)
- Keep a durable, resumable local session record

**First user:** a solo developer working on real projects (WordPress/PHP, Go).

## Core Ideas

1. **The agent is a partner, not an autoclicker.** It never writes to a file
   without an approved patch, and never runs a command without a policy
   decision (allowlist match, session grant, or explicit user approval).
2. **The workspace is a boundary.** Nothing outside the workspace root is
   ever touched — symlink escapes and `..` traversal are blocked outright.
3. **Everything is visible.** Every consequential action is shown in the TUI
   before it happens; every session is durable and resumable.
4. **Everything is cancellable.** No UI hang on a stuck tool or model call.
5. **Small, reviewable patches.** The unit of change is a unified diff the
   user can inspect, approve, or reject.
6. **Tool output is data, never instructions.** File contents and command
   output are treated as untrusted input, not as direction.
7. **Local-first.** JSONL session events on local disk only. No server, no
   telemetry, no cloud storage.

## Project Contract (locked for v0.1)

| Item | Value |
|---|---|
| Name | Kirsch |
| Language | Go |
| UI | Bubble Tea full-screen TUI (primary); non-interactive `kirsch run` later |
| License | MIT |
| Providers | One only for v0.1 (Anthropic) |
| Storage | JSONL session events, local disk only |
| CLI | stdlib `flag` + hand-rolled subcommands (no cobra) |
| Platforms | macOS + Linux, amd64/arm64 (no Windows in v0.1) |
| First user | Solo developer on real projects (WordPress/PHP, Go) |

## Explicitly Out of Scope for v0.1

Multiple providers, `git commit`/`git push` tools, MCP, LSP, subagents,
plugins, OS keychain, web UI, session branching/search, automatic compaction
UI tuning. Do not build these early.

## Repository Layout

```
plan/     everything needed to BUILD Kirsch — spec, architecture, ADRs,
          UI spec, per-milestone instruction sets. Scaffolding; goes away
          after v0.1.0.
doc/      everything needed to USE Kirsch — install, configuration, usage,
          troubleshooting. Written at Milestone 5; empty until then.
```

If you are unsure where something belongs: *is it for someone building Kirsch,
or someone using it?* Building → `plan/`. Using → `doc/`.

| Document | Status | Purpose |
|---|---|---|
| [`plan/README.md`](plan/README.md) | Live | Build index, progress tracker, rules for the builder |
| [`plan/kirsch-plan.md`](plan/kirsch-plan.md) | Locked (amended) | Complete v0.1 build spec with milestones |
| [`plan/architecture.md`](plan/architecture.md) | Settled | Why the codebase is shaped this way: dependency rules, interfaces, concurrency, cancellation, error model |
| [`plan/ui-spec-v0.1.md`](plan/ui-spec-v0.1.md) | Settled | Full TUI spec: layout, cards, modals, mode state machine, palette, timing, accessibility |
| [`plan/kirsch-ui-screens.md`](plan/kirsch-ui-screens.md) | Settled | Screen reference: every UI state as an 80-column character grid with per-region colour maps — the render target and golden-test source |
| [`plan/adr/`](plan/adr/) | Accepted | Seven architecture decision records |
| [`plan/milestone-0.md`](plan/milestone-0.md) | Ready | Instruction set — repo bootstrap + static TUI prototype |
| [`plan/milestone-1.md`](plan/milestone-1.md) | Complete | Instruction set — workspace engine + read-only tools |
| [`plan/milestone-2.md`](plan/milestone-2.md) | Ready | Instruction set — patches, commands, approvals |
| [`plan/milestone-3.md`](plan/milestone-3.md) | Draft | Instruction set — provider + agent loop |
| [`plan/milestone-4.md`](plan/milestone-4.md) | Outline | Real task loop + sessions |
| [`doc/`](doc/) | Planned | End-user documentation (Milestone 5) |

## Milestone Map

| Milestone | Delivers | Status |
|---|---|---|
| 0 | Repo skeleton + static TUI prototype | Complete |
| 1 | Workspace engine + read-only tools | Complete |
| 2 | Patches, commands, approvals | Not started — [instruction set ready](plan/milestone-2.md) |
| 3 | Provider + agent loop | Not started — [drafted](plan/milestone-3.md) |
| 4 | End-to-end tasks + durable sessions | Not started — [outlined](plan/milestone-4.md) |
| 5 | Polish, doctor, distribution, `v0.1.0` | Not started |

Live status is tracked in [`plan/README.md`](plan/README.md). Milestone 2 is
next and its instruction set is ready to execute.

## What works today

```
go run ./cmd/kirsch
```

Kirsch opens on a real repository, shows its project name, branch and dirty
state, and can read, list, search and diff it from inside the TUI. Every path
crosses a containment check before anything touches the filesystem, and the
path denylist (`.env`, `*.pem`, `*.key`, `.git/**`, `.kirsch/**`) is enforced in
one place so no tool can forget it.

There is deliberately **no** code anywhere in the binary that writes a file,
runs an arbitrary command, or calls a model. The only subprocesses are read-only
`git` and `rg`.

## License

[MIT](LICENSE) © 2026 Donovan Maidens
