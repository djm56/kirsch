# Security testing

What is scanned, what it has found, and how to triage what it reports.

```bash
npm run tools        # install the tooling (one-off)
npm run security     # vulnerabilities + static analysis + secret scan
npm run fuzz         # property-based testing of the untrusted-input surfaces
```

All of it runs in CI on every push. None of it needs a network at test time
beyond fetching the vulnerability database.

## Why this matters more here than usual

Kirsch reads repositories it did not write, on behalf of a language model, and
from Milestone 2 it will change files and run commands. Three consequences shape
the whole approach:

1. **Every byte of tool output is attacker-controlled.** A file's contents, a
   command's stdout, a search hit — all of it can be authored by whoever wrote
   the repository. ADR 0007 states the rule; the sanitiser and its fuzz targets
   enforce it.
2. **Path containment is the security model.** Not a defence in depth, *the*
   defence. Anything that weakens `workspace.Resolve` weakens everything.
3. **The user's secrets are in scope.** `.env` files, private keys and API
   tokens live in the repositories Kirsch reads. The path denylist, the config
   loader's refusal to accept credentials, and the debug log's redaction are all
   the same concern from different angles.

## The tools

| Tool | What it catches | Command |
|---|---|---|
| `govulncheck` | Known CVEs in dependencies and the standard library, with reachability analysis | `npm run security:vuln` |
| `gosec` | Static analysis: traversal, injection, weak permissions, hardcoded credentials | `npm run security:sast` |
| `secret-scan.sh` | Credentials committed to the repository | `npm run security:secrets` |
| `osv-scanner` | Dependency scanning against the OSV database | `npm run security:deps` |
| `go test -fuzz` | Property violations on untrusted input | `npm run fuzz` |

`govulncheck` is the one to trust most. It does **reachability** analysis rather
than matching version numbers, so it reports what this code actually calls
instead of everything in the dependency tree — which is the difference between a
finding and a wall of noise.

## What they have actually found

Not hypothetical. Every item below was a real defect in this codebase, and none
was found by reading the code.

### The path denylist was bypassable by changing case

`FuzzResolveNeverEscapes`, within seconds. `read_file(".ENV")` was permitted,
and on macOS and Windows — where filesystems are case-insensitive by default —
`.ENV` opens the same bytes as `.env`. The denylist was defeated on two of three
platforms by pressing shift.

Verified on the development machine before fixing: `.ENV` returned the contents
of `.env`.

**Why review missed it.** The hand-written table of denied paths encoded exactly
the same assumption the code did. Fuzzing asserts a *property* — whatever
`Resolve` returns is inside the workspace and not denylisted — rather than a
list of cases, so it was not constrained by what anyone had thought of. Plan §11
amendment 51.

### A `.gitignore` that is itself a symlink escaped the workspace

`gosec` G122. `filepath.WalkDir` does not follow symlinks while traversing, but
`os.ReadFile` follows them when opening — so a file literally named `.gitignore`
pointing outside the root was read and parsed as ignore rules. There was a
TOCTOU window besides: the entry could be swapped between the walk seeing it and
the read opening it.

Now read through `os.Root`, so the kernel enforces containment rather than this
package remembering to. Amendment 53.

### Two containment escapes in the standard library

