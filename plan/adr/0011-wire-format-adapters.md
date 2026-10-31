# ADR 0011 — Wire-format adapters

- **Status:** Proposed
- **Date:** 2026-10-30
- **Deciders:** Project owner (the operator), via the build plan (filed 2026-10-30); acceptance pending

## Context

Kirsch speaks one wire format today, the Anthropic Messages format, and ADR 0008 keeps one adapter for it. The built-in endpoints `opencode` and `anthropic` use that format. ADR 0008 treats the endpoint as configuration and the adapter as code.

The build plan (`plan/build-plan-v0.2-v0.6.md`) needs a second format. Its amendment A6 is filed here as ADR 0011. The build plan proposed this number as 0010; it was renumbered because 0009 was already taken (see ADR 0010 for the same renumbering on the catalog).

The inputs are these sections of the build plan:

- **Plan amendments and new ADRs**, row A6: one adapter package per `api` value, each translating to and from the neutral types. Per-model `compat` flags replace code branches. Thinking from Chat-format servers is decoded, shown, and never echoed.
- **Plan amendments and new ADRs**, row A3: the `api` key on `[provider.endpoints.<name>]`. The value `anthropic-messages` is the default. The value `openai-chat` arrives from M7.
- **Decisions already taken**, "Wire formats in order": Anthropic Messages (have); OpenAI Chat Completions (M7); OpenAI Responses only when a wanted model needs it. No Google format, no Bedrock or Vertex signing, and no OAuth device-flow logins.
- **Operator decisions**, O33: reasoning content from Chat-format servers is decoded to `BlockThinking`, shown dimmed, and never echoed back.
- **Milestone 7**, rows m7-d1, m7-d2, m7-d3 and m7-d5: recorded Chat-format fixtures, the `openai-chat` encode and decode, an adapter registry, and compat flags.

## Decision

Once this ADR is accepted, Kirsch adopts the following.

1. **One adapter package per `api` value.** Each adapter translates to and from the neutral types defined under ADR 0010. The adapter is selected by the `api` key. An unknown value fails at config load. The built-in endpoints keep `anthropic-messages`.
2. **Compat flags replace branches.** Per-model differences are rows in `models.toml`, under a `compat` table. The planned flags are `no_between_tools`, `no_parallel_tools`, `max_tokens_field` and `strict_tools`. The Haiku `between_tools` rejection becomes a compat row, not a code branch.
3. **Thinking from Chat-format servers is shown, never echoed.** Reasoning content, whether sent as `reasoning_content` or inside `<think>` tags, decodes to `BlockThinking`. It is displayed dimmed and is not sent back in later requests (O33).
4. **Responses format is deferred.** OpenAI Responses is added only when a wanted model needs it. This ADR does not add it.

## Consequences

**Positive**

- A new server that speaks an existing format needs configuration and a catalog row, not a new code path.
- Thinking from other servers cannot leak back into requests, because it is never echoed.

**Negative / risks**

- **Needed by M7, not M6.** Milestone 6 uses only the existing Messages adapter. Milestone 7 waits on this ADR being accepted.
- **Fixtures decide the detail.** The Chat-format behaviour is unverified until m7-d1 records real streams from the servers it names. Until then, the decode rules are what the plan says, not what a server does.
- **Compat flag set may grow.** The plan names these flags "at least", so the set may grow. Any further flag is a catalog change, and its test needs a fixture.
- Re-read against the code before the milestone that builds it. M7 builds on the catalog and role types that M6 reshapes.
