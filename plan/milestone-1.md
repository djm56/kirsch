# Milestone 1 — Instruction Set

> **Status: Ready for execution — blocked on Milestone 0.** This is the
> complete, ordered instruction set for Milestone 1 (workspace engine +
> read-only tools). Execute tasks in order. Nothing in this milestone touches
> an LLM, writes to a file, or executes a subprocess other than `git` and `rg`.

## Ground Rules (read first)

1. **Milestone 0 must be complete first.** Every box in
   [`milestone-0.md`](milestone-0.md) ticked, CI green.
2. **No writes, no arbitrary execution.** This milestone reads. `apply_patch`
   and `run_command` belong to Milestone 2 and must not appear, not even
   stubbed. The only subprocesses permitted are `git` (read-only subcommands)
   and `rg`.
3. **No provider, agent, session, or policy packages.** See the package build
   order table in plan §2.
4. **Locked decisions** (do not re-litigate):
   - Path denylist is enforced in `internal/workspace` in **this** milestone,
     not M2 (plan §8 M1, amendment 16b). A milestone that ships `read_file`
     without it ships a tool that can read `.env`.
   - `internal/app` is created **here**, not M3. The TUI must never import
     `internal/tool` (plan §2 hard rules).
   - Tool results use the envelope in plan §3 exactly — no per-tool shapes.
   - Every path argument of every tool goes through `workspace.Resolve`. No
     exceptions, no "this one is safe".
5. Every task has a checkable result. Do not move on with the previous one
   failing.

## Decisions Needing Owner Sign-off (before Task 5)

Two dependency choices this milestone cannot avoid. Flag them, don't guess:

- **`.gitignore` matching.** Recommended:
  `github.com/go-git/go-git/v5/plumbing/format/gitignore` — correct handling of
  nested ignore files, negation, anchoring and `**`, which a hand-rolled
  matcher gets wrong in ways that silently leak files into model context. Cost:
  a large module in `go.mod` (only the imported subpackage is compiled).
  Alternative: `github.com/sabhiram/go-gitignore` (tiny, but no nested-file
  semantics).
- **`vendor/` in Go projects.** Plan §3 ignores `vendor/` unconditionally.
  For a Go module that vendors deps this hides real, relevant source. Proposed:
  keep ignoring it by default, make it overridable later. Confirm.

---

## Task 1 — Test fixtures

Build these **first**; every later task is tested against them. All are plain
committed directories under `testdata/`.

### 1.1 Three gotchas to handle before writing any fixture

1. **Fixtures are not Git repos.** A nested `.git` directory cannot be
   committed. So `testdata/repo-*` are plain directories, and
   `git rev-parse --show-toplevel` will not work inside them. Root detection is
   therefore tested against a temporary repo created with `t.TempDir()` +
   `git init`; the fixtures are used with an explicit root override.
2. **The repo's own `.gitignore` will eat the fixtures.** A rule like
   `node_modules/` also matches `testdata/repo-node/node_modules/`, and Git
   cannot re-include a file whose parent directory is excluded. Write the
   project's ignore rules **root-anchored** — `/node_modules/`, `/vendor/` —
   so they cannot reach into `testdata/`. Verify with
   `git check-ignore -v testdata/repo-node/node_modules/left-pad/index.js`
   (must report no match).
3. **Symlinks must be relative** so they resolve identically on every machine,
   and the tests must skip with a clear message (not fail obscurely) if a
   checkout materialised them as plain files.

### 1.2 `testdata/repo-small`

General-purpose fixture for walking, reading, searching.

