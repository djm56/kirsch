# Configuration

Kirsch reads its settings from TOML files. Every setting has a built-in
default, so Kirsch runs with no configuration file at all. This page lists
every key, what it does, and where it may be set.

> **What this page covers.** The config file locations, the order in which
> the layers apply, the key reference, and the rules for secrets. Kirsch
> reads no configuration from command-line flags in v0.1. Options such as
> `--new`, `--debug`, and `--workspace` change one run and set no key on
> this page.

---

## Where the config files live

Kirsch reads a global file and a project file. Either may be missing; a
missing file is not an error.

| Scope | Path | Notes |
|---|---|---|
| Global | `$XDG_CONFIG_HOME/kirsch/config.toml` | Used when `XDG_CONFIG_HOME` is set and not empty. |
| Global | `~/.config/kirsch/config.toml` | Used otherwise. |
| Project | `.kirsch/config.toml` at the workspace root | The workspace is the enclosing Git repository, or the directory named with `--workspace`. |

The project file is read from the repository you are working in. Read the
[project file rules](#what-the-project-file-may-set) before you rely on it.

## Precedence

Kirsch applies these layers, in this order. Each layer changes only the keys
it sets. Keys that a layer does not mention keep the value from the layer
before it.

1. **Built-in defaults.** The values in the key reference below.
2. **Global file.** Your personal settings.
3. **Project file.** Accepted only for `context.project_files`. Every other
   key in it is ignored, with a warning that names the key. The warning goes
   to the [debug log](#debug-log).

Command-line flags are not a configuration layer. Kirsch reads no
configuration from command-line flags in v0.1, and no key on this page can
be set that way.

## What the project file may set

A project file may set `context.project_files` and nothing else. A
repository is untrusted input, and a project file overrides the global one,
so anything it could set, a hostile repository could set. Kirsch therefore
ignores every other key in a project file and names each ignored key in a
warning. That warning is written to the [debug log](#debug-log).

The rules for `context.project_files` itself are in the
[context table](#context). In short, each entry must be a relative path
inside the workspace.

A project file that sets a key whose name looks like a credential stops
startup. See [Secrets](#secrets).

## Secrets

**API keys are never read from a config file.** Name the environment variable
that holds the key with `api_key_env`, and export that variable in your shell.

```bash
export OPENCODE_API_KEY="..."   # or KIRSCH_OPENCODE_API_KEY, see below
```

Kirsch looks for the key in this order, and an empty value counts as unset:

1. `KIRSCH_` followed by the `api_key_env` name, for example
   `KIRSCH_OPENCODE_API_KEY`.
2. The `api_key_env` name itself, for example `OPENCODE_API_KEY`.

If both are set, the prefixed variable wins without a message. This lets
Kirsch run alongside other tools that use the unprefixed name. The
`/status` command names the variable that supplied the key, never its value.

Kirsch also refuses to start when any config file, global or project,
contains a key whose name looks like a credential. The match is
case-insensitive and looks at the last part of each key's name. `api_key`,
`apikey`, `token`, `secret`, and `password` all count, wherever they appear in
that part. The check covers keys Kirsch does not otherwise recognise, so
`max_tokens` stops startup as well as `api_key` does. Only the last part of a
dotted key is checked, so `provider.x.max_tokens` is checked as `max_tokens`
and stops startup. The error names the key.
The one exception is `api_key_env`, which holds a variable name, not a value.

## Key reference

Keys are written with their full dotted name, as they appear in the file.
`<name>` in an endpoint key stands for the endpoint's name, such as
`opencode`.

### `[provider]`

| Key | Default | Valid values | Scope | What it does |
|---|---|---|---|---|
| `provider.default` | `"opencode"` | The name of an endpoint defined under `[provider]`, built-in or your own | Global | Chooses the endpoint that serves model turns. Startup fails when no endpoint has this name. |

Each endpoint has the fields below. The built-in endpoints are
`opencode` and `anthropic`, and the table gives their defaults. You may define
endpoints under other names. A new endpoint has no defaults for the fields the
table marks as required, so those must be set. For a new endpoint, `thinking`
and `prompt_caching` default to off.

| Key | Default for `opencode` / `anthropic` | Valid values | Scope | What it does |
|---|---|---|---|---|
| `provider.<name>.base_url` | `https://opencode.ai/inference/go/anthropic/v1` / `https://api.anthropic.com/v1` | Required for a new endpoint. An `https` URL with any host. An `http` URL only on `localhost`, an IPv4 address in the `127.x.x.x` range, written in full, such as `127.0.0.1`, or `[::1]`. No username or password in the URL. | Global | The address Kirsch sends model requests to. |
| `provider.<name>.auth` | `x-api-key` / `x-api-key` | Required for a new endpoint. `x-api-key` or `bearer`. | Global | How the API key is sent with each request. |
| `provider.<name>.api_key_env` | `OPENCODE_API_KEY` / `ANTHROPIC_API_KEY` | Required for a new endpoint. A variable name. | Global | The name of the environment variable that holds the key. Never the key itself. |
| `provider.<name>.model` | `minimax-m3` / `claude-sonnet-5-5` | Required for a new endpoint. A model name the endpoint accepts. | Global | The model named in each request. |
| `provider.<name>.prompt_caching` | `true` / `true` (a new endpoint defaults to `false`) | `true` or `false` | Global | Adds cache markers to the parts of the system prompt that are marked for caching. |
| `provider.<name>.thinking` | `off` / `off` (a new endpoint defaults to `off`) | `off`, `low`, `medium`, or `high` | Global | The thinking level requested from the model. The exact request fields depend on the model. |

Startup checks every endpoint in the file, not only the one in use. A
malformed endpoint therefore stops startup even when `provider.default`
names a different one.

### `[policy]`

| Key | Default | Valid values | Scope | What it does |
|---|---|---|---|---|
| `policy.default_command_timeout_seconds` | `60` | A whole number from `1` to `300` | Global | The timeout for a command the agent runs, when the agent names no timeout of its own. |
| `policy.require_approval_for_patches` | `true` | `true` or `false` | Global | Whether applying a file change asks for your approval first. In this version every file change asks, whatever this is set to. See below. |
| `policy.require_approval_for_commands` | `true` | `true` or `false` | Global | Whether running a command asks for your approval first. Some commands always ask. See below. |
| `policy.allow_session_scoped_grants` | `true` | `true` or `false` | Global | Whether an approval can be granted for the rest of the session, not only once. Only commands can be granted: a file change and a shell are never granted. Grants are also off while another Kirsch is running in the same workspace. |
| `policy.env_passthrough` | `[]` | A list of variable names | Global | Extra environment variables passed to commands the agent runs, on top of `PATH`, `HOME`, and `LANG`. Names ending in `_TOKEN`, `_KEY`, or `_SECRET`, and names starting with `AWS_`, are removed even when listed. The match is case-sensitive: only the upper-case forms are matched, so a listed `my_token` reaches the command, and other secret-looking names, such as `GITHUB_PAT`, are not matched either. |

The 300-second limit applies to every command. A timeout above 300 is refused,
whether the agent names it or it comes from `default_command_timeout_seconds`.
The refusal comes after any approval prompt, so you may be asked to approve a
command that is then refused. Kirsch still starts with a larger value here, but
then every command that names no timeout of its own is refused. Keep this
setting at 300 or below.

**Command approvals.** `require_approval_for_commands` governs commands outside
a short built-in allowlist. Kirsch decides each command in this order:

1. Shells and runners always ask, even when the setting is `false`. The
   complete set is `sh`, `bash`, `zsh`, `dash`, `env`, `xargs`, and `nohup`,
   matched by file name, so `/bin/bash` counts too. Other shells, such as
   `fish`, `ksh`, and `csh`, and interpreters, such as `python`, are not in
   this set. They are treated like any other command: with the setting
   `false`, they run without asking.
2. The built-in allowlist runs without asking. It matches exact commands only,
   with nothing after them. The complete list is `go build`, `go test`,
   `git diff`, `git log`, and `ls`, and nothing else. So `go test ./...` and
   `ls -la` still ask.
3. Any other command asks when the setting is `true`, unless a session grant
   covers it. When the setting is `false`, every other command runs without
   asking.

**File-change approvals.** `require_approval_for_patches` is read at startup,
but the code that applies file changes never checks it. In this version every
file change asks for your approval, whatever the value. Setting it to `false`
does not turn that prompt off.

### `[context]`

| Key | Default | Valid values | Scope | What it does |
|---|---|---|---|---|
| `context.project_files` | `["AGENTS.md", "CLAUDE.md"]` | A list of paths relative to the workspace root | Global or project | The files Kirsch reads into the system prompt. Kirsch reads the first entry that is an ordinary file, and does not read the rest. A missing entry is skipped without a message. An entry is refused, with a warning, when it is an absolute path, when it leaves the workspace, or when it is a path Kirsch never reads, such as `.env` or `certs/server.key`, or anything inside a `.git` or `.kirsch` folder. An entry that is a folder or other special item is skipped with a warning. A list in a file replaces the default list; it does not add to it. |
| `context.max_project_context_bytes` | `32768` | A whole number of zero or more. Values above `32768` are treated as `32768`. | Global | The most bytes read from the chosen file. A longer file is cut at its last complete line within the limit, and a notice giving the limit is added. If no line break falls within the limit, only the notice is sent. At `0` that is always the case: Kirsch still picks the file, but sends only the notice. An empty file sends nothing. |

### `[session]`

| Key | Default | Valid values | Scope | What it does |
|---|---|---|---|---|
| `session.storage_dir` | `~/.local/share/kirsch/sessions` | A path. Only `~` and a path that starts with `~/` expand to your home folder. A relative path is taken from the folder you start Kirsch in. | Global | The folder for session data. Session logs go in `<folder>/sessions/<workspace-hash>`, one folder per workspace, and `index.json` and `usage.json` sit directly in `<folder>`. If the path ends in a folder named `sessions`, Kirsch treats that folder as the sessions folder and uses its parent as `<folder>`. |
| `session.auto_resume` | `true` | `true` or `false` | Global | When `true`, starting Kirsch resumes the most recent session for this workspace. If there is none, Kirsch starts a new session and shows the notice `no previous session — starting fresh`. If the resume fails for another reason, the notice says so and gives the reason. When `false`, or when another Kirsch is already running in this workspace, Kirsch starts a new session. The `--new` option on the command line also starts a new session. |

**Worked example, with the default.** Session logs go in
`~/.local/share/kirsch/sessions/<hash>`, one folder per workspace, where
`<hash>` is a 12-character hash of the workspace path. Each session is an
`<id>.jsonl` file in that folder. `index.json` and `usage.json` sit in
`~/.local/share/kirsch`, the folder above `sessions`. An empty
`session.storage_dir` falls back to that same folder, `~/.local/share/kirsch`.

### `[telemetry]`

| Key | Default | Valid values | Scope | What it does |
|---|---|---|---|---|
| `telemetry.debug_log` | `false` | `true` or `false` | Global | Writes the debug log on every run. Off by default. The `KIRSCH_DEBUG` variable and the `--debug` option turn it on for one run. See [Debug log](#debug-log). |

## Environment variables

The variables below change how Kirsch behaves. Kirsch reads others too, such as
`HOME`, to find your home folder.

| Variable | Effect |
|---|---|
| `KIRSCH_DEBUG` | Set to `1` to write the debug log for this run. Any other value does nothing. |
| `XDG_CONFIG_HOME` | Changes the global config directory, as described above. |
| `XDG_STATE_HOME` | Changes the directory for the debug log. |
| `KIRSCH_<NAME>` and `<NAME>` | The API key variables named by `api_key_env`. See [Secrets](#secrets). |

## Debug log

Kirsch writes a debug log only when you ask for one. It is off by default.
Turn it on for a single run with `--debug` or `KIRSCH_DEBUG=1`, or for every run
with `telemetry.debug_log = true`.

The log goes to `$XDG_STATE_HOME/kirsch/debug.log`, or to
`~/.local/state/kirsch/debug.log` when `XDG_STATE_HOME` is unset or empty. It
never writes to the screen, so it cannot disturb the display.

Warnings about configuration and project context go only to this log. With the
log off, a warning such as an ignored key in a project file, or a context file
that was refused, is not shown anywhere. Turn the log on to see those warnings.
[Using Kirsch](usage.md#starting-it) describes the same log.

## Startup checks

Kirsch checks the merged configuration before the interface opens. A failed
check stops startup and names the key at fault. The checks include, for
example:

- `provider.default` names an endpoint that exists.
- Each endpoint has the fields the table marks as required, and its `auth` and
  `thinking` hold valid values.
- Each `base_url` passes the rules in the endpoint table.
- `policy.default_command_timeout_seconds` is above zero. A value above 300
  passes this check. See the policy section.
- `context.max_project_context_bytes` is zero or more.

The credential check in [Secrets](#secrets) runs while the files are read,
before these checks.

An unknown key is a warning, not an error, unless its name looks like a
credential. A config file written for a newer Kirsch still starts an older one.

## Example

A global file that switches the default endpoint and turns on the debug log:

```toml
[provider]
default = "anthropic"

[telemetry]
debug_log = true
```

A project file that changes only the context files Kirsch reads:

```toml
[context]
project_files = ["AGENTS.md"]
```
