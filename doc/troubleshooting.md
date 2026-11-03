# Troubleshooting

When Kirsch will not start, or starts and then cannot send a prompt, begin
with `kirsch doctor`. It prints a line for each result its checks give. It sends
no prompt to a model and makes no network calls, so you can run it at any
time.

This page explains each line doctor can print, the common failures behind
them, and what to do about each one. For installing Kirsch, see
[install.md](install.md). For the settings named here, see
[configuration.md](configuration.md).

> **What this page covers.** The `kirsch doctor` report line by line, and the
> messages you see in the program for the failures people meet most often.
> Placeholders in angle brackets, such as `<path>`, stand for values that
> depend on your machine. Each one is explained where it first appears in its
> section.

---

## Reading the report

Run doctor inside your project folder, or pass `--workspace <path>` to check
another folder:

```bash
cd /path/to/your/project
kirsch doctor
```

The report has this shape:

```text
kirsch <version>
<status>  <check>: <detail>
doctor: <ok> ok, <warn> warn, <fail> fail
```

- `kirsch <version>` is the opening line. The version is the release version,
  or `0.1.0-dev` in a build that carries no release version. See
  [install.md](install.md) for how a build gets its version.
- `<status>` is `ok`, `warn`, or `fail`. Spaces follow it, as in the examples below.
- `<check>` is the name of the check, in lower case.
- `<detail>` is the message. Each one is quoted below.
- The last line counts the statuses.

The checks print in the order of the headings below. A `warn` line that
begins `skipped:` means the check did not run, because an earlier check left
it nothing to read. A config that did not load skips every check that reads
it, and a run with no workspace skips the checks that need a folder.

**Exit status.** Doctor exits `0` when no check fails, and `1` when at least
one check fails. A warning never causes a failure. Fix the `fail` lines first.

Example output from a repository with no key variables set. Your lines and
paths will differ:

```text
kirsch 0.1.0-dev
ok  git: found at /usr/bin/git
ok  rg: found at /usr/local/bin/rg
ok  repository: /home/you/project
ok  config: settings valid
warn  key: endpoint opencode: no key found (set KIRSCH_OPENCODE_API_KEY or OPENCODE_API_KEY); kirsch starts but cannot send a prompt
ok  backend: search_code uses ripgrep
ok  model: endpoint opencode, model minimax-m3: context window 128000 tokens
ok  context: AGENTS.md, 6854 bytes
ok  state: /home/you/.local/share/kirsch/sessions/3f9a1c07b2d4 exists and is writable
ok  lock: free
doctor: 9 ok, 1 warn, 0 fail
```

---

## git

```text
ok  git: found at <path>
```

`<path>` is the full path of the `git` program found on your `PATH`. Nothing
to do.

```text
fail  git: not found on PATH; kirsch needs git to find the repository
```

This appears when `git` is not on your `PATH` and you did not pass
`--workspace`. Kirsch will not start. Install Git, then open a new terminal so
the `PATH` change applies.

```text
warn  git: not found on PATH; kirsch starts with --workspace but the Git features are off, such as /diff, git_status, git_diff, the branch display and the dirty count
```

This appears when `git` is not on your `PATH` and you did pass
`--workspace`. Kirsch starts with its Git features off. Install Git to turn
them on.

---

## rg

```text
ok  rg: found at <path>
```

`<path>` is the full path of `rg`, which is ripgrep. Nothing to do.

```text
warn  rg: not found on PATH; search_code uses the pure-Go fallback
```