```
.gitignore                 -> "ignored/\n*.log\n"
README.md
main.go
pkg/util/strings.go
pkg/util/strings_test.go
pkg/util/.gitignore        -> "scratch.txt"        (nested ignore file)
pkg/util/scratch.txt       (ignored by the nested file)
deep/a/b/c/d/deep.txt      (depth 5 — exercises max_depth)
ignored/secret.txt         (ignored by root .gitignore)
app.log                    (ignored by pattern)
.env                       (denylist target)
.hidden.txt                (exercises include_hidden)
crlf.txt                   (CRLF line endings)
no-trailing-newline.txt
unicode.md                 (East Asian text + emoji — rune-width rendering)
binary.dat                 (contains NUL bytes — binary detection)
```

Do **not** commit a >2MB file for the size-cap test; generate it in
`t.TempDir()` at test time.

### 1.3 `testdata/repo-symlink-escape`

The allowed cases matter as much as the blocked ones — a containment check that
blocks legitimate internal symlinks is also a bug.

```
inside.txt
sub/target.txt
link-ok        -> sub/target.txt          MUST BE ALLOWED
link-dir-ok    -> sub                     MUST BE ALLOWED
link-escape    -> ../../../etc/passwd     workspace_violation
link-parent    -> ..                      workspace_violation
link-loop      -> link-loop               workspace_violation, must not hang
link-dangling  -> nowhere.txt             file_not_found, NOT a violation
```

### 1.4 Remaining fixtures

- **`testdata/repo-wordpress-plugin`** — `my-plugin.php` with a
  `Plugin Name: My Plugin` header comment, `includes/class-thing.php`,
  `vendor/autoload.php` (must be ignored by the walker).
