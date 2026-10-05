# Kirsch

> **Status: Milestones 0 and 1 complete; Milestone 2 accepted with deferred
> items (see [`plan/PROGRESS.md`](plan/PROGRESS.md)); Milestone 3 (the model)
> is in progress, with m3-d1 and m3-d2 done.** Kirsch reads and searches
> a real repository from inside the TUI, and, through the temporary `/patch`
> and `/run` debug commands, can apply a patch or run a command: a patch
> always waits for your approval, a command does unless it is on the short
> exact-match allowlist or you have approved its prefix for the session, and
> shells always ask. Every path crosses a containment check, the path
> denylist is enforced in one place, and the `kirsch` binary still calls no
> model: the Messages adapter is reached only by an opt-in live test until
> m3-d6 wires it in. The authoritative build spec is
> [`plan/spec/kirsch-plan.md`](plan/spec/kirsch-plan.md); progress is
> tracked in [`plan/PROGRESS.md`](plan/PROGRESS.md).
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
          troubleshooting. Mostly written at Milestone 5; `usage.md` landed
          early because the help overlay already points to it.
```

If you are unsure where something belongs: *is it for someone building Kirsch,
or someone using it?* Building → `plan/`. Using → `doc/`.

| Document | Purpose |
|---|---|
| [`plan/README.md`](plan/README.md) | Build index, rules for the builder |
| [`plan/PROGRESS.md`](plan/PROGRESS.md) | Live milestone status, deliverables, blocking, and claiming |
| [`plan/START-HERE.md`](plan/START-HERE.md) | Ingestion order for humans and AI assistants, first-contribution path |
| [`plan/spec/kirsch-plan.md`](plan/spec/kirsch-plan.md) | Complete v0.1 build spec with milestones and amendments |
| [`plan/spec/architecture.md`](plan/spec/architecture.md) | Why the codebase is shaped this way: dependency rules, interfaces, concurrency, cancellation, error model |
| [`plan/spec/ui-spec-v0.1.md`](plan/spec/ui-spec-v0.1.md) | Full TUI spec: layout, cards, modals, mode state machine, palette, timing, accessibility |
| [`plan/spec/kirsch-ui-screens.md`](plan/spec/kirsch-ui-screens.md) | Screen reference: every UI state as an 80-column character grid with per-region colour maps — the render target and golden-test source |
| [`plan/testing/`](plan/testing/) | Manual walkthroughs, automated-test reference, security tooling |
| [`plan/adr/`](plan/adr/) | Seven architecture decision records |
| [`plan/process/working-agreement.md`](plan/process/working-agreement.md) | Claiming, branches, commits, PRs, file ownership, and board discipline for shared work |
| [`plan/process/code-standards.md`](plan/process/code-standards.md) | Go code standards — file layout, naming, comments, error handling, testing, and pre-PR checklist |
| [`plan/milestones/milestone-0.md`](plan/milestones/milestone-0.md) | Instruction set — repo bootstrap + static TUI prototype |
| [`plan/milestones/milestone-1.md`](plan/milestones/milestone-1.md) | Instruction set — workspace engine + read-only tools |
| [`plan/milestones/milestone-2.md`](plan/milestones/milestone-2.md) | Instruction set — patches, commands, approvals |
| [`plan/milestones/milestone-3.md`](plan/milestones/milestone-3.md) | Instruction set — provider + agent loop |
| [`plan/milestones/milestone-4.md`](plan/milestones/milestone-4.md) | Instruction set — real task loop + sessions |
| [`doc/`](doc/) | End-user documentation: install, configuration, usage, troubleshooting |

## Progress

Milestones 0, 1 and 2 are complete. Milestone 2's deferred items are listed in [`plan/PROGRESS.md`](plan/PROGRESS.md), and Milestone 3 (provider and agent loop) is in progress: m3-d1 and m3-d2 are done. Live status — what's in progress, what's blocked, who owns each deliverable — is tracked in [`plan/PROGRESS.md`](plan/PROGRESS.md).

## What works today

```
go run ./cmd/kirsch
```

Kirsch opens on a real repository, shows its project name, branch and dirty
state, and can read, list, search and diff it from inside the TUI — and, through
the temporary `/patch` and `/run` debug commands, apply a patch or run a command
once you approve it. Every path crosses a containment check before anything
touches the filesystem, and the path denylist (`.env`, `*.pem`, `*.key`, `.git/**`,
`.kirsch/**`) is enforced in one place so no tool can forget it.

The `kirsch` binary deliberately calls **no** model yet: the Messages adapter in
`internal/provider/anthropic` is reached only by the opt-in live test (`npm run
test:live`) until m3-d6 wires it in. Kirsch writes a file or runs a command
only through the temporary `/patch` and `/run` debug commands: a patch always
asks, and a command asks unless it is on the exact-match allowlist or you have
approved its prefix for the session. Apart from commands run through `/run`,
the only subprocesses are read-only `git` and `rg`.

## Verifying it

```bash
npm run tools      # install the lint and security tooling (one-off)
npm run check      # fmt + vet + lint + test + screens
npm run security   # govulncheck + gosec + secret scan
npm run fuzz       # property tests over the untrusted-input surfaces
```

`npm` is a front door, not a dependency — every script is one line of `go` or
`bash`, and nothing in the build needs Node. See
[`plan/testing/`](plan/testing/) for the manual walkthroughs and for what the
suite deliberately does not cover.

## License

[MIT](LICENSE) © 2026 Donovan Maidens
