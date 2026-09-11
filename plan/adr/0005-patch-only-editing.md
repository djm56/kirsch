# ADR 0005 — Patch-only editing

- **Status:** Accepted (planning phase; no code yet)
- **Date:** 2026-09-11
- **Deciders:** Project owner

## Context

The agent must modify files, but principle #1 says: the agent never writes to
a file without an approved patch. Options for the write path:

1. **Direct write tools** (`write_file`, `edit_file` with string replace) —
   how many agents work, but edits are opaque unless the user diffs
   afterwards; approval happens on an argument blob, not a change.
2. **Patch-only editing** — the model emits unified diffs; Kirsch validates
   (dry-run, strict context match), renders the diff for approval, then
   applies.

## Decision

The only write path in v0.1 is **`apply_patch` with a unified diff**:

- Dry-run validation happens *before* the approval prompt; the user is never
  asked to approve a patch that would conflict.
- **Strict context matching:** if the file changed since the model last read
  it, application fails cleanly with `patch_conflict` (returned to the model
  as a tool result so it can re-read and retry).
- Diff paths are checked against the denylist and workspace containment
  (ADR 0004).
- The approval card shows files changed; `d` opens the colour-coded diff
  modal before deciding (ui-spec §4.1).
- Git write operations (`commit`, `push`) are not exposed to the model at all
  in v0.1.

## Consequences

**Positive**

- The object of approval is exactly the change — no gap between what was
  approved and what was written.
- Force-diffed review makes every session auditable after the fact.
- Conflict failures are graceful and model-recoverable, not corrupt states.
- Keeps the door open for a future `git commit` tool built on reviewed diffs.

**Negative / risks**

- Models sometimes emit malformed diffs; mitigated by dry-run validation plus
  `tool_input_invalid` self-correction loop (max 2 retries, then surface to
  user).
- Diff generation can be token-hungry for large changes; v0.1 targets small,
  reviewable patches anyway — this constraint is a feature.
- No whole-file rewrites via a single `write` call; large scaffolding tasks
  will produce big diffs. Accepted trade-off for v0.1.
