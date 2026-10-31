# ADR 0012 — Execution seam

- **Status:** Proposed
- **Date:** 2026-10-30
- **Deciders:** Project owner (the operator), via the build plan (filed 2026-10-30); acceptance pending

## Context

Kirsch runs its filesystem, search, git and command calls directly on the local machine, with no seam between the tools and the machine they act on. Path containment is set by ADR 0004 (workspace confinement), and approval by ADR 0006 (session-scoped approvals).

The build plan (`plan/build-plan-v0.2-v0.6.md`) adds a seam for Milestone 9, so that the TUI on one machine can work a repository on another. Its amendment A7 is filed here as ADR 0012. The build plan proposed this number as 0011; it was renumbered because 0009 was already taken (see ADR 0010 for the same renumbering on the catalog).

The inputs are these sections of the build plan:

- **Plan amendments and new ADRs**, row A7: an `Executor` interface behind every filesystem, search, git and command call. Containment is checked on the path string before any executor call. Patches travel as text. Local comes first, ssh second, and a container only if it is cheap.
- **Plan amendments and new ADRs**, row A2: new package `internal/executor` (M9).
- **Decisions already taken**, "Isolation is policy plus container, not a home-grown sandbox": `doc/sandboxing.md` documents devcontainer and Docker Sandbox use. Protected paths become executor-enforced in M9. An opt-in `bwrap` or `sandbox-exec` wrapper is deferred until real use shows the approval model is not enough.
- **Operator decisions**, O39: read-only protected paths are policy-only on the local executor, and executors with mounts enforce them. O40: the ACP SDK is `coder/acp-go-sdk`, and the ssh transport is the system `ssh` with `ControlMaster`.
- **Milestone 9**, rows m9-d1, m9-d2 and m9-d3: the local executor, protected paths, and an ssh executor that pipes patches to `git apply --check` and then `git apply` on the remote.

## Decision

Once this ADR is accepted, Kirsch adopts the following.

1. **One `Executor` interface.** Every read, list, search, git call, command and patch application goes through `Executor`. Tools and the workspace call it. They do not call the machine directly.
2. **Containment on the path string, first.** Containment is checked on the path string before any executor call. The executor is never asked to act on a path that failed that check.
3. **Patches travel as text.** A patch crosses the seam as text. It is not a file copied between machines.
4. **Order of executors.** Local comes first (m9-d1), then ssh (m9-d3). A container executor is built only if it is cheap.
5. **Isolation stays policy plus container.** Kirsch does not build its own OS sandbox. For executors with mounts, protected paths are enforced by the executor (O39). For the local executor, O39's default and m9-d2's Done when disagree, and this ADR does not choose between them (see Consequences).
6. **Contract row changes at the tag.** The Contract-table row "Providers" becomes "any Messages- or Chat-format endpoint; model chosen per role" at the M6 tag, not when this ADR is accepted. The spec line is not edited by this ADR.

## Consequences

**Positive**

- Approvals, the session log and containment can be checked in one place for every configuration, because every machine action passes the same seam.
- The ssh executor can be added without changing the tools that call it.

**Negative / risks**

- **Conflict on local protected paths.** O39's default is "policy-only on local". Milestone 9's m9-d2 "Done when" says the executor refuses writes to denylisted paths and `.git/**` itself, on the local machine too. They cannot hold together. The operator must decide before m9-d2 starts, and the answer belongs in an amendment. Until then, `doc/sandboxing.md` must not claim local enforcement.
- **Conflict with ADR 0004 on symlinks.** ADR 0004 Decision 1 and spec principle 3 (`plan/spec/kirsch-plan.md:27`) require containment to resolve symlinks (`filepath.EvalSymlinks`). A check on the path string alone does not, and a remote executor cannot resolve local symlinks. The operator must decide how the seam keeps principle 3 (for example, resolution inside the executor) before m9-d1, and the answer belongs in an amendment.
- **For you to decide: the contract-row timing.** Decision 6 dates the A7 contract-row change ("any Messages- or Chat-format endpoint") at the M6 tag, but the Chat-format adapter arrives with M7 (v0.3.0), and M6 uses only the Messages adapter. Move the change to the M7 tag or confirm M6.
- **Remote tools are a dependency.** The ssh executor needs `rg` and `git` on the remote. A missing one is a doctor failure (m9-d3), and the remote path depends on it.
- **Milestone 9 only.** Nothing in this ADR changes the behaviour of any milestone before Milestone 9.
- Re-read against the code before the milestone that builds it. The M9 seam is built on the tools and workspace as they stand at M8.
