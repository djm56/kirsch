# Kirsch

> **Status: Planning phase.** No code exists yet — the plan is complete and
> Milestone 0 is ready to start. The authoritative build spec is
> [`plan/kirsch-plan.md`](plan/kirsch-plan.md).

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
| [`plan/layout.md`](plan/layout.md) | Draft | Package layout, hard dependency rules, build order |
| [`plan/ui-spec-v0.1.md`](plan/ui-spec-v0.1.md) | Draft | Full TUI spec: layout, cards, modals, key bindings |
| [`plan/adr/`](plan/adr/) | Accepted | Seven architecture decision records |
| [`plan/milestone-0.md`](plan/milestone-0.md) | Ready | Instruction set — repo bootstrap + static TUI prototype |
| [`plan/milestone-1.md`](plan/milestone-1.md) | Ready | Instruction set — workspace engine + read-only tools |
| [`doc/`](doc/) | Planned | End-user documentation (Milestone 5) |

## Milestone Map

| Milestone | Delivers | Status |
|---|---|---|
| 0 | Repo skeleton + static TUI prototype | Not started |
| 1 | Workspace engine + read-only tools | Not started |
| 2 | Patches, commands, approvals | Not started |
| 3 | Provider + agent loop | Not started |
| 4 | End-to-end tasks + durable sessions | Not started |
| 5 | Polish, doctor, distribution, `v0.1.0` | Not started |

Live status is tracked in [`plan/README.md`](plan/README.md). We are **before
Milestone 0** — no code, no `go mod init`, no CI.

## License

[MIT](LICENSE) © 2026 Donovan Maidens
