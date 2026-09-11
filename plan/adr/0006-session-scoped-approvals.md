# ADR 0006 — Session-scoped approval grants

- **Status:** Accepted (planning phase; no code yet)
- **Date:** 2026-09-11
- **Deciders:** Project owner

## Context

The original v0.1 approval flow offered only `y` (approve once), `n` (reject),
and `d` (view diff). In practice a real task runs `go test ./...` five or six
times in a single session, and if that command is not on the built-in
allowlist the user approves the identical action over and over. The known
failure mode is not that this is annoying — it is that repeated identical
prompts train the user to approve reflexively, which destroys the value of the
approval mechanism for the one prompt that actually mattered.

The static allowlist in plan §4 does not solve this. It covers common cases
(`go test*`, `npm test`, …) but cannot anticipate a project's own script
(`./bin/verify`, `make check`, `vendor/bin/phpunit --filter=X`). Options:

1. **Keep `y`/`n` only** — maximally safe on paper, but produces approval
   fatigue, which is a real security regression rather than a UX one.
2. **Let the user edit the allowlist in config mid-session** — persistent,
   powerful, and exactly the wrong default: a decision made once in a hurry
   silently applies to every future session.
3. **Session-scoped grants** — approve once, remember for the life of *this*
   session only.

## Decision

Add a fourth approval outcome, `a` — *approve, and don't ask again this
session for this scope*.

- **Scope is an argv prefix**, exactly as shown on the approval card (e.g.
  `go test`), matched by the same engine as the static allowlist.
- **Offered for `run_command` only.** `apply_patch` never offers `a`. Every
  patch is approved individually, always — principle #1 is not negotiable, and
  unlike a command, no two patches are the same action.
- **Never for `sh -c` / `bash -c`**, and never for a bare wildcard. An
  arbitrary shell string is not a scope.
- **Memory + session log, never config.** Grants are recorded as
  `approval.scope_granted` events so `kirsch resume` restores them, and are
  gone when the session is. They are never written to `config.toml`; making
  one permanent is a deliberate act of editing the allowlist by hand.
- **Visible and revocable.** `/approvals` lists active grants and can clear
  them; `/status` shows the count.
- **Disabled when another instance holds the workspace lock** (plan §7), and
  disabled wholesale by `[policy].allow_session_scoped_grants = false`.

## Consequences

**Positive**

- The repeated-prompt fatigue path is removed without making any decision
  permanent behind the user's back.
- The blast radius of a mistaken grant is bounded by the session, and the
  session record shows exactly when it was granted and what ran under it.
- Projects with their own verification commands work well without the user
  hand-editing an allowlist before the first useful task.

**Negative / risks**

- It is a genuine relaxation: a user who grants `git` rather than `git status`
  has granted more than they probably meant. Mitigated by showing the exact
  matched prefix on the card before the grant, and by prefix-matching argv
  rather than substring-matching a string.
- More policy state to restore correctly on resume; if restoration were
  buggy in the *permissive* direction it would be a security bug, so the
  resume path is tested explicitly (M4 acceptance).
- Two mechanisms now grant command execution (static allowlist, session
  grant). Both funnel through one matcher in `internal/policy` to avoid the
  classic bug of two code paths disagreeing about what is allowed.
