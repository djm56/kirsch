# ADR 0003 — Provider abstraction with a fake for tests

- **Status:** Accepted (planning phase; no code yet)
- **Date:** 2026-09-11
- **Deciders:** Project owner

## Context

v0.1 supports exactly one provider (Anthropic), but the agent core must not
know that. The agent loop (state machine, tool-call handling, cancellation)
needs deterministic tests with zero API spend and zero network. Options:

1. **Agent imports the Anthropic client directly** — fastest to build, but
   couples the loop to one vendor's streaming format and makes deterministic
   testing impossible.
2. **Provider interface + adapters** — small interface; provider-specific
   delta accumulation lives inside the adapter.

## Decision

Define a minimal provider-neutral interface in `internal/provider`:

```go
Stream(ctx context.Context, req Request, onEvent func(StreamEvent)) error
```

with provider-neutral `StreamEvent` types: `TextDelta`, `ToolCallStart`,
`ToolCallDelta`, `ToolCallEnd`, `MessageDone`, `Error`. All provider-specific
accumulation/normalization lives inside the Anthropic adapter.
`internal/agent` depends on the interface only (hard rule in architecture.md §3).
A **`provider.Fake`** with scripted turns provides deterministic agent-loop
tests: approval gating, cancellation, hallucinated tools, max-turn guard.

Additional decisions baked in: retry policy (3x exponential backoff on
5xx/network, `Retry-After` on 429, immediate clear failure on 401/403 with
onboarding message, 400 logs the full request as a bug). API keys come from
environment variables only — never from config files (refuse and warn if a
key appears there).

## Consequences

**Positive**

- Agent loop fully testable offline, deterministically, free.
- Adding a second provider later (out of v0.1 scope) is a new adapter, not a
  refactor.

**Negative / risks**

- Interface design must not leak Anthropic-isms; if the neutral event set
  proves too narrow for a future provider, that's a v0.2+ problem and the
  interface may need revision (acceptable).
- One more layer of indirection for a single-provider release — justified by
  test value alone.
