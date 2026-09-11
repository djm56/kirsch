# ADR 0004 — Workspace confinement

- **Status:** Accepted (planning phase; no code yet)
- **Date:** 2026-09-11
- **Deciders:** Project owner

## Context

Kirsch reads files, applies patches, and runs commands inside a user's
repository. An agent that can wander outside the workspace root — via `..`
traversal, absolute paths, or symlinks planted inside the repo pointing
outside — is unacceptable. Secrets (`.env`, keys) must be off-limits to the
model.

## Decision

All file access is confined to the **workspace root** (Git root via
`git rev-parse --show-toplevel`, overridable with `--workspace`):

1. **Canonical path containment:** every path is resolved with
   `filepath.EvalSymlinks` and checked to be lexically/semantically inside
   the canonical root. Symlink escapes and `..` traversal are blocked and
   return `workspace_violation` tool errors.
2. **Built-in + `.gitignore` ignore rules** for listing/searching
   (`.git`, `node_modules`, `vendor`, `.kirsch`, …).
3. **Path denylist** (plan §4) blocks read/patch of `.env*`, `*.pem`,
   `*.key`, `.git/**`, `.kirsch/**` regardless of tool.
4. **`run_command` cwd confinement** plus environment filtering (pass through
   only `PATH`, `HOME`, `LANG` + allowlisted project vars; strip `*_TOKEN`,
   `*_KEY`, `*_SECRET`, `AWS_*`).
5. Enforced in `internal/workspace`/`internal/policy` — tools call the check,
   the check is not optional per-tool.

## Consequences

**Positive**

- Blast radius of any model mistake is the working tree, nothing else.
- Denylist guarantees secrets never enter model context via file tools.
- Testable in isolation with `testdata/repo-symlink-escape` fixture
  (Milestone 1 acceptance).

**Negative / risks**

- `EvalSymlinks` per access has a cost; acceptable at v0.1 scale, cache if it
  ever matters.
- Legitimate monorepo setups where useful code lives outside the Git root
  are not supported in v0.1 (`--workspace` override is the escape hatch).
- Filtering env vars can break some builds that need unusual vars; the
  project-config allowlist is the intended remedy.
