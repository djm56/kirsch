# kirsch-probe

A command-line tool to probe the OpenCode Go endpoint and gather facts about models, streaming, thinking blocks, caching, and error handling.

## Overview

`kirsch-probe` is an **opt-in** tool that the operator runs in their own shell with their own API key. It probes OpenCode Go's Messages endpoint across five models, testing nine scenarios per model:

- S1: Text completion
- S2: Tool call
- S3: Tool result round trip
- S4a–S4b: Prompt caching
- S5a–S5d: Thinking block forms
- S6: Version header presence (first model only)
- S0: Model listing (once)

**Total: 47 requests per run** (with all five models).

## Security

The probe follows strict security rules:

1. **Key from environment only:** Set `KIRSCH_OPENCODE_API_KEY` or `OPENCODE_API_KEY` in your shell. The prefixed variable takes precedence if both are set. Never pass the key as a flag or in a config file.

   To avoid leaving the key in shell history:
   ```bash
   read -rs OPENCODE_API_KEY && export OPENCODE_API_KEY
   ```

2. **Key is guarded on every output path:** The probe checks every file's contents before writing it and refuses to create a file that includes the key, which fails the run. Everything it prints to stdout or returns as an error has the key replaced with `<redacted>`, and server-derived text is redacted before it is truncated, so no key prefix is left behind. Note that a server which reflects the key into a response header therefore stops the run when that header file is refused.

3. **Redirects are never followed:** A redirect (3xx status) is recorded as an error and stops the probe immediately; it is never followed.

4. **Sequential requests with pauses:** Requests run 1 second apart (configurable) to respect subscription limits.

5. **Stop on auth/rate/gateway errors:** A 401, 403, 429, or 3xx response stops the entire run immediately. So does a 400 whose error `type` is `MissingSessionID` or contains `Session` (a gateway policy error: the client is not identifying itself properly, and every later request would fail the same way). A plain 400 for a bad request body does not stop the run, because S5's thinking forms are expected to 400 on some models. Everything collected so far is still written, and the stopping response's body is saved truncated to 4KB.

6. **Server-derived text is sanitised:** Anything the server controls that the probe prints or puts in a summary (a redirect's `Location`, an `error` event's message, a decode failure) is reduced to printable ASCII and length-capped. A redirect's `Location` is shown as its host only; a relative one shows `<no-host>` and one that cannot be parsed shows `<unparseable>`.

7. **`headers.json` keeps the full `Location`:** The saved response headers are the evidence record, so a redirect's `Location` is stored there verbatim (JSON-escaped, never printed to the terminal). Only `Set-Cookie` is dropped. A `Location` can carry a path or query string, so look at `*.headers.json` for any 3xx before you commit the directory.

## Identity headers