- **`testdata/repo-go-module`** — `go.mod` (`module example.com/calc`),
  `calc/divide.go` with a `Divide(a, b float64) (float64, error)` that has **no
  input validation** (Milestone 4's smoke test adds it), `calc/divide_test.go`.
  Go tooling ignores `testdata/`, so a nested `go.mod` will not break
  `go build ./...` — confirm this rather than assuming it.
- **`testdata/repo-node`** — `package.json` with a `test` script, `index.js`,
  `node_modules/left-pad/index.js` (must be ignored).
- **`testdata/repo-prompt-injection`** — normal `README.md` plus
  `src/helper.php` whose comment claims the user has pre-approved a dangerous
  action. Nothing in M1 acts on it; the fixture exists so §9.7's test has a
  target from M3 onward. Reading it must behave like reading any other file.
- **`testdata/detect/`** — loose files for project detection:
  `theme/style.css` (`Theme Name: My Theme`), `ambiguous/` (both `go.mod` and
  `package.json`, to pin precedence), `none/readme.txt`.

**Check:** `git status` clean after `git add testdata/`; every intended file is
tracked (`git ls-files testdata | wc -l` matches the fixture inventory);
symlinks report as symlinks (`git ls-files -s testdata | grep 120000`).

## Task 2 — `internal/config`

1. Dependency: `github.com/BurntSushi/toml`, chosen because
   `MetaData.Undecoded()` gives unknown-key reporting for free.
2. `config.Load(opts)` takes explicit file paths — **no reading of `$HOME` from
   inside the loader**, so tests never touch the real user environment.
3. Path resolution (done by the caller, not the loader):
   global `$XDG_CONFIG_HOME/kirsch/config.toml`, falling back to
   `~/.config/kirsch/config.toml`; project `<workspace>/.kirsch/config.toml`.
4. Precedence: **defaults → global → project → flags**, merged **per key**. A
   project file setting one key must not blank the global's other keys — the
   naive "unmarshal over the struct" approach does exactly that for nested
   tables, so verify it.
5. A missing config file is not an error. Every key has a default.
6. Unknown keys produce a warning naming the key and the file. They never fail
   startup — a config written for a newer Kirsch must still boot an older one.
7. **Secret refusal** (plan §5): if any key name matches
   `(?i)(api_?key|token|secret|password)` and carries a value, fail with a
   clear error naming the file and the key. `api_key_env` is exempt — it holds
   a variable *name*, not a value. Test both sides of that line.
8. `~` expansion for path-valued keys (`session.storage_dir`).
9. Parse and validate the `[provider]`, `[context]` and `[session]` blocks even
   though nothing consumes them until M3/M4 — a key that silently does nothing
   is worse than one that does not exist.

**Check:** `go test ./internal/config/...` green, including: precedence across
all four layers; per-key merge; unknown key warns but loads; every secret-shaped
key rejected; `api_key_env` accepted; missing files fine.

## Task 3 — `internal/telemetry`

1. `log/slog` with a `JSONHandler`.
2. Destination: `$XDG_STATE_HOME/kirsch/debug.log`, falling back to
   `~/.local/state/kirsch/debug.log`.
3. **Never stdout or stderr.** Anything written there corrupts the TUI's
   rendering. This is enforced by a test that captures `os.Stdout`/`os.Stderr`
   during a logging burst and asserts both are empty.
4. Enabled by `--debug`, `KIRSCH_DEBUG=1`, or `[telemetry].debug_log`. When
   disabled, the handler discards without formatting — no cost on the hot path.
5. Redaction rules, enforced at the logging helper rather than trusted to call
   sites: never log environment variable *values*; never log more than 256
   bytes of file content; never log a full tool `Content` field.
6. No rotation in v0.1. At 10MB, truncate and log a single warning.
7. Token/cost tracking is **not** built here — that arrives in M4.

**Check:** `go test ./internal/telemetry/...` green; with `--debug` the log file
is created and contains structured lines; without it no file is created; the
stdout/stderr test passes; a redaction test proves env values never appear.

## Task 4 — `internal/workspace`: root and containment

The security core of the whole project. Build it before anything that reads a
file.

### 4.1 Root detection

`workspace.Detect(flagWorkspace string) (*Workspace, error)`

- `--workspace` given → use it. Must exist and be a directory. No Git required.
- Otherwise `git rev-parse --show-toplevel` from the cwd.
- Neither → a clear onboarding error ("Kirsch runs inside a Git repository; use
  `--workspace <dir>` to override"), not a panic or a bare exit code.
- Store **both** `Root` (as supplied) and `CanonicalRoot`
  (`filepath.EvalSymlinks`). On macOS `/tmp` is a symlink to `/private/tmp`, so
  a containment check against a non-canonical root fails for every test that
  uses `t.TempDir()`. This is not a detail to discover later.

### 4.2 `Resolve` — the containment check

`(*Workspace) Resolve(rel string) (string, error)`

1. Empty path → `tool_input_invalid`.
2. **Absolute paths are rejected outright** — tools accept workspace-relative
   paths only.
3. `filepath.Clean(filepath.Join(root, rel))`.
4. `EvalSymlinks` fails on a path that does not exist, but `read_file` of a
   missing file must return `file_not_found`, not a containment error. So: walk
   up to the deepest **existing** ancestor, canonicalise that, then re-append
   the non-existent tail and check the result.
5. Containment test:
   `resolved == canonicalRoot || strings.HasPrefix(resolved, canonicalRoot + string(filepath.Separator))`.
   **The trailing separator is load-bearing** — without it
   `/home/me/project-evil` passes a prefix check against `/home/me/project`.
   Write the test for that case explicitly.
6. Symlink cycles surface as `ELOOP` from `EvalSymlinks` → `workspace_violation`.
   Never loop, never hang.

### 4.3 Path denylist

Enforced here, in `Resolve`, so no tool can forget it:
`.env`, `.env.*`, `*.pem`, `*.key`, `.git/**`, `.kirsch/**` → `workspace_violation`.

Matched against the workspace-relative path after cleaning, so
`./foo/../.env` is caught. A denylist checked before normalisation is not a
denylist.

### 4.4 Repository metadata

`Branch()`, `IsDirty()` via `git rev-parse --abbrev-ref HEAD` and
`git status --porcelain`. Cached, refreshed on demand — never shell out on
every TUI render. A non-Git workspace returns an empty branch and the header
omits it rather than showing an error.

**Check:** `go test ./internal/workspace/...` green, covering every row of the
`repo-symlink-escape` table in Task 1.3 (allowed cases included), the
`project-evil` prefix case, every denylist entry, and root detection against a
temporary `git init` repo.

## Task 5 — `internal/workspace`: ignore rules and project detection

1. **Built-in ignore, always applied and not overridable by a `.gitignore`
   negation:** `.git`, `.kirsch`, `node_modules`, `vendor`, `.DS_Store`.
2. **`.gitignore` support** using the library signed off above. Must handle
   nested ignore files, `!` negation, directory-only trailing `/`, `**`, and
   anchored vs floating patterns. Deeper files win.
3. **Walker:** `Walk(rel string, maxDepth int, includeHidden bool, fn)`.
   - Depth is relative to the start path, not the root.
   - **Symlinked directories are not followed** — this prevents both cycles and
     escapes, and is simpler than trying to detect them mid-walk.
   - Entries are returned in lexical order. Deterministic ordering matters:
     non-deterministic tool output makes both golden tests and model behaviour
     unstable.
4. **Project detection**, root-level only (bounded cost), returning an ordered
   slice with the primary type first:
   1. `Plugin Name:` in the first 8KB of any root-level `*.php` →
      `wordpress-plugin`
   2. `Theme Name:` in root `style.css` → `wordpress-theme`
   3. root `go.mod` → `go`
   4. root `package.json` → `node`

   Multiple matches are normal (a WordPress plugin with a `package.json` for
   its build step); record all of them, and pin the precedence with the
   `detect/ambiguous/` fixture.

**Check:** walking `repo-small` returns exactly the expected file set at each
`max_depth` and `include_hidden` combination; `ignored/`, `app.log`,
`pkg/util/scratch.txt`, `node_modules/` and `vendor/` never appear; each
`repo-*` fixture detects its expected project type; `detect/ambiguous/` returns
both types in the documented order.

## Task 6 — `internal/tool`: envelope and registry

```go
type Result struct {
    OK             bool
    Content        string   // sent to the model
    DisplaySummary string   // collapsed card in the UI
    Truncated      bool
    Error          *Error
    DurationMS     int64
}

type Tool interface {
    Name() string
    Schema() json.RawMessage
    Invoke(ctx context.Context, raw json.RawMessage) Result
}
```

1. `Invoke` **never returns an error and never panics.** The registry wrapper
   recovers panics and converts them to a result with an internal error kind.
   One malformed tool must not take down the TUI — prove it with a deliberately
   panicking fake tool in the tests.
2. `DurationMS` is measured by the registry wrapper, not by each tool.
3. Input decoding uses a typed struct with `DisallowUnknownFields`. On failure,
   return `tool_input_invalid` with a message naming the offending field — this
   text is what lets the model self-correct from M3, so write it for a reader.
4. Error kinds are the full plan §3.3 set. Kinds relevant to M1:
   `workspace_violation`, `tool_input_invalid`, `file_not_found`,
   `file_too_large`, `binary_file`, `cancelled`.
5. `Truncate(s string, maxBytes int) (string, bool)` cuts on a line boundary
   where possible and a rune boundary always — never mid-rune, which corrupts
   the terminal. For large outputs keep head and tail with the middle elided.
6. **Write the JSON schemas now**, even though nothing reads them until M3.
   They are the tool contract; retrofitting them later means retrofitting
   validation too.
7. Registry: `Register`, `Get`, `List` — `List` in deterministic order.

**Check:** `go test ./internal/tool/...` green, including the panicking-tool
recovery, unknown-field rejection, and rune-boundary truncation of a
multi-byte string.

## Task 7 — The five read-only tools

All five take their paths through `workspace.Resolve`. Implement per plan §3.

| Tool | Specifics |
|---|---|
| `read_file` | `path` required; optional `start_line`/`end_line`, 1-based inclusive. Default cap 500 lines; 2MB file limit → `file_too_large`; NUL byte in the first 8KB → `binary_file`; missing → `file_not_found`. Output is **line-numbered** (`N\t` prefix) — the model needs stable line references. |
| `list_files` | `path` default `"."`, `max_depth` default 2, `include_hidden` default false. Respects ignore rules. Caps at 1000 entries, then `Truncated`. Summary: path + `n files`. |
| `search_code` | `query` required; `regex` bool (default false = literal); optional `glob`; `max_results` default 50. Results as `path:line: text`, match text trimmed to 200 chars, sorted by path then line. |
| `git_status` | No input. `git status --porcelain`. A non-Git workspace returns `OK: false` with a clear message — not a crash. |
| `git_diff` | `staged` bool, optional `path`. 200KB cap with the standard truncation marker. |

**`search_code` backend parity (plan §3.4).** `rg --json` when `rg` is on
`PATH`, pure-Go fallback otherwise. The two must agree on ignore semantics and
result ordering. Write the differential test now, while both implementations
are fresh — a divergence found in M4 is a debugging afternoon.

**Check:** table-driven test over the registry asserting **every** path-taking
tool rejects `../escape` and every denylist entry; each tool's envelope shape
matches plan §3 exactly; the `rg` / fallback differential test passes on
`repo-small`; `read_file` on `binary.dat` returns `binary_file`, on a
3MB temp file returns `file_too_large`, on `crlf.txt` preserves CRLF.

## Task 8 — `internal/app`: the wiring layer

Created here, not M3, because the TUI must never import `internal/tool`.

1. `app.App` owns the root context and holds the workspace, config, telemetry
   handle, tool registry, and the `tea.Program` handle.
2. Typed messages to the TUI — `WorkspaceInfoMsg`, `ToolStartedMsg`,
   `ToolCompletedMsg`, `ErrorMsg`. The TUI imports **only** these types.
3. Tools run on their own goroutine; results reach the TUI **only** via
   `program.Send` (plan §2 hard rule).
4. Context hierarchy from plan §4: `rootCtx` → `sessionCtx` → `turnCtx`. There
   is no model call yet, but wire `turnCtx` now and exercise cancellation with
   a deliberately slow tool — cancellation retrofitted later is cancellation
   that does not work.
5. No agent, no provider, no session store.

**Check:** cancelling a slow tool via `Esc` returns control to the composer
within 1s and the card renders `⊘ cancelled`; no goroutine leak (`go test
-race`, and a leak check around the app lifecycle).

## Task 9 — TUI wired to real tools

1. Header shows the **real** project name, branch, and dirty marker.
2. Tool cards render real envelopes: summary line, `Enter` to expand,
   truncation markers, error kinds as error cards (ui-spec §3.3, §3.6).
3. **Temporary debug slash commands**, this milestone only:
   `/read <path>`, `/ls [path]`, `/search <query>`, `/gitstatus`, `/gitdiff`.
   They are scaffolding to satisfy the M1 acceptance criterion and are removed
   in M3 when the model drives tools. Label them `(debug)` in `/help` so nobody
   mistakes them for product surface.
4. `fake.go` stays, but only as fixture data for golden tests — the live path
   now runs on app messages.
5. Golden tests updated for real card content. Real envelopes must render in
   the **same shape** as the M0 grids in
   [`kirsch-ui-screens.md`](kirsch-ui-screens.md) — screen 02 for a running
   and a completed card, 07 for the 200-line cap, 08 for error cards and
   system notices. Real data changes the strings inside a card, never its
   structure; if a real envelope will not fit that shape, the screen reference
   is wrong and gets updated in this milestone's commit, not worked around in
   the renderer.

**Check:** from a real repository, `/search` and `/read` return correct results
rendered as tool cards; a denylisted path renders an error card reading
`workspace_violation`, not a panic or an empty card.

## Task 10 — Import-rule CI check

Plan §2's dependency rules are enforced by a test, not by code review.

1. A test that shells out to
   `go list -f '{{.ImportPath}} {{join .Imports " "}}' ./...` and asserts:
   - `internal/agent` imports none of `internal/tui`, `internal/provider`,
     `internal/tool`
   - `internal/tui` imports none of `internal/provider`, `internal/tool`,
     `internal/workspace`
2. `agent` and `provider` do not exist yet. The test must pass vacuously and
   still be in place — it is there to catch the violation on the day the
   package appears, which is exactly when nobody is thinking about it.
3. Add it to the CI `test` job.

**Check:** the test passes; adding a deliberate illegal import to
`internal/tui` makes it fail with a message naming the offending edge; remove
the deliberate import afterwards.

## Task 11 — Tests

1. **Unit:** path containment (every Task 1.3 row), denylist, config
   precedence, ignore matching, truncation, project detection.
2. **Tool integration:** every tool against the fixtures, asserting the exact
   envelope shape. No model involved.
3. **Differential:** `rg` backend vs pure-Go fallback on `repo-small`.
4. **TUI:** golden snapshots updated; synthetic key tests for the debug
   commands and for cancelling a slow tool.
5. **Architecture:** the Task 10 import-rule check.
6. `go test -race ./...` green — the app introduces the first real concurrency.

**Check:** `go test -race ./...` green; `gofmt -l .` empty; `go vet` and
`golangci-lint` clean; CI green.

## Task 12 — Final acceptance

- [ ] All six `testdata/` fixtures committed; `git check-ignore` confirms none
      are excluded by the project's own ignore rules; symlinks tracked as
      symlinks
- [ ] `internal/config` loads with full four-layer precedence; secret-shaped
      keys refused; unknown keys warn without failing
- [ ] `internal/telemetry` writes structured debug logs to file and **never**
      to stdout/stderr; redaction proven by test
- [ ] Workspace root detected via Git, overridable with `--workspace`;
      canonical root resolved (macOS `/tmp` case covered)
- [ ] Every symlink-escape row returns `workspace_violation`; every legitimate
      internal symlink is **allowed**; the dangling link returns
      `file_not_found`; the loop does not hang
- [ ] The `project-evil` prefix case is blocked
- [ ] Every path-denylist entry blocked, through every path-taking tool
- [ ] `.gitignore` (including nested files and negation) plus the built-in list
      respected by the walker; ordering deterministic
- [ ] Project type detected for all four fixture types; ambiguous case returns
      the documented precedence
- [ ] All five read-only tools implemented, returning the plan §3 envelope
      exactly, with correct error kinds
- [ ] `rg` and pure-Go `search_code` backends return identical results
- [ ] `internal/app` wires TUI ↔ tools; TUI imports no tool/workspace package
- [ ] Cancelling a slow tool returns within 1s and renders `⊘ cancelled`
- [ ] Debug slash commands read and search a real repository from inside the
      TUI, rendered as tool cards
- [ ] Import-rule check in place and enforced in CI
- [ ] `go test -race ./...`, `gofmt`, `go vet`, `golangci-lint` all clean; CI
      green
- [ ] **No patch, command-execution, provider, agent, session, or policy code
      exists anywhere in the repo**

**Definition of done:** every box checked, CHANGELOG updated, `plan/README.md`
progress table set to `☑ Complete`, and a commit or PR tagged so Milestone 2
starts from a known point.

---

## Explicitly Out of Scope (reminder)

No `apply_patch`, no `run_command`, no diff parsing, no command allowlist
(the *path* denylist is in scope; the *command* allowlist is not), no provider
or agent, no session store, no compaction, no token or cost tracking. If a task
seems to need one of these, stop — check the package build order table in plan
§2 and flag it rather than building ahead.
