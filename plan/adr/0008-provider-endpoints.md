# ADR 0008 — One Messages-format adapter, two configured endpoints

- **Status:** Accepted
- **Date:** 2026-10-02
- **Deciders:** Project owner

## Context

ADR 0003 fixed one provider for v0.1: Anthropic. Before Milestone 3 started,
the owner chose to develop and test against OpenCode Go instead. OpenCode Go is
a flat-rate subscription gateway: the Go plan costs $10/month, and the Go Plus
plan costs $40/month. Each plan sets per-model monthly limits, plus 5-hour and
weekly caps. The flat rate makes many live test runs affordable. Anthropic's
own API is to stay available.

OpenCode Go serves one base URL, `https://opencode.ai/zen/go/v1`, in three wire
formats (https://opencode.ai/docs/go/, read 2026-10-02):

- the Anthropic Messages format at `/messages`, for `minimax-m3`,
  `minimax-m2.7`, `qwen3.8-max`, `qwen3.8-flash` and `qwen3.7-plus`;
- the OpenAI Chat Completions format at `/chat/completions`, for its other
  open-model families (GLM, Kimi, DeepSeek and others);
- the OpenAI Responses format at `/responses`, for the Grok, GPT Luna and Muse
  Spark models.

Its documentation implies `Authorization: Bearer <key>`; the live probe found
that is rejected and `x-api-key` is required (see Built-in endpoints).
Anthropic's API uses the same Messages format, authenticates with `x-api-key`,
and requires an `anthropic-version` header
(https://platform.claude.com/docs/en/api/versioning, read 2026-10-02).

The options considered:

0. **Stay Anthropic-only and pay per token.** This leaves the plan unchanged,
   but every live test run costs money. That is what the owner wanted to avoid.
1. **One Messages-format adapter with a configurable endpoint.** This reaches
   OpenCode Go's Messages models and Anthropic direct with one wire format, and
   keeps Milestone 3's adapter design.
2. **One OpenAI Chat Completions adapter instead.** This reaches more of
   OpenCode Go's models. But it redesigns the adapter, the caching and the
   thinking tasks, and, because it is the only adapter, it drops Anthropic
   direct.
3. **Both adapters behind the provider interface.** This gives the widest model
   choice, at roughly double the provider work and fixtures for v0.1.

## Decision

Option 1. `internal/provider` gets a single adapter for the Anthropic Messages
wire format. An *endpoint* is configuration, not code. Each named endpoint
under `[provider.<name>]` sets these keys:

- `base_url`;
- `auth`, which is `bearer` or `x-api-key`;
- `api_key_env`;
- `model`;
- the existing `prompt_caching` and `thinking`.

`[provider].default` names the endpoint in use.

**Key variables.** `api_key_env` names the unprefixed variable. Kirsch reads
`KIRSCH_` + that name first and falls back to the unprefixed name. When both
are set, the prefixed one wins silently. Keys come from the environment only,
never from a config file. That rule and its refusal are unchanged from ADR 0003.

**Built-in endpoints.** Two endpoints ship built in:

- `opencode`, the default: base URL `https://opencode.ai/zen/go/v1`, `x-api-key`
  auth, `api_key_env = "OPENCODE_API_KEY"`. The key is read from
  `KIRSCH_OPENCODE_API_KEY`, then `OPENCODE_API_KEY`. The model is
  `minimax-m3`, chosen by the operator after the live probe of 2026-10-02
  (plan §11 amendment 80). The probe found that `Authorization: Bearer` is
  rejected with 401 "Missing API key.".
- `anthropic`: base URL `https://api.anthropic.com/v1`, `x-api-key` auth,
  `api_key_env = "ANTHROPIC_API_KEY"`. The key is read from
  `KIRSCH_ANTHROPIC_API_KEY`, then `ANTHROPIC_API_KEY`. The model is
  `claude-sonnet-5-5`, confirmed against the plan §5 model table
  (amendment 79).

A user may define further endpoints in the global config. A user-defined
endpoint must set `base_url`, `auth`, `api_key_env` and `model`, because there
is no default to inherit.

**Version header.** The adapter always sends the `anthropic-version` header,
which Anthropic's API requires on every request (see Context).
The live probe found that OpenCode Go accepts the header and does not require
it.

**Client identity (`opencode`).** OpenCode Go requires every request to carry
the client's own User-Agent and an `x-opencode-session` header holding a stable
ID per conversation (https://opencode.ai/docs/go/#where-can-i-use-it). Kirsch
sends `User-Agent: kirsch/<version>` and one random session ID per Kirsch
session, reused when that session is resumed, so routing and prompt caching
stay on one conversation. The session ID is not a secret and is never derived
from the key. Neither the User-Agent nor the session ID is a config key, so no
config file can set them. Without it the gateway returns
400 `MissingSessionID`, which Kirsch surfaces as a configuration error rather
than retrying.

**Trust boundary.** The whole `[provider]` section is honoured only from the
global config file and from command-line flags. That covers
`[provider].default`, every endpoint's `base_url`, `auth`, `api_key_env`,
`model`, `thinking` and `prompt_caching`, and the definition of new endpoints.
If the project config (`<workspace>/.kirsch/config.toml`) contains a
`[provider]` table or any key beneath it, Kirsch ignores all of it and prints
a warning naming the ignored keys. That warning is distinct from the
unknown-key warning. A project file cannot choose the endpoint, the
credential, the model or the thinking level.

Every `base_url` must use `https://`. The one exception is a loopback host,
written literally as `localhost`, an address in `127.0.0.0/8`, or `::1`.
No hostname is resolved to decide whether it is loopback. The host checked is
the one `net/url` parses from `base_url`, after any userinfo is removed, so
`http://localhost@evil.example/` is not loopback. IPv4-mapped forms such as
`::ffff:127.0.0.1` are not accepted. A host with a zone identifier, or an
empty host, is not loopback.

The reason: a repository must never be able to choose where the user's key and
source code are sent, which environment variable is sent as the credential, or
which model and thinking level the user pays for. ADR 0007 treats tool output
as untrusted input. This extends ADR 0007's premise, that a repository is
untrusted input, from model-facing content to configuration. Per-project
provider choice can be revisited later with explicit user confirmation; v0.1
does not offer it.

**Redirects.** The adapter does not follow HTTP redirects. A 3xx response is an
error naming the status and the `Location` host. Following a redirect would
forward the `x-api-key` header to the target. Go's HTTP client strips only
`Authorization`, `WWW-Authenticate` and `Cookie`, and only on a redirect to a
domain that is not an exact or subdomain match of the original. Custom headers
such as `x-api-key` are forwarded to any target.

**Proxies and certificates.** The proxy variables `HTTPS_PROXY`, `HTTP_PROXY`
and `NO_PROXY` (and their lowercase forms), and the certificate variables
`SSL_CERT_FILE` and `SSL_CERT_DIR`, are honoured as the user's own
environment, exactly as Go's standard library honours them on each platform.
They are not config keys, and a project file cannot set them. When a proxy is
in effect, `/status` names the proxy host.

**Config files and the model.** The model must not be able to move the trust
boundary by editing a config file. The file tools (`read_file`, `apply_patch`
and the rest) already refuse both files. `<workspace>/.kirsch/` is on the
workspace denylist, and the global file lies outside the workspace root.
`run_command` can write anywhere the user's account can, so for a command the
boundary is the user's approval. The approval card must show the full argv,
and a command touching `.kirsch/` or the global config directory gets no special
treatment in v0.1. That is a known limit, to be recorded in
`plan/testing/security.md` by Milestone 3 (Task 7 item 9).

**Key provenance.** `/status` names the environment variable that supplied the
key, for example `KIRSCH_OPENCODE_API_KEY`. It never shows the value, a prefix
of it, or its length.

**Implementation note.** The project layer must be decoded into a scratch
struct, and its `[provider]` keys reported and discarded, before the rest is
merged. Decoding it straight into the live config, as `mergeFile` does today,
would apply the values before any check could run.

**Thinking.** Current Claude models think by default, using adaptive thinking
(https://platform.claude.com/docs/en/build-with-claude/thinking and its
per-model configuration table, read 2026-10-02). On `claude-sonnet-5-5`, the
`anthropic` endpoint's default model:

- the manual `thinking.type: "enabled"` plus `budget_tokens` form returns a 400
  error;
- `thinking.type: "disabled"` returns a 400 error. `"between_tools"` is the
  form that turns off up-front thinking, at `high` effort or below;
- returned thinking blocks carry a `signature`.

Other models differ. The per-model table is the source, checked per model when
the plan §5 model table row is written.

The adapter must therefore preserve every thinking block a response returns,
and pass it back unchanged, signature included, in later requests within the
turn. This applies from the first tool turn, at every `thinking` setting
including `off`, because the model may think whatever the setting says.
`off` maps to `"between_tools"` on `claude-sonnet-5-5`, with effort held at
`high` or below. How effort is set, and how `low | medium | high` map onto
the request, is decided at Milestone 3, against the documentation.

Plan §6.5 ("Default `off`", thinking blocks handled only "when it is enabled")
and Milestone 3 Task 5 ("Default `off`") contradict this. This ADR governs
where they disagree; the plan §11 Amendment Log records the amendment.

The live probe found that `minimax-m3` thinks only when asked (`enabled` or
`between_tools`), returns no thinking with `disabled` or with no `thinking`
field, and reports cache reads (plan §11 amendment 80).

**Provider interface.** The provider interface and `provider.Fake` from ADR 0003
are unchanged. Every wire-specific detail stays inside the adapter and out of
the provider and agent packages' exported surfaces.

## Consequences

**Positive**

- Live testing is cheap enough to run often. Anthropic direct still works by
  changing one config value.
- Milestone 3's adapter, fixtures and acceptance list carry over with
  configuration added. Only the thinking request shape changes, as described
  above.
- Recorded fixtures come from a real endpoint without per-token cost.

**Negative / risks**

- Only OpenCode Go's Messages-format models are reachable. Its other models
  need a second adapter, which is outside v0.1's scope (option 3, later).
- **Gateway differences.** OpenCode Go's Messages endpoint is a gateway in
  front of non-Anthropic models. It may differ from Anthropic's API in stream
  event details, usage fields, cache reporting or thinking blocks. Fixtures
  must be captured from each endpoint actually used, and the adapter must
  tolerate missing optional fields.
- **Data flow.** Using `opencode` sends prompts and file contents to OpenCode's
  gateway and to whoever hosts the chosen model, not to Anthropic. Onboarding
  and `/status` must say so.
- **Usage limits.** The subscription's 5-hour limit is 20% of the monthly
  limit, and the weekly limit is 50%. A 429 caused by an exhausted limit can
  mean a lockout of hours or days. It must be surfaced as such, not retried as
  if it were transient. The adapter classifies a 429 as a lockout when the
  response identifies an exhausted usage limit by a signal the live probe has
  recorded. Absent such a signal, it applies ADR 0003's `Retry-After`
  back-off. The probe records which signal OpenCode Go sends.
- **Model figures.** The budget maths and the cost display need each model's
  context window, output limit and pricing, taken from that model's published
  documentation. OpenCode Go publishes model IDs only. Until a sourced row
  exists, the plan §5 unknown-model fallback applies: 128k context, a 4k output
  reserve and cost `?`. A flat-rate subscription has no per-token price, so the
  cost display for `opencode` shows usage rather than money.
- **Trust boundary.** Config loading gains a rule that must be implemented and
  tested: a `[provider]` table in a project file is ignored, with a warning
  naming its keys. `plan/testing/security.md` gains cases for a hostile
  `base_url`, a hostile `[provider].default`, a hostile model or thinking
  level, a redirecting endpoint, a userinfo or IPv4-mapped loopback
  `base_url`, and a `run_command` that writes a config file. One consequence
  for users: a project cannot pin its own model. That is the price of the
  boundary.
- **Partial supersession.** This decision supersedes in part two points of
  ADR 0003. First, "v0.1 supports exactly one provider (Anthropic)". Second,
  its 429 handling: honouring `Retry-After` as a transient back-off no longer
  covers a usage-limit 429, which is surfaced as a lockout (above). The rest of
  ADR 0003 stands.
