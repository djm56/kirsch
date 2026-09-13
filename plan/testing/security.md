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
