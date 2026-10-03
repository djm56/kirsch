# Architecture Decision Records

An **Architecture Decision Record** (ADR) documents a significant decision about how the system is shaped — the situation that prompted it, the options considered, what was chosen, and what it costs.

An ADR is written once, at the time the decision was made. Its body is not revised afterwards; only its Status line changes, to record that a later ADR has superseded it. It is a record of *why the system is not something else* — the thing that gets forgotten and causes people to re-litigate settled questions six months later. The code tells you what the system does; an ADR tells you why it is not built a different way.

The format is derived from Michael Nygard's [Architecture Decision Records](https://adr.github.io/), which originated the pattern and the name. The term is now a convention recognized across the industry, which is why the folder keeps the standard name — it makes the pattern recognisable to people who already know it.

## Format

Every ADR follows this structure:

```markdown
# ADR NNNN — Short title

- **Status:** one of [Accepted, Proposed, Deferred, Superseded, Rejected]
- **Date:** YYYY-MM-DD
- **Deciders:** who decided

## Context

The situation and the options considered.

## Decision

What was chosen and why.

## Consequences

What it costs and what it forecloses.
```

## When to Write One

Write an ADR for a decision that:

- **Is expensive to reverse** — the cost of changing your mind later is high.
- **Closes off alternatives** someone would otherwise reach for later.
- **Will be questioned again in six months** — the kind of choice that prompts "why didn't we just...?" when new team members arrive.

Not every choice deserves an ADR. A folder full of forty records is a folder nobody reads. Use judgment — if it is not significant enough to be questioned, it is probably not significant enough to record.

## When NOT to Write One

If the decision **changes the plan**, it goes in the Amendment Log in [`spec/kirsch-plan.md`](../spec/kirsch-plan.md) §11 instead.

Make this distinction crisp: an ADR records a decision about *how the system is shaped*; an amendment records a change to *what the plan says to build*. A reader must be able to tell which they need without ambiguity.

## ADR Index

| # | Title |
|---|---|
| 0001 | [Go + Bubble Tea for the TUI](0001-go-bubble-tea.md) |
| 0002 | [JSONL event log as session storage](0002-jsonl-session-store.md) |
| 0003 | [Provider abstraction with a fake for tests](0003-provider-abstraction.md) |
| 0004 | [Workspace confinement](0004-workspace-confinement.md) |
| 0005 | [Patch-only editing](0005-patch-only-editing.md) |
| 0006 | [Session-scoped approval grants](0006-session-scoped-approvals.md) |
| 0007 | [Tool output is untrusted input](0007-untrusted-tool-output.md) |
| 0008 | [One Messages-format adapter, two configured endpoints](0008-provider-endpoints.md) |

## Status Update

ADRs 0001–0007 were written during planning, and most still carry the status "Accepted (planning phase; no code yet)". That parenthetical is now stale, because Milestones 0–2 have shipped. Whether to update it is the operator's decision, not an agent's. One status has changed, at the operator's direction on 2026-10-02: ADR 0003 now records that ADR 0008 supersedes it in part.