The gateway only routes clients that identify themselves. Its docs ([Where can I use it](https://opencode.ai/docs/go/#where-can-i-use-it)) ask a client to send a user agent of its own, rather than a generic SDK or HTTP-library name, and a stable session ID per conversation. The probe sends both on every request, `/models` included:

- `User-Agent: kirsch-probe/0.1`. Go's default `Go-http-client/...` is never sent.
- `x-opencode-session: <id>`. The ID is 128 random bits from `crypto/rand`, hex-encoded (32 characters), generated fresh on every run, so no ID is reused across runs. It is not a secret; the API key is never recorded.

One ID per conversation: S2 and S3 share one, because S3 continues S2's conversation. S4-1 and S4-2 share one, because prompt caching is routed by session. Every other scenario, and `/models`, has its own.

The ID used is recorded for every scenario, as `session` in its `summary.json` entry and as a `<scenario>.session` file next to its other output. `/models` has only the `.session` file, in `_global/`.

The 2026-10-02 14:00 UTC run sent neither header. `x-api-key` was accepted (no 401), but every Messages request returned 400 `MissingSessionID`: "Request is missing x-opencode-session and cannot be routed efficiently."

## Usage

### Dry run (plan only)

```bash
npm run probe
```

Shows the planned requests and total count without connecting to the network.

### Live run

```bash
npm run probe -- -run
```

Executes all scenarios against the live endpoint. Output is saved to `testdata/probe/opencode/<YYYY-MM-DDTHHMMZ>/` (UTC, to the minute), a new directory per run so a re-run never mixes with an earlier run's files. A live run refuses an output directory that already exists and is non-empty, before any request is made.

### Custom flags

- `-base-url <URL>` — Change the endpoint (default: `https://opencode.ai/zen/go/v1`). For tests only; must be `https://` or a literal loopback host.
- `-models <list>` — Comma-separated model list (default: `minimax-m3,minimax-m2.7,qwen3.8-max,qwen3.8-flash,qwen3.7-plus`). Names must be unique (compared case-insensitively) and `_global` is reserved for the S0 output directory; either is refused before any request is made.
- `-out <dir>` — Output directory (default: `testdata/probe/opencode/<YYYY-MM-DDTHHMMZ>`, UTC). Must not exist or must be empty for a live run.
- `-auth <mode>` — How the key is sent: `x-api-key` (default; sends `x-api-key: <key>`) or `bearer` (sends `Authorization: Bearer <key>`). Exactly one auth header is sent, on every request including `/models`. Any other value is refused before any request is made. The mode is shown in the dry-run plan header, and recorded in `summary.json` (`auth`) and the `SUMMARY.md` heading; the key never is.

Example:

```bash
export OPENCODE_API_KEY=your_key
npm run probe -- -run -models minimax-m3,qwen3.8-max
```

## Output

Results are saved under `testdata/probe/opencode/<YYYY-MM-DDTHHMMZ>/`:

```
<YYYY-MM-DDTHHMMZ>/
├── _global/
│   ├── S0-models.headers.json
│   ├── S0-models.sse
│   ├── S0-models.session
│   └── S0-models.status
├── minimax-m3/
│   ├── S1-text.request.json
│   ├── S1-text.sse
│   ├── S1-text.headers.json
│   ├── S1-text.status
│   ├── S1-text.session
│   ├── S2-tool-call.request.json
│   ├── S2-tool-call.sse
│   ├── ...
├── qwen3.8-max/
│   ├── ...
├── summary.json
└── SUMMARY.md
```

- **`.request.json`** — The request body sent to the endpoint.
- **`.sse`** — The raw SSE response (server-sent events stream).
- **`.headers.json`** — Response headers (with `Set-Cookie` filtered out).
- **`.status`** — HTTP status code.
- **`.session`** — The `x-opencode-session` ID sent with that request (not a secret). Scenarios in one conversation share it.
- **`summary.json`** — Machine-readable summary of the probe run. A scenario's `error` field carries stream problems (an `error` event, an event that would not decode, a `tool_use` whose streamed input was not valid JSON) and the reason an S3 was skipped.
- **`SUMMARY.md`** — Human-readable summary table of results. A status shown as `200*` is a 200 whose stream carried an error or an undecodable event; the detail is in `summary.json`.

## Review and commit

After running the probe:

1. Review `SUMMARY.md` to confirm which models and scenarios succeeded.
2. Commit the `testdata/probe/opencode/` directory:

   ```bash
   git add testdata/probe/opencode/
   git commit -m "Add OpenCode Go endpoint probe results"
   ```

The recorded fixtures are used by Milestone 3's adapter tests to replay real streams without hitting the network again.

## Data flow

**What the probe sends to the network:**

- `User-Agent: kirsch-probe/0.1` and a random `x-opencode-session` ID (see Identity headers).
- Your API key in one auth header: `x-api-key` by default, or `Authorization: Bearer` with `-auth bearer`.
- Fixed literal prompts defined in the probe code (no repository data, no user input).
- Query model names from the `-models` flag.

**Where it goes:**

- To OpenCode Go's gateway at `https://opencode.ai/zen/go/v1`.
- From there to the chosen model's host (e.g., Minimax, Alibaba Qwen).

**What it stores locally:**

- Responses from the endpoint (SSE streams, headers, status codes).
- Fixed literal prompts that are defined in the probe code.
- Model outputs (thinking text, generated responses).
- No credentials (API keys, bearer tokens).

## Troubleshooting

### `no API key set`

```bash
read -rs OPENCODE_API_KEY && export OPENCODE_API_KEY
npm run probe -- -run
```

This avoids leaving the key in your shell history.

### `probe stopped: received 401 response`

Your API key is invalid or expired. Regenerate it from your OpenCode Go account.

**401 `Missing API key` → try `-auth bearer`.** The endpoint wants the key in a header it did not receive. The default `x-api-key` is an inference (OpenCode's docs name `@ai-sdk/anthropic` for this endpoint, which sends `x-api-key`) and is unconfirmed until a live run succeeds. The 2026-10-02 run sent `Authorization: Bearer` and got a 401 `Missing API key` on the first Messages request (`GET /models` returned 200). Re-run with the other mode:

```bash
npm run probe -- -run -auth bearer
```

### `probe stopped: received 400 response ... gateway error type MissingSessionID`

**400 `MissingSessionID` → headers not sent.** The gateway wants `x-opencode-session` (and a non-generic `User-Agent`) and did not get them. The probe sends both on every request, so seeing this again means the headers were removed or altered, for example by a proxy or a code change. The probe stops after the first such response rather than send the rest of the plan. The saved `.request.json`, `.sse` and `.session` files for that scenario show what was sent and returned; see Identity headers.

### `probe stopped: received 429 response`

Your subscription's usage limit (5-hour or weekly) has been reached. Wait for the limit window to reset, or upgrade your plan.

### Redirects in output

If a response status is 3xx, the probe stops and records the redirect. This is a network or infrastructure issue, not the probe. The terminal output shows only the host of the `Location`; the full header is in `_global/S0-models.headers.json` (see Security item 7).
