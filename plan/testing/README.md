# Kirsch — Testing

Everything about verifying Kirsch: the commands, the manual walkthroughs, what
the automated suite actually covers, and how the security tooling is wired.

| Document | Use it when |
|---|---|
| [`manual-milestone-0.md`](manual-milestone-0.md) | Sitting down to check the TUI by hand |
| [`manual-milestone-1.md`](manual-milestone-1.md) | Checking the workspace engine and read-only tools by hand |
| [`automated-tests.md`](automated-tests.md) | You want to know what the suite covers, and what it deliberately does not |
| [`security.md`](security.md) | Running the security tooling, or triaging what it reports |

## The commands

Kirsch is a Go project. `npm` is here only as a familiar front door — every
script is a thin wrapper around a `go` or `bash` command, and nothing in the
build depends on Node.

```bash
npm run tools        # install the security and lint tooling (one-off)

npm run check        # fmt + vet + lint + test + screens — the pre-commit gate
npm run ci           # check + security + race — what CI runs

npm test             # go test ./...
npm run test:race    # the same under the race detector
npm run test:cover   # coverage summary
npm run fuzz         # run every fuzz target for 30s each
npm run security     # vulnerabilities + static analysis + secret scan

npm start            # run Kirsch against the current repository
npm run start:debug  # ...with the debug log enabled
```

Without Node, read `package.json` — each script is one line and can be pasted
straight into a shell.

## What to run, and when

**Before every commit:** `npm run check`. Fast, and it is what CI will run.

**Before merging anything that touches paths, ignore rules or tool input:**
`npm run security` and `npm run fuzz`. Both have found real bugs in this
codebase, and both found them in inputs nobody had thought to write a test for:

- Fuzzing found the path denylist was bypassable by changing case — `.ENV`
  read the real `.env` on macOS (plan §11 amendment 51).
- `gosec` found that a `.gitignore` which is itself a symlink escaped the
  workspace (amendment 53).
- `govulncheck` found two containment escapes in the standard library and set
  the toolchain floor in `go.mod` (amendment 52).

**Before calling a milestone done:** the manual walkthrough for that milestone,
plus its acceptance checklist in `plan/milestone-N.md`. The walkthroughs cover
what automation cannot — whether the thing feels right, whether the colours are
legible on your terminal, whether an error message tells you what to do next.

## The one rule worth repeating

**A test double must not be more forgiving than the thing it replaces.**

Milestone 1 lost an afternoon to a deadlock that existed only in the real
program, because the test double for `program.Send` was a buffered channel and
could not express blocking. Every test passed. Plan §11 amendment 46 has the
detail; `automated-tests.md` describes how the harness was changed so it would
catch it.

The same rule applies to tests themselves. When a test is meant to catch a
specific bug, break the fix and confirm the test fails. Two tests in this
repository were verified that way, and both would otherwise have been decorative.