Kirsch's code search works without ripgrep. It uses its own built-in search
instead. This is a warning, not a failure. To use ripgrep, install it with
your system's package manager. See [install.md](install.md#ripgrep-optional).
The `search_code` part of the message is explained under
[Missing ripgrep](#missing-ripgrep) below.

---

## repository

```text
ok  repository: <root>
```

`<root>` is the folder Kirsch treats as the project root. That is the
enclosing Git repository, or the folder you named with `--workspace`.

```text
fail  repository: not in a Git repository
```

This appears when you run doctor outside a Git repository and did not pass
`--workspace`. Kirsch will not start. See
[Not in a repository](#not-in-a-repository) below.

```text
fail  repository: --workspace <path>: <error>
```

This appears when the folder named with `--workspace` cannot be used. The
detail is the error from the operating system, with the path shown in front
of it. For example, for a folder that does not exist:

```text
fail  repository: --workspace /nonexistent-kirsch-path: stat /nonexistent-kirsch-path: no such file or directory
```

If the path is a file and not a folder, the detail is
`--workspace <path> is not a directory`. Give Kirsch the path of a folder
that exists.

A rarer detail, `resolve workspace root <root>: <error>`, can also appear.

```text
warn  repository: <root> is not a Git repository; kirsch starts, but the Git features are off, such as /diff, git_status, git_diff, the branch display and the dirty count
```

This appears when `--workspace` names an existing folder that is not a Git
repository. Kirsch starts with its Git features off. `<root>` is that folder.

When `git` is missing, doctor leaves out the repository line, because the
`git` line already reports the cause.

---

## config

Kirsch reads a global settings file and a project settings file. The global file is
`$XDG_CONFIG_HOME/kirsch/config.toml`, or `~/.config/kirsch/config.toml` when
`XDG_CONFIG_HOME` is not set. The project file is `.kirsch/config.toml` in the
workspace. Each is described in [configuration.md](configuration.md).

The config check prints a `warn` line for each warning the loader raises.
It then prints either a `fail` line or an `ok` line.

```text
warn  config: <file>: unknown key "<key>" (ignored)
```

`<file>` is the config file the key is in. `<key>` is a setting Kirsch does not
know. It may be a typo, or a setting that a newer Kirsch understands. Kirsch
ignores it. Correct the spelling if the setting was meant to take effect.

```text
warn  config: <file>: key "<key>" ignored (a project config may set only [context].project_files)
```

This appears when the project file sets a key other than
`context.project_files`. Kirsch ignores the key. A project file may set only
that setting. To change any other setting, put it in your global file. The
same text appears when the project file holds a table Kirsch does not
recognise. In that case `<key>` is the table's name.

```text
warn  config: <file>: context.project_files must be a list of strings; ignored
```

This appears when `context.project_files` in the project file is not a list
of strings. Kirsch ignores the setting. Write it as a list, for example
`project_files = ["AGENTS.md"]`.

```text
fail  config: read <path>: <error>
fail  config: parse <path>: <error>
```

`read` appears when a settings file exists but cannot be read, usually
because of its permissions. `parse` appears when a file is not valid TOML.
`<path>` is the settings file's path. `<error>` is the operating system's message for `read`, and the parser's message for `parse`.
A type error also reaches you through
this line. For example, a `provider.default` that is not a string is reported
as `parse <path>: provider.default must be a string`. Fix the named file.

```text
fail  config: <file>: key "<key>" looks like a credential; Kirsch never reads secrets from config files, because config files get committed and shared. Set the value in an environment variable and name that variable with `api_key_env` instead.
```

This appears when a settings file holds a key whose name looks like a
credential, such as `api_key` or `token`. Kirsch refuses to start. Remove the
key from the file. Put the value in an environment variable, and name that
variable with `api_key_env`. The details are in [Secrets](configuration.md#secrets).

The message is longer than one line. Doctor joins its parts onto one line.

```text
fail  config: invalid settings (<consulted>): <error>
```

This appears when the settings parse, but the combined result is not valid.
`<consulted>` names the files that were checked: `global <path>` for the global
file, and `project <path>` for the project file when there is a workspace. It
reads `no config files` when no path is set. `<error>` is a message from the list below.

- `provider.default = "<name>"; endpoint does not exist`. Set
  `provider.default` to the name of an endpoint defined under `[provider]`.
- `provider.<name>.base_url is required`, and the same form for each other
  required field. Only an endpoint you define needs these, because
  the built-in endpoints supply them.
- `provider.<name>.auth = "<value>"; want x-api-key or bearer`.
- `provider.<name>.thinking = "<value>"; want one of off, low, medium, high`.
- `provider.<name>.base_url: <reason>`. The reason says which rule the
  address breaks, for example `base_url scheme must be https or http, got
  "ftp"` or `http is allowed only on localhost or loopback addresses`. The
  rules are in the Valid values column of `provider.<name>.base_url` in
  [configuration.md](configuration.md#provider).
- `policy.default_command_timeout_seconds = <n>; want a positive number`.
- `context.max_project_context_bytes = <n>; want zero or more`.

`<name>` in these messages is the endpoint's name, such as `opencode`.
`<value>` and `<n>` are the setting's value as you wrote it.

```text
ok  config: settings valid
ok  config: settings valid; project file not read (no workspace)
```

`settings valid` means the files parsed and the combined settings pass the
checks. Doctor cannot tell whether a given file was found, so the line does
not say. The `project file not read` form appears when no workspace was found, so the project
file was never read.

Config warnings appear in doctor's report. They do not appear in the program
unless you turn on the debug log. See [Debug log](configuration.md#debug-log).

---

## key

The key check reads the variable named by `api_key_env` for the default
endpoint. It prints the variable's name, and never the key itself.

```text
ok  key: endpoint <endpoint>: key found in <variable>
```

`<endpoint>` is the endpoint, `opencode` by default. `<variable>` is the
variable that supplied the key. It is `KIRSCH_<NAME>` when that variable is set,
and `<NAME>` otherwise. `<NAME>` is the endpoint's `api_key_env` value, which is
`OPENCODE_API_KEY` for the default endpoint.

```text
warn  key: endpoint <endpoint>: no key found (set KIRSCH_<NAME> or <NAME>); kirsch starts but cannot send a prompt
```

This is the line from the example report above, with `<endpoint>` as
`opencode` and `<NAME>` as `OPENCODE_API_KEY`. Kirsch starts, but it cannot send a prompt. The error you
see when you send one is described under [No API key](#no-api-key). Set the
variable as described there.

```text
warn  key: endpoint <endpoint>: api_key_env does not look like an environment variable name (value withheld); fix the setting
```

This appears when `api_key_env` holds something that is not a variable name.
A common cause is a key pasted into the setting by mistake. Doctor withholds
the value, so the line does not show it. Set `api_key_env` to a variable name
such as `OPENCODE_API_KEY`. If the value you pasted was a real key, treat that
key as exposed, and replace it at the provider. A settings file may have
been committed or shared.

```text
warn  key: skipped: config did not load
```

The config did not load, so there is no endpoint to check. Fix the `config`
line first.

---

## backend

```text
ok  backend: search_code uses ripgrep
```

Kirsch's `search_code` tool, which the model uses to search your code, is
using ripgrep. Nothing to do.

```text
warn  backend: search_code uses pure-go (rg is not on PATH)
```

`search_code` is using Kirsch's built-in search, because ripgrep is not on
your `PATH`. Search works. See [Missing ripgrep](#missing-ripgrep).

This line is never skipped. It does not read the config.

---

## model

```text
ok  model: endpoint <endpoint>, model <id>: context window <tokens> tokens
```

`<endpoint>` is the endpoint's name. `<id>` is the model's name as Kirsch's model table records it. That can
differ from the name in your config. `<tokens>` is the context window Kirsch
assumes for the model. Nothing to do.

```text
warn  model: endpoint <endpoint>, model <model>: not in the table; conservative fallback, context window <tokens> tokens
```

`<model>` is the model name from your config. Kirsch does not know this
model, so it uses a conservative context window. The program shows the same
situation as `Unknown model: cost display unavailable; conservative budget in use.`
The warning does not stop Kirsch. Check the model name in your config for a typo. If the name is correct, the warning remains.

```text
warn  model: skipped: config did not load
```

The config did not load, so there is no model to check. Fix the `config` line
first.

---

## context

The context check looks for the project files Kirsch reads into the model's
instructions. The default list is `AGENTS.md` and `CLAUDE.md`. See
[configuration.md](configuration.md#context).

```text
ok  context: <file>, <size> bytes
```

`<file>` is the first entry in `context.project_files` that is an ordinary
file in the workspace. `<size>` is its size on disk in bytes.

```text
ok  context: none; looked for <files>
```

None of the listed files is present. `<files>` is the list from
`context.project_files`, separated by commas. Kirsch runs without project
context. This is not a warning.

```text
ok  context: none; no project files configured
```

`context.project_files` is an empty list, so there is nothing to look for.

```text
warn  context: <file>, <size> bytes; <warning>
```

A file was chosen, but a candidate in the list was skipped. `<warning>` is one
from the list below. More than one can appear. Kirsch reads the
chosen file.

- `project context candidate "<name>" refused: <reason>`. Kirsch refuses a
  name that matches a secrets pattern such as `.env`, `*.pem` or `*.key`, a
  path inside `.git` or `.kirsch`, a path that leads outside the workspace, an
  absolute path, and an empty path. `<reason>` says which.
- `project context candidate "<name>" is not a regular file`. The candidate
  is a folder or another special item.

```text
warn  context: a candidate could not be read: <error>
```

Kirsch could not read a candidate, so it chose none. `<error>` is one of
the forms below, and any warnings from above follow it:
`stat project context candidate "<name>": <error>` or
`read project context candidate "<name>": <error>`.

Fix a context problem by correcting the name in `context.project_files`, then
checking that the file exists, is an ordinary file, and sits inside the
workspace. A path in `context.project_files` must be relative to the
workspace root.

```text
warn  context: skipped: config did not load
warn  context: skipped: no workspace
```

The `skipped: config did not load` form means the config did not load. The `skipped: no workspace` form means no workspace
was found, so there is no folder to look in. Fix the `config` or `repository`
line first.

---

## state

The state check looks at the folder where Kirsch saves sessions. That folder
is `session.storage_dir` in your config, and the default is
`~/.local/share/kirsch/sessions`. Kirsch keeps one folder for each project
inside the `sessions` folder, and keeps `index.json` in the folder above it.
The check looks at the project's folder, then at the folder above `sessions`
and its `index.json` file. See [configuration.md](configuration.md#session).

The `ok` lines say the folder is usable:

```text
ok  state: <dir> exists and is writable
ok  state: <dir> does not exist yet; kirsch will create it (<ancestor> is writable)
```

`<dir>` is the project's session folder. `<ancestor>` is the nearest folder
above it that already exists. A first run usually prints this form,
because Kirsch has not created the folder for that project. That is
normal.

The `warn` lines say the check could not be made on this system, or that
the folder has not been made. Kirsch starts.

```text
warn  state: <dir> exists; writability not checked on this platform
warn  state: <dir> does not exist yet; writability of <ancestor> not checked on this platform
warn  state: <folder> exists; writability not checked on this platform
warn  state: <index> exists; readability not checked on this platform
```

`<folder>` is the folder above `sessions`. `<index>` is `index.json` inside it.

The state check can also print these lines, when an earlier check left it
nothing to read:

```text
warn  state: skipped: config did not load
warn  state: skipped: no workspace
```

Fix the `config` line, or the `repository` line, first.

The `fail` lines are the ones to fix. Each one ends with a reason Kirsch
cannot start, or cannot create a new session:

```text
fail  state: cannot work out the state folder: home directory unknown; kirsch cannot start until this is fixed
```

Kirsch cannot find your home folder, and the session folder is under it.
Check that the `HOME` variable is set in the shell you start Kirsch from.
The `lock` line repeats the first part of the message, as a warning.

```text
fail  state: <dir> is not a folder; kirsch cannot start until this is fixed
```

Something that is not a folder sits where the session folder should be.
Rename that file, so Kirsch can make the folder.

```text
fail  state: <dir> is not writable; kirsch cannot create a new session, so any start that needs one fails until this is fixed
```

You do not have write access to the session folder. Give yourself write access
to it, or change `session.storage_dir` to a folder you can write to.

```text
fail  state: cannot read <dir>: <error>; kirsch cannot start until this is fixed
```

Kirsch cannot read the session folder. `<error>` is the operating system's
message, with the path left out. Check the folder's permissions.

```text
fail  state: <dir> does not exist and <ancestor> is not a folder; kirsch cannot start until this is fixed
fail  state: <dir> does not exist and <ancestor> is not writable; kirsch cannot start until this is fixed
fail  state: <dir> does not exist and <ancestor> cannot be read: <error>; kirsch cannot start until this is fixed
```

The session folder does not exist, and Kirsch cannot create it, because the
nearest folder above it cannot take a new folder.
Fix that folder, or change `session.storage_dir`.

```text
fail  state: cannot read <folder>: <error>; kirsch cannot start until this is fixed
fail  state: <folder> is not a folder; kirsch cannot start until this is fixed
fail  state: <folder> is not writable, so index.json cannot be updated; kirsch cannot create a new session, so any start that needs one fails until this is fixed
fail  state: cannot read <index>: <error>; kirsch cannot start until this is fixed
fail  state: <index> is a folder, not a file; kirsch cannot start until this is fixed
fail  state: <index> is not readable; kirsch cannot start until this is fixed
```

These cover the storage root and its `index.json` file, which Kirsch reads
and writes when it starts a session. Fix the named folder or file: give it the
permissions it needs, or move the item that is in its way. Rename rather
than delete a file you are unsure of.

---

## lock

The lock check asks whether another Kirsch is running in the same project.
Kirsch guards each project with a lock, so a second copy starts
with fewer features. See [Lock held](#lock-held) below.

```text
ok  lock: free
ok  lock: free (no lock file yet)
```

No other Kirsch holds the lock for this project. The `(no lock file yet)` form
appears before the lock file has been created. Each is normal.

```text
warn  lock: held by another kirsch in this workspace; a new start would skip resume and session grants
```

Another Kirsch is running in this project. A new start would skip resume and
session grants. Doctor does not disturb the other copy. See
[Lock held](#lock-held).

```text
warn  lock: free, but <lockPath> is not writable; the single-instance guard will be off
```

`<lockPath>` is the `.lock` file in the project's session folder. Kirsch can
take the lock only if it can write to that file. Otherwise it runs without
the single-instance guard. Give yourself write access to the file.

```text
warn  lock: could not check the lock: <error>
warn  lock: not checked on this platform
warn  lock: skipped: config did not load
warn  lock: skipped: no workspace
```

These mean the lock could not be checked, so the state is unknown. The
`could not check` form gives the operating system's message. The `skipped` forms have the
same meaning as on the other checks above.

---

## Common failures

### No API key

**What you see.** Kirsch opens, but it cannot send a prompt. When you send
one, an error card titled `auth` reads:

```text
no API key for endpoint "opencode" (set KIRSCH_OPENCODE_API_KEY or OPENCODE_API_KEY)
```

On some starts a notice in the transcript area also explains this. It shows
only while the transcript is empty, so it does not appear when Kirsch has
resumed an earlier session or has already printed a notice such as `no
previous session — starting fresh`.

**What to do.** Set the variable in the shell you start Kirsch from. Use the
bare name, or the prefixed one if you already use it for other tools:

```bash
export OPENCODE_API_KEY="<your-opencode-key>"
```

[install.md](install.md#set-up-your-api-key) has the details. Run
`kirsch doctor` to confirm: the `key` line should read `key found in`.

### Not in a repository

**What you see.** When you start Kirsch outside a Git repository, and without
`--workspace`, it stops before the screen opens. Kirsch prints its own message
after the program name:

```text
kirsch: Kirsch runs inside a Git repository.

Either cd into a repository, or point Kirsch at a directory explicitly:
    kirsch --workspace /path/to/project
```

This is the output of a run from an empty folder. Kirsch exits with status `1`.

`doctor` says the same thing in its own words:
`fail  repository: not in a Git repository` (see [repository](#repository)).

**What to do.** Either `cd` into the repository, or pass the folder with
`--workspace`:

```bash
kirsch --workspace /path/to/project
```

With `--workspace`, a folder that is not a Git repository works. Kirsch
starts with its Git features off. Doctor shows this as a `warn` line for the
`repository` check.

If you are sure you are inside a repository, check that
`git rev-parse --show-toplevel` works there. Kirsch treats any failure of that
command as 'not in a repository'.

### 401 or 403: rejected the API key

**What you see.** A prompt ends in an error that names the endpoint and the
status. Kirsch builds this text for a 401 and for a 403:

```text
provider "<endpoint>" rejected the API key (HTTP <status>); set KIRSCH_<NAME> or <NAME>
```

`<endpoint>` is the endpoint's name, `opencode` by default. `<status>` is
`401` or `403`. `<NAME>` is the endpoint's `api_key_env` value. The program
shows it in an error card titled `turn_error`, with extra text in
front of that message. For the default endpoint the card reads:

```text
agent: model stream: provider_error: auth: provider "opencode" rejected the API key (HTTP 401); set KIRSCH_OPENCODE_API_KEY or OPENCODE_API_KEY
```

**Why it can mislead.** The same message appears for a 401 and for a 403, whatever
the cause. The key is not always the problem. The endpoint may be
wrong, or the account may not have permission to use the model. The message
gives no way to tell these apart. So check in this order:

1. **The key.** Run `kirsch doctor`, and check the `key` line. Then confirm
   the variable is set in the shell you start Kirsch from, and that the value
   is the key you meant.
2. **The endpoint.** Check `base_url` and `auth` for the endpoint in your
   config. See [configuration.md](configuration.md#key-reference).
3. **The model.** Check the `model` setting. Your account may not allow the
   model you named.

Doctor cannot show a `401` or `403`, because it makes no network calls and
never sends a prompt.

### Missing ripgrep

**What happens.** Without ripgrep, `search_code` uses Kirsch's own search.
Search works. If ripgrep is on your `PATH` but fails during a search,
Kirsch also falls back to its own search, rather than failing the tool.

**What doctor shows.** Warnings, which do not fail doctor:

```text
warn  rg: not found on PATH; search_code uses the pure-Go fallback
warn  backend: search_code uses pure-go (rg is not on PATH)
```

These are the lines from the `rg` and `backend` checks.

**What to do.** Install ripgrep with your system's package manager. See
[install.md](install.md#ripgrep-optional).

### Lock held

**What you see.** A second Kirsch in the same project does not resume the
last session, and it turns off session grants. The program shows these
messages. Each line reads as follows:

- In the status bar, as a persistent warning: `another Kirsch is running here`
- When it would resume on launch: `auto-resume skipped — another Kirsch instance is running`
  (the notice that starts a new session in its place)
- When you run `kirsch resume`: `resume skipped — another Kirsch instance is running`

Session grants are switched off in this copy. The policy setting that
controls them is explained in
[configuration.md](configuration.md#policy).

If the lock cannot be taken for another reason, the status bar shows
`lock unavailable — single-instance guard off`.
In this case the guard is off, and Kirsch runs as normal.

**What to do.** If you want resume and grants in this project, close the
other Kirsch and start again. If you see the warning and expected no other
Kirsch to be running, look for one in another terminal. Running
`kirsch doctor` does not take the lock from a copy that is already running.

---

## If the report does not explain it

Run `kirsch doctor` and read the `fail` lines first. Fix each one, then run
doctor again. A `warn` line does not stop Kirsch, but it often explains
behaviour that looks wrong.

Config and project-context warnings do not show in the program. To see them,
turn on the debug log for one run with `kirsch --debug`, or with
`KIRSCH_DEBUG=1`. The log is written under `$XDG_STATE_HOME/kirsch/debug.log`,
or `~/.local/state/kirsch/debug.log`. See
[Debug log](configuration.md#debug-log).