`govulncheck`. First `GO-2026-4602` ("FileInfo can escape from a Root in `os`").
Then, *after* adopting `os.Root` for the fix above, `GO-2026-4970` ("Root escape
via symlink plus trailing slash in `os`").

The second is worth sitting with: hardening with `os.Root` created exposure to
an `os.Root` bug. **A mitigation is code, and code has vulnerabilities.** What
made both visible within minutes is that `govulncheck` runs against every build,
which is why the toolchain floor in `go.mod` is a security decision rather than
a housekeeping one. Amendment 52.

## Trust-boundary cases

These are the Milestone 3 trust boundaries, each with the test that guards it. m3-d2 wrote the config and URL cases below; m3-d6 adds `[context]` confinement, the `project_files` honoured/refused-entry boxes, and the `run_command`-writes-config known limit (ADR 0008).

### 1. Project files are an allowlist

A project file (`.kirsch/config.toml` in the workspace) may set **only** `[context].project_files`. Every other key in `[provider]`, `[policy]`, `[context]`, `[session]`, `[telemetry]`, or any other table is ignored with a warning naming the key, distinct from the unknown-key warning.

**Guarded by:** `internal/config/config_test.go` — `TestProjectFileIsAnAllowlist`, line 86–147.

**What the test asserts:**
- Line 109: Load succeeds despite every non-allowlisted key being present in the project file.
- Lines 115–124: The effective config for `Provider.Default`, `Provider.Endpoints["opencode"]`, `Policy.RequireApprovalForCommands`, `Policy.DefaultCommandTimeoutSeconds`, `Policy.EnvPassthrough`, `Context.MaxProjectContextBytes`, `Session.StorageDir`, and `Telemetry.DebugLog` remain at their defaults, unchanged by the project file.
- Lines 129–143: Every ignored key produces a warning with `Kind == WarnProjectIgnored`, covering `provider.default`, `provider.opencode.base_url`, `policy.require_approval_for_commands`, `policy.env_passthrough`, `context.max_project_context_bytes`, `session.storage_dir`, `telemetry.debug_log`, and `unknown_table`.

### 2. Wrong-typed `project_files` in a project file

A project file with `[context] project_files = "AGENTS.md"` (a string, not a list) starts normally, leaves the effective config unchanged, and produces a warning naming the key.

**Guarded by:** `internal/config/config_test.go` — `TestProjectFilesWrongTypeWarns`, line 159–169, and `TestProjectFilesWrongTypeMessage`, line 426–445.

**What the tests assert:**
- `TestProjectFilesWrongTypeWarns`, line 162: Load succeeds despite wrong type.
- Line 166: `project_files` defaults to `["AGENTS.md", "CLAUDE.md"]` and a warning is produced.
- `TestProjectFilesWrongTypeMessage`, line 441: The warning message says `"must be a list of strings"`, distinct from the project-allowlist message.

### 3. Debug log not enabled from project files

A project file with `[telemetry] debug_log = true` does not enable the debug log.

**Guarded by:** `internal/config/config_test.go` — `TestProjectFileIsAnAllowlist`, line 86–147.

**What the test asserts:**
- Line 122: `cfg.Telemetry.DebugLog` is `false`, even though the project file set it to `true`.

### 4. Credential-shaped keys refused in project files

A credential-shaped key (`api_key = "x"`) in a project file is refused by `checkSecrets`, not merely ignored.

**Guarded by:** `internal/config/config_test.go` — `TestProjectFileCredentialRefused`, line 150–156, and `TestSecretShapedKeysRefused`, line 223–257.

**What the tests assert:**
- `TestProjectFileCredentialRefused`, line 153: Load fails with an error when a project file contains `api_key`.
- `TestSecretShapedKeysRefused`, lines 224–243: Multiple credential-shaped keys (`api_key`, `apikey`, `token`, `secret`, `password`, `API_KEY`) in global config all produce errors.

### 5. URL validation: loopback-only for HTTP

`base_url` validation rejects non-loopback `http://`, `http://localhost@evil.example/`, `http://[::ffff:127.0.0.1]/`, and a hostname that merely resolves to loopback. It accepts `http://localhost`, `http://127.0.0.2`, `http://[::1]`.

**Guarded by:** `internal/config/config_test.go` — `TestValidateBaseURL`, line 172–192, and `TestValidateBaseURLEdgeCases`, line 448–464.

**What the tests assert:**
- `TestValidateBaseURL`, lines 177–181 (refused cases):
  - Line 178: `ValidateBaseURL("http://example.com/v1")` returns an error.
  - Line 178: `ValidateBaseURL("http://localhost@evil.example/")` returns an error.
  - Line 179: `ValidateBaseURL("https://user:pw@example.com/")` returns an error.
  - Line 179: `ValidateBaseURL("http://[::ffff:127.0.0.1]/")` returns an error.
- `TestValidateBaseURL`, lines 173–176 (accepted cases):
  - Line 174: `ValidateBaseURL("http://localhost:8080/v1")` succeeds.
  - Line 175: `ValidateBaseURL("http://127.0.0.1/v1")` succeeds.
  - Line 175: `ValidateBaseURL("http://127.0.0.2:9/v1")` succeeds.
  - Line 175: `ValidateBaseURL("http://[::1]:8/v1")` succeeds.
- `TestValidateBaseURLEdgeCases`, line 450 (accepted):
  - Line 449: `ValidateBaseURL("HTTP://localhost/v1")` succeeds (case-insensitive scheme).
- `TestValidateBaseURLEdgeCases`, lines 450–453 (refused):
  - Line 451: `ValidateBaseURL("http://127.1/")`, `http://127.00.0.1/`, `http://0.0.0.0/`, `http://localhost./` all return errors.
  - Line 452: IPv6 forms `[::ffff:7f00:1]`, `[::]`, and `[0:0:0:0:0:ffff:127.0.0.1]` all return errors.

**Not covered:** A hostname that merely resolves to loopback (e.g., `localtest.me` resolving to `127.0.0.1`) is tested in manual testing at `plan/testing/manual/milestone-3.md`, §2, "Case: Hostname resolving to loopback (refused)", but is not covered by a unit test; it is asserted by the manual walkthrough.

### 6. Redirects are errors; the adapter does not follow them

A test server returning a 3xx makes the adapter return an error naming the status and `Location` host; no second request is sent, and `x-api-key` never reaches the redirect target.

**Guarded by:** `internal/provider/anthropic/client_test.go` — `TestClientRedirectNotFollowed`, line 309–321, and `internal/provider/anthropic/client_fix_test.go` — `TestClientRedirectLocationHost`, line 176–189.

**What the tests assert:**
- `TestClientRedirectNotFollowed`, line 315: A 307 redirect produces an error with `KindRedirect` and `Status == 307`.
- Lines 318–320: The redirect target receives zero requests; the original server receives one.
- `TestClientRedirectLocationHost`, line 185: A 3xx redirect produces an error message that names the `Location` header's host (or a clipped version for long hostnames).

**Not covered:** The tests do not explicitly verify that `x-api-key` never reaches the redirect target through a spy on the redirect request; this is implied by "zero requests to the redirect target" but is not asserted on the request header content itself.

### 7. `[context].project_files` is honoured

A project file may set `[context] project_files = ["AGENTS.md", "CLAUDE.md"]` and the loader treats the list as FIFO: the first existing, resolvable regular file wins; absent candidates are skipped without warning; if none exist there is no project context and no error.

**Guarded by:** `internal/agent/prompt/loader_test.go` — `TestLoadProjectContext_FirstExistingCandidateWins`, line 87–144.

**What the test asserts:**
- Line 97: When both candidates exist, `AGENTS.md` is chosen.
- Line 100: The returned content is exactly the contents of `AGENTS.md`.
- Line 106: No warnings are produced for absent candidates.
- Line 116: After `AGENTS.md` is removed, `CLAUDE.md` wins.
- Lines 132–142: When no candidates exist, the chosen file, content, and size are all empty/zero and no warnings are emitted.

### 8. Refused `project_files` entries produce named warnings

A candidate that `workspace.Resolve` refuses — an absolute path, a `..` traversal, a symlink that escapes the root, or a denylisted path such as `.env`, `.kirsch/`, `.git/`, or a key file — does not stop loading. It produces a warning naming the refused candidate, and the loader continues with the next candidate.

**Guarded by:** `internal/agent/prompt/loader_test.go` — `TestLoadProjectContext_RefusedCandidatesProduceNamedWarnings`, line 149–182; `internal/workspace/workspace_test.go` — `TestSymlinkTable`, line 31–91 (symlink escape and `..` traversal), and `TestDenylist`, line 122–163 (denylisted paths including `.kirsch/config.toml` and `.env`).

**What the tests assert:**
- `TestLoadProjectContext_RefusedCandidatesProduceNamedWarnings`, line 163: The first allowed candidate (`valid.md`) is chosen despite four refused candidates before it.
- Line 166: The returned content is from `valid.md`, not from any refused path.
- Lines 173–180: Every refused candidate (`/abs.md`, `../outside.md`, `escaped-link`, `.env`) appears in exactly one warning.
- `TestSymlinkTable`, lines 55–60: Symlink escape, parent-directory symlink, and plain `..` traversal are all reported as workspace violations.
- `TestDenylist`, lines 136–138: `.kirsch/config.toml` and `.kirsch/sessions/abc.jsonl` are refused; lines 147–162 confirm ordinary project files are still allowed.

### 9. Non-regular `project_files` candidates are skipped with a warning

Directories and FIFOs (or any non-regular file) among the candidate list are skipped before being opened, producing a warning naming the candidate. The regular-file check must precede `Open` so that a FIFO cannot hang the loader.

**Guarded by:** `internal/agent/prompt/loader_test.go` — `TestLoadProjectContext_NonRegularFilesSkippedWithWarning`, line 187–210.

**What the test asserts:**
- Line 198: The regular candidate `valid.md` is chosen even though a directory and a FIFO appear earlier in the list.
- Line 201: The returned content is from `valid.md`.
- Line 204: A warning names the directory candidate `adir`.
- Line 207: A warning names the FIFO candidate `afifo`.

### 10. Cap truncation and clamp for project context

`max_project_context_bytes` is honoured up to a hard ceiling of 32768 bytes; the effective cap is `min(maxBytes, 32768)`. When content exceeds the cap, truncation happens at a line boundary and a marker `[project context truncated after N bytes]` is appended. Content that fits exactly at the cap carries no marker.

**Guarded by:** `internal/agent/prompt/loader_test.go` — `TestLoadProjectContext_CapTruncationAndClamp`, line 215–277.

**What the test asserts:**
- Lines 223–228: Content that fits exactly at the cap is returned unchanged and contains no truncation marker.
- Lines 238–241: Content over the cap is truncated at a line boundary and includes `[project context truncated after 10 bytes]`.
- Lines 257–263: A requested cap of 100000 is clamped to 32768; the returned content contains `[project context truncated after 32768 bytes]` and does not exceed the clamped cap plus the marker.
- Lines 273–275: A lower cap of 10 bytes is honoured with the marker `[project context truncated after 10 bytes]`.

### 11. Known limit: `run_command` can write config files

ADR 0008 states the limit directly: "`run_command` can write anywhere the user's account can, so for a command the boundary is the user's approval. The approval card must show the full argv, and a command touching `.kirsch/` or the global config directory gets no special treatment in v0.1." `run_command` therefore has no path-based refusal that would stop it from writing `<workspace>/.kirsch/config.toml`. The file tools (`read_file`, `apply_patch`, etc.) still refuse these paths through the workspace denylist.

**Guarded by:** `internal/tool/run_command_test.go` — `TestRunCommandCanWriteKirschConfig`, line 1813–1850.

**What the test asserts:**
- Line 1832: An approved `sh -c "mkdir -p .kirsch && echo wrote > .kirsch/config.toml"` command succeeds.
- Line 1835: The approver was called, so the boundary is the approval prompt, not a file-path refusal.
- Line 1838: The approval request carries the full three-element argv.
- Lines 1842–1845: `.kirsch/config.toml` exists inside the workspace after the command runs.
- Line 1847: The file contains exactly the bytes the command wrote.

**Inherited claim:** The file-tool refusal is guarded by `internal/workspace/workspace_test.go` — `TestDenylist`, line 136 (`".kirsch/config.toml"` is denied), and by the file-tool tests referenced in cases 1–6.

## Triage

### `govulncheck`

Findings are split into three tiers, and only the first is urgent:

- **"Your code is affected"** — a reachable call path exists. Fix it: usually a
  toolchain or dependency bump. This must be zero.
- **"in packages you import"** — present but not called. Worth a look; not a
  release blocker.
- **"in modules you require"** — transitive, unreachable. Noise for now.

When it is the standard library, bump `go.mod`'s version floor and say in the
commit that the reason is security rather than a feature. CI's `setup-go` reads
`go-version-file: go.mod`, so the pin propagates on its own.

### `gosec`

Most findings in this codebase are architectural false positives, and the reason
is always the same: **the path already crossed `workspace.Resolve`**. G304
("file inclusion via variable") fires on every `os.ReadFile` with a non-constant
path, which is every tool Kirsch has.

Suppress with a reason, never bare:

```go
// abs came from workspace.Resolve above: containment-checked, denylist-checked,
// symlinks followed and validated hop by hop. A raw path never reaches here.
body, err := os.ReadFile(abs) // #nosec G304 -- resolved by workspace.Resolve
```

There are eleven suppressions, each with a comment. **A suppression without a
stated reason is how a scanner stops being useful** — it becomes a thing people
add to make the build pass. If you cannot write the sentence explaining why the
finding does not apply, it probably applies.

Before suppressing, check the autofix suggestion. `gosec` recommended `os.Root`
for the G122 finding above, and it was right.

### Fuzzing

A crashing input is written to the package's `testdata/fuzz/` directory.
**Commit it.** It then runs as an ordinary test on every `go test`, which is how
the `.ENV` bypass became a permanent regression test rather than a memory.

Then write a named test for the specific case as well. The fuzz corpus proves
the bug stays fixed; a named test explains to the next reader *why* the code
looks the way it does.

## Adding a fuzz target

Worth doing whenever a function takes input from a file, a command, a model, or
a user. Assert properties, not outputs:

```go
func FuzzThing(f *testing.F) {
    f.Add("ordinary input")
    f.Add("\x1b[31mhostile\x1b[0m")   // seeds steer the fuzzer; make them nasty

    f.Fuzz(func(t *testing.T, in string) {
        out := Thing(in)
        // Assert what must ALWAYS be true, whatever the input.
        if strings.ContainsRune(out, 0x1b) {
            t.Fatalf("escape survived: %q -> %q", in, out)
        }
    })
}
```

The seeds matter more than they look. The fuzzer mutates them, so a corpus of
bland strings explores a bland space. `FuzzSanitize` seeds with OSC sequences,
8-bit CSI, ZWJ emoji and a 5,000-character line for exactly that reason.

## What is not covered

Stated plainly, so nobody mistakes silence for assurance.

- **No penetration testing of the agent loop.** It does not exist yet.
  Prompt-injection resistance is tested from Milestone 3 against
  `testdata/repo-prompt-injection`, which has been waiting since M1.
- **No supply-chain attestation.** Dependencies are pinned in `go.sum` and
  scanned, but nothing verifies provenance. Worth revisiting before v1.
- **No sandboxing.** `run_command` (Milestone 2) executes with the user's
  privileges. The controls are the allowlist, the approval prompt and
  environment filtering — not isolation. This is a deliberate v0.1 scope
  decision, and it is why every command is either allowlisted or approved.
- **Windows is unsupported** (plan §1), so nothing is tested there.
