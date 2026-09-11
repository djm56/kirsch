# ADR 0002 — JSONL event log as session storage

- **Status:** Accepted (planning phase; no code yet)
- **Date:** 2026-09-11
- **Deciders:** Project owner

## Context

Sessions must be durable and resumable, local-disk only (no server, no
database service). Options considered:

1. **SQLite** — robust queries, but a C dependency complicates cross-compile
   and adds binary weight for v0.1 needs.
2. **Single JSON file rewritten on save** — simple, but rewrite-on-save risks
   corruption on crash mid-write and loses append-friendliness.
3. **Append-only JSONL, one event per line, schema-versioned** — crash-safe
   (worst case: one torn line), trivially replayable into transcript state,
   greppable by humans, cheap `fsync` discipline.

## Decision

Store sessions as **append-only JSONL event logs** (one file per session) with
a `v` schema-version field on every line (plan §7). Rules:

- Single writer goroutine owns all writes.
- Buffered flush with `fsync` every ~500ms and on turn completion.
- On load, a corrupt line truncates the session at that point and marks it
  `recovered`; startup is never blocked by one bad file.
- Full history always persists in JSONL even after compaction (compaction
  affects only what is sent to the model).

## Consequences

**Positive**

- Crash-safe by construction; repair semantics are local and simple.
- Resume = replay events; no deserialization of a monolithic state blob.
- Human-inspectable with plain `grep`/`jq`.

**Negative / risks**

- No ad-hoc querying (session search/branching deferred beyond v0.1 —
  already out of scope).
- File size grows unboundedly per session; acceptable for v0.1, revisit
  rotation later.
- Schema evolution requires the `v` field discipline from day one.
