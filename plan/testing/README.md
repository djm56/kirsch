# Kirsch — Testing

Everything about verifying Kirsch: the commands, the manual walkthroughs, what
the automated suite actually covers, and how the security tooling is wired.

| Document | Use it when |
|---|---|
| [`manual/milestone-0.md`](manual/milestone-0.md) | Sitting down to check the TUI by hand |
| [`manual/milestone-1.md`](manual/milestone-1.md) | Checking the workspace engine and read-only tools by hand |
| [`manual/milestone-2.md`](manual/milestone-2.md) | Checking the approval flow and write tools by hand |
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

## Dividing the Test Burden — Two Developers

When two developers work on the same milestone, testing splits three ways:

### Automated Tests — With the Deliverable

Each deliverable owns its automated tests. The person who writes code is responsible for writing and maintaining the tests that go with it. `npm run check` before every commit is individual work, not shared.

The test suite is the specification: a test that passes means the code meets its specification. When a test fails, something in the code changed or something in the spec needs to be clarified first. Running tests locally is how you know the code is ready for review.

### CI — The Shared Arbiter

Pull requests run CI automatically. CI runs `npm run ci` — the full suite, including security tooling and the race detector — and it must pass before merging. Neither developer's local green is authoritative over CI. If your code passes locally but fails CI, there is something about the shared environment that you missed. Read the CI logs and fix it.

Never merge a pull request with red CI.

### Manual Walkthroughs — Where a Second Person Is Most Valuable

A manual walkthrough exists to catch what automation cannot — whether it feels right, whether the colours are legible on your terminal, whether an error message tells you what to do next. The person who built a deliverable is the worst person to judge that, because they know what they meant.

**Recommend that the manual walkthrough for a milestone is run by whoever did not build the bulk of it.** The fresh eye catches inconsistencies and surprises that the builder steps over. Where both developers built parts of a milestone, run the walkthrough independently and compare your findings — two independent runs of the same checklist is the single highest-value thing two people can do here that one person cannot.

### Security Tooling and Fuzzing — Specific Responsibility

**Before merging anything that touches paths, ignore rules or tool input:** `npm run security` and `npm run fuzz`. Both have found real bugs in this codebase, and both found them in inputs nobody had thought to write a test for:

- Fuzzing found the path denylist was bypassable by changing case — `.ENV` read the real `.env` on macOS (plan §11 amendment 51).
- `gosec` found that a `.gitignore` which is itself a symlink escaped the workspace (amendment 53).
- `govulncheck` found two containment escapes in the standard library and set the toolchain floor in `go.mod` (amendment 52).

Assign one developer to run security and fuzz on every PR that could affect paths, tool input handling, or external resources. The other continues with the next deliverable. It is not both developers' job.

### Milestone-Level Verification — Once at the End

A deliverable's "done when" statement is checked by its owner at handover — that is individual verification. The milestone's final acceptance checklist (Task 9 in each instruction set) is run **once**, at the end of the milestone, by whoever closes the milestone. It is not checked per deliverable.

## The Walkthrough Convention — How We Keep Records

The master walkthrough (`manual/milestone-N.md`) is always blank — it is the template. Completed runs are saved with the developer's name.

**To run a walkthrough:**

1. Copy the master to a named file: `cp manual/milestone-N.md manual/user-testing/milestone-N-<yourname>.md`
2. Tick your copy as you verify each item.
3. Commit both files: the blank master and your completed run in `user-testing/`.

**Why this matters:**

- Two people ticking the same file causes conflicts in git, and one person's run overwrites the other's record. Named copies prevent that.
- The master stays unticked. It is the template for the next run.
- Completed runs are kept, not deleted. They are the record of what was verified and by whom.
- Before you tick an item, you've actually checked it. A ticked box means "I tested this and it works".

## Outstanding Verification

Seven manual walkthrough checks remain unticked from Milestone 0 testing:

- `testing/manual/user-testing/milestone-0-donovan.md` — seven unticked items: all are the `Ctrl+G` / `g` / `G` re-pin checks. These were added by a later mission that built the re-pin feature, so they postdate the original walkthrough run rather than having been skipped.

The milestone-1 unticked item ("Outside a repository it refuses with a message naming `--workspace`") is the failure path for workspace detection. The operator has confirmed Milestones 0 and 1 are tested and is moving forward; close these unticked items as part of Milestone 2's acceptance work or afterward as a cleanup step.

## The One Rule Worth Repeating

**A test double must not be more forgiving than the thing it replaces.**

Milestone 1 lost an afternoon to a deadlock that existed only in the real
program, because the test double for `program.Send` was a buffered channel and
could not express blocking. Every test passed. Plan §11 amendment 46 has the
detail; [`automated-tests.md`](automated-tests.md) describes how the harness was changed so it would
catch it.

The same rule applies to tests themselves. When a test is meant to catch a
specific bug, break the fix and confirm the test fails. Two tests in this
repository were verified that way, and both would otherwise have been decorative.

## What to Run, and When

**Before every commit:** `npm run check`. Fast, and it is what CI will run locally before you push.

**Before a pull request:** Run the full pre-merge gate:
```bash
npm run check      # fmt + vet + lint + test + screens
npm run security   # (if touching paths or tool input)
npm run fuzz       # (if touching paths or tool input)
```

**Before calling a milestone done:** The manual walkthrough for that milestone (run by the developer who did not build most of it, or independently by both if you both built parts), plus the milestone's acceptance checklist in the relevant `milestones/milestone-N.md` file.
