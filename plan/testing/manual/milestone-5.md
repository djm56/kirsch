# Manual test — Milestone 5

Kirsch Milestone 5 is polish and release: the narrow-width status bar, the `kirsch doctor` command, `--version` and `--help`, the release tooling (build wrapper, goreleaser configuration, release workflow), the repository files, the user documentation under `doc/`, and the operator decisions that gate the conditional follow-through. The deliverables are defined in `plan/milestones/milestone-5.md`.

> **Status: skeleton.** This document was written at planning time (mission-20261023-06, 2026-10-23). It is expanded as each deliverable lands. Section 1's commands were run in mission-20261027-03, and the operator passed section 1 of the plain checklist on 2026-10-27. Section 2's commands were run in mission-20261026-01, and the operator passed section 2 of the plain checklist on 2026-10-27. Section 4's commands were run in mission-20261027-04, and the operator passed section 4 of the plain checklist on 2026-10-27, with a note (see the carried item under *Carried from Milestone 5 planning* in `plan/PROGRESS.md`). Section 5's commands were run in mission-20261027-06 (step 7) on 2026-10-27, and the operator passed section 5 of the plain checklist on 2026-10-28. Section 6's automated commands were run in mission-20261027-05 (step 5) on 2026-10-27, and the operator passed section 6 of the plain checklist on 2026-10-28; its hands-on steps need a live model turn and have not been run, and the operator checklist's section 6 has only automated steps. Section 9's commands were run in mission-20261027-08 on 2026-10-27, and the operator passed section 9 of the plain checklist on 2026-10-28. Section 16's automated check was run in mission-20261027-07 (step 3) on 2026-10-27; its removal control is recorded in mission-20261027-07 step 1, and the operator passed section 16 of the plain checklist on 2026-10-28. Section 3's automated checks were run in mission-20261028-03 (step 5) on 2026-10-28, from zsh after `source scripts/m5-test-env.sh`, and the output is in `.claude/memory/workspace/m5d5-step5-evidence.txt`; its hands-on step needs a live model turn and an API key. No key-free route was found in a 25-minute attempt (mission-20261028-03 step 6a), so the step has not been run; the operator tested section 3 of the plain checklist on 2026-10-29, and that checklist's quit-and-reopen check is checked at the final release check (section 37 of the checklist). Section 7's automated checks and hands-on commands were run in mission-20261028-03 (step 9) on 2026-10-28, from zsh, and the output is in `.claude/memory/workspace/m7d-step9-evidence.txt`; the operator tested section 7 of the plain checklist on 2026-10-29. Section 10's automated commands were run in mission-20261028-03 (step 13) on 2026-10-28, from zsh, and the output is in `.claude/memory/workspace/m10b-step13-evidence.txt`; the goreleaser snapshot was not run (goreleaser is not installed, and no install was allowed in that step), and the operator tested section 10 of the plain checklist on 2026-10-29. Section 8's automated checks were run in mission-20261028-03 (step 17) on 2026-10-28, from zsh, and the output is in `.claude/memory/workspace/m6d-step17-auto.txt`; the operator tested section 8 of the plain checklist on 2026-10-29, and its hands-on paste step has not been pre-run. Section 13's automated commands and golden check were run in mission-20261029-02 (step 4) on 2026-10-29, from zsh, and the output is in `.claude/memory/workspace/m7a-step4-runs.txt`. The fix round after the step 5 review (mission-20261029-02, step 4a) reran them, and its output is in `.claude/memory/workspace/m7a-step4a-runs.txt`; the operator passed section 13 of the plain checklist on 2026-11-03. Section 12's automated and hands-on commands were run in mission-20261029-03 (step 3) on 2026-10-29, from zsh, and the output is in `.claude/memory/workspace/m3b-step3-runs.txt`; the operator passed section 12 of the plain checklist on 2026-11-03. Section 24's automated commands were run in mission-20261030-01 (step 4c) on 2026-10-30, from zsh; the doctor runs are in `.claude/memory/workspace/m20261030-01-step4c-doctor-run.txt` and the test output in `.claude/memory/workspace/m20261030-01-step4c-runs.txt`. The step 3 files are superseded and not relied on. Its hands-on steps need a second live kirsch, and the crew did not run them. The operator's section 24 of the plain checklist runs them with two Terminal windows, and the operator passed it on 2026-11-03. The operator also passed sections 14, 17, 20, 21, 22 and 27 of the plain checklist on 2026-11-03; section 27 has one optional box not yet run. Section 11's automated command was run in mission-20261105-03 (step 3) on 2026-11-05, from zsh, and its output is pasted in the section; the operator passed section 11 of the plain checklist on 2026-11-09. Section 19's automated commands were run by Bishop on 2026-11-05 (mission-20261105-04), and the raw captured output, with its `exit=` lines, is pasted in the section, including the full `go test ./internal/app/` run. The operator passed section 19 of the plain checklist on 2026-11-09. Section 26's automated commands were run by Bishop on 2026-11-05 (mission-20261105-09, after step 3), and the raw captured output, with its `exit=` lines, is pasted in the section; the section needs no live step, and the operator passed section 26 of the plain checklist on 2026-11-09. Section 28's automated commands were run by @lambert on 2026-11-05 (mission-20261105-10, step 3), and the raw output, with its `exit=` lines, is pasted in the section; the section needs no live step, and the operator passed section 28 of the plain checklist on 2026-11-09. Section 30's automated commands were run by @lambert on 2026-11-05 (mission-20261105-11, step 3), and the raw output, with its `exit=` lines, is pasted in the section; the section needs no live step, and the operator passed section 30 of the plain checklist on 2026-11-09. Section 31's automated commands were run by @lambert on 2026-11-05 (mission-20261105-12, step 3), and the raw output, with its `exit=` lines, is pasted in the section; its one hands-on step is the operator's review of the golden diff, and the operator passed section 31 of the plain checklist on 2026-11-09. The operator confirmed the O23 reading on 2026-11-11. The operator passed sections 15, 18, 23 and 25 of the plain checklist on 2026-11-09. Section 33's automated commands were run by @lambert on 2026-11-05 (mission-20261105-13, step 1), and the raw output, with its `exit=` lines, is pasted in the section; it has no *Hands-on checks* block; the operator's check is the grep commands in section 33 of the plain checklist, which read `SECURITY.md`, and the operator passed section 33 of the plain checklist on 2026-11-09. Section 35's automated commands were run by @lambert on 2026-11-06 (mission-20261106-01, step 1), and the raw output, with its `exit=` lines, is pasted in the section; it has no *Hands-on checks* block; the operator's check is section 35 of the plain checklist, which reads section 7 of the spec, and the operator passed section 35 of the plain checklist on 2026-11-09. Every block marked `TO EXPAND` is a placeholder, and the expansion belongs to the step that lands the deliverable. The operator runs tests from the plain checklist `plan/testing/manual/user-testing/milestone-5-donovan.md`, set up with `source scripts/m5-test-env.sh`; see [`milestone-5-start-here.md`](../../milestones/milestone-5-start-here.md).

## How to read this document

- **Sections follow the order of work below.** Each section is one deliverable, titled `## N · <Title> (m5-dX)`. Split deliverables (m5-d3a and m5-d3b, for example) have their own sections.
- **Anchors.** The `§ m5-dX` hooks in `plan/milestones/milestone-5.md` resolve to the parenthesised ID in each heading. `§ m5-d3a` is the heading ending `(m5-d3a)`. The check block below proves this, and proves the order-of-work table and the known-limitations table against their sources.
- **Context.** The first line under each heading is a `**Context:**` line. Its label format is fixed by check (d) in *Mechanical checks*, which fails any Context line missing one of its labels. Run a section only in the state its Context line names. *Setup* in a Context line means the Setup terminal, or a new terminal or window that has rerun the whole Setup block, then the Guard functions block (its definitions, not only the `guard` call), then `guard`.
- **Automated checks** are copied from the acceptance boxes in `plan/milestones/milestone-5.md`, with the line they came from. A command is quoted only where the milestone gives one. A test box with no command says `TO EXPAND`. A command the document writes itself, and does not quote from the milestone, is marked by a first-line comment `# authored here` inside its block.
- **Hands-on checks** are numbered steps where the milestone gives enough detail. Otherwise a `TO EXPAND` block says what the operator will observe.
- **Live checks** appear only in sections whose deliverable has a live step. Each one is marked. A provider check contacts the provider and spends tokens. A `v*` tag pushed to GitHub publishes a public release, which is one-way. A manual run of the Release workflow is a dry run and publishes nothing. Run a live check only with the key, the fork or the tag its marking names.
- **Process-only boxes** (a decision recorded, the contents of a step report) are not repeated here. They belong to the mission brief.
- **Checkboxes** are `- [ ]` until run. Mark `- [✅]` when a check passes and `- [❌]` when it fails, as Milestone 4 does.
- **Placeholder convention.** `> **TO EXPAND when m5-dN lands:** …` marks every place where the detail depends on a decision that is not yet recorded (an O-number in `plan/milestones/milestone-5.md:61–85`) or on code that does not exist yet.
- **Golden screens.** The goldens are sized fences in `plan/spec/kirsch-ui-screens.md`. Use only the procedure in *Golden mechanism and regeneration procedure* (`plan/milestones/milestone-5.md:39–53`). `KIRSCH_REGEN=1` writes the current render to `/tmp/kirsch-grids.txt` (`internal/tui/regen_test.go:27`, as the milestone states). It changes no screen document. A person reconciles the grids by hand.
- **Saved copies.** Before any edit, a section saves the copy it names, with the file name it gives, and then edits. The names are the ones the milestone uses, except where a section says its name is chosen here.
- **Where expansions land.** Expansions land in this master file first. The operator's checklist, `plan/testing/manual/user-testing/milestone-5-donovan.md`, is a separate plain-language runbook: for each section, the delivering mission writes it fresh in the paste / you-should-see / tick style and runs every step first. It does not copy this file's wording.

**Mechanical checks.** Run this block before a run, and after any expansion. Each check is followed by a control in the same invocation: a copy with one planted fault, which the check must report. Every check prints PASS or FAIL, and the block ends with a verdict. The Wave column and the Hands-on and Live tokens of the Type column are checked mechanically. The Automated and Operator tokens of the Type column, and the Deliverable and Waits-on columns, are kept by hand against `plan/milestones/milestone-5.md:97–133`. Run this block under bash, not zsh: paste it into a bash shell, or save it to a file and run `bash <file>`. The block is wrapped in `bash <<'CHECK'` and `CHECK`, so `LC_ALL`, the `exit` in `plant` and the EXIT trap stay in a child shell. An abort then cannot close the terminal the block was pasted into.

```bash
bash <<'CHECK'
export LC_ALL=C
MAN=/Volumes/DATA/Github/kirsch/plan/testing/manual/milestone-5.md
MS=/Volumes/DATA/Github/kirsch/plan/milestones/milestone-5.md
SCR=$(mktemp -d) || exit 1
trap 'rm -f "$SCR"/*; rmdir "$SCR"' EXIT
FAILS=0

# plant SOURCE COPY AWK_PROGRAM: write a copy with one planted fault.
# It aborts unless the copy differs from the source, by at most one changed line.
plant() {
  awk "$3" "$1" > "$2" || { echo "ABORT: planting failed"; exit 1; }
  [ -n "$(diff "$1" "$2")" ] || { echo "ABORT: control copy identical to source"; exit 1; }
  [ "$(diff "$1" "$2" | grep -c '^[<>]')" -le 2 ] || { echo "ABORT: control changed more than one line"; exit 1; }
}

# verdict LABEL REAL CONTROL PLANTED: REAL must be empty; CONTROL must show the planted text.
verdict() {
  local r=PASS c=PASS
  [ -z "$2" ] || r=FAIL
  printf '%s\n' "$3" | grep -qF -- "$4" || c=FAIL
  if [ "$r" = PASS ] && [ "$c" = PASS ]; then echo "PASS  $1"; else echo "FAIL  $1 (real: $r, control: $c)"; FAILS=$((FAILS + 1)); fi
  if [ -n "$2" ]; then printf '      real output:\n%s\n' "$2"; fi
  return 0
}

# (a) anchors: § references in milestone-5.md against the section headings in $1
anchors() {
  comm -3 <(grep -oE '§ m5-d[0-9]+[a-z]?' "$MS" | sed 's/^§ //' | sort -u) \
          <(grep -E '^## [0-9]+ · .*\(m5-d[0-9]+[a-z]?\)$' "$1" | sed -E 's/.*\((m5-d[0-9]+[a-z]?)\)$/\1/' | sort -u)
}

# (b1) order-of-work rows: section number and ID, against the headings
order_check() {
  comm -3 <(awk -F'|' '/^\| [0-9]+ \| m5-d/ { s = $2; gsub(/ /, "", s); d = $3; sub(/^ +/, "", d); split(d, a, " "); print s, a[1] }' "$1" | sort -u) \
          <(grep -E '^## [0-9]+ · .*\(m5-d[0-9]+[a-z]?\)$' "$1" | sed -E 's/^## ([0-9]+) · .*\((m5-d[0-9]+[a-z]?)\)$/\1 \2/' | sort -u)
}

# (b2) Wave column against the wave list in milestone-5.md (the gate row reads "gate")
wave_expect() {
  awk '
    /^- \*\*Wave [0-9]:\*\*/ { w = $0; sub(/^- \*\*Wave /, "", w); sub(/:.*/, "", w); line = $0
      sub(/^- \*\*Wave [0-9]:\*\* /, "", line); gsub(/\([^)]*\)/, "", line)
      while (match(line, /m5-d[0-9]+[a-z]?/)) { print substr(line, RSTART, RLENGTH), w; line = substr(line, RSTART + RLENGTH) } }
    /^- \*\*Gate:\*\*/ { line = $0; gsub(/\([^)]*\)/, "", line); if (match(line, /m5-d[0-9]+[a-z]?/)) print substr(line, RSTART, RLENGTH), "gate" }' "$MS" | sort -u
}
wave_check() {
  comm -3 <(wave_expect) <(awk -F'|' '/^\| [0-9]+ \| m5-d/ { d = $3; sub(/^ +/, "", d); split(d, a, " "); w = $4; gsub(/ /, "", w); print a[1], w }' "$1" | sort -u)
}

# (b3) Type column, Hands-on and Live tokens only. Hands-on: a numbered hands-on item not tagged (operator).
# Live: a Live checks block, or a Context line that names key entry for a live step.
type_sections() {
  awk '
    function emit() { if (sec != "") print sec, H + 0, L + 0 }
    /^## [0-9]+ · / { emit(); sec = $2; H = 0; L = 0; inhands = 0; getline; getline; if (index($0, "key entered for this live step")) L = 1; next }
    /^\*\*Live checks\*\*/ { L = 1 }
    /^\*\*Hands-on checks\*\*/ { inhands = 1 }
    /^---$/ { inhands = 0 }
    inhands && /^[0-9]+\. / && !/^[0-9]+\. \(operator\)/ { H = 1 }
    END { emit() }' "$1" | sort -u
}
type_table() {
  awk -F'|' '/^\| [0-9]+ \| m5-d/ { s = $2; gsub(/ /, "", s); t = $5; print s, (t ~ /Hands-on/ ? 1 : 0), (t ~ /Live/ ? 1 : 0) }' "$1" | sort -u
}
type_check() { comm -3 <(type_sections "$1") <(type_table "$1"); }

# (c) known-limitations IDs against the Backlog
limits_check() {
  comm -3 <(sed -n '/^## Known limitations/,$p' "$1" | grep -E '^\| [CX][0-9]+' | sed -E 's/^\| ([CX][0-9]+).*/\1/' | sort -u) \
          <(sed -n '/^## Backlog/,/^## Out of scope/p' "$MS" | grep -E '^- \*\*[CX][0-9]+' | sed -E 's/^- \*\*([CX][0-9]+).*/\1/' | sort -u)
}

# (d) every section heading has a Context line that carries all of its labels
context_check() {
  awk '
    /^## [0-9]+ · / { h = $0; getline b; getline c
      if (c !~ /^\*\*Context:\*\*/) { print "no Context under: " h; next }
      n = split("Terminal:|Binary:|Working directory:|Workspace:|Key state:|Config state:", lab, "|")
      for (i = 1; i <= n; i++) if (index(c, lab[i]) == 0) print "Context lacks " lab[i] " under: " h }' "$1"
}

# (e) key state. A live section (Context names key entry for a live step) needs the unset line and the check.
# A launch section (Binary names kirsch or smoke-test, not "no kirsch launch") needs the check.
# LIVE lines list the live sections; MISSING lines are discrepancies.
keystate() {
  awk -v chk="keycount" \
      -v uns="unset ANTHROPIC_API_KEY KIRSCH_ANTHROPIC_API_KEY OPENCODE_API_KEY KIRSCH_OPENCODE_API_KEY" '
    function emit() {
      if (head == "") return
      if (live) print "LIVE     " head
      if (live && !(hasunset && hascheck)) print "MISSING unset or check in live section: " head
      if (launch && !hascheck) print "MISSING key-state check in launch section: " head
    }
    /^## [0-9]+ · / { emit(); head = $0; hascheck = 0; hasunset = 0; live = 0; launch = 0
      getline; getline; c = $0
      if (index(c, "key entered for this live step")) live = 1
      s = index(c, "Binary: "); e = index(c, "Working directory:")
      if (s > 0 && e > s && substr(c, s, e - s) ~ /kirsch|smoke-test/ && substr(c, s, e - s) !~ /no kirsch launch/) launch = 1
      next }
    $0 ~ ("^[ \t]*" chk "$") { hascheck = 1 }
    index($0, uns) { hasunset = 1 }
    END { emit() }' "$1"
}

echo "== (a) anchors"
plant "$MAN" "$SCR/a.md" '/^## 4 · /{ sub(/\(m5-d9\)$/, "(m5-d99)") } { print }'
verdict "(a) anchors against milestone-5.md" "$(anchors "$MAN")" "$(anchors "$SCR/a.md")" "m5-d99"

echo "== (b) order of work"
plant "$MAN" "$SCR/b1.md" '/^\| 16 \| m5-d13b /{ sub(/^\| 16 \|/, "| 17 |") } { print }'
verdict "(b1) order-of-work section numbers and IDs" "$(order_check "$MAN")" "$(order_check "$SCR/b1.md")" "17 m5-d13b"
plant "$MAN" "$SCR/b2.md" '/^\| 2 \| m5-d2 /{ sub(/\| 1 \|/, "| 3 |") } { print }'
verdict "(b2) Wave column against the milestone wave list" "$(wave_check "$MAN")" "$(wave_check "$SCR/b2.md")" "m5-d2 3"
plant "$MAN" "$SCR/b3.md" '/^\| 1 \| m5-d1 /{ sub(/Automated \+ Hands-on/, "Automated") } { print }'
verdict "(b3) Type column, Hands-on and Live tokens" "$(type_check "$MAN")" "$(type_check "$SCR/b3.md")" "1 0 0"

echo "== (c) known limitations"
plant "$MAN" "$SCR/c.md" '{ print } /^\| C2 \|/ { print "| C99 | planted | x |" }'
verdict "(c) known-limitations IDs against the Backlog" "$(limits_check "$MAN")" "$(limits_check "$SCR/c.md")" "C99"

echo "== (d) Context labels"
plant "$MAN" "$SCR/d.md" '/^\*\*Context:\*\*/ && !done && /Key state:/ { sub(/Key state:/, "Keys:"); done = 1 } { print }'
verdict "(d) every Context line carries the labels" "$(context_check "$MAN")" "$(context_check "$SCR/d.md")" "Context lacks Key state:"

echo "== (e) key state"
plant "$MAN" "$SCR/e.md" '/^## 2 · /{ s = 1 } /^## 3 · /{ s = 0 } { if (s && index($0, "keycount")) next; print }'
verdict "(e) key-state checks in live and launch sections" "$(keystate "$MAN" | grep '^MISSING')" "$(keystate "$SCR/e.md" | grep '^MISSING')" "## 2 ·"
plant "$MAN" "$SCR/e2.md" '/^## 2 · /{ s = 1 } /^## 3 · /{ s = 0 } { if (s && $0 ~ /^[ \t]*keycount$/) next; print }'
verdict "(e2) a launch section with only keycount_control is flagged" "$(keystate "$MAN" | grep '^MISSING')" "$(keystate "$SCR/e2.md" | grep '^MISSING')" "## 2 ·"
echo "live sections (the sections whose Context names key entry for a live step):"
keystate "$MAN" | grep '^LIVE'

if [ "$FAILS" -eq 0 ]; then echo "VERDICT: PASS (every check and control)"; else echo "VERDICT: FAIL ($FAILS check(s) failed)"; fi
CHECK
```

---

## Order of work

Sections run in wave order from `plan/milestones/milestone-5.md:327–335`. The operator gate (m5-d20) sits between wave 6 and wave 7. The *Deliverables* table is at `plan/milestones/milestone-5.md:97–133`. Check (b) in *Mechanical checks* compares the Section, Wave and Type columns with the section headings, the wave list at `plan/milestones/milestone-5.md:327–335`, and each section's blocks. The Deliverable and Waits-on columns are checked by hand against `plan/milestones/milestone-5.md:97–133`.

| Section | Deliverable | Wave | Type | Decisions it waits on |
|---------|-------------|------|------|-----------------------|
| 1 | m5-d1 Narrow-width status-bar fallback | 1 | Automated + Hands-on | O1, O23 |
| 2 | m5-d2 `--version` and `--help` | 1 | Automated + Hands-on | O4 |
| 3 | m5-d5 Persist `/approvals` clear | 1 | Automated + Hands-on + Operator | O11 |
| 4 | m5-d9 Terminal capability and resize tests | 1 | Automated + Operator | O18 |
| 5 | m5-d13a `doc/configuration.md` | 1 | Automated + Hands-on | O6 |
| 6 | m5-d19a Safety-test repair: C30, C35, C36 | 1 | Automated + Hands-on | O13 (scheduling) |
| 7 | m5-d3a Doctor: dispatch, grammar, version, exit codes, arch entry | 2 | Automated + Hands-on | m5-d2, O5 |
| 8 | m5-d6 Composer paste path and caret rows | 2 | Automated + Operator | O10; m5-d1 (file ownership: view.go, if O10 is wrap) |
| 9 | m5-d10a Build wrapper and version proof | 2 | Automated + Hands-on | m5-d2, O4 |
| 10 | m5-d10b goreleaser config and target-compile check | 2 | Automated + Hands-on + Operator | m5-d2, O2, O4, O20 |
| 11 | m5-d19b Safety-test repair: C29 | 2 | Automated + Hands-on | m5-d19a, O15 (C29), O13 |
| 12 | m5-d3b Doctor: `rg`, `git`, repository and config probes | 3 | Automated + Hands-on | m5-d3a, O5 |
| 13 | m5-d7a Soft-wrap fidelity: `wrap()` | 3 | Automated + Hands-on + Operator | m5-d1, m5-d6, O23 |
| 14 | m5-d8 Paste debounce | 3 | Automated + Operator | m5-d6, O9 |
| 15 | m5-d11 GitHub release workflow | 3 | Automated + Live + Operator | m5-d10b, O3 |
| 16 | m5-d13b Key-coverage test (S36) | 3 | Automated + Hands-on | m5-d13a |
| 17 | m5-d13c `doc/install.md` | 3 | Automated + Operator | m5-d2, m5-d10a, m5-d10b, O4, O6 |
| 18 | m5-d16 Fresh-machine no-key check | 3 | Automated + Operator | O14, O19, m5-d10a; m5-d2 and m5-d3a (file ownership, if O14 makes "not in a repo" a screen) |
| 19 | m5-d17a `/new` lifecycle: C51, C52 (optional) | 3 | Automated | m5-d5, O13 |
| 20 | m5-d4a Doctor: key, backend, model and project-context lines | 4 | Automated + Hands-on | m5-d3b, O5 |
| 21 | m5-d7b Soft-wrap fidelity: composer cut marker | 4 | Automated + Hands-on + Operator | m5-d6, m5-d7a, O17, O23 |
| 22 | m5-d14a `doc/usage.md` corrections | 4 | Automated + Hands-on | m5-d1, m5-d5, m5-d6, m5-d7a, O6, O10 |
| 23 | m5-d15 Front-door and tracker freshness | 4 | Automated + Hands-on | m5-d1, m5-d10a, m5-d10b, m5-d13c, O22 |
| 24 | m5-d4b Doctor: state directory and lock state | 5 | Automated + Operator | m5-d4a, O5 |
| 25 | m5-d12 Repository files and golden-regeneration note | 5 | Automated + Operator | m5-d10a, m5-d10b, m5-d8, O7, O8, O21 |
| 26 | m5-d17b `/new` busy state and turn fencing: C48, C49 (optional) | 5 | Automated | m5-d8, m5-d17a, O13 |
| 27 | m5-d14b `doc/troubleshooting.md` and `doc/README.md` | 6 | Automated + Live + Operator | m5-d3a, m5-d3b, m5-d4a, m5-d4b, m5-d13a, m5-d13c, O6 |
| 28 | m5-d18 Resume totals and compaction % (optional) | 6 | Automated | m5-d17a (file ownership only), O13 |
| 29 | m5-d20 Record the operator's decisions | gate | Operator | O12, O15, O16, and the decision list |
| 30 | m5-d21a Model table (conditional) | 7 | Automated | m5-d20 (X5, C9) |
| 31 | m5-d21b Help footer and goldens (conditional) | 7 | Automated + Operator | m5-d20 (X8, O16), O23 |
| 32 | m5-d21c Debug log clip (conditional) | 7 | Automated + Hands-on | m5-d20 (C24) |
| 33 | m5-d21d Debug log disclosure (conditional) | 7 | Automated + Operator | m5-d20 (C24), m5-d12 |
| 34 | m5-d21e ULID dependency (conditional) | 7 | Automated + Hands-on | m5-d20 (C40) |
| 35 | m5-d21f §7 amendment (conditional) | 7 | Automated + Operator | m5-d20 (C43, C46), m5-d5 |
| 36 | m5-d21g Lock behaviour (conditional) | 7 | Automated + Hands-on | m5-d20 (C8) |
| 37 | m5-d22 Demo, release smoke run, tag | 8 | Live + Operator + Sign-off | O12, O15, O16, m5-d21 deliverables that ran |

---

## Setup

Sections that run the `kirsch` binary or a test that reads configuration use the scratch environment below. Doc-only sections (greps on `doc/` and the root files) need no scratch environment. They still run from the checkout, with absolute paths, as their commands show.

**Where sessions go.** The session store defaults to `~/.local/share/kirsch/sessions` (`internal/config/config.go:113`, field `StorageDir` at `:73`, toml tag `storage_dir`). The field sits in `SessionConfig`, which is the `[session]` TOML section (`internal/config/config.go:31`, `toml:"session"`). Setting `XDG_STATE_HOME` does not move sessions. It moves only `debug.log` (`plan/testing/manual/milestone-4.md:1214`). The scratch config below therefore sets `storage_dir` under `$T_STATE`.

**Where the config file is.** With `XDG_CONFIG_HOME` set, the global file is `$XDG_CONFIG_HOME/kirsch/config.toml` (`internal/config/config.go:665–672`, `GlobalPath`). `~` in `storage_dir` is expanded (`config.go:294`), but the setup uses an absolute path.

```bash
# authored here
cd /Volumes/DATA/Github/kirsch
unset ANTHROPIC_API_KEY KIRSCH_ANTHROPIC_API_KEY OPENCODE_API_KEY KIRSCH_OPENCODE_API_KEY KIRSCH_LIVE_MODEL MY_TOKEN T_BROKEN
export T_CFG=$(mktemp -d)
export T_STATE=$(mktemp -d)
export WORKSPACE=$(mktemp -d)
export XDG_CONFIG_HOME="$T_CFG"
export XDG_STATE_HOME="$T_STATE"
git -C "$WORKSPACE" init
mkdir -p "$WORKSPACE/.kirsch"
mkdir -p "$T_CFG/kirsch"
printf '[session]\nstorage_dir = "%s"\n' "$T_STATE/sessions" > "$T_CFG/kirsch/config.toml"
```

The unset line clears the key variables, as the m5-d16 box does (`plan/milestones/milestone-5.md:1413`), and clears `MY_TOKEN` and `T_BROKEN`. Nothing in this document sets `T_BROKEN` now: section 7 uses a throwaway folder inside a subshell and leaves nothing behind. The name stays in the unset line because `scripts/m5-test-env.sh` still names it. That is a known leftover, carried in `plan/PROGRESS.md`. `KIRSCH_LIVE_MODEL` is unset as Milestone 4 does (`plan/testing/manual/milestone-4.md:40`).

**Guard.** Run this after Setup, and after any section that changes the config. The first `grep` must print one line, the `storage_dir` line under `$T_STATE`. The printf control must print `0`. If the first grep prints nothing, stop. A kirsch launch would then write to the real store. The guard functions in the next block check the environment and the file together. Run that block too.

```bash
grep -n "storage_dir = \"$T_STATE/sessions\"" "$T_CFG/kirsch/config.toml"
printf 'storage_dir = "/elsewhere"\n' | grep -c "storage_dir = \"$T_STATE/sessions\""
```

**Guard functions.** Authored here. Paste this block after Setup, in the same terminal. `guard` prints `GUARD PASS` or `GUARD FAIL`. The control must print `CONTROL PASS`.

```bash
# authored here
storage_ok() {   # exit 0 only when the scratch storage_dir line appears exactly once, under [session]
  awk -v want="storage_dir = \"$T_STATE/sessions\"" '
    /^\[/ { sec = $0 }
    $0 == want && sec == "[session]" { n++ }
    END { exit (n == 1 ? 0 : 1) }' "$1"
}
# keycount prints how many key variables are set. Expect 0.
keycount() {
  env | grep -c -E '^(KIRSCH_)?(ANTHROPIC|OPENCODE)_API_KEY='
}
# keycount_control plants one key line in a subshell and requires the count to be exactly 1.
keycount_control() {
  local got
  got=$(export OPENCODE_API_KEY=planted; keycount)
  if [ "$got" = 1 ]; then echo "CONTROL PASS: a planted key line counts as 1"; return 0; fi
  echo "CONTROL FAIL: a planted key line counted $got, not 1"; return 1
}
guard() {
  [ "$XDG_CONFIG_HOME" = "$T_CFG" ] && [ "$XDG_STATE_HOME" = "$T_STATE" ] || { echo "GUARD FAIL: XDG_CONFIG_HOME or XDG_STATE_HOME is not the scratch path"; return 1; }
  storage_ok "$T_CFG/kirsch/config.toml" || { echo "GUARD FAIL: storage_dir is not under [session] in the scratch config"; return 1; }
  keycount_control || { echo "GUARD FAIL: the key-count control did not print 1"; return 1; }
  [ "$(keycount)" = 0 ] || { echo "GUARD FAIL: a key variable is set"; return 1; }
  echo "GUARD PASS"
}
guard
CTRL=$(mktemp -d) && mkdir -p "$CTRL/kirsch" && printf '[policy]\nstorage_dir = "%s"\n' "$T_STATE/sessions" > "$CTRL/kirsch/config.toml"
storage_ok "$CTRL/kirsch/config.toml"; s=$?
[ "$s" = 1 ] && echo "CONTROL PASS: a [policy] line was refused" || echo "CONTROL FAIL: a [policy] line gave exit $s (refused is exactly 1)"
rm -f "$CTRL/kirsch/config.toml"; rmdir "$CTRL/kirsch" "$CTRL"
CTRL2=$(mktemp -d) && mkdir -p "$CTRL2/kirsch" && printf '[session]\n# storage_dir = "%s"\n' "$T_STATE/sessions" > "$CTRL2/kirsch/config.toml"
storage_ok "$CTRL2/kirsch/config.toml"; s=$?
[ "$s" = 1 ] && echo "CONTROL PASS: a commented storage_dir line was refused" || echo "CONTROL FAIL: a commented storage_dir line gave exit $s (refused is exactly 1)"
rm -f "$CTRL2/kirsch/config.toml"; rmdir "$CTRL2/kirsch" "$CTRL2"
```

**Model check before any live step.** The output must be empty. A model line is a stray override from an earlier section, and it breaks the live smoke test (`plan/testing/manual/milestone-4.md:1331`).

```bash
# authored here
grep -n -i 'model' "$T_CFG/kirsch/config.toml"
```

**Workspace.** `WORKSPACE` is a separate Git repository. The Kirsch checkout is never the workspace.

**Sessions.** Sessions are written under `$T_STATE/sessions`. Check with `ls "$T_STATE/sessions"` after a run.

**Terminals and windows.** Run every section in the Setup terminal. Any new terminal or window must rerun the whole Setup block, then paste the Guard functions block (its definitions, not only the `guard` call), then run `guard`, before it runs anything. A window without that scratch config falls back to `~/.local/share/kirsch/sessions` (`internal/config/config.go:113`). If its shell exports a key, that window can reach the live provider.

**Config changes.** A section that needs more config keys appends with `>>`, or writes its own file under its own directory and points `XDG_CONFIG_HOME` at it for one command. No section writes the Setup config with `>`. Every section that changes the config reruns the guard afterwards.

**Live key entry.** Use this only in a section whose Context line says `key entered for this live step`. Check (e) in *Mechanical checks* lists those sections, so no list is kept here. Never paste a key into a file or a chat. This is the Milestone 4 pattern (`plan/testing/manual/milestone-4.md:63–73`), and the key is unset again at the end of the section by the block that section ends with:

```bash
read -rs OPENCODE_API_KEY && export OPENCODE_API_KEY
```

**Cleanup.** Section 7 leaves no folder behind, so there is nothing to remove before Cleanup runs. The block below still unsets `T_BROKEN`, because `scripts/m5-test-env.sh` still names it. That is a known leftover (see the Setup note above). Then unset the variables when a session ends, and check the count. Authored here. It must print `CONTROL PASS`, then `0`.

```bash
# authored here
unset ANTHROPIC_API_KEY KIRSCH_ANTHROPIC_API_KEY OPENCODE_API_KEY KIRSCH_OPENCODE_API_KEY KIRSCH_LIVE_MODEL MY_TOKEN T_BROKEN
unset XDG_CONFIG_HOME XDG_STATE_HOME
keycount_control
keycount
```

Removing the scratch directories is the operator's step, and agents do not run it. Section 7 leaves no folder behind, so no `$T_BROKEN` removal is needed before the Cleanup block. After Cleanup, copy the removal line from `plan/testing/manual/milestone-4.md:60`: a recursive `rm` of `$T_CFG`, `$T_STATE` and `$WORKSPACE`. That line names no section 7 folder. The variable `MY_TOKEN` is unset above. If `T_BROKEN` is still set in an older terminal and names a folder, `m5_done` still lists that folder for removal. That is the known leftover in `scripts/m5-test-env.sh`.

---

## 1 · Narrow-width status-bar fallback (m5-d1)

**Context:** Terminal: Setup. Binary: none for the automated checks; `go run ./cmd/kirsch --workspace "$WORKSPACE"` for hands-on step 2. Working directory: /Volumes/DATA/Github/kirsch. Workspace: `--workspace "$WORKSPACE"` on the hands-on launch; n/a for the checks. Key state: all key variables unset, so no provider key is set and no turn runs in hands-on step 2. Config state: Setup, unchanged. The hands-on table assumes no session grants and no other kirsch running in the same workspace; either adds text to the bar.

**What landed**

O1 (`plan/PROGRESS.md`, Milestone 5 decisions) keeps `plan/spec/ui-spec-v0.1.md` §2.2: no token figure is shown below 80 columns. At 80 columns and wider, the status bar takes the first token figure that fits. With session totals, that is the session detail (`in … out … cache …`), then the session total `N tok`. With no session totals yet, it is the last call's figure as `N tok`, which is `0 tok` before any turn. If none fits, the bar drops to the model with grants, the model alone, or the family, with no token figure. Below 80 it shows no token figure at all, not even `N tok`. A cost figure appears only for per-token pricing, or as `$?` for unknown pricing; flat-priced models never show one. The Milestone 4 walkthrough ❌ (`plan/testing/manual/user-testing/milestone-4-donovan.md` §9 step 11) recorded a wrong expectation, not a code defect.

No product code changed. m5-d1 edited the spec sections §2.2 (the width rows, which now say no token figure below 80) and §2.3 (the truncation order, which below 80 starts at the grants indicator). It extended the existing test `TestSessionUsageNarrowLayoutFallsBackToSessionTotal` in `internal/tui/turn_messages_test.go`. The test already pinned 79 columns before m5-d1. m5-d1 added 60, 59 and 30 columns, and a cost assertion that no `$` figure appears below 80 columns. Only the test file changed in code.

The sized fences below 80 columns in `plan/spec/kirsch-ui-screens.md` were `40×11`, `38×6` and `40×6` when mission-20261027-03 ran. None moved. The file is byte-identical to the committed copy, `git show HEAD:plan/spec/kirsch-ui-screens.md`, which is valid as the pre-change baseline until the next commit that touches that file. No fence at 80 columns or wider moved either.

**Recorded:** O1 (`plan/PROGRESS.md`, Milestone 5 decisions) keeps ui-spec §2.2, no token figure below 80 columns, and corrects the Milestone 4 acceptance text. O23 (`plan/PROGRESS.md`, "Go with a") regenerates any golden that moves and has a human review the golden diff. No golden moved in m5-d1. Golden procedure: `plan/milestones/milestone-5.md:39–53`. The Milestone 4 walkthrough ❌ (`plan/testing/manual/user-testing/milestone-4-donovan.md:1042`) recorded a wrong expectation, resolved as O1 states.

**Key-state check, first.** Authored here. Run it before any kirsch launch in this section. It must print `CONTROL PASS`, then `0`.

```bash
# authored here
keycount_control
keycount
```

**Automated checks**

- [ ] `npm run check` exits 0 (`plan/milestones/milestone-5.md:459`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && npm run check
  ```

- [ ] Sized fences at 80 columns and wider are unchanged. The command extracts them from the saved copy and from the file, and compares them. Empty diff output means pass (`plan/milestones/milestone-5.md:441–458`). The command takes `$PRE` from the committed copy, `HEAD:plan/spec/kirsch-ui-screens.md`. That copy is the pre-change baseline only until the next commit that touches the file. After that commit, point `$PRE` at the commit before the change instead.

  ```bash
  # authored here
  PRE=$(mktemp); git -C /Volumes/DATA/Github/kirsch show HEAD:plan/spec/kirsch-ui-screens.md > "$PRE"
  NOW=/Volumes/DATA/Github/kirsch/plan/spec/kirsch-ui-screens.md
  wide() { awk '/^```text [0-9]+×[0-9]+$/ { split($2, d, "×"); keep = (d[1] + 0 >= 80); if (keep) print "@@FENCE " $2; infence = 1; next }
       /^```$/ { infence = 0; keep = 0; next }
       infence && keep { print }' "$1"; }
  wide "$NOW" | grep -c '^@@FENCE'
  diff <(wide "$PRE") <(wide "$NOW")
  # control: one character appended to the first line of the first wide fence in a scratch copy
  CTRL=$(mktemp)
  awk 'BEGIN { d = 0; m = 0 } /^```text 80×/ && !d { on = 1; d = 1; print; next } on && /^```$/ { on = 0 } on && !m { print $0 "X"; m = 1; next } { print }' "$PRE" > "$CTRL"
  diff <(wide "$PRE") <(wide "$CTRL")
  rm -f "$CTRL" "$PRE"
  ```

  The first output line is the count of wide fences. The control `diff` must print a difference. Without that, an empty diff proves nothing.

- [ ] At 60–79 columns and below 60, the status bar shows no token figure, as O1 specifies. The test is `TestSessionUsageNarrowLayoutFallsBackToSessionTotal` in `internal/tui/turn_messages_test.go`; its subtests named `below 80 columns: no tokens at all, …` cover 79, 60, 59 and 30 columns (`plan/milestones/milestone-5.md:438–439`). Command:

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -count=1 ./internal/tui -run TestSessionUsageNarrowLayoutFallsBackToSessionTotal -v
  ```

  Expected: `--- PASS: TestSessionUsageNarrowLayoutFallsBackToSessionTotal` and a `PASS` line for each subtest. The test renders the status row through the TUI's own `statusRow` path at each width. It proves the narrow-width rule. It does not run kirsch in a terminal.

- [ ] ui-spec §2.2 and §2.3 agree with the code (`plan/milestones/milestone-5.md:440`). Review step, not a command. The reviewer compares the §2.2 width rows with the band switch in `internal/tui/view.go` (the `case w >= 80`, `case w >= 60` and `case w >= 40` arms) and the token gate `if lay.ShowTokens` in `internal/tui/statusbar.go`. Below 80 there must be no token figure. At 80 and wider the figure is, in order: the session detail (`in … out … cache …`, with the cost when it fits) while session totals exist and the detail fits; otherwise the session total `N tok` while session totals exist; otherwise the last call's `N tok`, which is `0 tok` before any turn. If none of those fits, the bar shows no token figure. The reviewer also compares the §2.3 truncation order with the candidate ladder in `internal/tui/statusbar.go`.

**Hands-on checks**

1. **Record of mission-20261027-03, done (not an instruction).** Before the change, the sized fences below 80 columns were recorded by fence header: `40×11`, `38×6` and `40×6` (`plan/milestones/milestone-5.md:428`). The pre-change copy is `HEAD:plan/spec/kirsch-ui-screens.md`, as in the automated check above. The command below reproduces the list from the current file. Its verbatim output is pasted under *Notes / evidence*. It lists the same headers, so no fence moved.

   ```bash
   grep -n '^```text [0-9]*×' /Volumes/DATA/Github/kirsch/plan/spec/kirsch-ui-screens.md | awk -F'[ ×]' '$2+0 < 80'
   ```

   A later mission that changes a golden repeats this record before it changes anything. If the list is empty, the record says "no golden below 80 columns", and no fence changes.

2. Run the status bar at each width band that O1 names, and compare the bar with the O1 text.

   Launch from the checkout, with `$WORKSPACE` set by the setup block:

   ```bash
   cd /Volumes/DATA/Github/kirsch && go run ./cmd/kirsch --workspace "$WORKSPACE"
   ```

   Before each row of the table, resize the terminal and run `stty size`. It prints rows, then columns. `$COLUMNS` is not reliable here, because the shell does not export it by default. Quit with `/quit`.

   With no provider key no turn runs, so no usage is reported and the bar shows no session total. The expected status-bar row at each width is below. The table shows the left segment only. The render output in *Notes / evidence* shows the full row as the code draws it:

   | Columns | Status-bar row (`<model>` is the configured model) | Token figure |
   |---|---|---|
   | 100 | `<model> · idle · 0 tok` | `0 tok`, the last-call figure before any totals exist |
   | 80 | `<model> · idle · 0 tok` | `0 tok`, as above |
   | 79 | `<model> · idle` | none |
   | 60 | `<model> · idle` | none |
   | 59 | `<model> · idle` | none |
   | 40 | `<model> · idle` | none |
   | 30 | `<model> · idle` | none |

   The 80-column row shows `0 tok` because O1 allows a token figure at 80 and wider, and no turn has reported usage yet, so the last-call figure is the one shown. The no-figure rows below 80 are the O1 rule. The rows above were rendered by the status-bar code with no turn, in a scratch copy of the checkout and not in a live terminal (*Notes / evidence*). The session-total fallback (`N tok`) appears at 80 columns and wider only when session totals exist, the detail does not fit, and the total does. Scenario C in *Notes / evidence* shows it. The test covers both sides: the fallback at 80 columns, and no figure at 79 columns and below even with session totals present. A live check with a real turn is operator-owned and optional.

**Notes / evidence**

Render, run 2026-10-27 09:20 UTC by `@lambert` in mission-20261027-03 (step 6). The program was run in a scratch copy of the checkout, not in the repository: the working tree was copied with `rsync` (excluding `.git`, `.claude`, `node_modules`, `bin`), and a test file was added to that copy only. Command, from the scratch copy: `go test -count=1 ./internal/tui -run TestZZRenderStatusRow -v` (Go 1.26.10, darwin/arm64). The test calls `statusRow(layout())` through the helpers `sizedModel`, `drive` and `stripEscapes` that the existing narrow-layout test uses. Model `minimax-m2.7`, family `minimax`. The program prints each row between `[` and `]`; the brackets are not part of the row. Scenario A is the state of the walkthrough (no turn, no usage). Scenario B is one turn with usage reported (`UsageMsg` 100 in, 11 out; `SessionUsageMsg` 1000 in, 300 out, 8000 cache read, 2000 cache write, $0.04). Scenario C is scenario B with a 40-character model name, so the detail does not fit at 80 columns.

```text
== scenario A: no turn has run (no session totals, no usage) ==
100 cols: [minimax-m2.7 · idle · 0 tok]
80 cols: [minimax-m2.7 · idle · 0 tok]
79 cols: [minimax-m2.7 · idle]
60 cols: [minimax-m2.7 · idle]
59 cols: [minimax-m2.7 · idle]
40 cols: [minimax-m2.7 · idle]
30 cols: [minimax-m2.7 · idle]
== scenario B: one turn reported usage (session totals present) ==
100 cols: [minimax-m2.7 · idle · in 1.0k out 300 cache 8.0k/2.0k · $0.04]
80 cols: [minimax-m2.7 · idle · in 1.0k out 300 cache 8.0k/2.0k · $0.04]
79 cols: [minimax-m2.7 · idle]
60 cols: [minimax-m2.7 · idle]
59 cols: [minimax-m2.7 · idle]
40 cols: [minimax-m2.7 · idle]
30 cols: [minimax-m2.7 · idle]
== scenario C: session totals present, model name long enough that the detail does not fit ==
100 cols: [mmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmm · idle · in 1.0k out 300 cache 8.0k/2.0k · $0.04]
80 cols: [mmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmm · idle · 11.3k tok]
```

Reading the render against the rule: at 100 and 80 columns scenario A shows the last-call `0 tok`, and no session total exists yet. Scenario B shows the session detail at both widths, because the model name is short enough for the cost to fit. Scenario C shows the session total at 80 columns when the detail does not fit, and the detail at 100. Scenarios A and B show no token figure at any width below 80. Scenario C was rendered at 100 and 80 only. The hands-on table matches scenario A.

**Re-running the render.** The program is kept at `.claude/memory/workspace/m5-d1-zz_render_test.go` while mission-20261027-03 is live. It is not part of the repository. To find it, run `find .claude/memory/workspace -name m5-d1-zz_render_test.go`. The program uses the package's test helpers `sizedModel`, `drive` and `stripEscapes` from `internal/tui`, so it must be copied into `internal/tui` of a scratch copy of the module. `sizedModel` and `stripEscapes` are in `turn_messages_test.go`, and `drive` is in `view_smoke_test.go`. It also compiles against `Model.statusRow`, `Model.layout`, `UsageMsg` and `SessionUsageMsg`. A rename of any of those helpers or symbols would stop it compiling. To re-run it, copy the checkout into a new scratch directory (excluding `.git`, `.claude`, `node_modules` and `bin`), copy the file into that copy's `internal/tui/` as `zz_render_test.go`, and run `go test -count=1 ./internal/tui -run TestZZRenderStatusRow -v` from the scratch copy. Never copy it into the repository tree.

**Fence-list output (hands-on step 1).** Run from the checkout, verbatim:

````text
574:```text 40×11
590:```text 38×6
601:```text 40×6
````

Fence check, run 2026-10-27 09:22 UTC, with the command block above. The baseline is `HEAD:plan/spec/kirsch-ui-screens.md`. The first line is the count of wide fences in the file. The control diff must print a difference, and it does.

```text
wide fence count in NOW:
11
wide fence count in PRE:
11
--- diff PRE NOW (empty means pass):
exit 0
--- control diff PRE CTRL (must print a difference):
2c2
<  Kirsch ───────────────────────────────────────────────────────────────────────
---
>  Kirsch ───────────────────────────────────────────────────────────────────────X
exit 1
```

The `exit 1` on the control is the expected result of `diff`, which exits 1 when it finds a difference. Step report reference: mission-20261027-03, step 6.

---

## 2 · Version and help flags (m5-d2)

**Context:** Terminal: Setup. Binary: `bin/kirsch` and `bin/kirsch-999`, built from the checkout by the build block below (`bin/` is git-ignored) and run by absolute path, so nothing needs to be on PATH. Working directory: `$T_CFG`, outside any Git repository, for the binary runs; `/Volumes/DATA/Github/kirsch` for the build block and the `go test` check. Workspace: n/a. The check requires no repository, so `--workspace` is not passed. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

O4 is recorded in `plan/PROGRESS.md:346`. This is what the code now does.

`--version` prints `kirsch <version>` only. It prints no commit and no date. A version stamped with `-ldflags "-X main.version=…"` is printed as given. With no stamp, a `go install …@vX.Y.Z` build reports that module version with the leading `v` removed. A plain `go build` in the checkout keeps `0.1.0-dev`, because Go stamps a pseudo-version or `+dirty` there, and those are ignored (`resolveVersion` in `cmd/kirsch/main.go`). That `go install` path is covered by `TestResolveVersion`, which feeds version strings to the function; no real `go install` from a tag has been run.

`--version` and `--help` are handled before workspace detection and config loading, so both work outside a Git repository. The TUI shows the same resolved version.

Before this change, `--help` and `-h` printed the flag package's default usage and exited 1 (`kirsch: flag: help requested`), and `--version` was an undefined flag (exit 1). Reported by @hicks in mission-20261026-01, step 1, from a build of the pre-change code; not re-run since. The current status is the one the checks below verify (`plan/milestones/milestone-5.md:494` called it unverified).

Now `--help` and `-h` go to stdout and exit 0. `kirsch resume --help` also exits 0. An unknown flag prints the error and usage on stderr and exits 1. `kirsch help` (no dashes) is still an unknown command; the milestone does not ask for it.

The help text, exactly:

```text
Usage:
  kirsch [flags]
  kirsch resume [flags]

Commands:
  resume    resume the most recent session for this workspace

Flags:
  --workspace <path>  workspace root (defaults to the enclosing Git repository)
  --new               start a fresh session even when a previous session exists
  --debug             write a structured debug log
  --version           print the version and exit
  --help, -h          show this help and exit
```

**Note (2026-10-28, m5-d3a):** the help text above is the text from m5-d2. m5-d3a adds a `kirsch doctor` line under Usage and a `doctor` line under Commands, so the text printed now differs from the block above. The current help text is shown under *What landed* in §7, the m5-d3a section. The expectations below that quote the old text are kept as they were at m5-d2 and are stale for the help screen as it is now.

**Waits on:** O4, recorded (`plan/PROGRESS.md:346`), from `plan/milestones/milestone-5.md:98`.

**Key-state and repository checks, first.** Authored here. It must print `CONTROL PASS`, then `0`. The first `git` line must exit non-zero (`not a git repository`), which shows the scratch directory is outside any repository. The last line must print `true`: the positive control, the same command finding the workspace.

```bash
# authored here
keycount_control
keycount
git -C "$T_CFG" rev-parse --is-inside-work-tree; echo "exit=$?"
git -C "$WORKSPACE" rev-parse --is-inside-work-tree
```

Then build the binaries. `npm run build` runs `go build -o bin/kirsch ./cmd/kirsch` (`package.json:9`). Both files must be listed.

```bash
# authored here
cd /Volumes/DATA/Github/kirsch
npm run build
go build -ldflags "-X main.version=9.9.9" -o bin/kirsch-999 ./cmd/kirsch
export KIRSCH_BIN=/Volumes/DATA/Github/kirsch/bin/kirsch
ls -l bin/kirsch bin/kirsch-999
```

**Automated checks**

- [ ] `"$KIRSCH_BIN" --version` run outside a Git repository prints the version and exits 0 (`plan/milestones/milestone-5.md:498`). Run from a directory outside any Git repository. `$T_CFG` is one. The milestone does not name the directory.

  ```bash
  # authored here
  cd "$T_CFG"
  "$KIRSCH_BIN" --version
  echo $?
  ```

  Expected: `kirsch 0.1.0-dev`, then `0`, on a checkout with uncommitted changes or not sitting exactly on a `vX.Y.Z` tag. On a clean checkout exactly at tag `vX.Y.Z`, Go stamps that tag and the line reads `kirsch X.Y.Z`.

- [ ] A build with `-ldflags "-X main.version=9.9.9"` prints `9.9.9`, with no leading `v` (`plan/milestones/milestone-5.md:499`). The build command is the one in the build block above, which made `bin/kirsch-999`:

  ```bash
  # authored here
  cd "$T_CFG"
  /Volumes/DATA/Github/kirsch/bin/kirsch-999 --version
  echo $?
  ```

  Expected: `kirsch 9.9.9`, then `0`. No leading `v`.

- [ ] `"$KIRSCH_BIN" --help` lists the commands and the flags, and `echo $?` after it prints 0 (`plan/milestones/milestone-5.md:500–501`).

  ```bash
  # authored here
  cd "$T_CFG"
  "$KIRSCH_BIN" --help
  echo $?
  ERR=$(mktemp)
  "$KIRSCH_BIN" --help 2>"$ERR" >/dev/null; wc -c < "$ERR" | tr -d ' '
  "$KIRSCH_BIN" --nope 2>"$ERR" >/dev/null; wc -c < "$ERR" | tr -d ' '
  rm -f "$ERR"
  "$KIRSCH_BIN" resume --help >/dev/null; echo $?
  "$KIRSCH_BIN" --help | grep -c doctor
  "$KIRSCH_BIN" --help | grep -c resume
  ```

  Expected, in order: the help text in *What landed*, then `0`. Then `0` (nothing on stderr for `--help`). Then a non-zero count (`517` on 2026-10-26), the control: it proves the same capture does see stderr. Then `0` (`resume --help` exits 0). Then `2` (the Usage line `kirsch doctor` and the Commands line `doctor`; at m5-d2, before m5-d3a, `doctor` was not in the help and the count was `0`). Then `2`, the control: it proves the same grep finds a word that is there. `grep -c` exits 1 when the count is 0; the printed number is what counts.

- [ ] An unknown flag exits non-zero.

  ```bash
  # authored here
  cd "$T_CFG"
  "$KIRSCH_BIN" --nope
  echo $?
  ```

  Expected, exactly:

  ```text
  flag provided but not defined: -nope
  Usage:
    kirsch [flags]
    kirsch resume [flags]

  Commands:
    resume    resume the most recent session for this workspace

  Flags:
    --workspace <path>  workspace root (defaults to the enclosing Git repository)
    --new               start a fresh session even when a previous session exists
    --debug             write a structured debug log
    --version           print the version and exit
    --help, -h          show this help and exit
  kirsch: flag provided but not defined: -nope
  1
  ```

  The error appears twice, once from the flag parser and once from `kirsch:` in `main` (`cmd/kirsch/main.go:74`), the same as before this change. The usage text in the block above is the m5-d2 text. Since m5-d3a the usage text also names `kirsch doctor`, so this block is stale for the help screen as it is now (see the note under the help text above).

- [ ] `cmd/kirsch/main_test.go` covers both flags (`plan/milestones/milestone-5.md:502`). Command:

  ```bash
  # authored here
  cd /Volumes/DATA/Github/kirsch
  go test -count=1 -v ./cmd/kirsch -run 'TestRunVersionOutsideRepository|TestRunHelp|TestRunBadFlag|TestResolveVersion|TestLdflagsVersionBuild' 2>&1 | grep -E '^(--- |ok|FAIL)'
  ```

  Expected: a `--- PASS` line for each name in the list (`TestRunVersionOutsideRepository`, `TestRunHelp`, `TestRunBadFlag`, `TestResolveVersion`, `TestLdflagsVersionBuild`) and a final `ok  	github.com/djm56/kirsch/cmd/kirsch` followed by the run time. No `FAIL`.

**Hands-on checks**

1. O4 is recorded in `plan/PROGRESS.md:346` (see *What landed*). Read it before running the binary (`plan/milestones/milestone-5.md:490`).
2. Run `"$KIRSCH_BIN" --help` from `$T_CFG` with the key variables unset (Setup), then `echo $?`. Expected exit status `0`. Record what you see under *Notes / evidence*.

**Cleanup**

`rm -f /Volumes/DATA/Github/kirsch/bin/kirsch-999` (optional; `bin/` is git-ignored).

**Notes / evidence**

_Empty. Fill at run._

---

## 3 · Persist `/approvals` clear (m5-d5)

**Context:** Terminal: Setup. Binary: `go test` for the automated checks; `kirsch --workspace` and `kirsch resume --workspace` for the hands-on step, which the crew did not run (see *Hands-on checks*). Working directory: /Volumes/DATA/Github/kirsch. Workspace: `--workspace "$WORKSPACE"` on the hands-on launches: the launch that makes the grant, and the resume. Key state: all key variables unset for the automated checks and the key-state check; the hands-on step is run by the operator, in the operator's own terminal, with a key. Config state: Setup, unchanged.

**What landed**

Clearing approvals with `/approvals` is now saved in the session log. The confirmed clear writes an `approval.scopes_cleared` line (`internal/session/event.go:342`, `marshalApprovalScopesCleared`), and `App.ClearGrants` (`internal/app/app.go:530–547`) flushes that line to disk straight away, so a quit right after the clear cannot lose it. On resume, `LoadGrants` (`internal/session/grants.go:28`, rule in the doc comment at `:22–23`) discards every grant read before the clear line. Grants made after the clear still come back. The spec's session-events block lists the line (`plan/spec/kirsch-plan.md` §7, the JSONL block).

Things to know:

- `/new` writes no event. It clears the grants in memory (`internal/app/app.go:1010`), and it starts a new session file, which needs no clear line (the `O11` row, part (b)).
- With no session file open, a clear writes nothing and creates no file (`App.ClearGrants` checks for a nil store at `app.go:539`, and its comment at `:536–538` gives the reason).
- The flush runs on the UI event loop, inside the confirmed `/approvals` clear. A slow disk can pause the screen briefly, and only at a confirmed clear. This is an accepted trade, recorded by Bishop.

**Waits on:** nothing. O11 is recorded: the event name `approval.scopes_cleared` and `/new` writes no event. The `O11` row (part (b), which reads "Whether `/new` records one too. The milestone's own note says no, because `/new` starts a new file.") and the `O11 (a)` row (which reads "`approval.scopes_cleared`, the milestone's own example") are in *Milestone 5 — decisions recorded*, `plan/PROGRESS.md`. ADR 0006 makes revocability a requirement (`plan/adr/0006-session-scoped-approvals.md:45, :67–69`).

**Key-state check, first.** Authored here. Run it before any kirsch launch in this section. It must print `CONTROL PASS`, then `0`.

```bash
# authored here
keycount_control
keycount
```

**Automated checks**

- [✅] `go test -race ./internal/session/ ./internal/app/` passes in full, with no exclusion. `TestRunTurnApprovalToolUsesApprovalPath` is included (`plan/milestones/milestone-5.md:732`). Run by the crew in mission-20261028-03 (step 5) with `-count=1` added; each package reports `ok` (output under *Notes / evidence*).

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -race -count=1 ./internal/session/ ./internal/app/
  ```

- [✅] A round-trip test grants a command, clears the grants, reopens the session log, and asserts that no grants are returned (`plan/milestones/milestone-5.md:727`). The test is `TestResolveSessionGrantThenClearNotRestored` (`internal/app/resume_integration_test.go:496`). It first runs a control with no clear, and asserts the grant does come back there. Command:

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -count=1 ./internal/app/ -run 'ClearGrants|ResolveSessionGrantThenClear' -v
  ```

- [✅] A clear writes the new event into the session log, and the test reads the file (`plan/milestones/milestone-5.md:728`). The test is `TestRecordScopesClearedWritesEvent` (`internal/session/grants_test.go:174`), which reads the log and looks for `"type":"approval.scopes_cleared"`. Command:

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -count=1 ./internal/session/ -run 'LoadGrants|RecordScopesCleared' -v
  ```

- [✅] `LoadGrants` returns the grants before the clear event and none after it (`plan/milestones/milestone-5.md:729`). The milestone's wording is ambiguous on this point. The tests pin what ADR 0006 needs: grants recorded before the clear are dropped, and grants recorded after it are kept. `TestLoadGrantsClearLeavesNone` and `TestLoadGrantsClearThenGrantKeepsOnlyLater` (`internal/session/grants_test.go:148, :135`) pin those cases. Command: the one above.
- [✅] A session log with no clear event restores its grants as before (`plan/milestones/milestone-5.md:730`). The test is `TestLoadGrantsWithoutClearEventUnchanged` (`internal/session/grants_test.go:161`). Command: the one above.
- [✅] The §7 amendment lists the new event type (`plan/milestones/milestone-5.md:731`). The line is `{"v":1,"type":"approval.scopes_cleared"}` in the §7 JSONL block of `plan/spec/kirsch-plan.md:353`, and it matches what `marshalApprovalScopesCleared` writes (`internal/session/event.go:342`). Command:

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep 'approval.scopes_cleared' plan/spec/kirsch-plan.md
  ```

  The review of the amendment is the separate `@apone` step the milestone names (milestone step 4), reviewed in mission-20261028-03 (step 6 and its fix rounds). This tick records the grep and the comparison, not that review.

**Hands-on checks**

1. Launch `kirsch --workspace "$WORKSPACE"`. The launch that makes the grant needs the flag: it is where the grant is made, and without the flag the checkout becomes the workspace. Grant a command with `a`. Run `/approvals` and confirm the grant is listed. That is the positive control, and it shows the grant exists before the clear. If nothing is listed, stop: the run would prove nothing. Clear with `/approvals`, quit, run `kirsch resume --workspace "$WORKSPACE"`, then run `/approvals`. Expected after the clear and the resume: no grants listed (`plan/milestones/milestone-5.md:733`, operator). The session directory is keyed by the workspace hash (`internal/session/layout.go:78–79`), so the launches must name `$WORKSPACE`. A resume without the flag reads a different directory and lists nothing, which would pass for the wrong reason.

   > **Not run by the crew.** A grant is made when the agent asks to run a command and the operator answers the approval card with `a`. The agent asks inside a live model turn, which needs a provider key. The test setup (`scripts/m5-test-env.sh`) unsets the key variables, and the crew did not set one. The operator runs this step in their own terminal, with the key set there, and runs the key-state check above first. The expected result is the one stated in step 1. Until the operator has run it, this step is unverified; `milestone-5.md:733` stays unticked.

   > **Key-free route tried (mission-20261028-03 step 6a, W5): no key-free route found in a 25-minute attempt.** What was tried. Reading the grant path: a grant is written only by `App.Resolve`, which calls `store.RecordScopeGranted` (`internal/app/app.go:2245`) when the operator answers an approval card. The card is raised only by `App.Request`, which the agent's tool calls reach (`internal/tool/run_command.go:129`, `internal/tool/apply_patch.go:128`). So a grant needs a model turn, and so a provider key. The only fake provider (`internal/provider/fake.go`) serves the tests and cannot be selected from the binary or the config. Also tried: a session log written by hand, with an `approval.scope_granted` line and an `index.json` entry keyed by the workspace's resolved path. Kirsch read the entry, then appeared to wait for terminal replies (`]11;?`, `[6n`) that the non-interactive `script` run never sent, so nothing rendered and `/approvals` could not be driven. Whether a reply sender would let it render was not established. Not tried: a local endpoint (`doc/configuration.md` allows a custom endpoint with a `http://127.0.0.1…` `base_url`) returning a canned `run_command` tool call, with a dummy key value. That is not run now. The check stays with the operator's live run (`milestone-5.md:733`), and is checked at the final release check (section 37 of the checklist). Evidence: the W5 section of `.claude/memory/workspace/m5d5-step5-evidence.txt`.

**Notes / evidence**

Crew run: mission-20261028-03, step 5, 2026-10-28. Shell for every run below: zsh 5.9, after `source scripts/m5-test-env.sh`, which printed `Ready.`. The full output is in `.claude/memory/workspace/m5d5-step5-evidence.txt`. If the file has moved, it is under `.claude/memory/workspace/`. To find it, run `find .claude/memory/workspace -name m5d5-step5-evidence.txt`.

Key-state check (zsh, after source):

```
CONTROL PASS: a planted key line counts as 1
0
```

`keycount_control` exited 0. `keycount` printed `0` and exited 1. That exit comes from `grep -c` finding no match, the point recorded as S-4 in the testing review; the printed value is what counts.

Session package, `go test -count=1 ./internal/session/ -run 'LoadGrants|RecordScopesCleared' -v` (the `=== RUN` lines are left out; every `--- PASS` line is kept):

```
--- PASS: TestLoadGrantsReadsScopeGrantedEvents (0.02s)
--- PASS: TestLoadGrantsStopsAtTornLine (0.01s)
--- PASS: TestLoadGrantsEmptyFile (0.00s)
--- PASS: TestLoadGrantsMissingFile (0.00s)
--- PASS: TestLoadGrantsClearThenGrantKeepsOnlyLater (0.01s)
--- PASS: TestLoadGrantsClearLeavesNone (0.01s)
--- PASS: TestLoadGrantsWithoutClearEventUnchanged (0.01s)
--- PASS: TestRecordScopesClearedWritesEvent (0.01s)
PASS
ok  	github.com/djm56/kirsch/internal/session	0.223s
```

App package, `go test -count=1 ./internal/app/ -run 'ClearGrants|ResolveSessionGrantThenClear' -v`:

```
--- PASS: TestClearGrantsRoundTripAndEvent (0.12s)
--- PASS: TestClearGrantsAfterResumeBeforeAnyTurn (0.09s)
--- PASS: TestClearGrantsWithoutStoreCreatesNoSession (0.02s)
--- PASS: TestClearGrantsIsDurableWithoutClose (0.02s)
--- PASS: TestResolveSessionGrantThenClearNotRestored (0.09s)
PASS
ok  	github.com/djm56/kirsch/internal/app	0.537s
```

Race, `go test -race -count=1 ./internal/session/ ./internal/app/`, exit 0:

```
ok  	github.com/djm56/kirsch/internal/session	6.373s
ok  	github.com/djm56/kirsch/internal/app	98.164s
```

Hands-on step 1 was not run, and no no-model substitute was found. `/approvals` is a slash command inside the TUI (`internal/tui/update.go:1471`). `kirsch resume` runs `ResumeLastSession` as the app's startup hook (`chooseSessionInit` in `cmd/kirsch/main.go`), and the app is the TUI. Nothing outside the TUI prints the grants, so no command-line check can show the grants a resume restores. Nothing in this section was run against a live provider.

---

## 4 · Terminal capability and resize tests (m5-d9)

**Context:** Terminal: Setup for the hands-on step (`source scripts/m5-test-env.sh` run in that terminal); the automated checks need no Setup folders. Binary: `go test` for the automated checks; `bin/kirsch`, from `npm run build`, for the hands-on step. Working directory: /Volumes/DATA/Github/kirsch for the automated checks; the Setup practice project (`$WORKSPACE`) for the hands-on step. Workspace: `$WORKSPACE` for the hands-on step; n/a for the checks. Key state: all key variables unset (Setup clears them, and no step calls a model). Config state: Setup, unchanged (kirsch reads the Setup settings through `XDG_CONFIG_HOME`).

**What landed**

- `DetectCaps` (`internal/tui/styles.go:44–63`) turns colour off when `NO_COLOR` is set to a non-empty value, when `TERM` is `dumb` or empty, or when stdout is not a character device. An empty `NO_COLOR` is treated as unset, following the no-color.org convention. Before m5-d9 the code turned colour off for an empty value.
- New tests in `internal/tui/styles_test.go`: `TestDetectCapsColour` (the positive control, each colour-off case, and the empty-`NO_COLOR` case), `TestDetectCapsNoColourUnsetRestores` (a truly unset `NO_COLOR`, and the earlier value restored afterwards), and `TestDetectCapsUnicode` (the locale order `LC_ALL`, `LC_CTYPE`, `LANG`).
- `TestFrameInvariants` in `internal/tui/view_smoke_test.go` now also draws the help modal and a confirm prompt at every size in its sweep. At 80×24 and larger it checks that each overlay's own text is drawn and that the other overlay's text is not.

**Waits on:** nothing. O18 is recorded (2026-10-26, `plan/PROGRESS.md`, decisions table): an empty `NO_COLOR` is treated as unset. The tests above pin that choice (`plan/milestones/milestone-5.md:80`).

**Automated checks**

- [ ] `internal/tui/styles_test.go` covers each case in `plan/milestones/milestone-5.md:931–937`, and the positive control returns colour on (`plan/milestones/milestone-5.md:942`). Command: `go test -count=1 ./internal/tui -run 'TestDetectCapsColour' -v`. Expect: every subtest reads PASS, including `control:_xterm,_NO_COLOR_unset,_char_device` (`-v` prints the spaces as underscores), and the last line starts `ok`.
- [ ] The truly-unset `NO_COLOR` case passes, and the test asserts that the prior value is restored (`plan/milestones/milestone-5.md:943`). Command: `go test -count=1 ./internal/tui -run 'TestDetectCapsNoColourUnsetRestores' -v`. Expect: `--- PASS: TestDetectCapsNoColourUnsetRestores` and its `unset` subtest PASS. The test fails if the earlier value is not put back.
- [ ] The O18 behaviour is pinned by the empty-`NO_COLOR` case (`plan/milestones/milestone-5.md:944`). Command: `go test -count=1 ./internal/tui -run 'TestDetectCapsColour/NO_COLOR_empty' -v`. Expect: `NO_COLOR_empty_is_treated_as_unset_(O18)` PASS.
- [ ] (optional) The invariants sweep also runs with a modal and a confirm prompt open (`plan/milestones/milestone-5.md:945`). Command: `go test -count=1 ./internal/tui -run 'TestFrameInvariants' -v`. Expect: `--- PASS: TestFrameInvariants` and the last line starts `ok`.

Every check above in one run: `go test -count=1 ./internal/tui -run 'TestDetectCaps|TestFrameInvariants' -v`. The last line starts `ok`. That run also includes `TestDetectCapsUnicode`, which no check above names.

**Hands-on checks**

**Key-state check, first.** Authored here. Run it before any kirsch launch in this section. It must print `CONTROL PASS`, then `0`.

```bash
# authored here
keycount_control
keycount
```

1. (operator) A real resize with a modal open (`plan/milestones/milestone-5.md:946`).

   - Terminal: a Terminal window at least 100 columns and 30 rows, in the Setup terminal, from the kirsch folder: `"$KIRSCH_BIN" --workspace "$WORKSPACE"`.
   - Press Tab once (focus moves from the typing line to the card list; the bottom hint changes). Then press `?`.
   - Expected at 100 columns: a box whose top border reads `help`, whose rows list `composing`, `browsing`, `approval`, `modal` and `commands`, and whose bottom row ends `Esc or ? closes`.
   - Resize the window narrower, to about 60 columns, keeping it at least 30 rows tall. Expected: the box is still drawn, its top border still reads `help`, its bottom row still ends `Esc or ? closes`, and no line runs past the window edge.
   - Resize the window wider again. Expected: the box is drawn at the wider width, with the same text.
   - Press Esc: the box closes. Press Esc again: focus returns to the typing line. Type `/quit` and press Enter.
   - Trace: the title is `Title: "help"` from `internal/tui/update.go:959` (also `:993` and `:1459`); the footer text is built in `internal/tui/modal.go:294–329` (`modalExit` at `:333–338`). `Esc or ? closes` is not the shortest of its variants: `modal.go:304` and `:311` use `key` alone, which is the bare `Esc or ?` set at `modal.go:335`, and is shorter. The footer ends in `Esc or ? closes` at 100 and 60 columns in the dry run (verified at those sizes only). Dry run for this step (expect in a pseudo-terminal, screens replayed through a terminal emulator at 100×30 and 60×30, including a resize from 100×30 to 60×30 with help open): `.claude/memory/workspace/m4-step15-dryrun.txt`. That run did not use Terminal.app; the operator's run is the real check.

**Notes / evidence**

Operator run, 2026-10-27: every step of section 4 in the plain checklist passed. One note is open and not investigated: in step 2 (kirsch under `script` with `NO_COLOR=1`), each typed key left a new copy of the typing line on screen instead of redrawing it in place. The carried item in `plan/PROGRESS.md` (*Carried from Milestone 5 planning*) records the possible causes.

---

## 5 · `doc/configuration.md` (m5-d13a)

**Context:** Terminal: Setup. Binary: `bin/kirsch`, built from this checkout by `npm run build`, for hands-on check 3; none for the automated checks, which grep `doc/`. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

`doc/configuration.md` (new) is the settings guide, written for a user. It covers where the global and project config files live and the order the layers apply; that a project file may set only `context.project_files`; that API keys come only from the variables named by `api_key_env`; a key reference for each `[provider]`, `[policy]`, `[context]`, `[session]` and `[telemetry]` setting, with its default, valid values, scope and effect; the environment variables Kirsch reads; the debug log; the startup checks; and worked examples. It says once that v0.1 reads no configuration from command-line flags (`doc/configuration.md:41–42`), and that `--new`, `--debug` and `--workspace` change one run and set no key (`:9–11`).

**Code facts the review uncovered.** Each was read again against the code at step 7. The output is in the evidence file.

- **Timeout range.** Valid values run from 1 to 300. Startup refuses only a value of zero or less (`internal/config/config.go:652`). The 300 ceiling applies to each command (`internal/tool/run_command.go:147–160`), and a command that names no timeout takes the configured default first. A default above 300 therefore passes startup, and every command without its own timeout is then refused. The page says so in the policy section.
- **Command approval.** `ForCommand` (`internal/policy/policy.go:91–122`) decides in this order: shells and runners always ask, matched by file name (`policy.go:258–266`); the built-in allowlist runs exact matches only (`policy.go:75–84`); `require_approval_for_commands` allows the rest when it is false; session grants apply last.
- **File-change approval.** `require_approval_for_patches` is passed to `policy.New` (`internal/app/app.go:500`), but `ForPatch` (`internal/policy/policy.go:128`) has no caller outside the tests, and the patch approval request (`internal/tool/apply_patch.go:130`) is made whatever the setting. Every file change asks. The page says so (`doc/configuration.md:154–157`).
- **Warnings.** Config warnings go to `log.Warn` (`log.Warn("config", …)` in `run`, `cmd/kirsch/main.go`). That logger is disabled unless the debug log is on (the `debug` setting in `run`), so a warning about an ignored project key or a refused context file is shown nowhere by default. The page says so (`doc/configuration.md:208–211`).
- **Credential names.** `checkSecrets` runs whenever a config file is read (`internal/config/config.go:326`, `:376`). It refuses a key whose last name part matches `(?i)(api_?key|token|secret|password)` (`config.go:165`, `:468`), and a refusal stops startup. So `max_tokens` stops startup. Hands-on check 3 shows it.
- **Upper-case env strip.** `buildEnvironment` removes names ending in `_TOKEN`, `_KEY` or `_SECRET`, and names starting with `AWS_` (`internal/tool/run_command.go:398–401`, using `strings.HasSuffix` and `strings.HasPrefix`). The match is case-sensitive. `TestRunCommandLowerCaseSecretNamesPassThrough` (`internal/tool/run_command_test.go:601`) pins that a lower-case name passes through. The page's env row says so, added at step 7.

**Review history.** `@apone` reviewed the page against the code during mission-20261027-06. The mission journal, `.claude/memory/state/FLIGHT-RECORDER.md`, records the review history.

**Waits on:** O6, recorded on 2026-10-26: `@lambert` writes, and `@apone` reviews the claims against the code (`plan/PROGRESS.md`, Milestone 5 decisions).

**Automated checks**

- [ ] Every key in `internal/config/config.go:57–80` and every endpoint field in `config.go:219–259` is listed, with its default and valid values (`plan/milestones/milestone-5.md:1143`). The mechanical check is m5-d13b (its section below). The key-name cross-check below checks that each name taken from `config.go` (the struct tags, and the endpoint cases at `config.go:219–259`) appears on the page, as a backticked `.name` or as a `[section]` header. It does not check that the list is complete, and it does not check defaults. The defaults and valid values were compared by hand in hands-on check 2.

  ```bash
  # authored here
  cd /Volumes/DATA/Github/kirsch
  KEYS=$(mktemp)
  keycheck() {
    local page="$1" k n=0 bad=0
    { grep -o 'toml:"[a-z_]*"' internal/config/config.go | sed 's/toml:"//; s/"$//'
      sed -n '219,259p' internal/config/config.go | grep -o 'case "[a-z_]*":' | sed 's/case "//; s/":$//'; } | sort -u > "$KEYS"
    while IFS= read -r k; do
      n=$((n + 1))
      case " provider policy context session telemetry " in
        *" $k "*) grep -q -F "[$k]" "$page" || { echo "MISSING section [$k]"; bad=$((bad + 1)); } ;;
        *) grep -q -F ".$k\`" "$page" || { echo "MISSING key .$k"; bad=$((bad + 1)); } ;;
      esac
    done < "$KEYS"
    echo "names checked: $n, missing: $bad"
  }
  keycheck doc/configuration.md
  grep -q -F '`provider.default`' doc/configuration.md && echo "provider.default present"
  # control: the session.auto_resume row removed from a copy; must print MISSING key .auto_resume
  grep -v -F '`session.auto_resume`' doc/configuration.md > "$KEYS.control"
  keycheck "$KEYS.control"
  ```
- [ ] `grep -n -i 'flag' doc/configuration.md` prints at least one line. Every line it prints is a sentence that promises no config flag (`plan/milestones/milestone-5.md:1144`). Paste the output.

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n -i 'flag' doc/configuration.md
  ```

- [ ] `grep -n -i -w -E 'milestones?|ADRs?' doc/configuration.md` prints nothing. The milestone's own pattern, `'milestone|ADR'` (`plan/milestones/milestone-5.md:1145`), misses plurals, and the control below shows it. Control: the printf line must print `an ADR`, `ADRs` and `milestones`, and not `headroom`. The old pattern on the same input prints only `an ADR`.

  ```bash
  # authored here
  cd /Volumes/DATA/Github/kirsch && grep -n -i -w -E 'milestones?|ADRs?' doc/configuration.md
  printf 'an ADR\nADRs\nmilestones\nheadroom\n' | grep -i -w -E 'milestones?|ADRs?'
  printf 'an ADR\nADRs\nmilestones\nheadroom\n' | grep -i -w -E 'milestone|ADR'
  ```

**Hands-on checks**

**Key-state check, first.** Authored here. Run it before any kirsch launch in this section. It must print `CONTROL PASS`, then `0`.

```bash
# authored here
keycount_control
keycount
```

1. Read the page as a user, not a builder. Confirm that `KIRSCH_DEBUG` and the config path are present (`plan/milestones/milestone-5.md:1136`), and that API keys are read from the variables named by `api_key_env` (`plan/milestones/milestone-5.md:1138`). The config path is on line 22, `KIRSCH_DEBUG` on line 193, and the secrets rule on lines 63 and 71–73.

   ```bash
   # authored here
   cd /Volumes/DATA/Github/kirsch && grep -n 'KIRSCH_DEBUG\|XDG_CONFIG_HOME\|api_key_env' doc/configuration.md
   ```

2. Compare each documented default with `Defaults()` (`internal/config/config.go:84–118`). Every default on the page matches `Defaults()`: the built-in endpoints, the policy and context values, the storage folder, `auto_resume` and `debug_log`. The sentence that a new endpoint defaults `thinking` and `prompt_caching` to off is not a `Defaults()` value. It rests on the comment on `EndpointConfig` and on the review.

   ```bash
   # authored here
   cd /Volumes/DATA/Github/kirsch && grep -n '^| `' doc/configuration.md | cut -c1-90
   sed -n '84,118p' internal/config/config.go | grep -E 'Default:|BaseURL|Auth:|APIKeyEnv|Model:|DefaultCommandTimeout|RequireApproval|AllowSession|EnvPassthrough|ProjectFiles|MaxProject|StorageDir|AutoResume|DebugLog|PromptCaching'
   ```

3. A settings file with a credential-looking name stops kirsch from starting. This is the same check as operator checklist step 4. It needs `bin/kirsch` built from this checkout (`npm run build`) and the test setup (`source scripts/m5-test-env.sh`). The command writes a throwaway file in a new folder and leaves the settings file alone.

   ```bash
   # authored here
   ( d=$(mktemp -d); mkdir -p "$d/kirsch"; printf '[provider]\nmax_tokens = 1\n' > "$d/kirsch/config.toml"; XDG_CONFIG_HOME="$d" "$KIRSCH_BIN" --workspace "$WORKSPACE" ; echo "result: $?"; rm -f "$d/kirsch/config.toml"; rmdir "$d/kirsch" "$d" )
   ```

   Expected: a message that names the key `provider.max_tokens` as one that looks like a credential, and `result: 1`. Control: the same file with `max_steps` in place of `max_tokens` does not print the credential message. `max_steps` is not a Kirsch key, so config ignores it as an unknown key, and the control shows the name rule, not a real setting. Run with `</dev/null` and no controlling terminal, the control exits 1 with `could not open a new TTY: open /dev/tty: device not configured`. That message comes from the missing terminal, not from the config file. This control was not run in a Terminal window. The refusal comes from the name, not the file.

**Notes / evidence**

Run 2026-10-27 in mission-20261027-06, step 7, by `@lambert`. The key cross-check and the mechanical checks ran under bash; the operator-checklist run ran in zsh after `source scripts/m5-test-env.sh`; the evidence file shows each command as run. The verbatim output of every run is in `.claude/memory/workspace/archive-mission-20261027-07/m6-step7-evidence.txt`. If that file has moved again, it is under `.claude/memory/workspace/`, found with `find .claude/memory/workspace -name m6-step7-evidence.txt`. The short results are:

- Word check, corrected pattern: no output, exit status 1. Control: `an ADR`, `ADRs`, `milestones`. The old pattern on the same control input prints only `an ADR`.
- Flag grep: lines 9, 41 and 42, each saying that v0.1 reads no configuration from command-line flags.
- Key cross-check: no `MISSING` line on the page. Control with the `session.auto_resume` row removed: `MISSING key .auto_resume`.
- Hands-on check 3: `result: 1`, and the credential message for `provider.max_tokens`. Control with `max_steps`, run with no controlling terminal: no credential message, then `could not open a new TTY: open /dev/tty: device not configured` and `result: 1`.
- Tests: `TestRunCommandLowerCaseSecretNamesPassThrough` and `TestRunCommandAllStripPatterns` pass.

The step 7 control run left one throwaway folder, `/var/folders/nb/46t8bdcd3050lnvz_6066k_40000gn/T/tmp.QbML9dtjkf`, holding one file, `kirsch/config.toml`. The operator removes it, after checking the path, with `rm -f /var/folders/nb/46t8bdcd3050lnvz_6066k_40000gn/T/tmp.QbML9dtjkf/kirsch/config.toml`, then `rmdir /var/folders/nb/46t8bdcd3050lnvz_6066k_40000gn/T/tmp.QbML9dtjkf/kirsch /var/folders/nb/46t8bdcd3050lnvz_6066k_40000gn/T/tmp.QbML9dtjkf`.

---

## 6 · Safety-test repair for `run_command`: C30, C35, C36 (m5-d19a)

**Context:** Terminal: Setup. Binary: `go test` for the automated checks. The hands-on steps need a live model turn and were not run for m5-d19a (see *Hands-on checks*). Working directory: /Volumes/DATA/Github/kirsch. Workspace: `--workspace "$WORKSPACE"` on the hands-on launch. Key state: all key variables unset. Config state: Setup, plus the `[policy]` block that hands-on step 1 appends, which step 3 removes again, so the Setup config is restored before the section ends. The guard reruns after the append and after the restore.

**What landed**

- `internal/tool/run_command_test.go`: the child's stdin is checked to be `/dev/null` (`TestRunCommandStdinIsDevNull`). A timed-out command and a cancelled command must each leave no grandchild process running (`TestRunCommandTimeoutProcessGroupDead`, `TestRunCommandCancellationProcessGroupDead`). A new test checks that lower-case names such as `my_token`, `api_key` and `db_secret` pass through, because the match is case-sensitive (`TestRunCommandLowerCaseSecretNamesPassThrough`).
- `internal/tui/approval_unspecified_test.go` (new file): `TestApprovalUnspecifiedCardShowsRequestName` checks that the approval card shows the request's name. `TestApprovalUnspecifiedCardBlankFieldsKnownDefect` pins the current blank card for an unspecified operation and is marked as a known defect.
- No production code changed.

**Mutant check, recorded at step 3.** Each mutant was applied to a throwaway overlay, not the working tree, and the named test failed under it. These were not rerun in step 5. The verbatim output is in `.claude/memory/workspace/m5-step3-evidence.txt`, which is scratch and is archived at the next mission start.

| Test | Mutant | Failing output (verbatim) |
|---|---|---|
| `TestRunCommandStdinIsDevNull` | `cmd.Stdin = devNull` replaced by an empty reader | `run_command_test.go:233: child stdin is not /dev/null; probe printed "STDIN_NOT_DEVNULL\n"` |
| `TestRunCommandTimeoutProcessGroupDead` | the signal goes to `pid`, not the group `-pid` | `run_command_test.go:922: still alive 3s after the command ended: grandchild sleep(pid 83329)` |
| `TestRunCommandCancellationProcessGroupDead` | `Setpgid: true` set to `false` | `run_command_test.go:963: still alive 3s after the command ended: grandchild sleep(pid 83373)` |
| `TestRunCommandLowerCaseSecretNamesPassThrough` | the key is upper-cased before the strip test | `run_command_test.go:622: lower-case variable "my_token" was stripped; matching is meant to be case-sensitive` |
| `TestApprovalUnspecifiedCardShowsRequestName` | `Description` dropped from the sent request | `approval_unspecified_test.go:94: message Description = "", want "reticulate splines"` |
| `TestApprovalUnspecifiedCardBlankFieldsKnownDefect` | the unspecified branch fills `Subject` and `Detail` | `approval_unspecified_test.go:140: Subject = "unspecified operation"; pinned empty (known defect) - if filled, update this test` |

Under the lower-case mutant, `TestRunCommandAllStripPatterns` still passed. The lower-case test is the one that catches it. Cite each test by its function name; the line numbers inside the pasted failure lines above are from `internal/tool/run_command_test.go` as it stood when the mutants ran, before a formatting fix shifted the file by two lines.

**Open product defect (C36, not fixed here).** An approval request for an unspecified operation is sent with an empty subject and detail (`internal/app/app.go`, near line 2140). The card then shows blank rows and a `[d] view diff` action with no diff behind it. After the request is resolved, the collapsed line shows an empty subject (`internal/tui/cards.go:263`, read from the code, not run). It is not established whether a user can reach this path: `internal/policy/policy.go:27` says the zero value "must refuse", and `internal/app/app.go:2141` says it "should not occur in practice". Fixing it changes `internal/app/app.go`, so it is outside m5-d19a.

**Waits on:** O13 (scheduling of the safety group). No other dependency (`plan/milestones/milestone-5.md:123`).

**Key-state check, first.** Authored here. Run it before any kirsch launch in this section. It must print `CONTROL PASS`, then `0`.

```bash
# authored here
keycount_control
keycount
```

**Automated checks**

- [ ] Each C30 test fails under its named mutant and passes on the real code (`plan/milestones/milestone-5.md:1557`). The tests are `TestRunCommandTimeoutProcessGroupDead`, `TestRunCommandCancellationProcessGroupDead` and `TestRunCommandStdinIsDevNull`. Command (real code): `go test -count=1 ./internal/tool -run 'TestRunCommandStdinIsDevNull|TestRunCommandAllStripPatterns|TestRunCommandLowerCaseSecretNamesPassThrough|TestRunCommandTimeoutProcessGroupDead|TestRunCommandCancellationProcessGroupDead' -v`. This is the same command the operator runs in section 6 of the plain checklist, so it also runs the secret-name tests in the next box. The mutant for each test is in the table under *What landed*.
- [ ] The lower-case cases assert that `my_token`, `api_key` and `db_secret` are not stripped, and the suite passes (`plan/milestones/milestone-5.md:1558`). Command: `go test -count=1 ./internal/tool -run 'TestRunCommandLowerCaseSecretNamesPassThrough|TestRunCommandAllStripPatterns' -v`.
- [ ] The C36 test asserts that the blank approval card for an unspecified operation shows the operation name and no empty field (`plan/milestones/milestone-5.md:1559`). Command: `go test -count=1 ./internal/tui -run 'TestApprovalUnspecifiedCard' -v`. **This box is not met by the product.** The card still has empty fields for an unspecified operation (see the open defect under *What landed*). `TestApprovalUnspecifiedCardBlankFieldsKnownDefect` pins that blank output, so it passes now and fails once the product is fixed. Leave this box unticked until then.
- [ ] `go test -count=1 ./internal/tool/ ./internal/tui/` passes (`plan/milestones/milestone-5.md:1560`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -count=1 ./internal/tool/ ./internal/tui/
  ```

**Hands-on checks**

_Not run for m5-d19a._ These steps need a live model turn to call `run_command`, and steps 1 and 3 change the config, which step 3 restores. No live run was done. The automated checks above cover the stripping rules with no model in the loop. They do not show what a model-driven command receives, so the `MY_TOKEN` check below remains unrun.

1. Run `run_command` with `MY_TOKEN` exported in the kirsch process environment and listed in `[policy] env_passthrough`. The TOML section is `policy` (`internal/config/config.go:29`), and the key is `env_passthrough` (`internal/config/config.go:62`). Set `LANG` in the environment too (`plan/milestones/milestone-5.md:1562`). Check that `LANG` is set: `echo "$LANG"` must print a value.

   ```bash
   # authored here
   export MY_TOKEN=dummy-token-for-test
   [ -e "$T_CFG/config.setup.bak" ] || [ "$(grep -c '^\[policy\]' "$T_CFG/kirsch/config.toml")" != 0 ] || cp "$T_CFG/kirsch/config.toml" "$T_CFG/config.setup.bak"
   grep -q '^\[policy\]' "$T_CFG/kirsch/config.toml" || printf '\n[policy]\nenv_passthrough = ["MY_TOKEN"]\n' >> "$T_CFG/kirsch/config.toml"
   grep -c '^\[policy\]' "$T_CFG/kirsch/config.toml"
   guard
   ```

   The backup is taken only when no backup exists and the config has no `[policy]` table, so a rerun cannot save a policy-bearing config as the backup. The append runs only when no `[policy]` table exists. A second run would write a second table, which BurntSushi/toml rejects, so kirsch exits 1 and fails closed. The count must print `1`, and `guard` must print `GUARD PASS`. The backup made first is what step 3 restores.

2. Expected: the child sees `LANG`, and does not see `MY_TOKEN` even though it is listed (`internal/tool/run_command.go:379–384, :386–401`). The `LANG` arrival is the control. An absent `MY_TOKEN` alone proves nothing.

   > **Not run for m5-d19a.** Invoking the child command needs a live model turn, and none was run. `milestone-5.md:1562` does not say how to do it without one. Do not run this step with any key variable set.

3. Restore the Setup config, so the later sections see the state their Context lines name. Authored here.

   ```bash
   # authored here
   cp "$T_CFG/config.setup.bak" "$T_CFG/kirsch/config.toml"
   unset MY_TOKEN
   grep -c '^\[policy\]' "$T_CFG/kirsch/config.toml"
   printf '[session]\nstorage_dir = "%s"\n' "$T_STATE/sessions" | cmp - "$T_CFG/kirsch/config.toml" && echo "CMP PASS: config matches the Setup config"
   guard
   ```

   The `grep -c` line must print `0`, and the `cmp` line must print `CMP PASS`. `guard` cannot see a `[policy]` table, so the `grep -c` line is the evidence that the table is gone. `guard` must print `GUARD PASS`.

**Notes / evidence**

Automated checks run on 2026-10-27 in mission-20261027-05 (step 5), after `source scripts/m5-test-env.sh`. Every automated command above exited 0 on the real code. Output: `.claude/memory/workspace/m5-step5-evidence.txt` (scratch; archived at the next mission start). The hands-on steps were not run.

---

## 7 · Doctor: dispatch, report grammar, version line, exit codes, arch entry (m5-d3a)

**Context:** Terminal: Setup. Binary: `bin/kirsch`, built from the checkout by `npm run build` (the build block in §2) and run by absolute path as `"$KIRSCH_BIN" doctor`. Working directory: `$T_CFG`, outside any Git repository, for the doctor runs; /Volumes/DATA/Github/kirsch for the build and the `go test` checks. Workspace: n/a for the doctor runs. Hands-on steps 1 and 5 pass `$WORKSPACE` as a `--workspace` argument, and the key-state block checks it with `git`. Key state: all key variables unset. Config state: Setup, unchanged. The broken-settings runs use a throwaway folder that is made and removed inside a subshell in hands-on step 1, so the Setup settings are not touched.

**What landed**

`kirsch doctor` is a subcommand. It runs after flag parsing and before workspace detection and config load, so it still prints its report outside a project folder, and when the settings file is broken. `--help` and `--version` given with `doctor` are answered by flag handling, with the help text or the version line and no report (`cmd/kirsch/main.go`, function `run`).

The report grammar and exit codes are the ones recorded as O5 and O5 detail, in *Milestone 5 — decisions recorded* in `plan/PROGRESS.md`:

- The line `kirsch <version>` is printed before any check. The version is the same version that `--version` prints.
- Each check gets a line in the form `<status>  <name>: <detail>`, with a double space after the status. The status is `ok`, `warn` or `fail`.
- The summary line is `doctor: <n> ok, <n> warn, <n> fail`.
- Exit status 0 when no check fails, including a run where some checks warn. Exit status 1 when any check fails. A failing run prints no extra `kirsch:` line, because the report already says what failed.

**Changed 2026-10-29 by m5-d3b.** Doctor now runs checks for `git`, `rg`, the repository and the settings file, after the version line. It prints the summary, and it exits 1 when a line says `fail`. Those checks are described in §12. A check does not always print exactly one line: `repository` prints none when git is missing, unless `--workspace` names a path that `workspace.Detect` rejects, and `config` prints one `warn` line for each unknown key as well as its own line. At m5-d3a there were no checks: doctor printed the version line and `doctor: 0 ok, 0 warn, 0 fail`, and exited 0. The m5-d3a instructions in `plan/milestones/milestone-5.md` still require every check line to use the same grammar, and m5-d3b does.

The help screen now reads as below. The help text in §2 is the m5-d2 text, and §2 carries a note about the change. The usage line `kirsch doctor [flags]` is the m5-d3b text, from 2026-10-29; at m5-d3a the line read `kirsch doctor`.

```text
Usage:
  kirsch [flags]
  kirsch resume [flags]
  kirsch doctor [flags]

Commands:
  resume    resume the most recent session for this workspace
  doctor    check the installation and report problems

Flags:
  --workspace <path>  workspace root (defaults to the enclosing Git repository)
  --new               start a fresh session even when a previous session exists
  --debug             write a structured debug log
  --version           print the version and exit
  --help, -h          show this help and exit
```

**Waits on:** m5-d2 (help text and flag dispatch order) and O5 (doctor design: package placement, output grammar, fail versus warn, exit codes, side-effect policy), from `plan/milestones/milestone-5.md:99`.

**Key-state and repository checks, first.** Authored here. Run it after `guard`, as the hands-on commands below are. It must print `CONTROL PASS`, then `0`. The first `git` line must exit non-zero, and the last must print `true`.

```bash
# authored here
keycount_control
keycount
git -C "$T_CFG" rev-parse --is-inside-work-tree; echo "exit=$?"
git -C "$WORKSPACE" rev-parse --is-inside-work-tree
```

**Automated checks**

- [✅] `kirsch doctor` is dispatched ahead of `workspace.Detect` and config load (`plan/milestones/milestone-5.md:544`). Test: `TestRunDoctorBeforeWorkspaceAndConfig`, which runs doctor in a folder that is not a repository with a broken settings file, and checks that the plain run fails on the same settings file. Command:

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -count=1 ./cmd/kirsch/ -run 'TestRunDoctorBeforeWorkspaceAndConfig' -v
  ```

- [✅] The version line shows the version (`plan/milestones/milestone-5.md:545`). Test: `TestRunDoctorBeforeWorkspaceAndConfig`, which checks that the first line of the report is `kirsch <version>`. (m5-d3b, 2026-10-29: the earlier test, `TestRunDoctorSkeletonReport`, pinned the whole report while no probe existed. It was removed. Its name is no longer in the list of tests in `cmd/kirsch`, so the command that named it ran no test.) Command:

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -count=1 ./cmd/kirsch/ -run 'TestRunDoctorBeforeWorkspaceAndConfig' -v
  ```

- [✅] The help output names `doctor` (`plan/milestones/milestone-5.md:546`). Test: `TestRunHelp`, which asserts the Commands entry in `cmd/kirsch/main_test.go`. Command:

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -count=1 ./cmd/kirsch/ -run 'TestRunHelp' -v
  ```

- [✅] Exit status matches O5 (`plan/milestones/milestone-5.md:547`). Tests: `TestRunDoctorExitSentinel` (a failing result returns the doctor sentinel after the report, and a result that only warns returns no error) and `TestFailed` in `internal/doctor`. Commands:

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -count=1 ./cmd/kirsch/ -run 'TestRunDoctorExitSentinel' -v
  cd /Volumes/DATA/Github/kirsch && go test -count=1 ./internal/doctor/ -run 'TestFailed' -v
  ```

  The exit status 1 itself is set in `main`. (m5-d3b, 2026-10-29: a probe can fail now. `TestDoctorBinaryExitStatus` in `cmd/kirsch` builds the binary and runs `kirsch doctor` as a subprocess, so the exit status is checked there. §12 shows it from the terminal.)

- [✅] The `internal/arch` test passes with the new package entry (`plan/milestones/milestone-5.md:548`). Test: `TestImportRules`. The milestone names `internal/arch/imports_test.go` but gives no command line; this is the command used:

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -count=1 ./internal/arch/ -v
  ```

- [✅] `plan/spec/architecture.md` names the new package (`plan/milestones/milestone-5.md:549`). Checked by grep, below. The review of that line is an `@apone` step, and this section does not claim it.

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n 'internal/doctor' plan/spec/architecture.md
  ```

**Hands-on checks**

**Changed 2026-10-29 by m5-d3b.** Any expectation below that quotes the m5-d3a report is stale, and the current results are in §12. That covers the summary counts, the exit statuses, the usage line and the `--workspace` run. Hands-on 1 now prints a `fail` line for config and exits 1 for the broken settings file, where it expected exit 0; the plain run still exits 1. Hands-on 1 also prints `fail  repository: not in a Git repository`, because `$T_CFG` is outside any repository, and its summary reads `doctor: 2 ok, 0 warn, 2 fail` (run 9 of step 3a in `.claude/memory/workspace/m3b-step3a-runs.txt`, a scratch file archived at the next mission start). The operator's section 7 note shows the same lines. Hands-on 3 now exits 1 outside a repository, because the repository check fails there. Hands-on 4 now reads `4:  kirsch doctor [flags]`. Hands-on 5 expected a `--workspace` value not to change the report; with m5-d3b it does. The steps and their expected text are kept as they were at m5-d3a. The carried item (d) in the Notes below is the m5-d3a finding. **Dated note, 2026-10-29 (mission-20261029-06, step 3b):** the summary counts quoted in this section are out of date since m5-d4a as well, because the summary counts the key, backend, model and context lines. Measured on 2026-10-29, from the kirsch folder with no key set and an empty settings folder, the summary reads `doctor: 7 ok, 1 warn, 0 fail` (`.claude/memory/workspace/m20261029-06-step3b-runs.txt`, part 4, scratch). The count outside a repository with no flag is `doctor: 5 ok, 2 warn, 1 fail` (part 6). The same applies to §11 hands-on 1 and 2, whose summary lines are older still. **Dated note, 2026-10-30 (m5-d4b, step 4g):** the summary counts quoted in the paragraph above, `5 ok, 2 warn, 1 fail` and `7 ok`, are stale and are not the current figures. The current figures from the step 4c doctor runs (`.claude/memory/workspace/m20261030-01-step4c-doctor-run.txt`) are `doctor: 9 ok, 1 warn, 0 fail` for run B, with the kirsch folder as workspace and a writable state folder, and `doctor: 5 ok, 4 warn, 1 fail` for run E, outside a repository with no flag. See the notes at `plan/testing/manual/milestone-5.md:1098` and `plan/testing/manual/milestone-5.md:1109`.

The hands-on commands below were run in zsh, after the Setup and the Guard functions. The automated commands were run in zsh with no setup, because they need none. The verbatim output is in `.claude/memory/workspace/m7d-step9-evidence.txt`. If the file has moved, it is under `.claude/memory/workspace/`, because the workspace is archived at the next mission start. To find it, run `find .claude/memory/workspace -name m7d-step9-evidence.txt`.

1. Outside a repository, with a broken settings file (authored here; it covers `plan/milestones/milestone-5.md:536`). Doctor still prints the report and exits 0. The plain run with the same settings file refuses to start and exits 1. The settings file is made in a throwaway folder inside the subshell, so the Setup settings are untouched, and the folder is removed when the subshell ends:

   ```bash
   # authored here
   cd "$T_CFG"
   ( d=$(mktemp -d); mkdir -p "$d/kirsch"; printf '[session\nstorage_dir = "%s"\n' "$T_STATE/sessions" > "$d/kirsch/config.toml"; XDG_CONFIG_HOME="$d" "$KIRSCH_BIN" doctor; echo "exit=$?"; XDG_CONFIG_HOME="$d" "$KIRSCH_BIN" --workspace "$WORKSPACE"; echo "exit=$?"; rm -f "$d/kirsch/config.toml"; rmdir "$d/kirsch" "$d" )
   ```

   Expected: the report, then `exit=0`. Then a line starting `kirsch: parse`, which names the `config.toml` in the throwaway folder and `toml: line 2: expected '.' or ']' to end table name`, then `exit=1`. The folder path differs on each run. Observed, with the folder path shortened to `…`:

   ```text
   kirsch 0.1.0-dev
   doctor: 0 ok, 0 warn, 0 fail
   exit=0
   kirsch: parse …/kirsch/config.toml: toml: line 2: expected '.' or ']' to end table name, but got '\n' instead
   exit=1
   ```

   **Dated note, 2026-10-29 (mission-20261029-06, step 3b):** the `doctor: 0 ok` line in the block above is the output of an earlier doctor that ran no checks, the version that operator section 7 describes. It is kept as recorded. The summary has counted real checks since m5-d3b, so it is no longer the current result. **Dated note, 2026-10-30 (m5-d4b, step 4e):** since m5-d4b the summary also counts the `state` and `lock` lines, so the `doctor: 0 ok` sample above, and the `doctor: 2 ok, 0 warn, 2 fail` figure in the m5-d3b notes, are out of date. This broken-settings run was not measured again after m5-d4b, so no current figure is given for it here. The run measured at step 4c outside a repository with no flag (run E, `.claude/memory/workspace/m20261030-01-step4c-doctor-run.txt`) reads `doctor: 5 ok, 4 warn, 1 fail`, with settings that read, so it is a different run.

2. From the kirsch folder, which is inside a repository (authored here):

   ```bash
   # authored here
   cd /Volumes/DATA/Github/kirsch
   "$KIRSCH_BIN" doctor
   echo "exit=$?"
   ```

   Expected: `kirsch 0.1.0-dev`, `doctor: 0 ok, 0 warn, 0 fail`, then `exit=0`. **Dated note, 2026-10-29 (mission-20261029-06, step 3b):** this count is out of date. Since m5-d3b and m5-d4a the summary counts every check line, and the current run from this folder with no key set reads `doctor: 7 ok, 1 warn, 0 fail` (`.claude/memory/workspace/m20261029-06-step3b-runs.txt`, part 4). **Dated note, 2026-10-30 (m5-d4b, step 4e):** with the `state` and `lock` lines counted, the same run with `--workspace` set to the kirsch folder, with no key set and a writable state folder, reads `doctor: 9 ok, 1 warn, 0 fail` (step 4c, run B, `.claude/memory/workspace/m20261030-01-step4c-doctor-run.txt`).

3. From `$T_CFG`, which is outside any repository (authored here). This is the same command run from the test folder:

   ```bash
   # authored here
   cd "$T_CFG"
   "$KIRSCH_BIN" doctor
   echo "exit=$?"
   ```

   Expected: the same report as step 2, then `exit=0`. The git check in the key-state block above prints its fatal message from this folder. That message is the expected outcome, and it is not a failure of doctor.

4. The help screen (authored here):

   ```bash
   # authored here
   cd "$T_CFG"
   "$KIRSCH_BIN" --help | grep -n doctor
   ```

   Expected, exactly:

   ```text
   4:  kirsch doctor
   8:  doctor    check the installation and report problems
   ```

5. Flags given with `doctor` (authored here; it checks an assumption behind the carried items below):

   ```bash
   # authored here
   cd "$T_CFG"
   "$KIRSCH_BIN" doctor --workspace "$WORKSPACE"; echo "exit=$?"
   ```

   Expected: the same report as step 2, then `exit=0`. In this run the `--workspace` value, which names an existing folder, did not change the report. The step tries an existing folder only.

**Notes / evidence**

Run for mission-20261028-03, step 9, on 2026-10-28, in zsh 5.9 from the kirsch folder, after `source scripts/m5-test-env.sh`. The `guard` function printed `GUARD PASS` before the runs and after them. The binary was built by `npm run build` (exit 0) from a checkout whose HEAD is `7244c58` with uncommitted changes, so the version line reads `kirsch 0.1.0-dev`.

- Hands-on 1 matched: the report, `exit=0`, then the refusal from the plain run and `exit=1`.
- Hands-on 2 and 3 matched: the same report in each folder, each ending with `exit=0`.
- Hands-on 4 matched: the grep lines above.
- Hands-on 5 matched the report, with `exit=0`. In that run the `--workspace` value did not change the report. The step has no control, so it does not show that the flag is ignored in general.
- The automated commands above were run in zsh, exactly as printed, with `-v`. Each `-run` command printed `--- PASS` for every test it names, and each package printed `ok`. The `internal/arch` command has no `-run`, so it ran every test in that package; `TestImportRules` and `TestGuardedPackagesAreReal` both passed. The `grep` line printed line 45 of `plan/spec/architecture.md`. Step 10a re-ran these commands, the key-state block and the hands-on commands; the command text and the output are in `.claude/memory/workspace/m7d-step9-evidence.txt`, as Block 6 (step 10a).
- Not run in m5-d3a: a failing check. No probe could fail then, so the exit status 1 from the binary was not observed. Dated note, 2026-10-29 (m5-d3b): a probe can fail now. §12 shows the exit status 1 from the terminal, and `TestDoctorBinaryExitStatus` checks it in a test.

Carried to m5-d3b. The fix-before-lifting items are in *Carried from Milestone 5 planning* in `plan/PROGRESS.md`, under the m5-d3b entry. They are named here, lettered (a) to (d) as the *Carried from Milestone 5 planning* entry in `plan/PROGRESS.md` letters them:

- *(a) The changes-nothing test* (`TestRunDoctorChangesNothing`) is vacuous as written. It needs a write mutant, controls in each root, walk errors surfaced, and the size and mtime of each file recorded.
- *(b) The forward-looking comments* on `doctorProbes` and in `run` in `cmd/kirsch/main.go`, and the comment above the `internal/doctor` entry in `internal/arch/imports_test.go`, are rewritten when the probes they describe land.
- *(c) The unknown `Status`* fails open (DIR-002). `doctor.Failed` counts only `StatusFail`, and `doctor.Report` counts no unknown status, so an unknown status is left out of the summary and does not fail the run. This is read from the code, not run.
- *(d) The suggestions*: a subprocess exit test, which runs the built binary and checks exit status 1; in the one run of hands-on 5, a `--workspace` value given with `doctor` did not change the report (that run had no control, so it does not show that the flag is ignored in general); and the usage line `kirsch doctor` has no `[flags]`; multi-line details are not handled by the line grammar; the arch mutant for `internal/doctor` has not been run.

**Status of the carried items, dated 2026-10-29 (m5-d3b, mission-20261029-03, step 3a).** The items above are kept as they were written, except item (b). Item (b) was rewritten to cite symbols instead of line numbers; its original cites were `cmd/kirsch/main.go:76–77`, `cmd/kirsch/main.go:158–160` and `internal/arch/imports_test.go:43–44`. This is what the source and the runs in `.claude/memory/workspace/m3b-step3a-runs.txt` show (scratch, archived at the next mission start):

- (a) Closed in part. `TestRunDoctorChangesNothing` now has a positive control and a comparison control in each root, and its snapshot records size, mtime and a checksum. A write mutant (`mw`) was reported by @hicks at step 1 to fail the test; no saved output; not re-run. That part stays open until a saved run exists.
- (b) Closed in `cmd/kirsch/main.go`: the comments on `doctorProbes` and in `run` now describe the probes. Open in the `internal/doctor` entry of `internal/arch/imports_test.go`, whose comment still says the probes are still to come. That stays true until m5-d4a and m5-d4b land. **Dated note, 2026-10-30 (m5-d4b, step 4c):** this has closed. The comment above the `internal/doctor` entry in `internal/arch/imports_test.go` now describes the import rule and its one exception, `internal/agent/prompt`, and does not say the probes are still to come. That was closed by m5-d4a, so the condition "until m5-d4a and m5-d4b land" is met. The text above is kept as it was written.
- (c) Closed. `Failed` and `Report` count a status outside the three known values as a failure. `TestReportCountsUnknownStatusAsFail` and `TestRunNeverReturnsUnknownStatus` cover it.
- (d) Closed, except for one item. `TestDoctorBinaryExitStatus` is the subprocess exit test. `TestRunDoctorHonoursWorkspaceFlag` covers `--workspace`. The usage line reads `kirsch doctor [flags]`. `TestRunDoctorMultiLineConfigErrorIsOneLine` covers multi-line details. Open: the arch mutant for `internal/doctor` was reported by @hicks at step 1 to fail `TestImportRules` only when the run sets `GOFLAGS=-overlay=…`. A plain `-overlay` run passes, because `TestImportRules` shells out to `go list`, which does not read the overlay. The saved runner `.claude/memory/workspace/m3b-run.sh` (scratch, archived at the next mission start) uses plain `-overlay`, so it does not reproduce the failure. No saved output of the failing run exists; not re-run.

---

## 8 · Composer paste path and caret rows (m5-d6)

**Context:** Terminal: Terminal.app and iTerm2, each after rerunning Setup in that window, for the hands-on step. The automated checks run in any terminal where `source scripts/m5-test-env.sh` has been run. Binary: `go test` for the automated checks; `"$KIRSCH_BIN"` for the hands-on step. Working directory: /Volumes/DATA/Github/kirsch. Workspace: `--workspace "$WORKSPACE"` on the hands-on launch. Key state: all key variables unset; the hands-on step sends no message, so no model is called. Config state: Setup, unchanged.

**What landed**

- `Composer.rows` scrolls when the input has more lines than the composer has rows. The composer's height limit is the clamp in `contentLines`.
- The window is worked out from the caret's line alone, with no stored scroll position. While the caret is on a line in the top window, the window shows the top lines. Past that, the window shows the lines ending on the caret's line, so the caret sits on the bottom row and the window moves by as many lines as the caret does. With no caret (unfocused, or busy) the window shows the top lines.
- The `>` prompt shows only while line 0 is in the window. A full window drops the dim hint, so the caret row is never what gives way.
- `view.go` is unchanged. O10 chose scroll, not wrap.
- The tests are in `internal/tui/paste_path_test.go`: `TestPasteLandsVerbatim`, `TestLargePasteConfirmAcceptAndDecline`, `TestMultiLinePasteStartingWithSlashIsNotACommand`, `TestComposerScrollsToTheCaret`, `TestComposerWindowFollowsTheCaretLineByLine`, `TestComposerWindowStaysAtTheTopWithoutACaret`, `TestComposerHintYieldsToAFullWindow`.

**Known, operator decision (W1):** a pasted tab becomes spaces, because the textarea's sanitiser rewrites a tab. Space indentation lands verbatim. Tab indentation does not. The milestone's acceptance says indentation lands verbatim. The crew has not resolved this. It is for the operator to decide, and it is carried in `plan/PROGRESS.md` under m5-d6.

**Waits on:** O10 (decided: scroll) and m5-d1 for `view.go`, which m5-d6 leaves alone because O10 is not wrap.

**Key-state check, first.** Authored here. Run it before any kirsch launch in this section. It must print `CONTROL PASS`, then `0`.

```bash
# authored here
keycount_control
keycount
```

**Automated checks**

Run by the crew at step 17 of mission-20261028-03, in zsh after `source scripts/m5-test-env.sh`. The full output is in `.claude/memory/workspace/m6d-step17-auto.txt`.

- [✅] A space-indented multi-line paste lands verbatim (`TestPasteLandsVerbatim`) (plan/milestones/milestone-5.md:772). Command:

  ```bash
  # authored here
  go test -count=1 ./internal/tui/ -run '^TestPasteLandsVerbatim$' -v
  ```

- [ ] Tab indentation lands verbatim — not met: tabs become spaces (operator decision, W1). Milestone cite: plan/milestones/milestone-5.md:772.

- [✅] A paste over 8 KB shows the y/n prompt, and accept and decline are tested. Test: `TestLargePasteConfirmAcceptAndDecline`, with subtests `accept` and `decline` (plan/milestones/milestone-5.md:773). Command:

  ```bash
  # authored here
  go test -count=1 ./internal/tui/ -run '^TestLargePasteConfirmAcceptAndDecline$' -v
  ```

- [✅] A multi-line paste starting with `/` is not a command (plan/milestones/milestone-5.md:774). Test: `TestMultiLinePasteStartingWithSlashIsNotACommand`. Command:

  ```bash
  # authored here
  go test -count=1 ./internal/tui/ -run '^TestMultiLinePasteStartingWithSlashIsNotACommand$' -v
  ```

- [✅] A 12-line paste leaves the caret row visible (plan/milestones/milestone-5.md:775). Tests: `TestComposerScrollsToTheCaret` and `TestComposerWindowFollowsTheCaretLineByLine`. Command:

  ```bash
  # authored here
  go test -count=1 ./internal/tui/ -run '^(TestComposerScrollsToTheCaret|TestComposerWindowFollowsTheCaretLineByLine)$' -v
  ```

  The test passes. Whether the real binary behaves the same is open; see the hands-on step below.

- [✅] Existing composer tests pass (plan/milestones/milestone-5.md:776). Command:

  ```bash
  # authored here
  go test -count=1 ./internal/tui/
  ```

- [✅] With no caret the window stays at the top, and the hint yields to a full window. Tests: `TestComposerWindowStaysAtTheTopWithoutACaret` and `TestComposerHintYieldsToAFullWindow`. Command:

  ```bash
  # authored here
  go test -count=1 ./internal/tui/ -run '^(TestComposerWindowStaysAtTheTopWithoutACaret|TestComposerHintYieldsToAFullWindow)$' -v
  ```

**Hands-on checks**

1. (operator) Paste a 20-line trace in Terminal.app and in iTerm2. Expected: newlines are kept, nothing acts as a key, and the typing area shows the last lines with the cursor after the last line. Pressing Up moves the window up a line at a time until the top line shows; after that the window holds while the caret moves inside it. Run Setup in each window before starting. Sample text to copy: `for i in {1..20}; do printf '  trace %02d\n' "$i"; done`, pasted into the kirsch typing area with Cmd-V and not submitted with Enter. Esc clears the typing area when no turn is in progress.

   **Not pre-run.** The crew could not run this step; it needs your real Terminal. A pseudo-terminal pre-run was attempted inside a 25-minute time-box, and it did not establish the result. Keys reached kirsch: a typed `a` before the paste echoed in attempts 6 to 8. After a 12-line or 20-line bracketed paste (attempts 2, 3, 4, 6 and 8), the last output the probe captured showed the top five pasted lines, `trace 01` to `trace 05`, and no later line. That frame is recorded directly for attempts 2 and 3, which list `trace 01` to `trace 05` by name. For attempts 4, 6 and 8 it is inferred: five trace tokens were recorded, with `trace 01` present and `trace 10` absent. Attempt 1 is not counted: it started before the banner, and its controls failed. That window is what a caret at or above line 4 gives, and it is also what a composer that never scrolled would show. The tests expect the last lines (for example `trace 07` to `trace 11` for 12 lines), so the evidence fits both a scroll that does not engage in the real binary after a real paste and a harness artefact. The output was 914 to 915 bytes after a 12-line or 20-line paste (attempts 2, 3, 4, 6 and 8), and 933 bytes after a 5-line paste (attempt 7). In attempt 3, 14 of the 15 Up presses produced no output, and the fifteenth produced 84 bytes that include `trace 05`. A typed `Z` after the paste did not appear in attempts 4, 6 and 8, and did appear after a 5-line paste in attempt 7. SIGTERM did not stop the child in attempts 2, 3 and 8. It did in attempts 4, 6 and 7. Attempt 5 ended before the kill step. The probe drains output in bounded windows (`.claude/memory/workspace/m6d-paste-probe.py`). The mission-20261028-03 step 18 review, reading the probe code, found no obvious buffer-block cause. One possible harness cause is that nothing answers terminal queries; that is a hypothesis, not a finding. The check that settles it is this step in your real Terminal. The cause is unresolved. The attempts and their output are in `.claude/memory/workspace/m6d-step17-paste-attempts.txt`. The operator's plain steps are in section 8 of `plan/testing/manual/user-testing/milestone-5-donovan.md`.

   Tab note, in plain words: if your pasted text used tabs, they show as spaces. That is known and waiting for your decision (W1).

   Recovery, in plain words: If kirsch stops responding after the paste (keys do nothing), close this Terminal tab or window, then write what you saw in section 8 of the operator checklist under *Anything odd?* and tell Bishop. Nothing in this step touches a model or your files.

**Notes / evidence**

Automated checks, run by the crew at step 17. The lines below are verbatim from the `-run 'Paste|Composer'` run in `.claude/memory/workspace/m6d-step17-auto.txt`:

```text
--- PASS: TestHistoryUpInMiddleOfComposer (0.00s)
--- PASS: TestPasteLandsVerbatim (0.01s)
--- PASS: TestLargePasteConfirmAcceptAndDecline (0.02s)
--- PASS: TestMultiLinePasteStartingWithSlashIsNotACommand (0.01s)
--- PASS: TestComposerScrollsToTheCaret (0.01s)
--- PASS: TestComposerWindowFollowsTheCaretLineByLine (0.01s)
--- PASS: TestComposerWindowStaysAtTheTopWithoutACaret (0.00s)
--- PASS: TestComposerHintYieldsToAFullWindow (0.01s)
--- PASS: TestPendingCardT6Paste (0.00s)
--- PASS: TestGAndCapitalGAreLiteralInTheComposer (0.00s)
--- PASS: TestComposerCtrlGRePins (0.02s)
--- PASS: TestQuestionMarkIsLiteralInComposer (0.00s)
--- PASS: TestSubmitHandsComposerTextToCallback (0.00s)
PASS
ok  	github.com/djm56/kirsch/internal/tui	0.331s
```

The whole package, run in the same script: `ok  	github.com/djm56/kirsch/internal/tui	6.760s`.

Key-state check, run in the same shell, from the same script:

```text
CONTROL PASS: a planted key line counts as 1
0
```

`keycount` exits with status 1 when it counts 0, because `grep -c` does that. The printed `0` is the result.

Hands-on step 1: not run by the crew. The pseudo-terminal attempts are recorded above, under *Hands-on checks*.

---

## 9 · Build wrapper and version proof (m5-d10a)

**Context:** Terminal: Setup. Binary: `bin/kirsch`, built by `scripts/build.sh` into /Volumes/DATA/Github/kirsch/bin. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

- `scripts/build.sh` (new). Its header, `scripts/build.sh:1-29`, is the authority for the rules. An argument is stamped as the bare version after one leading `v` is removed. The value must match `X.Y.Z`, optionally followed by `-suffix` or `+suffix`, or the script exits 2. More than one argument exits 2.
- With no argument, the script stamps only when git is present, the repository is the script's own, `git status --porcelain` succeeds and is empty, and HEAD is exactly on a `v` tag of that shape. Otherwise it builds unstamped (`scripts/build.sh:55-69`).
- An unstamped build reports `0.1.0-dev`, unless Go's build info records a clean release tag. That fallback is `resolveVersion` (`cmd/kirsch/main.go:43-59`), and it is O4(b): an unstamped `go install` build reports its module version with the leading `v` removed (`plan/PROGRESS.md`, *Milestone 5 — decisions recorded*). O4 does not describe the checkout stamping above. That rule comes from the script.
- Milestone tags such as `m3` and `m4` are never stamped (`scripts/build.sh:25–26`).
- `package.json:9` (`build`) is now `bash scripts/build.sh`.
- The operator copy is section 9 of `plan/testing/manual/user-testing/milestone-5-donovan.md`. Its steps were run on 2026-10-27; see *Notes / evidence*.

**Waits on:** m5-d2 (the ldflags variable) and O4, from `plan/milestones/milestone-5.md:109`.

**Key-state check, first.** Authored here. Run it before any kirsch launch in this section. It must print `CONTROL PASS`, then `0`.

```bash
# authored here
keycount_control
keycount
```

**Automated checks**

- [ ] `bash scripts/build.sh 9.9.9 && bin/kirsch --version` prints `9.9.9`, with no leading `v` (`plan/milestones/milestone-5.md:980`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && bash scripts/build.sh 9.9.9 && bin/kirsch --version
  ```

- [ ] `npm run build` runs `scripts/build.sh` (`plan/milestones/milestone-5.md:981`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && npm run build
  ```

  The review step confirms that `package.json:9` (`build`) now points at `bash scripts/build.sh` (`plan/milestones/milestone-5.md:975–976`).

- [ ] With no argument, the script stamps only on a clean exact `v` release tag on HEAD. Otherwise the binary reports `0.1.0-dev` unless Go's build info records a clean release tag (`plan/milestones/milestone-5.md:982`; `scripts/build.sh:1-29`; `cmd/kirsch/main.go:43-59`). Command, from the kirsch folder:

  ```bash
  cd /Volumes/DATA/Github/kirsch && npm run build && bin/kirsch --version
  ```

  On a tree with changes and no tag on HEAD, the expected output is `kirsch 0.1.0-dev`. On a clean exact `v` release tag, it shows that tag's version. `--version` cannot tell a stamped build from an unstamped one on the same tag; the distinguishing check is listed under *Notes / evidence*.

**Hands-on checks**

1. No hands-on steps are given in the milestone. Milestone text: "Prove the version with a known value" (`plan/milestones/milestone-5.md:977`). The `bin/kirsch` run in the automated check starts from the checkout. This run takes the absolute path from `$T_CFG`, outside the checkout. Authored here:

   ```bash
   # authored here
   cd "$T_CFG" && /Volumes/DATA/Github/kirsch/bin/kirsch --version; echo "exit=$?"
   ```

   Expected output, after *Before you start* and a normal build (step 4 of section 9 in the operator copy), from an unstamped checkout:

   ```
   kirsch 0.1.0-dev
   exit=0
   ```

   On an exact release tag with no changes, the first line shows that tag's version instead.

**Notes / evidence**

Operator-step run, 2026-10-27, mission-20261027-08. Every step of section 9 in the operator copy was run in zsh after *Before you start*, in one shell session. The commands and verbatim output are in `m8-step3-runs.txt` in the archived workspace folder for mission-20261028-03 (find it with `find .claude/memory/workspace -name m8-step3-runs.txt`; two copies come back, so use the one inside the archive folder) (archived; the same name in `.claude/memory/workspace/` now holds mission-20261029-04's own step 3 runs). The key-state check printed `CONTROL PASS`, then `0`. The stamped runs printed `kirsch 9.9.9` and `kirsch 9.9.8`. The bad shape printed `build.sh: version argument '1.2' is not X.Y.Z[-suffix|+suffix]`, then `result: 2`. The normal build printed `kirsch 0.1.0-dev`. The `-ldflags` count printed `0`. The hands-on command printed `kirsch 0.1.0-dev` and `exit=0`. Both runs, with their verbatim output, are in the archived file named above, under `step 4a run`. These are evidence of mission-20261027-08. The archived file sits in the archived workspace folder for mission-20261028-03; the nested archive folders are kept as they were moved.

Developer proofs. They are in the workspace, so they go to the archive at the next mission start.

- **Clean exact-tag case.** `m8-step2a-evidence.txt` in the archived workspace folder for mission-20261028-03 (find it with `find .claude/memory/workspace -name m8-step2a-evidence.txt`), section b (lines 62–77). The distinguishing artefact is `go version -m bin/kirsch | grep -- -ldflags`. A stamped build prints `build -ldflags="-X main.version=0.1.0"` (line 70). A plain `go build` on the same tag prints `kirsch 0.1.0` with no `-ldflags` line (lines 74–76), because `resolveVersion` reads the tag from build info. So `--version` alone cannot tell the two apart.
- **Bad-shape refusals.** Section g of the same file (lines 132–156). Accepted shapes, including `v1.2.3` and `1.2.3-rc.1`: section h (lines 158–173). Empty argument: `m8-step3-runs.txt` in the archived workspace folder for mission-20261028-03 (find it with `find .claude/memory/workspace -name m8-step3-runs.txt`; use the copy inside the archive folder), extra block (lines 66–68; checked in step 3b). `bash scripts/build.sh ''` prints `build.sh: version argument '' is not X.Y.Z[-suffix|+suffix]` and exits 2.
- **Nested checkout.** Section j (lines 193–214). The script builds unstamped inside a checkout that sits in a repository tagged `v9.9.9` (line 198).
- **Git absent.** Section k (lines 216–243). With no `git` on PATH the build is unstamped. The control on the same clone with the real PATH stamps `0.1.0` (lines 232–237).
- **Git status failure.** Section a (lines 28–60). The script prints `build.sh: git status failed; building unstamped` (line 45), and with `GOFLAGS=-buildvcs=false` the build is unstamped (lines 43–48). Without that flag, Go's own VCS stamping stops the build with exit 1 (lines 51–56). This is carried item (a) in `plan/PROGRESS.md`.
- **Several `v` tags on HEAD.** Section e (lines 106–116). This is carried item (b).
- **`--help` gets a shape error.** Section g (lines 142–144). This is carried item (c).
- **A tracked change or an untracked file stops stamping.** Section c (lines 79–92), and `m8-step1-evidence.txt` lines 27–35.
- **Milestone tags and other non-version tags.** `m8-step1-evidence.txt` lines 16–41 (`m3`, `m4`), section d (lines 94–104, `m9` ignored), and section f (lines 118–130, `v1-notes`).

---

## 10 · goreleaser config and target-compile check (m5-d10b)

**Context:** Terminal: Setup. Binary: none; the target check builds into a temporary folder under `bin/` and removes it. The goreleaser snapshot is operator-owned and was not run here. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: not used, because no kirsch is launched in this section. Config state: Setup, unchanged. Shell for every run in *Notes / evidence*: zsh 5.9.

**What landed**

- `.goreleaser.yaml` (new). Its header comments are the authority. It uses goreleaser's version 2 schema, which needs goreleaser v2.6 or later (`archives.formats`). It builds `./cmd/kirsch` only, with `CGO_ENABLED=0`, for the `goos` and `goarch` lines in the file. Its ldflags set `main.version` to `{{.Version}}`. Archives are tar.gz with goreleaser's default name template. Checksums are sha256. The changelog comes from git.
- `scripts/ci.sh targets`, also `npm run ci:targets`. It reads its targets from the `goos` and `goarch` lines of `.goreleaser.yaml`. Before it reads them, `check_goreleaser_shape` refuses some file shapes it cannot model. That check is a denylist, not an allowlist (see the carried items). It compiles `./cmd/kirsch` for each target and prints `target os/arch: ok` or `FAILED`. It fails on any failure and on zero targets. An EXIT trap removes its temporary folder under `bin/`.
- The `targets` job in `.github/workflows/ci.yml` runs `bash scripts/ci.sh targets`. `bash scripts/ci.sh all` includes the same job.
- goreleaser was not installed on this machine on 2026-10-28. The snapshot run is operator-owned; see the hands-on check below.

**Waits on:** m5-d2 (ldflags variable), O2 (goreleaser shape), O4 (version string), O20 (target-compile method), from the m5-d10b row of the deliverable table in `plan/milestones/milestone-5.md`. The target list is the Release bullet in `plan/spec/kirsch-plan.md`, matched by its text `Release via **goreleaser**` (line 482 on 2026-10-28). The milestone file's cites to the spec were one line low (it said :481 for this bullet), and step 14a swept them to the live lines. Spec line numbers can move again, so the bullet's text is the stable key: it names the four targets. The check has run on this machine, with the result in *Notes / evidence*. Nothing was pushed in this step, so the GitHub run has not happened.

**Automated checks**

- [✅] `.goreleaser.yaml` builds only `./cmd/kirsch`, with `CGO_ENABLED=0` and `-X main.version={{.Version}}` (`plan/milestones/milestone-5.md`, m5-d10b acceptance). Run in mission-20261028-03 step 13 from zsh; output in `.claude/memory/workspace/m10b-step13-evidence.txt`, sections C and C2; the control runs for this block are in section J (step 14a). Command, from the kirsch folder:

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n '^ *main:' .goreleaser.yaml
  cd /Volumes/DATA/Github/kirsch && grep -n -E 'CGO_ENABLED|main.version' .goreleaser.yaml
  # Control: a copy with one probe build line added under `main: ./cmd/kirsch`. The count must be above 0.
  cd /Volumes/DATA/Github/kirsch && CTRL=.claude/memory/workspace/m10b-goreleaser-probe-control.yaml && awk '{print} /^    main: \.\/cmd\/kirsch$/{print "    main: ./cmd/kirsch-probe"}' .goreleaser.yaml > "$CTRL" && awk '/^builds:/{f=1;next} /^[^[:space:]#]/{f=0} f' "$CTRL" | grep -v '^[[:space:]]*#' | grep -c kirsch-probe
  # Real check: must print 0.
  cd /Volumes/DATA/Github/kirsch && awk '/^builds:/{f=1;next} /^[^[:space:]#]/{f=0} f' .goreleaser.yaml | grep -v '^[[:space:]]*#' | grep -c kirsch-probe
  ```

  Expected: the `main: ./cmd/kirsch` line; the `CGO_ENABLED=0` line and the `-X main.version={{.Version}}` line (the `CGO_ENABLED|main.version` grep also prints a comment line that mentions `main.version`, which is expected); the control line prints a number above 0; and `0` from the probe count. That count is the number of lines containing `kirsch-probe` in the builds section outside comments. `grep -c` exits 1 when it counts nothing, which is expected for the probe count.
- [✅] Every target in the Release bullet of `plan/spec/kirsch-plan.md` compiles, and the config names exactly that set (`plan/milestones/milestone-5.md`, m5-d10b acceptance). Run in mission-20261028-03 step 13 from zsh: the target lines are in `m10b-step13-evidence.txt` section A, and the set comparison is in section D. Command:

  ```bash
  cd /Volumes/DATA/Github/kirsch && bash scripts/ci.sh targets; echo "result: $?"
  # Control: one side has an extra target, so this must print a `> darwin/386` line and `control exit=1`.
  cd /Volumes/DATA/Github/kirsch && diff <(sed -n "$(grep -n 'Release via' plan/spec/kirsch-plan.md | cut -d: -f1)p" plan/spec/kirsch-plan.md | grep -o -E '[a-z]+/(amd64|arm64)' | sort) <(sed -n 's/^ *goos: *\[\(.*\)\] *$/\1/p' .goreleaser.yaml | tr ',' '\n' | tr -d ' ' | while read o; do sed -n 's/^ *goarch: *\[\(.*\)\] *$/\1/p' .goreleaser.yaml | tr ',' '\n' | tr -d ' ' | while read a; do echo "$o/$a"; done; done | sort; echo darwin/386); echo "control exit=$?"
  # Real check: must print only SAME SET.
  cd /Volumes/DATA/Github/kirsch && diff <(sed -n "$(grep -n 'Release via' plan/spec/kirsch-plan.md | cut -d: -f1)p" plan/spec/kirsch-plan.md | grep -o -E '[a-z]+/(amd64|arm64)' | sort) <(sed -n 's/^ *goos: *\[\(.*\)\] *$/\1/p' .goreleaser.yaml | tr ',' '\n' | tr -d ' ' | while read o; do sed -n 's/^ *goarch: *\[\(.*\)\] *$/\1/p' .goreleaser.yaml | tr ',' '\n' | tr -d ' ' | while read a; do echo "$o/$a"; done; done | sort) && echo "SAME SET"
  ```

  Expected: a `target os/arch: ok` line for each target in the Release bullet, then the line `ci.sh: all ... targets compile` and `result: 0`. The control `diff` prints a `> darwin/386` line and then `control exit=1`, which shows the comparison can fail. The real `diff` command prints only `SAME SET`, with no diff lines above it.
- [✅] A forced failure in one target makes the CI check exit non-zero. The forced failure is reverted and recorded (`plan/milestones/milestone-5.md`, m5-d10b acceptance). Ticked on recorded evidence, not re-run in step 13, because step 13 is docs work. `.claude/memory/workspace/m11-step12a-evidence.txt`, lines 157–171: the forced file made `target linux/arm64: FAILED` and `exit=1`. Lines 228–241 are the removal and the rerun: the `zz_` count of 0 at line 229 shows the file is gone, and `exit=0` at line 241 is the green rerun. The log does not show the `rm` command itself. No `zz_` file was in `cmd/kirsch` on 2026-10-28 (checked at step 14a). Command: none run here.

**Hands-on checks**

1. Check that the target run left no folder behind. Paste:

   ```bash
   cd /Volumes/DATA/Github/kirsch && mkdir bin/targets.control && find bin -maxdepth 1 -name 'targets.*' -print; rmdir bin/targets.control; echo "control done"
   cd /Volumes/DATA/Github/kirsch && find bin -maxdepth 1 -name 'targets.*' -print; echo "leftover check done"
   ```

   Expected: the control line prints `bin/targets.control` before `control done`, which shows the search can see a leftover folder. The real check prints no path before `leftover check done`. The EXIT trap in `scripts/ci.sh` removes the folder after each run, so a path in the real check means the trap did not run.

2. (operator) `goreleaser release --snapshot --clean` produces `dist/` with archives and a checksums file, and the native binary reports its version (`plan/milestones/milestone-5.md`, m5-d10b operator box). `dist/` is gitignored (`.gitignore:10`). The snapshot also shows whether the stamped version is bare. Confirming which ldflags were used, and whether goreleaser's default ldflags are replaced, needs `go version -m` on the built binary (the command is under m5-d22 in `plan/milestones/milestone-5.md`). Those points are unverified until it runs (see the carried items). Not run here: goreleaser was not installed on 2026-10-28, and this step may not install anything. In step 13 `command -v goreleaser` printed nothing and returned 1 (evidence section F2). The snapshot is owed at m5-d22, the operator's final release check.

   ```bash
   cd /Volumes/DATA/Github/kirsch && goreleaser release --snapshot --clean
   ```

**Notes / evidence**

Crew run, 2026-10-28, mission-20261028-03 step 13, zsh 5.9, from /Volumes/DATA/Github/kirsch. The commands above, with their verbatim output and exit status, are in `.claude/memory/workspace/m10b-step13-evidence.txt`, sections A, B, C, C2, D, E, F, F2 and H (section H holds the commands as they were printed at step 13). The control runs for the blocks above, and the run of the checklist's section 10 commands after `source scripts/m5-test-env.sh`, are in section J (step 14a). After the next mission starts, find it with `ls .claude/memory/workspace/archive-mission-*/m10b-step13-evidence.txt`.

Recorded by @hicks at mission-20261028-03 steps 11 and 12a. Not re-run in step 13.

Step 14a (docs fix round 1, 2026-10-28): the checklist's section 10 commands were run in zsh after `source scripts/m5-test-env.sh`, and again without it. Setup made no difference to their output or exit codes; the only extra lines are Setup's own `Ready.` block (evidence section J4). The declaration of Setup in the checklist's section 10 is kept.

- Forced failure and removal: `.claude/memory/workspace/m11-step12a-evidence.txt`, lines 157–171 (forced failure) and 228–241 (the removal, shown by the `zz_` count of 0 at line 229, and the green rerun). After the next mission starts, `ls .claude/memory/workspace/archive-mission-*/m11-step12a-evidence.txt`.
- Mutants for the shape refusals and the zero-target guard, and the trap checks: `.claude/memory/workspace/m11-step12a-evidence.txt`. After the next mission starts, `ls .claude/memory/workspace/archive-mission-*/m11-step12a-evidence.txt`.

Snapshot run: not run, and why. Goreleaser was not installed on this machine on 2026-10-28, and no install was allowed in step 13. The snapshot is owed at m5-d22.

Carried items. Each is recorded in `plan/PROGRESS.md` under *Carried from Milestone 5 planning*, in the m5-d10b entry. Items (a) and (b) are warnings. Items (c) to (e) are notes, not warnings.

- (a) The shape check is a denylist, not an allowlist. A `- ignore:` written with a list dash passes it, and so do a bare `-` entry in `builds:` and build keys the check does not name. The direction is an allowlist.
- (b) The awk that reads `.goreleaser.yaml` is untested on CI's awk.
- (c) A custom ldflags line drops goreleaser's default `-s -w`. Noted, not a defect.
- (d) `.Version` in snapshot builds is unverified until the operator's snapshot run.
- (e) The operator's snapshot run is owed at m5-d22.

---

## 11 · Safety-test repair: C29 (m5-d19b)

**Context:** Terminal: Setup. Binary: `go test`. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged. The relax branch edits a milestone file, not the config.

**What landed**

- **Branch taken: relax.** The operator's row `O15 (C29 only)` in `plan/PROGRESS.md` (*Milestone 5 — decisions recorded*) chose it, and that row is dated 2026-10-29, before this mission. The relaxed value is 3 s after the deadline. The row `C29 detail` in the same file records it (2026-11-05).
- **Files changed:** `plan/milestones/milestone-2.md`, Task 9. The timeout box is relaxed and unticked, with the old text quoted beside it. The cancelled-process box keeps its tick unset and carries a note. The stdin box carries the C34 reword (row `C34`, 2026-11-03) and stays unticked. Dated Update lines were added. The plan and tracker files were updated too: `plan/PROGRESS.md`, `plan/milestones/milestone-5.md`, `plan/milestones/milestone-5-start-here.md`, and the two checklists, `plan/testing/manual/milestone-5.md` and `plan/testing/manual/user-testing/milestone-5-donovan.md`.
- **No code and no test changed.** The tighten branch was not taken, so `internal/tool/run_command.go` and `internal/tool/run_command_test.go` are as they were.
- **Suspected gap (read from the code, not run).** After SIGTERM, `killProcessGroup` in `internal/tool/run_command.go` (about :361–369) polls the group leader only. Once the leader is gone it returns without sending SIGKILL to the group. A child or grandchild that ignores SIGTERM may therefore survive when its leader exits on SIGTERM. No test checks this. It is listed under *Open from m5-d19b* in `plan/PROGRESS.md`. Fixing it is a code step outside the relax branch.

**Waits on:** m5-d19a (same test file), O15 (the C29 decision, recorded before this step runs), O13, from `plan/milestones/milestone-5.md:124`.

**Before any edit.** On the relax branch only, save a verbatim copy of `plan/milestones/milestone-2.md` to `.claude/memory/workspace/m5-d19b-milestone-2.pre.md` (DIR-020; `plan/milestones/milestone-5.md:1601`). The milestone gives no name for this copy. The name here is chosen, and the milestone's relax box is the one to edit. Done in mission-20261105-03: the copies are in `.claude/memory/workspace/m5-d19b-step3-pre/`, the `milestone-2.md` copy among them, and `cmp` matched each one to its source before the edit.

**Automated checks**

- [ ] If O15 tightens: not taken (relax branch). No tighten command is run in this section. (`plan/milestones/milestone-5.md:1604`)
- [ ] If O15 relaxes: `plan/milestones/milestone-2.md:422–431` states 3 s after the deadline, and no test asserts 500 ms for that case (`plan/milestones/milestone-5.md:1605`). Command, run from the kirsch folder on 2026-11-05 in mission-20261105-03. No model is called and no key is read:

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -count=1 ./internal/tool -run 'TestRunCommandTerminateThenKill$|TestRunCommandTimeoutWithinDeadline$' -v; echo "exit=$?"
  ```

  Real output, pasted verbatim:

  ```
  === RUN   TestRunCommandTimeoutWithinDeadline
  --- PASS: TestRunCommandTimeoutWithinDeadline (1.02s)
  === RUN   TestRunCommandTerminateThenKill
  --- PASS: TestRunCommandTerminateThenKill (3.02s)
  PASS
  ok  	github.com/djm56/kirsch/internal/tool	4.241s
  exit=0
  ```

  `TestRunCommandTimeoutWithinDeadline` requires the call to end 1–2 s after it started, which is within 1 s after its 1 s limit. `TestRunCommandTerminateThenKill` requires the whole call to take no more than 4 s, so with its 1 s limit, at most 3 s past it. Its child ignores SIGTERM, so the call is forced to stop about 2 s after the limit, which is the 2 s grace (`internal/tool/run_command.go`). The 3 s box has 1 s of margin on that.
- [ ] O15's C29 decision is recorded in `plan/PROGRESS.md` before step 1 runs (`plan/milestones/milestone-5.md:1606`). Process check; reviewed, not run. The O15 row is dated 2026-10-29, before mission-20261105-03 ran this step.

**Hands-on checks**

1. Branch taken: relax (row `O15 (C29 only)`, 2026-10-29). The tighten branch was not taken, so `internal/tool/run_command.go` and `internal/tool/run_command_test.go` are unchanged by this mission. The relax branch touches `plan/milestones/milestone-2.md`: the timeout box at `:422–431` and the stdin box at `:439–442`. The other files it touched are listed under *What landed*. Every box in this section is unticked; ticking them is the operator's.

**Notes / evidence**

_Empty. Fill at run._

---

## 12 · Doctor: `rg`, `git`, repository and config probes (m5-d3b)

**Context:** Terminal: Setup. Binary: `"$KIRSCH_BIN" doctor`, built by `npm run build` (the build block in §2). Working directory: the kirsch folder for the project run; a fresh temporary folder outside any repository for the `--workspace` pair; the kirsch folder for the broken-settings and unknown-key runs, each with `XDG_CONFIG_HOME` set to its own folder for that one command. Workspace: the `--workspace` run names a fresh folder outside any repository, not `$WORKSPACE`, because the case under test is a non-Git folder. Key state: all key variables unset. Config state: Setup, unchanged; the broken and unknown-key runs use their own settings folders, as m5-d3a did.

**What landed**

`kirsch doctor` now runs its probes after the version line, in this order: `git`, `rg`, `repository` and `config`. Each prints its line in the m5-d3a grammar, and the summary line counts them. The number of lines is not fixed: `repository` prints none when git is missing, unless `--workspace` names a path that `workspace.Detect` rejects, and `config` prints one `warn` line for each unknown key, before its own line. The probes are `Run` in `internal/doctor/probes.go`. The report and the pass or fail rule are `Report` and `Failed` in `internal/doctor/doctor.go`. The probes only read: they create nothing, take no lock and make no network call. `--workspace` given with `doctor` now applies, through `workspace.Detect`. The m5-d3a report did not use the flag. m5-d4a later added lines after the config line (`key`, `backend`, `model` and `context`, see §20). The expected results below describe each check by its property, so they hold with those lines present.

The pass and fail rule is O5, recorded on 2026-10-29 (*Milestone 5 — decisions recorded* in `plan/PROGRESS.md`): a check fails when the problem would stop kirsch starting, and warns when kirsch still starts but works less well. The O5 detail row of 2026-10-29 settles the two cases that depend on `--workspace`. The lines, with the result each one gets:

| Check | Line as printed | Result | When |
|---|---|---|---|
| git | `ok  git: found at <path>` | ok | git is on PATH |
| git | `fail  git: not found on PATH; kirsch needs git to find the repository` | fail | git is missing and there is no `--workspace` |
| git | `warn  git: not found on PATH; kirsch starts with --workspace but the Git features are off, such as …` | warn | git is missing and `--workspace` is given |
| rg | `ok  rg: found at <path>` | ok | rg is on PATH |
| rg | `warn  rg: not found on PATH; search_code uses the pure-Go fallback` | warn | rg is missing |
| repository | `ok  repository: <root>` | ok | a repository is found, or `--workspace` names one |
| repository | `fail  repository: not in a Git repository` | fail | no `--workspace`, and no repository above the current folder |
| repository | `warn  repository: <folder> is not a Git repository; kirsch starts, but the Git features are off, such as …` | warn | `--workspace` names an existing folder that is not a repository |
| repository | `fail  repository: <error from workspace.Detect>` | fail | `--workspace` names a path that `workspace.Detect` rejects. Examples: a path that does not exist (`--workspace <path>: <error>`), a path that is not a directory (`--workspace <path> is not a directory`), and a root that cannot be resolved (`resolve workspace root <path>: <error>`). Every other `workspace.Detect` error takes this branch too, and the line carries that error's text |
| config | `ok  config: settings valid` | ok | the settings load and validate |
| config | `ok  config: settings valid; project file not read (no workspace)` | ok | no workspace was found, so the project file is not read |
| config | `warn  config: <file>: unknown key "<key>" (ignored)` | warn | one line for each key kirsch does not know |
| config | `fail  config: parse <file>: <TOML error>` | fail | the settings file does not parse |
| config | `fail  config: invalid settings (<files consulted>): <error>` | fail | the settings parse but do not validate |

The Git feature list in the warnings is the one in `gitFeaturesOff`. Its docblock calls it "examples … not a complete list".

The summary line is `doctor: <n> ok, <n> warn, <n> fail`. The exit status is 1 when any line is `fail`, and 0 otherwise. A warning does not change the exit status. A status the report does not know counts as a failure (`Failed`). A failing check does not stop the others: the config check runs even after the repository check fails, and it reads the project file only when a workspace was found.

Known limit: a `git` that is present but errors (for example, "dubious ownership") is reported as `not in a Git repository`, because `workspace.Detect` treats any failure of git as no repository. Doctor cannot tell the two apart (`repositoryResults`, in `internal/doctor/probes.go`, names this limit).

**Waits on:** m5-d3a (dispatch and report grammar) and O5, from `plan/milestones/milestone-5.md:100`.

**Key-state and repository checks, first.** Authored here. It must print `CONTROL PASS`, then `0`. The first `git` line must exit non-zero, and the last must print `true`.

```bash
# authored here
keycount_control
keycount
git -C "$T_CFG" rev-parse --is-inside-work-tree; echo "exit=$?"
git -C "$WORKSPACE" rev-parse --is-inside-work-tree
```

**Automated checks**

- [✅] With the PATH shim lacking `rg`, doctor prints a warn line that names the pure-Go fallback (`plan/milestones/milestone-5.md:588`). Tests: `TestRunDoctorRGMissing` (through `run`, with the shim) and `TestRunDefaultLookPathUsesPATH` (the real `exec.LookPath`, with the shim). `TestRunRGMissing` injects a `LookPath` and does not use the shim. Command:

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -count=1 ./cmd/kirsch ./internal/doctor -run '^(TestRunDoctorRGMissing|TestRunRGMissing|TestRunDefaultLookPathUsesPATH)$' -v
  ```

  Output: `--- PASS` for each name the command selects (the times differ between runs).

- [✅] With the PATH shim lacking `git`, doctor prints a git-missing line and not "not in a Git repository" (`plan/milestones/milestone-5.md:589`). Tests: `TestRunDoctorGitMissing` (through `run`, with the shim, and with a control that git is found without it) and `TestRunGitMissing` (an injected `LookPath`; with no flag it fails, with `--workspace` it warns). Command:

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -count=1 ./cmd/kirsch ./internal/doctor -run '^(TestRunDoctorGitMissing|TestRunGitMissing)$' -v
  ```

- [✅] Outside a repository, doctor prints "not in a Git repository" and every check that does not need one (`plan/milestones/milestone-5.md:590`). This is the no-flag case. An explicit `--workspace` to a non-Git folder is a warning, not a failure, by the O5 detail row (`TestRunNonGitWorkspaceVersusNoFlag`). Tests: `TestRunDoctorOutsideRepository` and `TestRunNotInRepositoryRunsOtherChecks`. Command:

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -count=1 ./cmd/kirsch ./internal/doctor -run '^(TestRunDoctorOutsideRepository|TestRunNotInRepositoryRunsOtherChecks)$' -v
  ```

- [✅] A broken `config.toml` produces a line naming the file and the parse error, and doctor does not exit on it (`plan/milestones/milestone-5.md:591`). Tests: `TestRunDoctorBrokenConfigDoesNotExitEarly`, `TestRunBrokenGlobalConfig` and `TestRunBrokenProjectConfig`. Command:

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -count=1 ./cmd/kirsch ./internal/doctor -run '^(TestRunDoctorBrokenConfigDoesNotExitEarly|TestRunBrokenGlobalConfig|TestRunBrokenProjectConfig)$' -v
  ```

- [✅] An unknown config key produces a warn line (`plan/milestones/milestone-5.md:592`). Tests: `TestRunDoctorUnknownConfigKeyWarns` and `TestRunUnknownConfigKey`. Command:

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -count=1 ./cmd/kirsch ./internal/doctor -run '^(TestRunDoctorUnknownConfigKeyWarns|TestRunUnknownConfigKey)$' -v
  ```

- [✅] `--workspace` is honoured (`plan/milestones/milestone-5.md:593`). Tests: `TestRunDoctorHonoursWorkspaceFlag`, which drives `run` from a folder outside any repository with `--workspace` pointing at a real one; its control is the same folder without the flag, which fails. `TestRunNonGitWorkspaceVersusNoFlag` covers the non-Git folder. Command:

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -count=1 ./cmd/kirsch ./internal/doctor -run '^(TestRunDoctorHonoursWorkspaceFlag|TestRunNonGitWorkspaceVersusNoFlag)$' -v
  ```

Each command above was run in zsh, and each printed `--- PASS` for every name it selects. The output of the step 3a run is in `.claude/memory/workspace/m3b-step3a-runs.txt`, run 11. The output of the step 3 run is in `.claude/memory/workspace/m3b-step3-runs.txt`. Both are scratch files, archived at the next mission start.

**Hands-on checks**

The missing-tool tests use a PATH shim, not an emptied PATH (DIR-009, `plan/milestones/milestone-5.md:585`). The shim is built by `shimPathWithout`, in `internal/doctor/probes_test.go` and again in `cmd/kirsch/main_test.go`. It is a temporary folder of symlinks to every executable on the PATH except the tool under test, and PATH is set to that folder for the test. The control, quoted from `plan/milestones/milestone-5.md:585`: under the same shim PATH, `LookPath("git")` succeeds in the `rg` test, and `LookPath("rg")` still fails. The tests assert that control before they read any result, so a missing-`rg` result does not count until it holds.

Each hands-on step below makes its folders in the system temporary area with `mktemp -d`. These commands do not remove them. The operator removes them.

1. From the kirsch folder, which is inside a repository (authored here):

   ```bash
   # authored here
   cd /Volumes/DATA/Github/kirsch
   "$KIRSCH_BIN" doctor; echo "exit=$?"
   ```

   Expected: `kirsch 0.1.0-dev`, then the check lines for `git`, `rg`, `repository` (showing the kirsch folder) and `config` (with `settings valid`), then the m5-d4a lines (`key`, `backend`, `model` and `context`; the `key` line starts `warn` when no key is set). No line starts `fail`, the summary line reports no failure, and the run ends `exit=0`. The git and rg paths depend on the machine. If rg is missing, its line starts `warn`.

2. A folder outside any repository, with `--workspace` (authored here). The `git` line checks that the folder is outside any repository. Step 3 is the control, and it runs from the same folder:

   ```bash
   # authored here
   d=$(mktemp -d)
   cd "$d"
   git -C "$d" rev-parse --is-inside-work-tree; echo "exit=$?"
   "$KIRSCH_BIN" doctor --workspace "$d"; echo "exit=$?"
   cd /Volumes/DATA/Github/kirsch
   ```

   Expected: a line starting `fatal: not a git repository`, then `exit=128`. Then the repository line starts with `warn`, and it names the folder and says the Git features are off. No line starts `fail`, and the run ends `exit=0`.

3. The same folder with no flag, run from inside it (authored here):

   ```bash
   # authored here
   cd "$d"
   "$KIRSCH_BIN" doctor; echo "exit=$?"
   cd /Volumes/DATA/Github/kirsch
   ```

   Expected: `fail  repository: not in a Git repository`, a summary line that reports a failure, then `exit=1`. Steps 2 and 3 are each other's control: the same kind of folder, and the only difference is the flag.

4. A settings file that does not parse, in its own folder (authored here; the Setup settings are not touched):

   ```bash
   # authored here
   c=$(mktemp -d); mkdir -p "$c/kirsch"; printf '[session\n' > "$c/kirsch/config.toml"
   cd /Volumes/DATA/Github/kirsch
   XDG_CONFIG_HOME="$c" "$KIRSCH_BIN" doctor; echo "exit=$?"
   XDG_CONFIG_HOME="$c" "$KIRSCH_BIN" --workspace /Volumes/DATA/Github/kirsch; echo "exit=$?"
   ```

   Expected: the git, rg and repository lines start with `ok`. The config line starts with `fail  config: parse`, names the file, and includes `toml: line 2`. The key, model and context lines read `warn` with `skipped: config did not load`. The config line is the only `fail`, and the run ends `exit=1`. The plain run then prints `kirsch: parse` naming the same file, then `exit=1`.

5. An unknown key, in its own folder (authored here):

   ```bash
   # authored here
   c2=$(mktemp -d); mkdir -p "$c2/kirsch"; printf 'bogus_setting = 1\n' > "$c2/kirsch/config.toml"
   cd /Volumes/DATA/Github/kirsch
   XDG_CONFIG_HOME="$c2" "$KIRSCH_BIN" doctor; echo "exit=$?"
   ```

   Expected: a `warn  config:` line that names the file and says `unknown key "bogus_setting" (ignored)`, then `ok  config: settings valid`. The unknown key is a warning and not a failure, so the summary reports at least one warning and no failure, and the run ends with `exit=0`. The count of warnings is not fixed: any other warning in the report adds to it, and with no key set the `key` line is one of them.

6. A settings value that parses but does not validate, and a missing `--workspace` path (authored here):

   ```bash
   # authored here
   cd /Volumes/DATA/Github/kirsch
   c3=$(mktemp -d); mkdir -p "$c3/kirsch"; printf '[provider]\ndefault = "nowhere"\n' > "$c3/kirsch/config.toml"
   XDG_CONFIG_HOME="$c3" "$KIRSCH_BIN" doctor; echo "exit=$?"
   "$KIRSCH_BIN" doctor --workspace /nonexistent-m3b-step3-path; echo "exit=$?"
   ```

   Expected: `fail  config: invalid settings (…): provider.default = "nowhere"; endpoint does not exist`, with the config line as the only `fail`, then `exit=1`. Then `fail  repository: --workspace /nonexistent-m3b-step3-path: stat …: no such file or directory`, `exit=1`.

7. The help screen (authored here):

   ```bash
   # authored here
   "$KIRSCH_BIN" --help | grep -n doctor
   ```

   Expected: `4:  kirsch doctor [flags]` and `8:  doctor    check the installation and report problems`.

**Notes / evidence**

Run for mission-20261029-03, step 3, on 2026-10-29, in zsh 5.9, from the kirsch folder. The commands are in `.claude/memory/workspace/m3b-step3-run.zsh`, with the second and third passes in `m3b-step3-run2.zsh` and `m3b-step3-run3.zsh`. Their output, verbatim, is in `.claude/memory/workspace/m3b-step3-runs.txt`. These workspace files are archived at the next mission start. The binary came from `npm run build` (exit 0), unstamped, so the version line reads `kirsch 0.1.0-dev`. The checkout was at `686d3c4` with uncommitted changes. The runs set `XDG_CONFIG_HOME` to a fresh empty folder, so the developer's own settings file was not read.

- The hands-on steps matched the expected lines above, as they were printed in the step 3 runs, except where the step 3b note below says otherwise.
- Automated checks: each command name in the automated list passed, with no `--- FAIL`. The retired name `TestRunDoctorSkeletonReport` selects no test, and `TestRunDoctorBeforeWorkspaceAndConfig` selects one, in the same run.
- Not run: a real Mac with git removed from the PATH by hand. The shim tests cover that case. The "git present but erroring" limit is not tested, and it is recorded as a carried item in `plan/PROGRESS.md`.
- The folders made in these runs are listed at the end of the runs file. Nothing in the runs removed them.
- Config state, corrected 2026-10-29 by step 3a of mission-20261029-03: the step 3 runs above set `XDG_CONFIG_HOME` to a fresh empty folder, not to the Setup folder that this section's Context names. Step 3a reran the §12 commands in the declared Setup context, after `source scripts/m5-test-env.sh` and `guard`, with the outputs in `.claude/memory/workspace/m3b-step3a-runs.txt` (scratch, archived at the next mission start). That covers every operator §12 step, hands-on 6 with its new `cd` line, and every automated command in this section. The other hands-on steps were not rerun in step 3a. Operator §7 step 5 was rerun from `$T_CFG` in the same way, and its output is in the operator checklist.
- Step 3b (docs fix round 2, 2026-10-29, mission-20261029-03): hands-on steps 2 and 3 were rerun with the `cd` lines added to them, and the checks in the operator's section 12 step 7 were rewritten to add a package count and a failure count. The rewritten step 7 was run with a pattern that selects nothing, and its first number was `0`. That run is recorded in `.claude/memory/workspace/m3b-step3b-runs.txt` (scratch, archived at the next mission start). The five-count rule it tested was replaced in step 3c, below.
- Step 3c (docs fix round 3, 2026-10-29, mission-20261029-03): operator section 12 step 7 now has two commands. Command A counts the tests the pattern selects (`go test -list ... | grep -c '^Test'`), and must print a number that is not `0`. Command B runs the same tests unpiped and ends with `echo "exit $?"`, and must end `exit 0`. The tick box names both conditions. Four runs show each failure case. Run 1, the good case: A printed 33 and B ended `exit 0`. Run 2, a pattern that selects nothing: A printed `0`, and B still ended `exit 0`, with `[no tests to run]` on both package lines, so command A is what catches it. Run 3, a compile error injected into `internal/doctor/probes.go` through `-overlay` (a syntax error, not a type error): A printed `0`, and B printed a `#` line and `FAIL` lines with `[build failed]` for both packages, and ended `exit 1`. Run 4, one selected test made to fail through `-overlay` (`TestFailed`): A printed 33, and B printed `--- FAIL: TestFailed`, a `FAIL` line for `internal/doctor`, and ended `exit 1`. The runs are in `.claude/memory/workspace/m3b-step3c-runs.txt` (scratch, archived at the next mission start). The overlay files were scratch copies in `.claude/memory/workspace/m3b-step3c-mut/`; no repository file was edited.

- Docs fix round 1 (mission-20261029-06, step 3a): the expected results above that quoted summary counts were stale once m5-d4a added its lines, so they now describe each check by its property. The hands-on 1 report from the kirsch folder, with no key set, was measured on 2026-10-29 and prints the new lines, with its `key` line starting `warn`, and it ends `exit=0` (`.claude/memory/workspace/m20261029-06-step3a-runs.txt`, scratch, archived at the next mission start). The §20 hands-on 3 and 7 runs of step 3a used folders under the workspace with `GIT_CEILING_DIRECTORIES` set to the workspace root, so git treated them as outside any repository (see §20, *Notes / evidence*). The system-temporary folders the hands-on steps make with `mktemp -d` are outside any repository without that setting.

---

## 13 · Soft-wrap fidelity: `wrap()` (m5-d7a)

**Context:** Terminal: Setup. Binary: `go test`; `KIRSCH_REGEN=1 go test` for the grids. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

- `wrap` (`internal/tui/view.go`) keeps leading indentation and every run of interior spaces. It breaks lines at spaces and hard-breaks only a word wider than the frame, at the frame width in cells. The run of spaces at a break is consumed, so no continuation line starts with a space, and there is no hanging indent. Indentation of the frame width or more is clamped to one cell short of the frame. Tabs expand to 4-column stops inside `wrap`. Other Unicode spaces (newline, carriage return, NBSP, U+3000) become one plain space, so the width measured is the width drawn. A rune wider than the frame is emitted alone. Before this change `wrap` looped forever on such a rune; that is fixed, and `TestWrapWideRunes` pins it.
- Callers whose output moves when their text has indentation or repeated spaces: `renderUser` and `renderAssistant` (the user and assistant text, `internal/tui/cards.go:113, :124`), `renderError` (the error message only, `cards.go:335`; the hint is appended after the wrap, `:336–338`), and `renderNotice` (the notice text, `cards.go:348`). The onboarding rows (`internal/tui/onboarding.go:52, :61, :69, :77`) also call `wrap`, but with fixed single-spaced strings, so their output does not move. The list is derived by this command, and re-run it before the list is trusted:

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n 'wrap(' internal/tui/*.go | grep -v _test | grep -v 'func wrap'
  ```

  The same grep also prints `composer.go:77` and `:83`. Those lines are comments about textarea's own wrap, not callers, so they are not in the list above.
- Golden fences that moved: **none.** `plan/spec/kirsch-ui-screens.md` is byte-identical to its pre-change copy (see *Before any edit*), and `git diff --stat 686d3c4 -- plan/spec/kirsch-ui-screens.md` prints nothing. The operator's check is that this file has no changes (`plan/testing/manual/user-testing/milestone-5-donovan.md`, section 13, step 3).
- Bundled fix, a pre-existing defect: streamed assistant text after the first chunk was not sanitised, so escape sequences, control characters and new lines in later chunks reached the screen or were lost. Now `TextBlock.Raw` keeps the whole streamed text, and `Transcript.AppendText` recomputes `Lines` with `SanitizeLines` over all of `Raw` on every delta, so escapes, CRLF and multi-byte runes split across deltas are handled. The raw text of the assistant's streamed reply (`TextBlock.Raw`) is released when the turn ends. A cancelled turn releases it too, unless `cancelTurn` returns early because a new session is pending (`m.newSessionPending`) or a compaction is running (`m.compacting`). Those two early returns come before the clear. The sites that clear it are listed by this command, which at writing prints `update.go:443` (the `TurnCompleteMsg` case), `:464` (the `TurnErrorMsg` case) and `:1389` (in `cancelTurn`, `:1358`):

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n 'Text.Raw = ""' internal/tui/*.go
  ```

  `TurnCancelledMsg` (`update.go:474`) reaches the clear at `:1389` through `cancelTurn`, and a second call to `cancelTurn` is at `:743`. Thinking `Raw` is kept (see below). Thinking text is sanitised when read, by `ThinkingCard.BodyLines`, over all of `ThinkingCard.Raw`.
- Known limits, carried and not fixed: thinking `Raw` is never released, because no stream-end hook exists for thinking cards. Error-card messages and notices still reach `wrap` unsanitised, as before this mission (the `wrap` docblock says so). The per-delta re-sanitise cost grows with the square of the reply length; it is recorded in *Carried from Milestone 5 planning* in `plan/PROGRESS.md`.

**Waits on:** O23 (golden handling, recorded first), m5-d1 (both touch `view.go`), and m5-d6 (`composer.go`, `update.go` order), from `plan/milestones/milestone-5.md:105`.

**Before any edit.** Save the pre-change copy of `plan/spec/kirsch-ui-screens.md` to `.claude/memory/workspace/m7a-kirsch-ui-screens.pre.md`, after m5-d1 has landed and before this edit (`plan/milestones/milestone-5.md:809`). The milestone requires the save but gives no name, so this name is chosen here, with the `m7a-` prefix that mission-20261029-02 gives its workspace files.

**Automated checks**

- [ ] A pasted 200-column indented trace keeps its indentation (`plan/milestones/milestone-5.md:812`). What is tested: `TestWrapIndentedTrace200` checks an indented line wider than 200 columns, wrapped at 200, 80 and 33 columns, at the `wrap` level. `TestTranscriptWrapsIndentedLines` checks indentation through `appendBlock` and `View` on a shorter line that is wider than its 40-column frame. No test drives a 200-column line through the transcript, so the box stays unticked. Command:

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -count=1 -v ./internal/tui -run 'TestWrapIndentedTrace200|TestTranscriptWrapsIndentedLines' | grep -E '^--- (PASS|FAIL)'
  ```

  Expected: a `PASS` line for each test the command names, and no `FAIL`.
  Left unticked on purpose: the evidence is weaker than the box. The step 4a run (`.claude/memory/workspace/m7a-step4a-runs.txt`, section H) printed both `PASS` lines, but `TestWrapIndentedTrace200` tests `wrap` alone, and the transcript test uses a shorter line in a 40-column frame, so no run takes a 200-column line through the transcript.
- [✅] Runs of interior spaces survive in the transcript (`plan/milestones/milestone-5.md:813`). Covered by `TestWrapKeepsInteriorSpaceRuns` and `TestTranscriptKeepsIndentAndSpaceRuns`, which drives `appendBlock` and `View` on a user card and an assistant block. Command:

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -count=1 -v ./internal/tui -run 'TestWrapKeepsInteriorSpaceRuns|TestTranscriptKeepsIndentAndSpaceRuns' | grep -E '^--- (PASS|FAIL)'
  ```

  Expected: a `PASS` line for each test the command names, and no `FAIL`.
  Ticked on the step 4a run (`.claude/memory/workspace/m7a-step4a-runs.txt`, section H): each command printed `PASS` for each test it names, and no `FAIL`.
- [✅] `wrap` has direct tests in `internal/tui/wrap_test.go`, and they pass (`plan/milestones/milestone-5.md:814`). The group is every `TestWrap*` test plus the two transcript tests above. Command:

  ```bash
  cd /Volumes/DATA/Github/kirsch && W='TestWrap|TestTranscriptKeepsIndentAndSpaceRuns|TestTranscriptWrapsIndentedLines'; go test -count=1 -v ./internal/tui -run "$W" > .claude/memory/workspace/m7a-step4-wrap-v.txt 2>&1; echo "tests in group: $(go test -list "$W" ./internal/tui | grep -c '^Test')"; grep -c '^--- PASS' .claude/memory/workspace/m7a-step4-wrap-v.txt; grep -c '^--- FAIL' .claude/memory/workspace/m7a-step4-wrap-v.txt
  ```

  Expected: the group size, then the passing count, which should equal the group size, then `0` failures. All three numbers come from this command, not hand-kept. The command writes the scratch log `.claude/memory/workspace/m7a-step4-wrap-v.txt`. Scratch files under `workspace/m7a-*` are archived at the next mission start, so copy the numbers into the runs file for the record.
  Ticked on the step 4a run (`.claude/memory/workspace/m7a-step4a-runs.txt`, section F): the group has 18 tests, 18 `PASS` and 0 `FAIL`.
- [✅] Streamed text is sanitised across chunks (added here; the milestone has no box for it). The group is every `Streamed*` test, `TestTranscriptStreamedTabNeverReachesScreen`, the `AppendText` and `AppendThinking` seeding and refusal tests, `TestStreamEndReleasesRaw` and `TestThinkingBodyLines`. Command:

  ```bash
  cd /Volumes/DATA/Github/kirsch && S='Streamed|AppendText|AppendThinking|StreamEndReleasesRaw|ThinkingBodyLines'; go test -count=1 -v ./internal/tui -run "$S" > .claude/memory/workspace/m7a-step4-stream-v.txt 2>&1; echo "tests in group: $(go test -list "$S" ./internal/tui | grep -c '^Test')"; grep -c '^--- PASS' .claude/memory/workspace/m7a-step4-stream-v.txt; grep -c '^--- FAIL' .claude/memory/workspace/m7a-step4-stream-v.txt
  ```

  Expected: the group size, then the passing count, which should equal the group size, then `0` failures. The group's selection is derived by `go test -list` in the same command. The command writes the scratch log `.claude/memory/workspace/m7a-step4-stream-v.txt`, which is archived at the next mission start, like the other `m7a-*` scratch files.
  Ticked on the step 4a run (`.claude/memory/workspace/m7a-step4a-runs.txt`, section G): the group has 12 tests, 12 `PASS` and 0 `FAIL`.
- [✅] Golden check, procedure step 5 (`plan/milestones/milestone-5.md:53`). The whole-package run is the evidence that the screens still match: it runs the golden tests with the rest of `internal/tui`.

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -count=1 ./internal/tui
  cd /Volumes/DATA/Github/kirsch && npm run screens
  ```

  Expected: the first command prints `ok` with the package path and a time, and no `FAIL` line. The second prints two lines starting with `>` (npm naming the script it runs), then `kirsch-ui-screens.md: grids OK, content OK`.
  Ticked on the step 4a run (`.claude/memory/workspace/m7a-step4a-runs.txt`, section D): the whole-package run printed `ok` with no `FAIL`, and `npm run screens` printed `kirsch-ui-screens.md: grids OK, content OK`.

**Hands-on checks**

1. Record the sized fences that move by header (`plan/milestones/milestone-5.md:809`). The pre-change copy is named in *Before any edit* above. Result: none moved. The git comparisons are made against `686d3c4`, the pre-mission baseline commit, so they stay valid after later commits. The control is `internal/tui/view.go`, which does change since `686d3c4`, so its `git diff --stat` shows a line:

   ```bash
   cd /Volumes/DATA/Github/kirsch && cmp plan/spec/kirsch-ui-screens.md .claude/memory/workspace/m7a-kirsch-ui-screens.pre.md && echo "screens: identical to pre-change copy"
   cd /Volumes/DATA/Github/kirsch && git diff --stat 686d3c4 -- plan/spec/kirsch-ui-screens.md
   cd /Volumes/DATA/Github/kirsch && git diff --stat 686d3c4 -- internal/tui/view.go
   cd /Volumes/DATA/Github/kirsch && go test -count=1 ./internal/tui
   cd /Volumes/DATA/Github/kirsch && npm run screens
   ```

   The `cmp` command reads the scratch copy `.claude/memory/workspace/m7a-kirsch-ui-screens.pre.md`, which is archived at the next mission start. It works only until then; after that, use the `686d3c4` comparison alone.

   The first command prints the `identical` line. The second prints nothing. The third prints one line naming `internal/tui/view.go`. The fourth prints `ok` and no `FAIL`. The fifth prints two lines starting with `>`, then `kirsch-ui-screens.md: grids OK, content OK`.

2. Regenerate and reconcile, by the procedure at `plan/milestones/milestone-5.md:49–53`:

   ```bash
   cd /Volumes/DATA/Github/kirsch && KIRSCH_REGEN=1 go test ./internal/tui -run TestGenerateGrids
   ```

   The grids go to `/tmp/kirsch-grids.txt`. A person reconciles each affected fence in `plan/spec/kirsch-ui-screens.md`. Not run in mission-20261029-02 step 4: no fence moved, so there was nothing to reconcile, and no regeneration was needed.

3. (operator) A person reviews the golden diff before the step closes (`plan/milestones/milestone-5.md:816`). No fence moved, so there is no golden diff to review. The operator's check is step 3 of section 13 in the plain checklist: the reference-screens file has no changes against `686d3c4`, `go test -count=1 ./internal/tui` prints `ok`, and `npm run screens` prints `grids OK, content OK`. If the first command prints anything, the operator does not tick; they write what they saw under *Anything odd?* and tell Bishop, because a later piece may have changed the reference-screens file. The second command must print a line naming `internal/tui/view.go`, so that the control is seen to work.

**Notes / evidence**

Run in mission-20261029-02, step 4 and its fix round, step 4a, on 2026-10-29, from zsh. The output is in `.claude/memory/workspace/m7a-step4a-runs.txt`. The step 4 verbose logs survive as `.claude/memory/workspace/m7a-step4a-prior-wrap-v.txt` and `.claude/memory/workspace/m7a-step4a-prior-stream-v.txt`. The step 4a verbose logs are `m7a-step4-wrap-v.txt` and `m7a-step4-stream-v.txt`. Workspace files are archived at the next mission start.

- Wrap group (`TestWrap*` and the two transcript tests): section F of the runs file records 18 tests, 18 `PASS` and 0 `FAIL`.
- Streamed group: section G records 12 tests, 12 `PASS` and 0 `FAIL`, `TestTranscriptStreamedTabNeverReachesScreen` included.
- `go test -count=1 ./internal/tui` (whole package): `ok`. This is the run that proves the screens still match.
- Golden check: `plan/spec/kirsch-ui-screens.md` is byte-identical to its pre-change copy, and `git diff --stat 686d3c4` prints nothing for it. The positive control, `git diff --stat 686d3c4 -- internal/tui/view.go`, prints one line. `npm run screens` prints `kirsch-ui-screens.md: grids OK, content OK`.
- Goldens that moved: **none.** Nothing was regenerated.
- Not run: `KIRSCH_REGEN=1` (no fence moved). No kirsch launch and no live provider run; none of these steps needs a key.

---

## 14 · Paste debounce (m5-d8)

**Context:** Terminal: Setup. Binary: `go test` for the checks; `kirsch` for the optional hands-on paste check (below). Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a for the checks; `--workspace "$WORKSPACE"` on the hands-on launch. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

- **O9 branch taken (2026-10-29): amend the spec, do not build the 20 ms joining.** Bracketed paste already gives one message per paste in the normal case, so the debounce would add nothing there.
- **Code:** `PasteDebounce` is deleted from `internal/tui/styles.go`. `grep -rn PasteDebounce internal/` prints nothing.
- **Spec:** `plan/spec/ui-spec-v0.1.md` §7.2 and §11 now describe one update per bracketed paste, and no debounce. The "Approval card appearance … Never debounced" row is unchanged.
- **Tests:** `internal/tui/paste_nobracket_test.go` adds `TestUnbracketed5KBPasteLandsWholeThroughKeyMessages` and `TestUnbracketedMultiLinePasteCurrentBehaviour`. Both drive `Model.Update` with key messages built to match bracketed-off delivery. Bubble Tea's own reader is not run.
- **Limits when bracketed paste is off** (the terminal does not send pastes as one block):
  - A multi-line paste sends its first line to the model as a message. The rest of the paste is dropped while the turn is busy. `TestUnbracketedMultiLinePasteCurrentBehaviour` pins this. A test that feeds Enter key messages shows this; that a pasted newline arrives as Enter is read from Bubble Tea's source, not run. It is current behaviour, not a wanted one, and it is an open operator decision (see `plan/PROGRESS.md`, *Open from m5-d8*).
  - A tab arrives as the Tab key. The arrival as `tea.KeyTab` is read from Bubble Tea's source (`keyHT`, `key.go`) and is not run. Tab either completes a unique slash-command name or moves focus to the transcript (the `tea.KeyTab` case in `keyComposing`, `internal/tui/update.go`). It does neither when the typing area holds a slash name that matches several commands or none (`completeSlash` is not unique, `internal/tui/composer.go`); the rest of the paste is then typed in as normal. When focus moves, the rest of the paste goes to `keyBrowsing`, which puts nothing into the composer, and the letters `d`, `g`, `G` and `?` act as browsing commands there (`d` opens detail, `g` and `G` move the view, `?` opens help). A later Tab in the same paste moves focus back to the typing area (`keyBrowsing` handles `tea.KeyTab` and calls `setBase(BaseComposing)`), so the pasted text is split between the two. What kirsch does with the key is read from `update.go` and is not tested.
  - A multi-byte character (for example `é`) that falls on a terminal read boundary can be lost, because the rune never arrives as a rune: its bytes arrive as unknown-input messages. This is read from Bubble Tea's reader (`detectOneMsg`, `readAnsiInputs` in `key.go`, bubbletea v1.3.10) and is not tested. Kirsch cannot fix it without changing the reader.
  - A 20 ms debounce would not have fixed the multi-byte limit. That is why the spec change was chosen.

**Waits on:** m5-d6 (the paste path) and O9 (paste debounce), from `plan/milestones/milestone-5.md:107`. Bracketed paste already gives one message in the normal case (`github.com/charmbracelet/bubbletea@v1.3.10/key_sequences.go:106–118`, the Bubble Tea module pinned at `go.mod:8`). The milestone now cites `github.com/charmbracelet/bubbletea@v1.3.10/key_sequences.go:106–118` (`plan/milestones/milestone-5.md:71`, and the *m5-d8 — Paste debounce* section's *Items covered* and *Must-know-before*), the same file this section cites.

**Before any edit.** Save verbatim copies of `internal/tui/styles.go` to `.claude/memory/workspace/m5-d8-styles.pre.go` and of `plan/spec/ui-spec-v0.1.md` to `.claude/memory/workspace/m5-d8-ui-spec.pre.md` (*m5-d8 — Paste debounce*, *Before any edit*, in `plan/milestones/milestone-5.md`). The controls below run on the committed baseline `686d3c4`, not on these copies (dated note, 2026-10-29, mission-20261029-04 step 3a). The copies are scratch and are archived at the next mission start.

**Key-state check, first.** Authored here. Run it before any kirsch launch in this section. It must print `CONTROL PASS`, then `0`.

```bash
# authored here
keycount_control
keycount
```

**Automated checks**

- Implement branch: not taken. O9 chose the amend branch, so there is no 20 ms test.
- [✅] Amend branch, the greps. Each prints nothing and ends `exit 1` (no match), run 2026-10-29 in mission-20261029-04. The pattern is case-sensitive on purpose. `grep -n -i debounce` also matches the unrelated "Never debounced" row.

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -rn PasteDebounce internal/; echo "exit $?"
  cd /Volumes/DATA/Github/kirsch && grep -n 'Paste debounce' plan/spec/ui-spec-v0.1.md; echo "exit $?"
  ```

- [✅] Controls, run against the committed baseline `686d3c4` (dated note, 2026-10-29, mission-20261029-04 step 3a). The first prints the declaration at line 22 and ends `exit 0`. The second prints the §7.2 bullet at line 565 and the §11 row at line 758, and ends `exit 0`. Run output: `.claude/memory/workspace/m8-step3a-runs.txt` (scratch, archived at the next mission start).

  ```bash
  cd /Volumes/DATA/Github/kirsch && git show 686d3c4:internal/tui/styles.go | grep -n PasteDebounce; echo "exit $?"
  cd /Volumes/DATA/Github/kirsch && git show 686d3c4:plan/spec/ui-spec-v0.1.md | grep -n 'Paste debounce'; echo "exit $?"
  ```

  Expected, in order: `22:` followed by the `PasteDebounce` declaration, then `exit 0`; `565:` and `758:` lines containing `Paste debounce`, then `exit 0`. Do not tick this box if the lines differ. A different line means the command or the commit is wrong, and the check has not been run.

- [✅] Unbracketed 5 KB paste lands whole, for single-line text through `Model.Update`. The text the test feeds is words, spaces and a two-byte character, with no tabs. The test is `TestUnbracketed5KBPasteLandsWholeThroughKeyMessages`. The run pattern is found with `go test -list`, then the test runs by name.

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -list '^TestUnbracketed' ./internal/tui | grep '^TestUnbracketed'
  cd /Volumes/DATA/Github/kirsch && go test -list '^TestUnbracketed' ./internal/tui | grep -c '^TestUnbracketed'; echo "exit $?"
  cd /Volumes/DATA/Github/kirsch && go test -count=1 -run '^TestUnbracketed5KBPasteLandsWholeThroughKeyMessages$' ./internal/tui; echo "exit $?"
  ```

  The first command prints the two names `TestUnbracketed5KBPasteLandsWholeThroughKeyMessages` and `TestUnbracketedMultiLinePasteCurrentBehaviour`. The second prints `2`, not `0`, and ends `exit 0`. The third must end `exit 0`. The first two match operator section 14, step 4. The third names the 5 KB test alone; operator section 14, step 5 runs the whole `^TestUnbracketed` set, and both tests must pass there. Failing cases, saved verbatim in `.claude/memory/workspace/m20261029-04-step3b-runs.txt` (scratch, archived at the next mission start): a pattern that selects nothing prints no names and `0`, ending `exit 1`; a scratch copy with one planted fault, run through `-overlay`, ends `exit 1`.

- [✅] Multi-line unbracketed paste is pinned as current behaviour, not fixed. `TestUnbracketedMultiLinePasteCurrentBehaviour` passes and ends `exit 0`. A pass here means the defect is recorded, not that it is resolved.

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -count=1 -run '^TestUnbracketedMultiLinePasteCurrentBehaviour$' ./internal/tui; echo "exit $?"
  ```

**Hands-on checks**

1. (operator) Optional, not run: bracketed paste off, paste 5 KB, and the UI stays responsive.

   **Only if your terminal can turn bracketed paste off.** Some terminals offer a setting that stops applications from turning it on. The name and place of that setting differ by terminal, and some terminals have none. If yours has none, skip this step. It is not in the operator checklist, so there is no box to tick there; if you run it, report the result to the crew.

   **Single line only.** Copy one line of about 5 KB with no line breaks. Do not paste multi-line text. Do not include tabs. What you would see with a tab: a Tab in the typing line finishes a slash command name only when the name matches exactly one command, and otherwise moves the cursor focus up to the transcript (the bottom line of the screen changes). After that, the rest of the paste does not appear in the typing line, and a later Tab in the same paste moves focus back to it, so the text ends up split between the two. The arrival of a tab as the Tab key is read from Bubble Tea's source and is not run; what kirsch does with it is read from its code and is not tested. With bracketed paste off, a multi-line paste sends its first line to the model as a message, and the rest is dropped.

   Expected: the text lands whole in the typing area, nothing is sent, and kirsch keeps responding to keys. Do not press Enter.

**Notes / evidence**

- Run 2026-10-29, mission-20261029-04: the grep pair and the baseline controls, the list check, the 5 KB test and the pinned multi-line test, as in the commands above. Saved output: `.claude/memory/workspace/m8-step3a-runs.txt` (scratch, archived at the next mission start).
- Re-run 2026-10-29, mission-20261029-04 step 3b (docs fix round 2): the list check in its two-line form, the 5 KB test, the pinned multi-line test (the whole `^TestUnbracketed` set, which passes), and the two failing forms (a pattern that selects nothing; `-overlay` with one planted fault in a scratch copy). Saved output: `.claude/memory/workspace/m20261029-04-step3b-runs.txt` (scratch, archived at the next mission start).
- Not run: the optional hands-on step 1 (no kirsch launch; it is not in the operator checklist). No live provider run.
- Key-state check not run in the agent shell. On 2026-10-29, `env | grep -c '^ANTHROPIC_API_KEY='` printed `1` in that shell (the count only; the value is not recorded here; output in `.claude/memory/workspace/m20261029-04-step3c-runs.txt`). `keycount` counts such lines (`scripts/m5-test-env.sh`, `keycount`), so it cannot print `0` there. This has no bearing on §14, whose checks do not start kirsch; the operator runs the key-state check in a clean Terminal.

---

## 15 · GitHub release workflow (m5-d11)

**Context:** Terminal: Setup. Binary: none for the workflow, which runs on GitHub; the archive checks run `npm run release:snapshot`, which writes `dist/` and publishes nothing. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset; this block makes no model call. Config state: Setup, unchanged.

**What landed**

- `.github/workflows/release.yml` is new. It runs when a `v*` tag is pushed, or when started by hand from the Actions tab. A tag run builds the release and publishes it. A manual run (`workflow_dispatch`) builds a snapshot, publishes nothing, and uploads the archives and checksums as an artifact named `release-dry-run`. On a tag push, a step named `tagged commit is on main` runs first. It fetches `main` and fails the run, before anything is built or published, unless the tagged commit is on `main`. A manual run skips that step.
- `scripts/goreleaser-version` is new. It holds the goreleaser version that the workflow and the snapshot both use: `v2.19.0`. v2.20.0 was published on 2026-11-04 (`gh api repos/goreleaser/goreleaser/releases/tags/v2.20.0 --jq .published_at` printed `2026-11-04T12:48:50Z`, run by Bishop on 2026-11-04). It was deliberately not pinned. The pin rule is the **Pin rule (Bishop, 2026-11-04)** bullet in *Open from m5-d11* in `plan/PROGRESS.md`: the pin is not a release published on the day it is chosen.
- `scripts/third-party-licenses.sh` is new. It runs before the build, as a goreleaser `before:` hook. It copies the licence and notice files of every module that `./cmd/kirsch` links into `build/licenses/`, and Go's own licence. A module with no licence file fails the release. `build/` is gitignored.
- `scripts/release-snapshot.sh` is new, with `npm run release:snapshot`. It builds the release archives into `dist/` on your machine and publishes nothing. Nothing is installed. The first run downloads goreleaser and, only if go.mod's toolchain is older than goreleaser needs, a newer Go toolchain, so it takes minutes.
- `.goreleaser.yaml` gains the `before:` hook and an explicit archive `files:` list: `LICENSE`, `README.md`, `CHANGELOG.md`, `NOTICE`, and `build/licenses/**/*` into `licenses/`. The list is the source of truth for what an archive holds. The archive also holds the binary, `kirsch`.

**Trigger, permissions and pinning, as O3 records them.** O3 (`plan/PROGRESS.md`, *Milestone 5 — decisions recorded*) says the workflow runs when a `v*` tag is pushed, can be started by hand for a dry run, has permission only to create the release, and pins its tools to fixed versions. The file's `on:` block:

```yaml
on:
  push:
    tags: ['v*']
  workflow_dispatch:
```

- Trigger: a `v*` tag push, and a manual dispatch. There is no `pull_request` trigger, so no release build runs on a pull request. A tag push must name a commit that is on `main`. The `tagged commit is on main` step fetches `main` and fails the run, with nothing published, when it is not. A manual dispatch skips that step.
- Permissions: the workflow default is `contents: read`. Only the `release` job raises it to `contents: write`, which creating a release and uploading its assets needs.
- Pinning: actions are pinned by major tag, as `ci.yml` pins them. The grep in the third *Automated checks* item lists each `uses:` line, and every one must carry a version tag. The goreleaser version is read from `scripts/goreleaser-version`, which must hold a bare `vX.Y.Z`, or the step fails.

**Waits on:** m5-d10b (goreleaser config) and O3 (release workflow: trigger, permissions, pinning, dispatch), from `plan/milestones/milestone-5.md:111`. Repository Actions settings must allow write. That is an operator setting (`plan/milestones/milestone-5.md:1073`).

**Automated checks**

Each check is run with its control in the same command. A control is a planted copy of the text, which the same grep must catch. Run them from the kirsch folder.

- [✅] `release.yml` triggers on `v*` tags, and on dispatch only if O3 says so (`plan/milestones/milestone-5.md:1064`). The milestone gives no command line, so the grep is authored here.

  ```bash
  # authored here
  cd /Volumes/DATA/Github/kirsch && grep -n -A3 '^on:' .github/workflows/release.yml
  cd /Volumes/DATA/Github/kirsch && grep -n -E '^  pull_request' .github/workflows/release.yml; echo "exit=$?"
  printf 'on:\n  pull_request:\n' | grep -n -E '^  pull_request'
  ```

  Expected: the first command prints lines 6 to 9, `on:`, `  push:`, `    tags: ['v*']` and `  workflow_dispatch:`. The second prints nothing and `exit=1`, so there is no `pull_request` trigger. The third is the control, and it must print `2:  pull_request:`.

- [✅] The write permission is scoped as O3 records (`plan/milestones/milestone-5.md:1065`). The milestone gives no command line, so the grep is authored here.

  ```bash
  # authored here
  cd /Volumes/DATA/Github/kirsch && grep -n -E '^permissions:' -A1 .github/workflows/release.yml
  cd /Volumes/DATA/Github/kirsch && grep -n -E '^ +contents: write$' .github/workflows/release.yml
  printf 'permissions:\n  contents: write\n' | grep -n -E '^ +contents: write$'
  ```

  Expected: `12:permissions:` with `13-  contents: read` under it, the workflow default. Then one line, `29:      contents: write`, which is in the `release` job only. The control must print `2:  contents: write`. The pattern is anchored to a key line, so the comments that mention `contents: write` do not match.

- [✅] Action pinning matches O3 (`plan/milestones/milestone-5.md:1066`). The goreleaser version check is in the same block.

  ```bash
  # authored here
  cd /Volumes/DATA/Github/kirsch && grep -n 'uses:' .github/workflows/release.yml
  cd /Volumes/DATA/Github/kirsch && grep -n 'uses:' .github/workflows/release.yml | grep -c -v '@v[0-9]'; echo "exit=$?"
  printf '      - uses: actions/checkout\n' | grep -n 'uses:' | grep -v '@v[0-9]'
  cd /Volumes/DATA/Github/kirsch && grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' scripts/goreleaser-version
  printf 'latest\n' | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$'; echo "exit=$?"
  ```

  Expected: four `uses:` lines, `33:      - uses: actions/checkout@v4`, `50:      - uses: actions/setup-go@v5`, `76:      - uses: goreleaser/goreleaser-action@v7` and `88:        uses: actions/upload-artifact@v4`. The count of lines with no version tag prints `0` and `exit=1`. Its control prints `1:      - uses: actions/checkout`, an unpinned line. The version prints `v2.19.0`. Its control, `latest`, prints nothing and `exit=1`.

- [✅] (authored here, not a milestone box) The archives hold the notices and the licence texts. Run after `npm run release:snapshot`, which writes `dist/`. This checks the archive against the `files:` list in `.goreleaser.yaml`.

  ```bash
  # authored here
  cd /Volumes/DATA/Github/kirsch && npm run release:snapshot; echo "exit=$?"
  cd /Volumes/DATA/Github/kirsch && tar -tzf dist/kirsch_*_darwin_arm64.tar.gz | sort | grep -E '^(kirsch|LICENSE|README\.md|CHANGELOG\.md|NOTICE|licenses/go/LICENSE)$'
  cd /Volumes/DATA/Github/kirsch && tar -tzf dist/kirsch_*_darwin_arm64.tar.gz | grep -c '^licenses/'
  cd /Volumes/DATA/Github/kirsch && for f in dist/kirsch_*.tar.gz; do echo "$(tar -tzf "$f" | wc -l | tr -d ' ') $f"; done
  printf 'NOTICE\nsecret.txt\n' | grep -c -E '^(kirsch|LICENSE|README\.md|CHANGELOG\.md|NOTICE|licenses/go/LICENSE)$'
  ```

  Expected: `exit=0` from the snapshot, then the names in the order `sort` gives on the machine these were run on: `CHANGELOG.md`, `kirsch`, `LICENSE`, `licenses/go/LICENSE`, `NOTICE`, `README.md`. Then `29`, the count of files under `licenses/`. Then one line per archive, each `34` followed by the archive name. The control prints `1`: `NOTICE` matches and `secret.txt` does not. The `sort` order depends on the locale, so a different order on another machine is not a fault; a missing name is. The `29` and the `34` are the counts on 2026-11-04. They change when the libraries change or the archive's `files:` list changes, so the check is the shape of the output, not these numbers.

- [✅] (authored here, not a milestone box) The licence script prints its count line. It writes `build/licenses/`, which Git ignores.

  ```bash
  # authored here
  cd /Volumes/DATA/Github/kirsch && bash scripts/third-party-licenses.sh | grep -c -E '^third-party-licenses: [0-9]+ modules \+ Go → build/licenses$'
  printf 'third-party-licenses: several modules\n' | grep -c -E '^third-party-licenses: [0-9]+ modules \+ Go → build/licenses$'
  ```

  Expected: the first prints `1`. The control is a planted line with no count, and it must print `0`. The `28` in the line is the module count on 2026-11-04. It changes when the libraries change, so the check is the line's shape, not the number.

**Live checks**

> Live GitHub release run. Spends no model tokens. A **manual run** of the *Release* workflow on `main` is a dry run. It builds a snapshot, publishes no release, and uploads the archives and checksums as an artifact. It is safe on the real repository, not only on a fork. A **`v*` tag push is ONE-WAY**: it publishes a public release. The tag run belongs to m5-d22, and nothing in this section pushes a tag. The manual run needs the commit pushed first.

- [ ] (operator) A manual run on `main` is green, its run page lists an artifact named `release-dry-run`, and the Releases page shows no new release. Command: none. The check is on GitHub, in the Actions tab. `plan/milestones/milestone-5.md:1067` says a dispatch run "produces a release"; O3 makes a dispatch a dry run that publishes nothing, so this box is read as the dry run, and the milestone is not edited here.
- [ ] (operator, m5-d22) A `v*` tag push produces a public release with archives and checksums. ONE-WAY, and not run here. Command: none.

**Notes / evidence**

**Crew run, 2026-11-04 (mission-20261104-01, steps 1, 2a, 2c and 2e; review step 2f APPROVED).** These results are the crew's reports. Lambert did not re-run them; the step 3 re-run below covers the automated checks.

- `npm run release:snapshot` succeeded. The tail of its output:

  ```
    • calculating checksums
    • writing artifacts metadata
    • release succeeded after 7s
    • thanks for using GoReleaser!
  ```

- It wrote four archives and a checksums file. Each archive has 34 entries: `kirsch`, `LICENSE`, `README.md`, `CHANGELOG.md`, `NOTICE`, and 29 files under `licenses/`, which are 28 modules plus `licenses/go/LICENSE`.
- `bash scripts/third-party-licenses.sh` prints `third-party-licenses: 28 modules + Go → build/licenses`.
- `./dist/kirsch_darwin_arm64_v8.0/kirsch --version` printed `kirsch m4-SNAPSHOT-f6d0cd0`. The version stamp reaches the binary in a snapshot. Whether a real `v0.1.0` tag prints `0.1.0` or `v0.1.0` is not verified until the first tag, which is m5-d22's.
- `npm run ci:targets` ends `ci.sh: all 4 targets compile`.
- Controls: a module with no licence file exits 1, naming the module. The arguments `.`, `..` and `"$PWD/"` each exit 2. A symlinked `build` exits 2, and the target's `keep-me` file survives. Without the guard that file is deleted.

**Re-run, 2026-11-04 (mission-20261104-01, step 3, Lambert).** The commands in *Automated checks* were run from `/Volumes/DATA/Github/kirsch` at commit `f6d0cd0`, with the working tree uncommitted. The outputs, as printed:

```
### on: block (grep -n -A3 '^on:')
6:on:
7-  push:
8-    tags: ['v*']
9-  workflow_dispatch:

### pull_request absence (exit=1 means none found)
exit=1
### control
2:  pull_request:

### permissions (grep -n -E '^permissions:' -A1)
12:permissions:
13-  contents: read

### contents: write as a key
29:      contents: write
### control
2:  contents: write

### uses: lines
33:      - uses: actions/checkout@v4
50:      - uses: actions/setup-go@v5
76:      - uses: goreleaser/goreleaser-action@v7
88:        uses: actions/upload-artifact@v4
### unpinned count
0
exit=1
### control
1:      - uses: actions/checkout

### goreleaser-version
v2.19.0
### control (exit=1, nothing printed)
exit=1

### npm run release:snapshot (tail)
  • release succeeded after 7s
  • thanks for using GoReleaser!
exit=0

### names, sort | grep (darwin_arm64)
CHANGELOG.md
kirsch
LICENSE
licenses/go/LICENSE
NOTICE
README.md
### licenses/ entries
29
### entries per archive
34 dist/kirsch_m4-SNAPSHOT-f6d0cd0_darwin_amd64.tar.gz
34 dist/kirsch_m4-SNAPSHOT-f6d0cd0_darwin_arm64.tar.gz
34 dist/kirsch_m4-SNAPSHOT-f6d0cd0_linux_amd64.tar.gz
34 dist/kirsch_m4-SNAPSHOT-f6d0cd0_linux_arm64.tar.gz
### archive filter control
1

### licence count line
1
### licence count control
0

### checksums file
dist/kirsch_m4-SNAPSHOT-f6d0cd0_checksums.txt

### kirsch --version
kirsch m4-SNAPSHOT-f6d0cd0
```

**Crew run of operator step 2, 2026-11-04 15:34 UTC (Lambert, mission-20261104-01, step 4c).** Warm cache; first-run output not captured. The command was `npm run release:snapshot; echo "result: $?"`, run from `/Volumes/DATA/Github/kirsch` on the same Mac. The output has no `go: downloading` line and no goreleaser download line, so the tools were already on this machine. It shows the shape of a warm run, not what a first run prints. Full raw output:

```
> kirsch@0.1.0-dev release:snapshot
> bash scripts/release-snapshot.sh

go: github.com/goreleaser/goreleaser/v2@v2.19.0 requires go >= 1.27.3; switching to go1.27.3
  • starting release
  • skipping announce, publish, and validate...
  • cleaning distribution directory
  • loading environment variables
  • getting and validating git state
    • using tags                                     previous=m3 current=m4
    • pipe skipped or partially skipped              reason=disabled during snapshot mode
  • parsing tag
    • current tag is not semver                      error=invalid semantic version tag=m4
    • pipe skipped or partially skipped              reason=validation is disabled
  • setting defaults
  • snapshotting
    • building snapshot...                           version=m4-SNAPSHOT-f6d0cd0
  • running before hooks
    • running                                        hook=bash scripts/third-party-licenses.sh
  • ensuring distribution directory
  • setting up metadata
  • writing release metadata
  • loading go mod information
  • build prerequisites
  • building binaries
    • building                                       paths=cmd/kirsch binaries=kirsch target=linux_arm64_v8.0
    • building                                       paths=cmd/kirsch binaries=kirsch target=linux_amd64_v1
    • building                                       paths=cmd/kirsch binaries=kirsch target=darwin_amd64_v1
    • building                                       paths=cmd/kirsch binaries=kirsch target=darwin_arm64_v8.0
  • archives
    • archiving                                      name=dist/kirsch_m4-SNAPSHOT-f6d0cd0_darwin_arm64.tar.gz
    • archiving                                      name=dist/kirsch_m4-SNAPSHOT-f6d0cd0_linux_arm64.tar.gz
    • archiving                                      name=dist/kirsch_m4-SNAPSHOT-f6d0cd0_linux_amd64.tar.gz
    • archiving                                      name=dist/kirsch_m4-SNAPSHOT-f6d0cd0_darwin_amd64.tar.gz
  • calculating checksums
  • writing artifacts metadata
  • release succeeded after 8s
  • thanks for using GoReleaser!
result: 0
```

**Open for the operator.** None of these blocks the automated checks above.

- The first live run is m5-d22's tag run, and a `v*` tag push is ONE-WAY. The manual run in the Live checks is the step to do first. It needs the commit pushed.
- Repository Actions settings must allow write (O3; `plan/milestones/milestone-5.md:1073`, the m5-d11 *Must-know-before*). The crew did not check this setting, and the workflow asks for `contents: write` at job level.
- The GitHub wording in operator step 5 has not been verified by the crew: the *Actions* tab, *Release*, and *Run workflow*. *Release* is the workflow's `name:` and `release-dry-run` is the artifact name in the file, but GitHub's own button labels are not in the file and were not checked. The operator copy asks the operator to stop and tell Bishop if *Run workflow* cannot be found.
- A tag push must name a commit that is on `main`. The `tagged commit is on main` step fails any tag run on a commit that is not, before anything is published. Push `main` first, then tag a commit that is on it. The first tag is m5-d22's. The tag gate has not been exercised on a runner, so its behaviour is read from `release.yml`, not observed.
- W-2: a `v*` tag on a commit that is on `main`, but whose CI failed, still publishes. O3 does not require a CI-passed gate, so whether to add one is the operator's call.
- The milestone's box at `plan/milestones/milestone-5.md:1067` says a dispatch run "produces a release". O3 makes a dispatch a dry run. The box is read here as the dry run, and the milestone is not edited. Bishop should decide whether its wording changes.

**Carried, non-blocking.**

- The comment in `scripts/third-party-licenses.sh` says "a link there means the output has been pointed somewhere unexpected". That claims more than the check does.
- The symlink guard runs before the `go list` calls, not next to the `rm`.
- The arch axis of the module union has no mutant coverage. `go list` with an unknown GOARCH gives the same module set, so no control separates the arches.
- Action pins are by major tag, as O3 records. A commit-SHA pin for `goreleaser-action` in the write job was suggested and not taken.

_Record the run URL and the release page at the live run, not any credential._

---

## 16 · Key-coverage test (S36) (m5-d13b)

**Context:** Terminal: Setup. Binary: none (no kirsch launch; the checks run `go test`). Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged. The removal control edits a copy of `doc/configuration.md`, never the real page.

**What landed**

m5-d13b adds the test file `internal/config/key_coverage_test.go`. It changes no product code, and it does not change `doc/configuration.md`: the step 1 diff lists that test file alone, and the page is byte-identical to its pre-step copy (the evidence is under *Notes / evidence* below). The file holds these tests:

- `TestKeyCoverage_PageDocumentsEveryKey` fails, naming each item, for every settings section and every key in the code that `doc/configuration.md` does not list. It reads the page from the module root, which `moduleRoot` finds by walking up to `go.mod`.
- `TestKeyCoverage_DerivedListsAreSane` is the positive control. It fails if the derived lists are empty, if a known section (`policy`, `telemetry`) or one key from each extraction route is missing, or if the matcher accepts a page that is empty or that names a key only in prose. A table row is accepted.

The keys come from the sources below, read by `documentedKeys`:

- `reflect` over `Config`: each section's `toml` tag, and each tagged field of its struct (for example `policy.env_passthrough`, from `PolicyConfig`).
- `go/ast` over `ProviderConfig.UnmarshalTOML`, because the provider keys are decoded by hand. The `case` labels give the endpoint fields (`provider.<name>.base_url`), and the `key ==` comparison gives `provider.default`.

"Documented" has the exact meaning `missingFromPage` gives it: a section needs a heading ``### `[section]` `` on the page, and a key needs a table row beginning ``| `dotted.key` |``. A key named in a paragraph does not count.

The section structs sit within `config.go:57–80`: `PolicyConfig` starts at `:57` and `TelemetryConfig` at `:78`. `Config` starts at `:27`. The provider decoder is `ProviderConfig.UnmarshalTOML`, which starts at `config.go:176`; its endpoint `switch fkey` runs `config.go:219–259`. This section cites symbols, and the test reads symbols, not line numbers.

**Done:** m5-d13a (the page the test reads) is done in mission-20261027-06, so the page exists. Source of the dependency: `plan/milestones/milestone-5.md:114`.

**Automated checks**

- [✅] The test passes against the current page. Acceptance source: `plan/milestones/milestone-5.md:1178`. Command: `go test -count=1 ./internal/config -run KeyCoverage -v` (run from the module root). The milestone names `internal/config/key_coverage_test.go` and gives no command line; the command is authored here. Output of the run in mission-20261027-07 step 3, 2026-10-27:

  ```
  === RUN   TestKeyCoverage_DerivedListsAreSane
  --- PASS: TestKeyCoverage_DerivedListsAreSane (0.00s)
  === RUN   TestKeyCoverage_PageDocumentsEveryKey
  --- PASS: TestKeyCoverage_PageDocumentsEveryKey (0.00s)
  PASS
  ok  	github.com/djm56/kirsch/internal/config	0.605s
  ```

- [ ] Removing one key from `doc/configuration.md` makes the test fail. Acceptance source: `plan/milestones/milestone-5.md:1179`. The control was recorded in mission-20261027-07 step 1 (the removal run is in the hands-on item below). It was not re-run in mission-20261027-07 step 3, so this box stays unticked until it is.

**Hands-on checks**

1. The removal-and-revert control (`plan/milestones/milestone-5.md:1175`). Recorded in mission-20261027-07 step 1, not re-run in step 3.
   - **Key removed:** `policy.env_passthrough`. The page names it only in its table row, so deleting that row is the whole edit. The key appears on one line of the page, that row (the `grep` is under *Notes / evidence*).
   - **Exact edit:** in a copy of `doc/configuration.md`, delete the one line that begins ``| `policy.env_passthrough` |``. Nothing else changes.
   - **Run and result** (recorded in mission-20261027-07 step 1, control 1; the copy was a scratch page, not the real one):

     ```
     === RUN   TestKeyCoverage_DerivedListsAreSane
     --- PASS: TestKeyCoverage_DerivedListsAreSane (0.00s)
     === RUN   TestKeyCoverage_PageDocumentsEveryKey
         key_coverage_test.go:202: doc/configuration.md does not document:
               key policy.env_passthrough
     --- FAIL: TestKeyCoverage_PageDocumentsEveryKey (0.00s)
     FAIL
     FAIL	github.com/djm56/kirsch/internal/config	0.219s
     FAIL
     exit=1
     ```

     The removal is a page fault, not an extraction fault: `TestKeyCoverage_DerivedListsAreSane` passes in the same run, as its line above shows.
   - **Revert:** the removal was made in a scratch copy, so the real page was never edited. Control 3 restored the scratch copy, checked it against the repo page with `cmp`, and ran the tests (recorded in mission-20261027-07 step 1):

     ```
     page identical to repo page
     === RUN   TestKeyCoverage_DerivedListsAreSane
     --- PASS: TestKeyCoverage_DerivedListsAreSane (0.00s)
     === RUN   TestKeyCoverage_PageDocumentsEveryKey
     --- PASS: TestKeyCoverage_PageDocumentsEveryKey (0.00s)
     PASS
     ok  	github.com/djm56/kirsch/internal/config	0.212s
     exit=0
     ```

**Notes / evidence**

- The automated check output is in the Automated checks block above. It is from the run in mission-20261027-07 step 3.
- Evidence for *What landed*, taken in mission-20261027-07 step 5 on 2026-10-27:

  `git status --short -- internal cmd doc`:

  ```
   M doc/README.md
   M doc/usage.md
   M internal/tool/run_command_test.go
   M internal/tui/styles.go
   M internal/tui/view_smoke_test.go
  ?? doc/configuration.md
  ?? internal/config/key_coverage_test.go
  ?? internal/tui/approval_unspecified_test.go
  ?? internal/tui/styles_test.go
  ```

  Only `internal/config/key_coverage_test.go` is this milestone's work in that list. The step 1 diff has one file header, for that test file. The other entries predate mission-20261027-07's steps. `doc/configuration.md` is untracked from m5-d13a (mission-20261027-06).

  `cmp doc/configuration.md` against the copy taken before step 1 of this mission reports the two files identical.

  `grep -n env_passthrough doc/configuration.md`:

  ```
  128:| `policy.env_passthrough` | `[]` | A list of variable names | Global | Extra environment variables passed to commands the agent runs, on top of `PATH`, `HOME`, and `LANG`. Names ending in `_TOKEN`, `_KEY`, or `_SECRET`, and names starting with `AWS_`, are removed even when listed. The match is case-sensitive: only the upper-case forms are matched, so a listed `my_token` reaches the command, and other secret-looking names, such as `GITHUB_PAT`, are not matched either. |
  ```

  The key therefore appears only on that line, the table row.
- Step 1 also ran a removal of the `provider.<name>.thinking` row together with the `[telemetry]` heading, and mutants of the extraction and of the matcher (a broken extraction that yields no sections, an AST route that drops the endpoint case labels, and a matcher that accepts any line containing the key). Those runs are not reproduced in this manual, and nothing here relies on them.
- An early expected result for the removal run said the key is also named in prose. That was wrong: the page names it only in its table row, so the run above shows the page fault alone. The prose-rescue property is checked by the prose-only control in `TestKeyCoverage_DerivedListsAreSane` (`key_coverage_test.go:180–183`), not by the removal run.
- The gates `npm run check`, `npm run fmt:check` and `npm run lint` were run in step 1 and are not repeated here.
- The optional operator demonstration (a throwaway copy with one row removed, run by the operator) is not in the operator checklist. It was not run first in this step, and the checklist does not include steps that have not been run.

---

## 17 · `doc/install.md` (m5-d13c)

**Context:** Terminal: the Setup terminal for the automated check and the operator runbook steps in `plan/testing/manual/user-testing/milestone-5-donovan.md`. The `go install` and clone-build steps there use throwaway folders under `T_CFG`. The clean-machine run is a clean machine or container, at the final release check, and Setup does not apply there. Binary: `bin/kirsch` from `npm run build` in the checkout, plus the `go install` binary and the clone-build binary from the runbook. Working directory: /Volumes/DATA/Github/kirsch for the grep checks; the clean machine for the clean-machine run. Workspace: n/a. Key state: all key variables unset on the clean machine. Config state: none in the Setup terminal.

**What landed**

- The install page is `doc/install.md`. It covers the supported platforms, what Kirsch needs on the machine (Git, and ripgrep as an optional search tool), the install routes (a release archive, `go install`, and a build from source), the API key setup, and `kirsch doctor`.
- The release-archive route is checked by the operator at the final release check (section 37 of this checklist).
- The `go install` version text, as O4 (b) records it: a `go install` build reports its module version with the leading `v` removed when Go records a clean release tag. Otherwise it reports `0.1.0-dev`. The code is `devVersion` at `cmd/kirsch/main.go:33` and `resolveVersion` at `cmd/kirsch/main.go:52`. The decision is row O4 in `plan/PROGRESS.md`, under *Milestone 5 — decisions recorded*.
- The module is public. `go install github.com/djm56/kirsch/cmd/kirsch@latest` resolves today to the pseudo-version `v0.0.0-20261029062614-686d3c4b6cff` (commit `686d3c4`), and that build reports `kirsch 0.1.0-dev`. That snapshot has the doctor command but none of its checks (`git ls-tree -r --name-only 686d3c4 -- internal/doctor` lists only `doctor.go` and `doctor_test.go`), and the page's doctor example is not checked against it.

**Waits on:** m5-d2 (the `--version` text), m5-d10a and m5-d10b (archive names and build wrapper), O4, and O6, from `plan/milestones/milestone-5.md:115`. The O4 (b) rule above is in effect, so a `go install` build of an untagged commit reports `0.1.0-dev`.

**Key-state check, first.** Authored here. Run it before any kirsch launch in this section. It must print `CONTROL PASS`, then `0`. Read the printed `0`; `keycount` exits 1 when it prints 0, because `grep -c` does.

```bash
# authored here
keycount_control
keycount
```

**Automated checks**

- [ ] `doc/install.md` states what a `go install` build reports, as O4 (b) records it: the leading `v` removed from a clean release tag, otherwise `0.1.0-dev` (`plan/milestones/milestone-5.md:1218`). Command: `grep -n -E '0\.1\.0-dev|without its leading' doc/install.md`. Expected: every printed line is in the `go install` section (the table and the sentence under it) or in the build-from-source section. Review step against the code: the review step checks the sentences against `devVersion` and `resolveVersion` in `cmd/kirsch/main.go`.

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n -E '0\.1\.0-dev|without its leading' doc/install.md
  ```

- [ ] `grep -n -i -w -E 'milestones?|ADRs?' doc/install.md` prints nothing. The milestone's own pattern, `'milestone|ADR'` (`plan/milestones/milestone-5.md:1219`), misses plurals, and the control below shows it. Control: the printf line prints `an ADR`, `ADRs` and `milestones`, and not `headroom`. The old pattern on the same input prints only `an ADR`.

  ```bash
  # authored here
  cd /Volumes/DATA/Github/kirsch && grep -n -i -w -E 'milestones?|ADRs?' doc/install.md
  printf 'an ADR\nADRs\nmilestones\nheadroom\n' | grep -i -w -E 'milestones?|ADRs?'
  printf 'an ADR\nADRs\nmilestones\nheadroom\n' | grep -i -w -E 'milestone|ADR'
  ```

**Hands-on checks**

1. (operator) In the Setup terminal, work through the steps of section 17 in `plan/testing/manual/user-testing/milestone-5-donovan.md`, in order. The install steps use throwaway folders under `T_CFG`.
2. (operator) At the final release check, section 37, do the clean-machine run: on a clean machine or in a container, with no `kirsch` on its `PATH` and no API key variables set, follow `doc/install.md` from the top. This includes the release-archive route once a release exists, and the checksum commands. Record the machine or container, and the commands the page gave, in *Notes / evidence*.

**Notes / evidence**

_Empty. Fill at run._

---

## 18 · Fresh-machine no-key check (m5-d16)

**Context:** Terminal: Setup for the `cmp` and the `env -u` run; a clean environment for the onboarding run (O19). Binary: `/Volumes/DATA/Github/kirsch/bin/kirsch` from `scripts/build.sh` (m5-d10a), run under `env -i` by its absolute path from outside the checkout. Working directory: /Volumes/DATA/Github/kirsch for `cmp`; the temporary Git repository the script creates for onboarding. Workspace: n/a. Key state: all key variables unset. The `env -u` command below enforces it. Config state: Setup, unchanged.

**What landed**

- The check is `scripts/no-key-check.sh`, run as `npm run check:no-key` (`package.json`). It builds `bin/kirsch` with `scripts/build.sh`, creates a temporary home and a temporary Git repository, and starts the binary inside that repository with `env -i HOME=<temporary home> PATH=$PATH TERM=xterm-256color`. It removes both when the run ends.
- It drives the program through `script(1)`, which gives it a pseudo-terminal (O19). Both argument forms are handled: the BSD form, which runs on macOS and is the form that has been run, and the util-linux form, which has only been checked for shell syntax and has not been run on Linux.
- It passes when the line `No API key: set KIRSCH_OPENCODE_API_KEY → OPENCODE_API_KEY` is in the output the terminal received during the run. It checks the emitted output, not the final screen, so a line that was drawn and later scrolled away still matches.
- It takes about 35 seconds on the Mac where it was written. The first output reaches the capture about 30 seconds after launch. Nothing is typed into the composer, so it sends no request to a model.
- Exit codes: `0` pass; `1` Kirsch drew frames and the line is missing, or the run timed out after a frame; `2` the check could not complete; `130` interrupted. The header of `scripts/no-key-check.sh` lists each case.
- O14 is unchanged. "Not in a repo" keeps its message, `Kirsch runs inside a Git repository.`, from `internal/workspace/workspace.go`, and the process exits 1. The automated check below shows it.

**Product change, outside the planned file list.** On a fresh start, the notice `no previous session — starting fresh` used to hide the no-key onboarding and the empty session. Now the onboarding (or the empty session) stays while the transcript holds only notices and every notice row fits below it. The notices are listed beneath it after one blank row. If a notice row would not fit, the normal scrollable transcript shows, with nothing cut off. This includes notices from user commands such as `/status`. The first message, a tool card, an error or any other item ends it, as before. The composer placeholder is not shown below 40 columns, or when the transcript region has no rows. The rule is decided for each frame, so `/new`, which empties the transcript, or a larger window can bring the onboarding back. Code: `internal/tui/model.go` (`introRows`), `internal/tui/view.go:654`, and the header of `internal/tui/empty.go`. Spec: `plan/spec/ui-spec-v0.1.md` §2 and §7.5.

**What a tester sees at 80 by 24 on a fresh start.** The screens below were captured on 2026-11-05 by a scratch run outside the repository. The binary ran under `env -i` in a new temporary Git repository with a new temporary home. The output was drawn by a terminal emulator at the size shown, and the run was ended with two `Ctrl+C`. Under `env -i` there is no UTF-8 locale, so the wordmark is letter-spaced and the bullets are `.`. Every run made with `env -i`, as the commands in this section are, shows the same. Trailing blank rows are left out.

Fresh start, nothing typed (80 by 24):

```text
 Kirsch -----------------------------------------------------------------------
 repo - master

 K I R S C H

 v0.1.0-dev . terminal-native coding agent

 . No API key: set KIRSCH_OPENCODE_API_KEY → OPENCODE_API_KEY.
 . Keys are never read from config files.

 . Using opencode sends prompts and file contents to OpenCode's gateway and to
 whoever hosts the chosen model, not to Anthropic.

 . no previous session - starting fresh





 ------------------------------------------------------------------------------
 minimax-m3 . idle . 0 tok
 ------------------------------------------------------------------------------
 > Ask anything (Enter to send, /help for help)
```

Read it top to bottom: the onboarding comes first, then one blank row, then the notice line. The composer shows the placeholder.

After `/status` is typed and entered at the same size (80 by 24), the notices are too tall to fit under the onboarding, so the normal transcript shows instead. The composer shows the cursor, not the placeholder. The session line differs on every run:

```text
 Kirsch -----------------------------------------------------------------------
 repo - master

 . no previous session - starting fresh

 . model minimax-m3 . endpoint opencode . base_url opencode.ai . key source  .
 context none . branch master . dirty 0 . session
 20261105T090741-06GTNXP3YVE34JPWKYQNTRE4TR . tokens in 0 out 0 cache 0 .
 compaction - . grants none . Using opencode sends prompts and file contents to
 OpenCode's gateway and to whoever hosts the chosen model, not to Anthropic.









 ------------------------------------------------------------------------------
 minimax-m3 . idle . 0 tok
 ------------------------------------------------------------------------------
 > _
```

The same `/status`, at 80 by 40, leaves the onboarding showing, with the notices beneath it. Rows 14 to 20 of that screen:

```text
 . no previous session - starting fresh

 . model minimax-m3 . endpoint opencode . base_url opencode.ai . key source  .
 context none . branch master . dirty 0 . session
 20261105T090825-06GTNXVGPMHC777CAETS3EH74M . tokens in 0 out 0 cache 0 .
 compaction - . grants none . Using opencode sends prompts and file contents to
 OpenCode's gateway and to whoever hosts the chosen model, not to Anthropic.
```

Rows 36 to 39 of that screen, where the composer shows the placeholder again:

```text
 ------------------------------------------------------------------------------
 minimax-m3 . idle . 0 tok
 ------------------------------------------------------------------------------
 > Ask anything (Enter to send, /help for help)
```

The narrow case, 39 columns by 24 rows, a fresh start with nothing typed. The transcript region reads `terminal too narrow`, and the composer shows the cursor and no placeholder:

```text
 Kirsch ------------------------------
 repo
 terminal too narrow
















 -------------------------------------
 minimax-m3 . idle
 -------------------------------------
 > _
```


**Baseline.** `.claude/memory/workspace/m5-d16-smoke-test.pre.sh` is `scripts/smoke-test.sh` as it stood before m5-d16 changed anything. The `cmp` below compares against it.

**Key-state check, first.** Authored here. Run it before any kirsch launch in this section. It must print `CONTROL PASS`, then `0`.

```bash
# authored here
keycount_control
keycount
```

**Automated checks**

- [ ] The no-key check passes. Property: the onboarding line was written to the terminal during a run with no key. Expected: the matched line `. No API key: set KIRSCH_OPENCODE_API_KEY → OPENCODE_API_KEY.`, then `no-key-check: PASS (the line was emitted to the terminal)`, then `exit=0`. npm first prints its own `> kirsch@0.1.0-dev check:no-key` lines. The run takes about 35 seconds. The output is pasted into the step report (`plan/milestones/milestone-5.md:1410`).

  ```bash
  # authored here
  cd /Volumes/DATA/Github/kirsch && npm run check:no-key; echo "exit=$?"
  ```

- [ ] The check can fail. Property: a run whose expected line is wrong ends in FAIL, so a PASS means the line was seen. The copy must sit in `scripts/`, because the script finds its repository from its own place. It differs from the script in one line only, the `want=` line (`scripts/no-key-check.sh:141`), where `KIRSCH_OPENCODE_API_KEY` becomes `KIRSCH_FALSIFIED_KEY`. The `sed` writes the copy and leaves the script alone. Expected: `diff` reports the one changed line (a `<` line and a `>` line) and exits 1. The falsifier run takes about 35 seconds. It prints the FAIL line first, `no-key-check: FAIL (Kirsch drew frames, ...; the terminal output never carried: No API key: set KIRSCH_FALSIFIED_KEY ...)`, and then the last 40 rows of Kirsch's captured output (`tail -n 40`, `scripts/no-key-check.sh:423`). Both go to stderr, so the FAIL line sits on the line above those 40 rows. That text has no final newline, so `exit=1` lands on the same row as the last of those 40 rows. The copy is then removed, and its absence is checked against a control: `test -e` on the real script must exit 0 first. If the run stops part way, for example with `Ctrl+C`, the copy stays in `scripts/`. Run the `rm` line by hand, then run the `test -e` pair: `test -e scripts/no-key-check.sh; echo "control exit=$?"` must print exit 0, then `test -e scripts/no-key-check.falsify.sh; echo "exit=$?"` must print exit 1, which shows the copy is gone.

  ```bash
  # authored here
  # Falsifier: one changed line, a FAIL with exit 1, then the copy is removed.
  cd /Volumes/DATA/Github/kirsch && sed '/^want=/s/KIRSCH_OPENCODE_API_KEY/KIRSCH_FALSIFIED_KEY/' scripts/no-key-check.sh > scripts/no-key-check.falsify.sh && diff scripts/no-key-check.sh scripts/no-key-check.falsify.sh; echo "diff exit=$? (expect 1, one changed line)"
  cd /Volumes/DATA/Github/kirsch && bash scripts/no-key-check.falsify.sh; echo "exit=$? (expect 1; the FAIL line is the line above the 40 rows, and this line shares the last row)"
  cd /Volumes/DATA/Github/kirsch && rm -- scripts/no-key-check.falsify.sh; echo "rm exit=$?"
  cd /Volumes/DATA/Github/kirsch && test -e scripts/no-key-check.sh; echo "control exit=$? (expect 0, the real script is present)"
  cd /Volumes/DATA/Github/kirsch && test -e scripts/no-key-check.falsify.sh; echo "gone exit=$? (expect 1, the copy is absent)"
  # Convenience only, not a control: must not list no-key-check.falsify.sh
  cd /Volumes/DATA/Github/kirsch && git status --short scripts/
  ```

- [ ] The script refuses an argument with exit 2, before it builds or starts anything. Property: exit 2 (the check could not complete) is reported apart from exit 0 (pass). Expected: the usage line `usage: bash scripts/no-key-check.sh   (takes no arguments)` and `exit=2`.

  ```bash
  # authored here
  cd /Volumes/DATA/Github/kirsch && bash scripts/no-key-check.sh extra; echo "exit=$?"
  ```

- [ ] The temporary home and Git repository are removed when the run ends. Property: no `kirsch-no-key.*` directory is left under the temporary directory. The block makes its own control first: a directory made the way the script makes one must be listed, with exit 0, before the real listing counts. The block removes its control directory. If the block stops part way, `rmdir` the path the control printed, since that directory is empty. Run it after the check above has finished. Expected: the control lists its directory and ends `exit=0`. The real listing prints nothing, its error from `ls` shows on the terminal (`No such file or directory` in bash, `no matches found` in zsh), and it ends `exit=1`.

  ```bash
  # authored here
  # Control first: a directory made the way the script makes one must be listed, with exit 0.
  CTRL="$(mktemp -d "${TMPDIR:-/tmp}/kirsch-no-key.XXXXXX")" && echo "control directory: $CTRL"
  ls -d "${TMPDIR:-/tmp}"/kirsch-no-key.*; echo "control exit=$? (expect 0, the directory above is listed)"
  rmdir "$CTRL"; echo "control removed exit=$?"
  # Real check: no kirsch-no-key.* directory may be left. Expect nothing listed, the ls error, and exit=1.
  ls -d "${TMPDIR:-/tmp}"/kirsch-no-key.*; echo "exit=$? (expect 1, nothing listed)"
  ```

  The `exit=1` on the real listing is the macOS result. BSD `ls`, which macOS ships, exits 1 for a missing path. GNU `ls` (for example `gls` from GNU coreutils) exits 2 for the same case, so on Linux that line reads `exit=2`. Nothing is listed either way. Bishop ran both commands on 2026-11-05 09:55 UTC on macOS 26.5.1, and the output is pasted below verbatim:

  ```
  $ ls -d /nonexistent-kirsch-m5d16; echo "ls exit=$?"
  ls: /nonexistent-kirsch-m5d16: No such file or directory
  ls exit=1
  $ gls --version | head -1
  ls (GNU coreutils) 9.12
  $ gls -d /nonexistent-kirsch-m5d16; echo "gls exit=$?"
  gls: cannot access '/nonexistent-kirsch-m5d16': No such file or directory
  gls exit=2
  ```
- [ ] `scripts/smoke-test.sh` is unchanged against its saved copy. The saved copy is `.claude/memory/workspace/m5-d16-smoke-test.pre.sh` (`plan/milestones/milestone-5.md:1412`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && cmp -s .claude/memory/workspace/m5-d16-smoke-test.pre.sh scripts/smoke-test.sh; echo "exit=$?"
  cd /Volumes/DATA/Github/kirsch && git ls-files --error-unmatch -- scripts/smoke-test.sh
  ```

  The `; echo "exit=$?"` on the `cmp` line is an addition, not in the milestone. It shows the exit status. The control, as a command, appends one byte to a copy of the tracked file. The `cmp` must then exit 1 (authored here):

  ```bash
  # authored here: a copy with one byte appended must differ, so cmp exits 1
  CTRL=$(mktemp) && cp /Volumes/DATA/Github/kirsch/scripts/smoke-test.sh "$CTRL" && printf 'x' >> "$CTRL"
  cmp -s "$CTRL" /Volumes/DATA/Github/kirsch/scripts/smoke-test.sh; echo "control exit=$? (expect 1)"; rm -f "$CTRL"
  ```

- [ ] `scripts/smoke-test.sh` still exits 2 without a key. The command strips every key variable from the child with `env -u`, so the script sees no key. It then exits 2 at `scripts/smoke-test.sh:16–22`, whatever the shell holds. With any one key variable set in the child, the script runs `TestSmokeLive` against the live provider and spends tokens (`plan/milestones/milestone-5.md:1413`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && env -u ANTHROPIC_API_KEY -u KIRSCH_ANTHROPIC_API_KEY -u OPENCODE_API_KEY -u KIRSCH_OPENCODE_API_KEY bash scripts/smoke-test.sh; echo "exit=$?"
  ```

  The `; echo "exit=$?"` is as the milestone gives it (`plan/milestones/milestone-5.md:1416`). The printed `exit=` line is pasted.

- [ ] The "not in a repo" case keeps its message and exits 1, as O14 records (`plan/milestones/milestone-5.md:1420`). Property: outside a Git repository, Kirsch prints the message and exits without a screen. The message is in `internal/workspace/workspace.go`. Needs `bin/kirsch`, built by `npm run build` or by the check above. Command below. Expected: the lines `kirsch: Kirsch runs inside a Git repository.`, then `Either cd into a repository, or point Kirsch at a directory explicitly:`, then `    kirsch --workspace /path/to/project`, then `exit=1`.

  ```bash
  # authored here
  NR="$(mktemp -d)" && cd "$NR" && env -i HOME="$HOME" PATH="$PATH" TERM=xterm-256color /Volumes/DATA/Github/kirsch/bin/kirsch; echo "exit=$?"; cd /Volumes/DATA/Github/kirsch && rmdir "$NR"
  ```

**Hands-on checks**

1. (operator) Run `npm run check:no-key` on a clean machine, or in a container, with no key set, as in the first automated check above. Paste its last lines into *Notes / evidence* (`plan/milestones/milestone-5.md:1421`).
2. (operator) In an 80 by 24 window, start Kirsch by hand on a fresh start, with no key, in a new temporary folder and a new temporary home. Compare the screen with the first sample under *What a tester sees*. Then type `/status` and press Enter, and compare with the second sample. Quit with two `Ctrl+C`.

   ```bash
   # authored here
   FAKE_HOME="$(mktemp -d)" && cd "$(mktemp -d)" && git init -q . && env -i HOME="$FAKE_HOME" PATH="$PATH" TERM=xterm-256color /Volumes/DATA/Github/kirsch/bin/kirsch
   ```

3. (operator) In an 80 by 40 window, repeat step 2 in a window 40 rows tall. After `/status`, the onboarding should still show, with the notices beneath it. Quit with two `Ctrl+C`.

   The throwaway folders from steps 2 and 3 are left for the operator to remove.

**Notes / evidence**

_Empty. Fill at run._

---

## 19 · `/new` lifecycle: C51, C52 (optional) (m5-d17a)

**Context:** Terminal: Setup. Binary: `go test` only; no kirsch launch in this section. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a; the tests use temporary folders (a full `go test ./internal/app/` run left the number of session files under `~/.local/share/kirsch/sessions` unchanged; output under *Automated checks*). Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

**Choice recorded for C51: cleanup, not lazy creation.** A session file is still created when a session starts, and it is removed when that session is abandoned with nothing written to it. The choice rests on these reasons:

- In the code: the `NewSession` docblock says that when the new store cannot be created, "the error and the App is unchanged" (`internal/app/app.go:1212`). That rule predates this mission, and the cleanup choice is consistent with it.
- Not in the code: the author's reasoning, which is that cleanup keeps the session id visible right after `/new`.

The approach is recorded in @hicks's step 1 report and in row 1 of `.claude/memory/missions/mission-20261105-04/PROGRESS.md` ("C51 cleanup approach"). Lazy creation was the other option and was not built.

What the code does, with the cites checked against `internal/app/app.go`:

- A session that this process creates carries an origin. It is made at start-up by `ensureStore` (`:973`) or `StartNewSession` (`:1002`), or by `/new` through `NewSession` (`:1239`). The origin is set in `createSessionStore` (`:1142`; the assignment is at `:1170`).
- An abandoned session with nothing written is removed in these cases: on quit, in `Close` (`:890`, through `abandonEmpty` at `:1055`); when `/new` replaces it, in `installSession` (`:1303`); and when the store cannot be opened, in `discardFailedCreate` (`:1198`).
- `abandonEmptyLocked` (`:1063`) deletes the 0-byte file, then calls `restoreIndexLocked` (`:1085`, call at `:1077`). That function restores `index.json` only when the entry still names the abandoned session (`:1091`). The restore point comes from `restorePoint` (`:1111`), which walks past sessions that are already removed. If there was no earlier entry, or the restore target's file is gone, the workspace entry is deleted.
- A session that received any event is never removed (`TestNewSessionThatReceivedAnEventSurvivesClose`). A resumed session is never removed (`TestResumedSessionIsNeverDeleted`).
- `Close` releases the workspace lock only after the removals. The comment is at `internal/app/app.go:922–924`.

C52: `/new` sets a fresh provider session id. `randomSessionID()` (`internal/app/app.go:1039`) makes 16 random bytes as hex. `NewSession` sets it after a successful install (`:1281`). `cmd/kirsch/main.go` is unchanged.

File ownership: the planned ownership was `internal/app/app.go` only. The change also adds `internal/app/session_orphan_test.go`, changes one line in `internal/app/session_ops_test.go`, and changes `cmd/kirsch/newsession_wiring_test.go`, which now records one real event with `ClearGrants()` (`:99`) before `/new`.

User-visible change, as the code does it (read from the code, not seen in a run): a session file that this Kirsch created and that nothing was written to is removed when you quit (`Close`, `internal/app/app.go:890`) or when `/new` replaces it (`installSession`, `internal/app/app.go:1303`). A session that received any event, or that was resumed, is never removed. When the removed session is the one `index.json` names for this workspace (at quit, the session you were in), the entry is put back to what it held before Kirsch created that session, skipping any earlier empty sessions that `/new` already removed, or deleted if there was nothing earlier or that earlier session's file is gone (`restoreIndexLocked`, `internal/app/app.go:1085`; `restorePoint`, `internal/app/app.go:1111`). After `/new`, the entry names the new session; at quit it goes back as described only if that session is still empty. So, after a normal quit, the next launch does not find an empty session that this Kirsch created. Which notice that launch shows depends on how it is started (`chooseSessionInit`, `cmd/kirsch/main.go:277–310`). `/new` also sets a new provider session id (`randomSessionID()`, `internal/app/app.go:1039`, set in `NewSession` at `internal/app/app.go:1281`).

**Waits on:** m5-d5 (the session layer) and O13 (scheduling of the optional group), from `plan/milestones/milestone-5.md:120`.

**Automated checks**

- [ ] A test shows that quitting during `/new` leaves no empty session file (`plan/milestones/milestone-5.md:1456`). Test: `TestCloseAfterNewSessionLeavesNoEmptySession` (`internal/app/session_orphan_test.go:54`). Command and output below.
- [ ] A test shows that after `NewSession` the provider session id differs from the id before it (`plan/milestones/milestone-5.md:1457`). Test: `TestNewSessionRotatesTheProviderSessionID` (`internal/app/session_orphan_test.go:160`). Command and output below.
- [ ] `go test ./internal/app/` passes (`plan/milestones/milestone-5.md:1458`). Left unticked for the operator. The full-package run below is Bishop's capture on 2026-11-05 (mission-20261105-04, step 4a), with its `exit=` line.

```bash
# authored here
go test -count=1 ./internal/app/; echo "exit=$?"
```

```
ok  	github.com/djm56/kirsch/internal/app	92.542s
exit=0
```

The capture above (92.542s) predates the step 5 test-isolation fix.

After the step 5 test-isolation fix (Bishop's run, 2026-11-05; it counts the session files):

```bash
b=$(find ~/.local/share/kirsch/sessions -name '*.jsonl' | wc -l); go test -count=1 ./internal/app/; echo "exit=$?"; a=$(find ~/.local/share/kirsch/sessions -name '*.jsonl' | wc -l); echo "session files before=$b after=$a"
```

```
ok  	github.com/djm56/kirsch/internal/app	112.224s
exit=0
session files before=    6251 after=    6251
```

The named tests, and the `/new` test through the real wiring, run by Bishop on 2026-11-05 (mission-20261105-04, step 4a). The commands are authored here, and they are not quoted from the milestone. Each output is pasted as captured, with its `exit=` line.

```bash
# authored here
go test -count=1 ./internal/app/ -run 'TestCloseAfterNewSessionLeavesNoEmptySession$|TestNewSessionThatReceivedAnEventSurvivesClose$|TestFreshLaunchSessionIsAbandonedLikeANewOne$|TestResumedSessionIsNeverDeleted$|TestNewSessionRotatesTheProviderSessionID$' -v; echo "exit=$?"
```

```
=== RUN   TestCloseAfterNewSessionLeavesNoEmptySession
--- PASS: TestCloseAfterNewSessionLeavesNoEmptySession (0.08s)
=== RUN   TestNewSessionRotatesTheProviderSessionID
--- PASS: TestNewSessionRotatesTheProviderSessionID (0.08s)
=== RUN   TestNewSessionThatReceivedAnEventSurvivesClose
--- PASS: TestNewSessionThatReceivedAnEventSurvivesClose (0.06s)
=== RUN   TestResumedSessionIsNeverDeleted
--- PASS: TestResumedSessionIsNeverDeleted (0.08s)
=== RUN   TestFreshLaunchSessionIsAbandonedLikeANewOne
=== RUN   TestFreshLaunchSessionIsAbandonedLikeANewOne/empty_startup_then_quit
=== RUN   TestFreshLaunchSessionIsAbandonedLikeANewOne/empty_startup_then_/new_then_quit
=== RUN   TestFreshLaunchSessionIsAbandonedLikeANewOne/startup_with_an_event_then_/new_then_quit_(control)
--- PASS: TestFreshLaunchSessionIsAbandonedLikeANewOne (0.17s)
    --- PASS: TestFreshLaunchSessionIsAbandonedLikeANewOne/empty_startup_then_quit (0.05s)
    --- PASS: TestFreshLaunchSessionIsAbandonedLikeANewOne/empty_startup_then_/new_then_quit (0.06s)
    --- PASS: TestFreshLaunchSessionIsAbandonedLikeANewOne/startup_with_an_event_then_/new_then_quit_(control) (0.06s)
PASS
ok  	github.com/djm56/kirsch/internal/app	0.736s
exit=0
```

Output of the `cmd/kirsch` test (`cmd/kirsch/newsession_wiring_test.go:72`), run by Bishop on 2026-11-05 (mission-20261105-04). The command is authored here.

```bash
# authored here
go test -count=1 ./cmd/kirsch/ -run 'TestSlashNewThroughRealWiringStartsANewSession$' -v; echo "exit=$?"
```

```
=== RUN   TestSlashNewThroughRealWiringStartsANewSession
--- PASS: TestSlashNewThroughRealWiringStartsANewSession (0.16s)
PASS
ok  	github.com/djm56/kirsch/cmd/kirsch	0.669s
exit=0
```

**Hands-on checks**

None is written in this section. The automated checks above cover the behaviour. A hands-on quit-during-`/new` step, with the session directory listing before and after, has not been run and has no captured output, so it is not written. It is carried below.

**Notes / evidence**

Mutant record. The mutant results come from the step reports of @hicks (steps 2a and 2c) and @vasquez (steps 2e and 2g), summarised in the m5-d17a rows of this mission's PROGRESS (`.claude/memory/missions/mission-20261105-04/PROGRESS.md`). The raw outputs were not saved to disk, and no mutant was re-run for this section. Each test in the table below failed when the change in its row was made (as reported); the change and test columns come from the step reports. The mutant labels are the step reports' own. M1 here is the step 2a mutant, not the step 1 mutant with the same number (the `abandonEmpty` early return, the C51 fix) (@hicks, step 1 report):

| Mutant | Change made | Test that failed (`internal/app/session_orphan_test.go`) | Failing message (verbatim where it is a literal string) |
|---|---|---|---|
| M1 (step 2a) | C52 rotation removed | `TestNewSessionRotatesTheProviderSessionID` (:160; message at :191) | `provider session id "abababababababababababababababab" unchanged by /new` |
| M3 | size check dropped | `TestNewSessionThatReceivedAnEventSurvivesClose` (:229) | description, not the literal message: the session file is deleted, so a stat fails |
| M6 | startup origin dropped | `TestFreshLaunchSessionIsAbandonedLikeANewOne` (:322) | description, not the literal message: the file is left and index.json still points at it |
| M7 | resume gets an origin | `TestResumedSessionIsNeverDeleted` (:261; message at :292) | `resumed empty session removed by Close` |
| MW1 (2g) | `abandonMu` taken late | `TestCloseDuringInstallWaitsForTheReplacedSessionToGo` (:782; message at :864) | `Close returned (workspace lock free) while the replaced empty session … was still on disk` [format string, truncated] |
| MD | `discardFailedCreate` call removed | `TestFailedStoreOpenDiscardsTheHalfMadeSession` (:980; message at :1023) | `half-made session file … on disk = true, want false` |

Carried items, not closed by this section:

- A hands-on quit-during-`/new` step is not written, because it has no captured output.
- Review suggestions from the m5-d17a review, not graded here: (1) no deterministic mutant covers the full span of the `abandonMu` hold in `installSession` (`internal/app/app.go:1329–1338`); (2) comment wrapping in the `abandonMu` field comment (`internal/app/app.go:481–490`) and in the `NewSession` docblock (`:1207–1238`); (3) `createSessionStore` says "It touches no App state" (`internal/app/app.go:1129`), and the suggested wording is "mutates no App fields".
- Test ordering race, product side: `Submit` and `RunTurn` send through separate `sendAsyncSequence` calls, so a late `BudgetMsg` can leave a stale estimate on the status line (`internal/tui/update.go:449`, `:470`, `:478`). `TestSubmitSendsBudgetMsgForSmallRequest` was changed in m5-d17a to find the BudgetMsg by type; no run shows the new assertion failing (the mutant run was denied by the permission system). The race is listed under *Open from m5-d17a* in `plan/PROGRESS.md`.
- Comment carried: the comment above `TestSubmitSurfacesContextOverflowWhenEstimateExceedsBudget` (`internal/app/app_test.go:179–184`) says the first message is the BudgetMsg, which the product does not guarantee. Listed under *Open from m5-d17a* in `plan/PROGRESS.md`.
- Unverified: whether `RunTurn`'s streamed deltas can arrive out of order (`emit` at `internal/app/app.go:2120–2124`). Listed under *Open from m5-d17a* in `plan/PROGRESS.md`.
- The cross-process `index.json` read-modify-write has no lock (`ReadIndex` at `internal/session/index.go:28`, `WriteIndex` at `:44`). This predates m5-d17a and is listed under *Open from m5-d17a* in `plan/PROGRESS.md`.

---

## 20 · Doctor: key, backend, model and project-context lines (m5-d4a)

**Context:** Terminal: Setup. Binary: `"$KIRSCH_BIN" doctor`, built by `npm run build` (the build block in §2); `go test` for the checks. Working directory: stated inside each hands-on step, because it differs from step to step. Workspace: stated inside each hands-on step, because it differs from step to step. Key state: all key variables unset; hands-on 1 sets one dummy value for one command at a time, never a real key. Config state: Setup, unchanged, except where a step names its own settings folder for that one command, or reads the operator's own settings (`env -u XDG_CONFIG_HOME`).
**What landed**

After the config line, `kirsch doctor` prints the lines `key`, `backend`, `model` and `context`, in this order. The summary line counts them with the rest. Each uses the m5-d3a grammar, `<status>  <check>: <detail>`. The probes are `resolvedResults` and its helpers in `internal/doctor/probes.go`. The lines below are as the binary prints them. Every line in the table was seen in a measured run (*Notes / evidence*, and `.claude/memory/workspace/m20261029-06-step3-runs.txt` and `.claude/memory/workspace/m20261029-06-step3a-runs.txt`, both scratch and archived at the next mission start), except the row for an unreadable candidate, which is read from the code.

| Check | Line as printed | Result | When |
|---|---|---|---|
| key | `ok  key: endpoint opencode: key found in <VAR>` | ok | a key is found. `<VAR>` is `KIRSCH_OPENCODE_API_KEY` when both variables are set, because the `KIRSCH_` name wins |
| key | `warn  key: endpoint opencode: no key found (set KIRSCH_OPENCODE_API_KEY or OPENCODE_API_KEY); kirsch starts but cannot send a prompt` | warn | neither variable is set |
| key | `warn  key: endpoint opencode: api_key_env does not look like an environment variable name (value withheld); fix the setting` | warn | the `api_key_env` setting is not shaped like an upper-case variable name. The setting's text is not printed |
| backend | `ok  backend: search_code uses ripgrep` | ok | `rg` is on PATH |
| backend | `warn  backend: search_code uses pure-go (rg is not on PATH)` | warn | `rg` is not on PATH. The backend line still prints when config did not load, because it does not read the config |
| model | `ok  model: endpoint opencode, model minimax-m3: context window 128000 tokens` | ok | the model is in the table. The line prints the table's id and window |
| model | `warn  model: endpoint opencode, model <id>: not in the table; conservative fallback, context window <n> tokens` | warn | the model is not in the table |
| context | `ok  context: AGENTS.md, <n> bytes` | ok | the first candidate that loads is named, with its size on disk |
| context | `ok  context: none; looked for AGENTS.md, CLAUDE.md` | ok | no candidate exists |
| context | `ok  context: none; no project files configured` | ok | `project_files` is empty |
| context | `warn  context: none; looked for .env, AGENTS.md; project context candidate ".env" refused: path ".env" is outside the workspace or denied: matches the denied pattern .env` | warn | a candidate was refused (here `.env`, a denied name) or is not a regular file (a directory, for example, which reads `project context candidate "AGENTS.md" is not a regular file`). When no candidate is chosen the line starts `none; looked for`. When a later candidate is chosen, the line reads `AGENTS.md, <n> bytes` and then the warning, for example `warn  context: AGENTS.md, 20 bytes; project context candidate ".env" refused: path ".env" is outside the workspace or denied: matches the denied pattern .env` |
| context | `warn  context: a candidate could not be read: <loader error>` | warn | the loader could not stat or read a candidate. The line is read from the code (`contextResult` in `internal/doctor/probes.go`), not run. The `none` prefix is dropped, because a candidate was there |
| key, model, context | `warn  <check>: skipped: config did not load` | warn | the settings file did not load (the config line is a fail). The line guesses no default |
| context | `warn  context: skipped: no workspace` | warn | no workspace was found (the repository line is a fail), so no project file is read |

**Status, by the O5 rule.** The rule (O5 later checks, recorded 2026-10-29 in *Milestone 5 — decisions recorded* in `plan/PROGRESS.md`) is: a check fails if the problem would stop kirsch from starting, and warns if kirsch still starts but works less well. None of the new lines can stop kirsch from starting. `run` reads no key, model or context file before the terminal screen is built, so none of them is a fail. The code comment that says so is on `resolvedResults` in `internal/doctor/probes.go`, and it is read from the code, not run. Each warn case below is one where kirsch starts and works less well. The word *measured* after a row means the line was seen in a run in *Notes / evidence*; *read from the code* means the reason that kirsch starts was read from the code, not run:

| Case | Result | Why it is a warning, not a failure |
|---|---|---|
| no key | warn | kirsch starts and shows its onboarding notice, but refuses each prompt (read from the code: the comment on `keyResult` in `internal/doctor/probes.go`). The warn line is measured. |
| `api_key_env` not a variable name | warn | kirsch starts (read from the code: `config.Validate` accepts any non-empty name). The setting is not printed (measured). |
| `rg` missing: the `backend` line | warn | search still works with the pure-Go fallback (read from the code: `search_code` falls back, `internal/tool/searchcode.go`). The line is measured. **Bishop's decision under O5, listed for the operator to overrule.** |
| `rg` missing: the `rg` line (m5-d3b) | warn | the same condition, as in m5-d3b. It makes the summary count two warnings for one missing tool. Measured. |
| model not in the table | warn | kirsch starts and uses the table's conservative window (read from the code: the fallback in `provider.LookupModel`, and the `modelResult` comment says `App.CheckOnboarding` shows the same notice). The warn line is measured. |
| candidate refused or unreadable | warn | kirsch starts without that context (read from the code: the `contextResult` comment). The refused line is measured; the unreadable case is read from the code, not run. |
| settings did not load | warn | nothing was checked, so a line cannot say "ok". The config line is the fail. Measured. |
| `--workspace` names no usable path | warn | the repository line is the fail. Measured. |

Doctor exits 1 only when a check fails. The new lines never change the exit status. Measured: every run with a warning and no failure ended `exit=0`. The run with `exit=1` in *Notes / evidence* (hands-on 4) has its failure on the config line.

**Known limits, carried and not fixed in m5-d4a.**

- A lower-case `api_key_env` that kirsch can use is flagged. `ResolveKey` (in `internal/config/config.go`) reads the name as written, so a lower-case name such as `my_key` resolves in kirsch. The key line warns that the name is not an environment variable name. Measured with `my_key`: the line is `warn  key: endpoint opencode: api_key_env does not look like an environment variable name (value withheld); fix the setting`. That kirsch sends a prompt with such a key is read from the code, not run.
- `keyVarNames` (`cmd/kirsch/main_test.go`) is derived from the built-in endpoints, and no test pins its list of names to what the production resolver reads. A key variable the resolver reads but the built-in endpoints do not list would not be cleared by `isolatedEnv`. This is review item W-B from the step 2b review, mission-20261029-06.

**Waited on:** m5-d3b (done 2026-10-29, mission-20261029-03; the config-load result) and O5 (recorded 2026-10-28, with the later-check rule of 2026-10-29). Both are in place.

**Baseline.** The comment check below compares with commit `686d3c4`, not with the copies saved before the edit at `.claude/memory/workspace/m5-d4a-searchcode.pre.go` and `.claude/memory/workspace/m5-d4a-app.pre.go` (`plan/milestones/milestone-5.md:624`). Those copies are archived at the next mission start, and the commit stays. `686d3c4` is the commit before m5-d4a's edits. It already holds m5-d5's change to `app.go`, so a diff against it shows m5-d4a's edits and nothing else.

**Key-state and repository checks, first.** Authored here. It must print `CONTROL PASS`, then `0`. The first `git` line must exit non-zero, and the last must print `true`.

```bash
# authored here
keycount_control
keycount
git -C "$T_CFG" rev-parse --is-inside-work-tree; echo "exit=$?"
git -C "$WORKSPACE" rev-parse --is-inside-work-tree
```

**Automated checks**

- [✅] The key line names the winning variable or says none, and never prints the key value. A test sets a known value and asserts the output does not contain it (`plan/milestones/milestone-5.md:633`). Tests: `TestKeyValueNeverPrinted`, `TestKeyLineWithholdsNonNameAPIKeyEnv` (the second also covers a name that is not a variable name). Command: `go test -count=1 -run '^(TestKeyValueNeverPrinted|TestKeyLineWithholdsNonNameAPIKeyEnv)$' ./internal/doctor`. Output: an `ok` line for `./internal/doctor`.
- [✅] With `KIRSCH_OPENCODE_API_KEY` and `OPENCODE_API_KEY` each set and unset, the key line names the winner or says none (`plan/milestones/milestone-5.md:634`). Tests: `TestKeyLineNamesWinner` (every combination), `TestKeyLineNilGetenvIsEmptyEnvironment`. Command: `go test -count=1 -run '^(TestKeyLineNamesWinner|TestKeyLineNilGetenvIsEmptyEnvironment)$' ./internal/doctor`. Output: an `ok` line for `./internal/doctor`.
- [✅] A model id not in the table prints `not in the table` and the conservative fallback (`plan/milestones/milestone-5.md:635`). Test: `TestModelLine` (its `not in the table` subtest). Command: `go test -count=1 -run '^TestModelLine$' ./internal/doctor`. Output: an `ok` line for `./internal/doctor`.
- [✅] With `AGENTS.md` present, the context line names it and its size. With it absent, the line says `none` (`plan/milestones/milestone-5.md:636`). Test: `TestContextLine` (its subtests cover present, absent, a directory, a refused name, an empty list and an unreadable file). Command: `go test -count=1 -run '^TestContextLine$' ./internal/doctor`. Output: an `ok` line for `./internal/doctor`.
- [✅] The backend line reads `ripgrep` with `rg` on PATH and `pure-go` without, under the PATH shim (`plan/milestones/milestone-5.md:637`). Test: `TestBackendLineUnderPathShim`. Command: `go test -count=1 -run '^TestBackendLineUnderPathShim$' ./internal/doctor`. Output: an `ok` line for `./internal/doctor`.
- [✅] With config not loaded, the key, model and context lines each read `skipped: config did not load` (`plan/milestones/milestone-5.md:638`). Tests: `TestResolvedLinesSkippedWhenConfigDidNotLoad` (in `internal/doctor`) and `TestRunDoctorSkippedLinesWhenConfigDidNotLoad` (through the binary's `run`, in `cmd/kirsch`). Command: `go test -count=1 -run '^(TestResolvedLinesSkippedWhenConfigDidNotLoad|TestRunDoctorSkippedLinesWhenConfigDidNotLoad)$' ./internal/doctor ./cmd/kirsch`. Output: an `ok` line for each of the two packages.
- [✅] The comments describe doctor lines that exist, and the files change only in comments (`plan/milestones/milestone-5.md:639–653`). The comparison is with commit `686d3c4` (see *Baseline* above). Ticked on the run in *Notes / evidence*: every changed content line of both files is a `//` comment, and the control line is reported.

  ```bash
  # authored here
  BASE=$(mktemp -d)
  git -C /Volumes/DATA/Github/kirsch show 686d3c4:internal/tool/searchcode.go > "$BASE/searchcode.go"
  git -C /Volumes/DATA/Github/kirsch show 686d3c4:internal/app/app.go > "$BASE/app.go"
  check_comments() { diff -U0 "$1" "$2" | grep -E '^[-+]' | grep -vE '^(---|\+\+\+) ' | grep -vE '^[-+][[:space:]]*//'; }
  changed_lines() { diff -U0 "$1" "$2" | grep -E '^[-+]' | grep -vcE '^(---|\+\+\+) '; }
  check_comments "$BASE/searchcode.go" /Volumes/DATA/Github/kirsch/internal/tool/searchcode.go
  check_comments "$BASE/app.go" /Volumes/DATA/Github/kirsch/internal/app/app.go
  changed_lines "$BASE/searchcode.go" /Volumes/DATA/Github/kirsch/internal/tool/searchcode.go
  changed_lines "$BASE/app.go" /Volumes/DATA/Github/kirsch/internal/app/app.go
  # control: the baseline app.go with one non-comment line appended
  cp "$BASE/app.go" "$BASE/app-control.go"; printf 'var controlProbe = 1\n' >> "$BASE/app-control.go"; check_comments "$BASE/app.go" "$BASE/app-control.go"
  ```

  The two `check_comments` commands on the real files print nothing on a pass. Their exit status is 1 because grep found no line to print, and that is the pass. The two `changed_lines` commands print a count of at least 1 each, because the edit changes both files. The control must print `+var controlProbe = 1`.

**Hands-on checks**

1. No key, then a dummy key, one command at a time (operator section 20, step 2). Confirm the key line names the variable and never prints the value (`plan/milestones/milestone-5.md:627`). O5 records that doctor makes no network call (`plan/PROGRESS.md`, *Milestone 5 — decisions recorded*, row O5, 2026-10-28), so a dummy value is enough for this step. No real key is needed or used. The dummy is set for one command only, in the same form as operator section 20, step 4.

   Working directory: the kirsch folder. Workspace: the kirsch folder, with no `--workspace` flag. Settings: Setup, unchanged; the dummy is set for one command at a time.

   ```bash
   cd /Volumes/DATA/Github/kirsch
   "$KIRSCH_BIN" doctor; echo "exit=$?"
   KIRSCH_OPENCODE_API_KEY=dummy-not-a-key "$KIRSCH_BIN" doctor | grep "key:"
   KIRSCH_OPENCODE_API_KEY=dummy-not-a-key "$KIRSCH_BIN" doctor | grep -c dummy-not-a-key
   KIRSCH_OPENCODE_API_KEY=dummy-not-a-key "$KIRSCH_BIN" doctor | grep -c KIRSCH_OPENCODE_API_KEY
   ```

   You should see, first, `warn  key: endpoint opencode: no key found (set KIRSCH_OPENCODE_API_KEY or OPENCODE_API_KEY); kirsch starts but cannot send a prompt` and `exit=0`. Then, with the dummy set for one command, the key line `ok  key: endpoint opencode: key found in KIRSCH_OPENCODE_API_KEY`, the count `0` (the dummy value is not in the output), and the count `1` (the positive control: the variable's name is found once, so the `0` means the search ran).

2. A model that is not in the table (`plan/milestones/milestone-5.md:635`). The model line warns and names the fallback window. Doctor uses a settings folder of its own for this step.

   Working directory: the kirsch folder. Workspace: the kirsch folder, with no flag. Settings: `$c2`, for this command only.

   ```bash
   c2=$(mktemp -d)
   mkdir -p "$c2/kirsch"
   printf '[provider.opencode]\nmodel = "no-such-model"\n' > "$c2/kirsch/config.toml"
   cd /Volumes/DATA/Github/kirsch
   XDG_CONFIG_HOME="$c2" "$KIRSCH_BIN" doctor; echo "exit=$?"
   ```

   You should see `warn  model: endpoint opencode, model no-such-model: not in the table; conservative fallback, context window 128000 tokens`, and `exit=0`. The window is the table's conservative fallback in this build (128000 tokens, measured). The summary counts that warning, and any others (for example the key warning, if no key is set).

3. No `AGENTS.md` (`plan/milestones/milestone-5.md:636`). The context line says `none`. The folder is outside any repository, so the repository line warns, as it does in section 12.

   Working directory: `$c3`, a temporary folder outside any repository. Workspace: `$c3`, given with `--workspace`. Settings: Setup, unchanged.

   ```bash
   c3=$(mktemp -d)
   cd "$c3"
   "$KIRSCH_BIN" doctor --workspace "$c3"; echo "exit=$?"
   cd /Volumes/DATA/Github/kirsch
   ```

   You should see `warn  repository:` naming the folder and saying it is not a Git repository, `ok  context: none; looked for AGENTS.md, CLAUDE.md`, and `exit=0`.

4. A settings file that does not read (`plan/milestones/milestone-5.md:638`). The key, model and context lines each read `skipped: config did not load`.

   Working directory: the kirsch folder. Workspace: the kirsch folder, with no flag. Settings: `$c4`, for this command only.

   ```bash
   c4=$(mktemp -d)
   mkdir -p "$c4/kirsch"
   printf '[session\n' > "$c4/kirsch/config.toml"
   cd /Volumes/DATA/Github/kirsch
   XDG_CONFIG_HOME="$c4" "$KIRSCH_BIN" doctor; echo "exit=$?"
   ```

   You should see `fail  config: parse` naming the file, then `warn  key: skipped: config did not load`, `warn  model: skipped: config did not load` and `warn  context: skipped: config did not load`. The backend line still prints, because it does not read the settings. The exit status is 1, and the cause is the config line's fail, not any line of this section.

5. The operator's own settings, with no settings override (operator section 20, step 3). This run reads the settings file of the Mac it runs on, so the model line shows whatever model those settings name.

   Working directory: the kirsch folder. Workspace: the kirsch folder, with no flag. Settings: the operator's own, through `env -u XDG_CONFIG_HOME`.

   ```bash
   cd /Volumes/DATA/Github/kirsch
   env -u XDG_CONFIG_HOME "$KIRSCH_BIN" doctor; echo "exit=$?"
   ```

   Expected: one `model:` line for the model the settings name. It starts `ok` if kirsch lists the model, and `warn  model:` with `not in the table; conservative fallback` if it does not. Either way the run ends `exit=0`.

6. A refused candidate, then a candidate chosen after it (authored here; no operator section runs this, and it measures the context rows of the table).

   Working directory: the kirsch folder. Workspace: `$w6`, given with `--workspace` on both commands. Settings: `$c6`, for these two commands only; it names `project_files`.

   ```bash
   # authored here
   c6=$(mktemp -d); mkdir -p "$c6/kirsch"; printf '[context]\nproject_files = [".env", "AGENTS.md"]\n' > "$c6/kirsch/config.toml"
   w6=$(mktemp -d); printf 'x\n' > "$w6/.env"
   XDG_CONFIG_HOME="$c6" "$KIRSCH_BIN" doctor --workspace "$w6" | grep 'context:'
   printf 'project notes\n' > "$w6/AGENTS.md"
   XDG_CONFIG_HOME="$c6" "$KIRSCH_BIN" doctor --workspace "$w6" | grep 'context:'
   ```

   Expected: first, `warn  context: none; looked for .env, AGENTS.md; project context candidate ".env" refused: path ".env" is outside the workspace or denied: matches the denied pattern .env`. Second, the same refusal after `warn  context: AGENTS.md, <n> bytes;`, where `<n>` is the size of the file just written. The `.env` file is refused by its name and is never read.

7. No workspace found: a folder outside any repository, with no `--workspace` flag (authored here).

   Working directory: `$d7`, a temporary folder outside any repository. Workspace: none is found, because no flag is given. Settings: Setup, unchanged. The step 3a measurement used an empty settings folder instead; see *Notes / evidence*.

   ```bash
   # authored here
   d7=$(mktemp -d); cd "$d7"
   "$KIRSCH_BIN" doctor; echo "exit=$?"
   cd /Volumes/DATA/Github/kirsch
   ```

   Expected: `fail  repository: not in a Git repository`, then `warn  context: skipped: no workspace`, and `exit=1`. The `fail` is the repository line's; the context line only reports the skip.

8. **Key-state check** (authored here; operator section 20, step 5). Working directory: the kirsch folder. Workspace: n/a. The dummy value is removed. The unset line clears every key variable, and the count after it must print `CONTROL PASS`, then `0`. The next section that checks keys runs the count again before any key is entered.

   ```bash
   # authored here
   unset ANTHROPIC_API_KEY KIRSCH_ANTHROPIC_API_KEY OPENCODE_API_KEY KIRSCH_OPENCODE_API_KEY
   keycount_control
   keycount
   ```

9. **The automatic tests for this piece, with a count first** (authored here; operator section 20, step 6). Working directory: the kirsch folder. Workspace: n/a. The first two commands print two counts, and they must be equal: the number of test names in `PAT`, and the number of tests `go test -list` selects from it. Neither count is written here, because the pattern is the source of the names. A difference means the pattern selects a different set of tests from the names it carries, and the step fails.

   ```bash
   cd /Volumes/DATA/Github/kirsch
   PAT='^(TestKeyLineNamesWinner|TestKeyLineNilGetenvIsEmptyEnvironment|TestKeyValueNeverPrinted|TestKeyLineWithholdsNonNameAPIKeyEnv|TestModelLine|TestContextLine|TestBackendLineUnderPathShim|TestResolvedLinesSkippedWhenConfigDidNotLoad|TestContextLineSkippedWithoutWorkspace|TestResolvedLinesNeverFail|TestRunDoctorKeyLine|TestRunDoctorModelLine|TestRunDoctorContextLine|TestRunDoctorBackendLine|TestRunDoctorSkippedLinesWhenConfigDidNotLoad|TestIsolatedEnvClearsKeyVariables|TestAllowedIsExact|TestImportRules)$'
   printf '%s\n' "$PAT" | grep -o 'Test[A-Za-z0-9_]*' | wc -l
   go test -list "$PAT" ./internal/doctor ./cmd/kirsch ./internal/arch | grep -c '^Test'
   go test -count=1 -run "$PAT" ./internal/doctor ./cmd/kirsch ./internal/arch; echo "exit $?"
   ```

   Expected: two equal counts (the names in `PAT`, then the tests listed), then one `ok` line for each of `./internal/doctor`, `./cmd/kirsch` and `./internal/arch`, and `exit 0`. A line starting `FAIL`, or a last line other than `exit 0`, fails the step.

10. **A pattern that selects nothing** (authored here; operator section 20, step 7). Working directory: the kirsch folder. Workspace: n/a. It shows why step 9's count comes first.

   ```bash
   cd /Volumes/DATA/Github/kirsch
   go test -count=1 -run '^NoSuchTestXYZ$' ./internal/doctor; echo "exit $?"
   ```

   Expected: one `ok` line that ends `[no tests to run]`, then `exit 0`. Go counts "no tests to run" as a success, so this command passes with no test run; step 9's count is what catches it.

**Notes / evidence**

Run on 2026-10-29 by mission-20261029-06, from the agent's Bash tool, in two passes. The first pass (step 3) is in `.claude/memory/workspace/m20261029-06-step3-runs.txt` (scratch, archived at the next mission start). Step 3a (docs fix round 1) reran the hands-on commands and changed this section; its run script is `.claude/memory/workspace/m20261029-06-step3a-tmp/run.zsh` and its output, verbatim, is `.claude/memory/workspace/m20261029-06-step3a-runs.txt` (both scratch, archived at the next mission start). The step 3a script unsets the four key variables in its own process only, uses dummy values, and never reads a real key. It sets `XDG_CONFIG_HOME` to an empty folder under the workspace (`cfg-empty`) unless a run names another settings folder; hands-on 5 reads the operator's own settings through `env -u XDG_CONFIG_HOME`. The workspace sits inside the kirsch repository, so the script sets `GIT_CEILING_DIRECTORIES` to the workspace root: without it, git climbs from a workspace folder into the kirsch repository, and a folder meant to be outside any repository is not. The step 3 pass wrote the `mktemp -d` folders of its own runs to the system temporary area (`/var/folders/...`), not to the workspace, and its runs header said otherwise; that claim was wrong and is corrected in the runs file. Step 3a creates its folders with `mkdir -p` under the workspace. The setup script itself was not sourced, because it makes its folders under the system temporary area.

- Hands-on 1 (no key, then a dummy key): the no-key run prints the `warn  key` line naming both variables and ends `exit=0`. The dummy run, in the form of operator step 4, prints `ok  key: endpoint opencode: key found in KIRSCH_OPENCODE_API_KEY`; the count of the dummy value in its output is `0`, and the positive control counts `1` for `KIRSCH_OPENCODE_API_KEY`.
- Failing run for operator step 2 (a key set, so the no-key warn line must be absent): `grep -c '^warn  key:'` prints `0` in step 3a's runs file, so step 2's box is not ticked when a key is set.
- Failing run for operator step 3 (settings that do not read): with `XDG_CONFIG_HOME` pointing at a broken folder, the run prints `fail  config: parse` and ends `exit=1`, so step 3's `exit=0` check fails. Both are in the step 3a runs file.
- Failing run for the dummy-key step (a leak): a scratch copy of `internal/doctor/probes.go` that prints the value, built with `go build -overlay` under the workspace (no repository file edited), prints `ok  key: endpoint opencode: key found in KIRSCH_OPENCODE_API_KEY = dummy-not-a-key`. `grep -c dummy-not-a-key` on its output prints `1`, not `0`, so the check catches the leak. The step 3a runs file holds this run.
- Hands-on 2 (model not in the table): the run in the file prints the `warn  model` line with `no-such-model` and the 128000-token fallback, and ends `exit=0`. The developer's own settings name `claude-haiku-5-5`, which is also not in the table; that run also warns.
- Hands-on 3 (no `AGENTS.md`): the repository line warns, the context line reads `ok  context: none; looked for AGENTS.md, CLAUDE.md`, and the run ends `exit=0`.
- Hands-on 4 (settings that do not read): `fail  config: parse`, then the key, model and context lines each read `skipped: config did not load`, the backend line still prints, and the run ends `exit=1`. The failure is the config line's.
- Hands-on 5 (the operator's own settings): the model line warns `model: endpoint opencode, model claude-haiku-5-5: not in the table; conservative fallback` on the Mac the agent ran on, and the run ends `exit=0`.
- Hands-on 6 (context rows, step 3a): a refused `.env` in `project_files`, with no later candidate, gives `warn  context: none; looked for .env, AGENTS.md; project context candidate ".env" refused: ...`. With `AGENTS.md` present as well, the line reads `warn  context: AGENTS.md, 20 bytes; project context candidate ".env" refused: ...`. `project_files = []` gives `ok  context: none; no project files configured`.
- Hands-on 7 (no workspace, step 3a): from a folder outside any repository, with no flag, the repository line is `fail`, the context line is `warn  context: skipped: no workspace`, and the run ends `exit=1`.
- Test selection and runs: the pattern in operator step 6 selects tests across `./internal/doctor`, `./cmd/kirsch` and `./internal/arch`. On 2026-10-29 in step 3c, its count of names in `PAT` and its `go test -list` count were equal, both 18 (`.claude/memory/workspace/m20261029-06-step3c-runs.txt`, part 1). The same count was taken in step 3b (`.claude/memory/workspace/m20261029-06-step3b-runs.txt`, part 7). The same pattern runs `ok` for all three packages and ends `exit 0`.
- Failing runs: a pattern that selects nothing prints `0` from the count, and its test command still ends `exit 0` on an `ok` line that ends `[no tests to run]` (step 3c, `.claude/memory/workspace/m20261029-06-step3c-runs.txt`, part 4), which is why the count comes first. A `PAT` with one name misspelled prints 18 names and 17 tests listed, and its test command still ends `exit 0` (part 3), so the two counts must be compared. An overlay that changes one format string on the known branch of the model line fails `TestModelLine/known` and ends `exit 1` (the step 3 runs file, `.claude/memory/workspace/m20261029-06-step3-runs.txt`, scratch and archived at the next mission start).
- Comment check: the diff of both files against `686d3c4` changes at least one line in each file, and every changed line is a `//` comment. The control line is reported as `+var controlProbe = 1`. The counts are not written here, because they move with each comment edit. The step 3b run of this check is `.claude/memory/workspace/m20261029-06-step3b-comments-runs.txt` (scratch, archived at the next mission start); parts 9a and 9b show that no changed line is a non-comment, parts 9c and 9d show at least one changed line in each file, and part 9e is the control.
- Key-state check, last: not run in the agent shell, because the Setup functions (`keycount` and `keycount_control`) are not loaded there. The step 3a run script counted the exported key variables after its `unset`, and the count printed `0` (a count only; no value was read or printed). The check itself is for the operator's Setup terminal.
- **Step 3b (docs fix round 2, 2026-10-29, mission-20261029-06).** Measured with a fresh build, `go build -o .claude/memory/workspace/m20261029-06-step3b-kirsch ./cmd/kirsch` (unstamped, so the version line reads `kirsch 0.1.0-dev`). The script is `.claude/memory/workspace/m20261029-06-step3b-run.zsh` and its output is `.claude/memory/workspace/m20261029-06-step3b-runs.txt` (scratch, archived at the next mission start). Each kirsch run used an empty settings folder under the workspace, not the Setup settings file (the `go test` runs read no settings):
  - Hands-on 1 shape, from the kirsch folder, no key set (part 4): `doctor: 7 ok, 1 warn, 0 fail`, and `exit=0`. The one warning is the `key` line. The `context` line reads `AGENTS.md, 6687 bytes`, the size of that file on the day of the run. **Dated note, 2026-10-30 (m5-d4b, step 4c):** since m5-d4b the summary also counts the `state` and `lock` lines, so this sample now reads `doctor: 9 ok, 1 warn, 0 fail` with a writable state folder (step 4c, run B, in `.claude/memory/workspace/m20261030-01-step4c-doctor-run.txt`, with a scratch settings folder). The count moves with the key, `rg` on PATH, the state folder and the lock.
  - Outside a repository with no flag, in a folder under the workspace with the git ceiling set to the workspace root (part 6): `doctor: 5 ok, 2 warn, 1 fail`, and `exit=1`. The fail is the repository line. The warnings are the `key` line and `context: skipped: no workspace`. **Dated note, 2026-10-30 (m5-d4b, step 4c):** measured again at step 4c (run E, `.claude/memory/workspace/m20261030-01-step4c-doctor-run.txt`), with the `state` and `lock` lines counted, this run reads `doctor: 5 ok, 4 warn, 1 fail` and `exit=1`. The two new warnings are `warn  state: skipped: no workspace` and `warn  lock: skipped: no workspace`. The fail is still the repository line.
  - Hands-on 9 (part 7): the count is `18`, and the run ends `exit 0` across the three packages. Hands-on 10 (part 8): the pattern selects nothing, and it prints `[no tests to run]` with `exit 0`.
- **Settings folders for the step 3a runs, corrected in step 3b.** Hands-on 7 ran with the empty settings folder (`cfg-empty`). Hands-on 6 did not: it ran with its own settings folders (`cfg-ctx` and `cfg-nopf`), which name `project_files`. Neither used the Setup settings file. The §20 hands-on steps above now say which folder each command uses.
- **Pre-edit copy of this file, step 3a (W9).** Step 3a's pre copy of this file was overwritten during step 3a, because its basename matched the milestone file's, and it was rebuilt by reversal. The rebuilt copy, `.claude/memory/workspace/m20261029-06-step3a-pre/plan_testing_manual_milestone-5.md`, has sha256 `226f3c3e80a37ad6363496b3fc5ccb953f264e593e8cff0dfb1fbc504cb55488` (recomputed with `shasum -a 256` in step 3c, `.claude/memory/workspace/m20261029-06-step3c-runs.txt`, part 5). No hash of this file was recorded before step 3a edited it, so that rebuild is not verified against a pre-3a value. The only earlier hash of this file in the workspace is `567bd2dabba5818475364ee92ed0de85b14b925bb16ee0177f4e7825691184a2`, in `.claude/memory/workspace/m20261029-06-step3-pre-hashes.txt` (line 7, for `m20261029-06-step3-pre/plan_testing_manual_milestone-5.md`). That copy was taken before step 3, not before step 3a, so it does not verify the rebuild, and the rebuilt copy does not match it. No hash of the state after step 3 was found in the workspace. From step 3b on, the pre-edit hash of this file is recorded before the first edit, in `.claude/memory/workspace/m20261029-06-step3b-pre-hashes.txt`: `f7b581d621335a99ae9deb12971f44a9fbd1475c6b5a4b63e3e8f85bddff4889`, which is the state after step 3a.

---

## 21 · Soft-wrap fidelity: composer cut marker (m5-d7b)

**Context:** Terminal: Setup. Binary: `go test`; `npm run screens`; for hands-on 3, `$KIRSCH_BIN`, built by `npm run build` (hands-on 3 rebuilds it first). Working directory: /Volumes/DATA/Github/kirsch. Workspace: `$WORKSPACE` (Setup B) for hands-on 3; n/a for the rest. Key state: all key variables unset (kirsch in hands-on 3 needs no key and sends nothing). Config state: Setup, unchanged.

**What landed**

O17 is recorded as the `⋯` marker (`plan/milestones/milestone-5.md`, decisions table). `(*Composer).rows` in `internal/tui/composer.go` ends every composer row that is cut with the warning-styled marker (`g.Trunc`: `⋯`, or `...` in ASCII, ui-spec §10.2). Each row is cut before `View()`'s hard edge in `exactly` (`internal/tui/view.go`), which drops cells with no marker. A row whose text fits is returned unchanged and unmarked. One exception: when the caret sits at the end of its line, with no character under it, the caret's own cell can sit one cell past the edge; `exactly` drops that cell with no marker, so every typed character stays visible. A caret over a zero-width character mid-text takes a cell of its own, so that row is cut and marked even though its text fits; `TestComposerCaretOverWideAndZeroWidthAtTheBoundary` pins it. The cut applies to every composer row. A wide character at the cut leaves the row one cell short of the edge. The caret past the cut on a long line is not visible; v0.1 has no horizontal scrolling (ui-spec §7.3).

**Fences that moved:** none. No golden changed. The 686d3c4 control pair below shows it.

**Waits on:** m5-d6 (composer work), m5-d7a (both touch `view.go`), O17 (the marker), and O23 (golden handling), from `plan/milestones/milestone-5.md:106`.

**Automated checks**

- [x] At 40 columns no composer row is cut without the marker O17 records (`plan/milestones/milestone-5.md`, `## m5-d7b`, acceptance). Command:

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -count=1 -run 'TestComposerCut|TestComposerFit|TestComposerBusyFirst|TestComposerCaretOver' ./internal/tui; echo "exit $?"
  ```

  Run 2026-10-29: `ok`, `exit 0`. Failure case: the same command, run with `-overlay` against a copy of `internal/tui/composer.go` whose marker argument is `""`, prints `FAIL` and `exit 1`. The real file was not changed.
- [x] Golden check, procedure step 5 (`plan/milestones/milestone-5.md`, *Golden mechanism and regeneration procedure*):

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -count=1 ./internal/tui; echo "exit $?"
  cd /Volumes/DATA/Github/kirsch && npm run screens; echo "exit $?"
  ```

  Run 2026-10-29: `go test` prints `ok` and `exit 0`; `npm run screens` prints `kirsch-ui-screens.md: grids OK, content OK` and `exit 0`. No fence moved, so no regeneration was run.

**Hands-on checks**

1. No golden moved, so no regeneration ran. The 686d3c4 control pair: the screens file must print nothing, and the composer file must print a line (it changed; that is code, not a golden).

  ```bash
  cd /Volumes/DATA/Github/kirsch && git diff --stat 686d3c4 -- plan/spec/kirsch-ui-screens.md; echo "exit $?"
  cd /Volumes/DATA/Github/kirsch && git diff --stat 686d3c4 -- internal/tui/composer.go; echo "exit $?"
  ```

  Run 2026-10-29: the first prints nothing, `exit 0`; the second prints `internal/tui/composer.go | 62 ++++++++++++++++++++++++++++++++++++++++++++----` with `1 file changed, 57 insertions(+), 5 deletions(-)`, `exit 0`.
2. (operator) The golden diff review is this section's screens check. With no fence moved there is no golden diff to read, and the operator checklist (section 21, step 4) asks for the same check against the 686d3c4 baseline.
3. Not run by the crew. (operator) Run `npm run build; echo "exit $?"` first, and go on only on `exit 0`, so `$KIRSCH_BIN` is the current build. Then, in a window 40 columns wide (`stty size` shows a second number from 40 to 47; below 40 kirsch prints `terminal too narrow`; at 48 or more the placeholder shows whole, see the width note below), start kirsch as operator section 21, step 5 does (`"$KIRSCH_BIN" --workspace "$WORKSPACE"`, after *Before you start*, with Setup B having said `Ready.`). Before typing, the empty typing area shows the dim placeholder cut to the window: `> Ask anything (Enter to send, /help ⋯`, ending in `⋯`. Type one line longer than the window without pressing Enter. The row must end in `⋯` at the right edge, one column in from the window's edge. Esc empties the typing area, which then shows that same placeholder line again. `/quit` only when the typing area shows the dim `Ask anything` line again. The crew has no real terminal and could not resize one, so this step is for the operator.

**Notes / evidence**

Run 2026-10-29, with m5-d7b uncommitted on top of 686d3c4.

- Moved goldens: none. `git diff --stat 686d3c4 -- plan/spec/kirsch-ui-screens.md` printed nothing.
- Automated: cut tests `ok`, `exit 0`; failure case (marker emptied through `-overlay`) `FAIL`, `exit 1`; `go test -count=1 ./internal/tui` `ok`, `exit 0`; `npm run screens` `grids OK, content OK`, `exit 0`.
- Build: `npm run build` `exit 0`. A copy with a syntax error, through `go build -overlay`, `exit 1`. The operator's step 1 now ends `echo "exit $?"` and is not ticked on a non-zero exit. Failing run through npm, with the real build script and no source edit: `npm run build -- not-a-version; echo "exit $?"` prints `build.sh: version argument 'not-a-version' is not X.Y.Z[-suffix|+suffix]` and `exit 2` (`.claude/memory/workspace/m20261029-05-step3a-runs.txt`, R1). R1 is a stand-in for a failing build, not a compile failure: it shows that npm passes a failing build script's non-zero exit through, using an argument error. The compile-failure case is the `go build -overlay` run above. Npm's own exit code is the rule the operator checks.
- Failing marker-test run with no test matched: `go test -count=1 -run 'TestComposerNoSuchMarkerTestName' ./internal/tui` prints `ok ... [no tests to run]` and `exit 0`. The operator's step 3 names `[no tests to run]` as a do-not-tick case (runs R5).
- Screens: the operator's step 4 control, `git diff --stat 686d3c4 -- internal/tui/composer.go`, prints `1 file changed, 57 insertions(+), 5 deletions(-)` (R10). A failing run of the screens lint was made on scratch copies of `scripts/lint-screens.py`, `internal/tui/view.go` and `plan/spec/kirsch-ui-screens.md`, with one grid row one column short. It was `python3 scripts/lint-screens.py` run directly on the scratch copies, the command that `npm run screens` wraps, and not `npm run screens` itself. A `cmp` of the scratch copy of the script against `scripts/lint-screens.py` exits 0, so the two are identical. It prints `screen 05 (80×20): ragged box borders` and `result: 1`. The real screens file was not edited (R7; the copy's control run, R6, prints `grids OK, content OK`).
- Width, read from the code and not run on a terminal. Arithmetic, labelled as such: the placeholder row is `> ` plus the 44-cell `Placeholder` in `internal/tui/composer.go`, 2 + 44 = 46 cells. Content width is the window width minus 2 (`framePadding` in `internal/tui/view.go`, `framePad = 1`). The row is cut while 46 is more than the content width, so it is cut at 40 to 47 columns and shows whole at 48 or more. At 40 columns the content width is 38, and the cut row is 37 cells of text plus `⋯`, which `TestComposerCutAcrossModesAt40` pins as `> Ask anything (Enter to send, /help ⋯`. The right pad is one column, so the `⋯` sits one column in from the window's right edge. The operator's width check is `stty size`, second number from 40 to 47.
- Zero-width caret mid-text, the case in the spec and CHANGELOG: `TestComposerCaretOverWideAndZeroWidthAtTheBoundary` passes in both Unicode and ASCII modes, with the `zwsp/36 letters cut` case (38 cells with the caret, cut and marked) (R11).
- Measured: at 40 columns the content width is 38 cells (the cut rows in the failure output are 38 cells wide).
- Hands-on at 40 columns: not run (no real terminal in the crew's shell).

---

## 22 · `doc/usage.md` corrections (m5-d14a)

**Context:** Terminal: Setup. Binary: none. The checks are `grep`, `printf` and `git show b2be4ba:`. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

Each correction below has a check under *Automated checks*. The text was written from the review files in `.claude/memory/missions/mission-20261031-03/` (`review-step2.md` and `review-step2b.md`). Those files cite `internal/tui/statusbar.go` for the status-bar narrowing, `internal/tui/update.go` for the Tab order, `internal/policy/policy.go` for command approval, and `cmd/kirsch/main.go` for the lock.

- **Starting it.** The paragraph under the start-up commands says `kirsch resume` restores the most recent session with its session grants. It also says that when another Kirsch is already running in the workspace, `kirsch resume` starts a fresh session instead and says so.
- **The screen, Sizes.** Below 80 columns the status bar drops its token figures. As the width shrinks further, it shortens its left-hand text, starting with the grant count, and keeps the state for as long as it can. No ladder of named steps is written out.
- **The screen, Status bar.** A persistent warning is never removed by narrowing. On a very narrow terminal a long warning is cut at the right edge.
- **The screen, Composer.** The composer scrolls up and down to keep the line being typed in view. It does not scroll sideways. A line wider than the terminal is cut at the edge with a `⋯` marker.
- **The screen, under the status-bar mock.** The sentence says the `cache` figures are tokens read from the cache, then tokens written to it.
- **Pasting.** A terminal that does not send a paste as one block delivers each tab as a Tab key press. The first one moves focus to the cards, and the text after it is read as card keys.
- **An approval is pending.** The example of a typed command is `/new`. The old `/run` example is gone.
- **The approval model.** A patch always waits, and no setting turns that off. A command runs without a card when it is exactly one of the built-in allowlist commands, with no further arguments, or when a session grant covers it. Every other command waits unless `policy.require_approval_for_commands` is `false`, and shells and runners wait regardless. A grant cleared with `/approvals` is recorded in the session log as cleared, so it stays cleared after `kirsch resume`.
- **Slash commands, `/files`.** The row says a resumed session starts the list empty and does not rebuild it from the earlier transcript.
- **The debug commands section.** The heading `### Debug commands — temporary` and its table are removed.
- **The approval sentence.** The sentence beginning "Nothing consequential happens silently" is replaced. The replacement drops that opening phrase and names the exact-match allowlist and session grants.
- **The missing-`rg` claim.** Removed. No code in `internal/tui` or `internal/app` writes a warning for a missing `rg` (see the emitter search under *Hands-on checks*).
- **The spec.** The session grants paragraph in `plan/spec/kirsch-plan.md` gained the sentence ending "stays cleared after `kirsch resume`".

**Automated checks**

The line numbers in the expected output below belong to the guide as it stands at this review. They move whenever the guide is edited. The text after each number is what is checked.

- [ ] The audit names a section covering key bindings, one covering slash commands, and one covering the approval model (`plan/milestones/milestone-5.md:1259`). Record each heading in the step report.

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n '^## \|^### ' doc/usage.md
  ```

  The printed list is the audit record. It must contain `## Keys` with its `### ` subsections, `## Slash commands` and `## The approval model`, and it must not contain a `### Debug commands` line. Expected output, as printed:

  ```
  20:## Starting it
  46:## The screen
  108:## Keys
  113:### Composing — the default
  142:### Browsing — after `Shift+↑` or `Tab`
  164:### An approval is pending
  183:### A modal is open
  196:### A confirm prompt is up
  211:### Everywhere
  228:## Scrolling and pinning
  254:## Slash commands
  277:## Tool cards
  302:## The approval model
  343:## When colour or Unicode is not available
  363:## If something looks wrong
  ```

- [ ] The status-bar mock still shows cache read and write, and the sentence after it explains the figures (`plan/milestones/milestone-5.md:1260`). Command:

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n 'cache 8.9k/1.2k\|figures are tokens read' doc/usage.md
  ```

  Expected output, as printed (the mock line and the sentence under it):

  ```
  61: claude-sonnet-5 · ⠋ thinking · in 12.4k out 980 cache 8.9k/1.2k · $0.04   status bar
  66:The `cache` figures are tokens read from the cache, then tokens written to it.
  ```

- [ ] The `/files` row says a resumed session starts the list empty (`plan/milestones/milestone-5.md:1261`). Command:

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n '^| `/files`' doc/usage.md
  ```

  Expected output, as printed:

  ```
  265:| `/files` | A notice listing the files changed by approved patches this session. A resumed session starts this list empty and does not rebuild it from the earlier transcript |
  ```

- [ ] The section headed `### Debug commands — temporary` is gone, and so is its table. No `/read <path>`, `/ls [path]`, `/search <query>`, `/gitstatus` or `/gitdiff` line remains (`plan/milestones/milestone-5.md:1262`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n '/read <path>\|/ls \[path\]\|/search <query>\|/gitstatus\|/gitdiff\|### Debug commands' doc/usage.md
  printf '/read <path>\n' | grep -n '/read <path>\|/ls \[path\]\|/search <query>\|/gitstatus\|/gitdiff'
  ```

  The first command prints nothing. The second prints `1:/read <path>`, which shows the search does find the command when it is present.

  The same check as commands, which print PASS or FAIL (authored here). The control reads the pre-change text at `b2be4ba`:

  ```bash
  # authored here
  test "$(grep -c '/read <path>\|/ls \[path\]\|/search <query>\|/gitstatus\|/gitdiff\|### Debug commands' /Volumes/DATA/Github/kirsch/doc/usage.md)" = 0 && echo "PASS real" || echo "FAIL real"
  test "$(git -C /Volumes/DATA/Github/kirsch show b2be4ba:doc/usage.md | grep -c '/read <path>\|/ls \[path\]\|/search <query>\|/gitstatus\|/gitdiff\|### Debug commands')" -ge 1 && echo "PASS control" || echo "FAIL control"
  ```

  Expected: `PASS real` and `PASS control`.

- [ ] "Starting it" names `kirsch resume` and `kirsch --new` (`plan/milestones/milestone-5.md:1263`). Command:

  ```bash
  cd /Volumes/DATA/Github/kirsch && sed -n '/^## Starting it/,/^## /p' doc/usage.md | grep -n 'kirsch resume\|kirsch --new'
  ```

  Expected output, as printed. The numbers count lines from the heading `## Starting it`:

  ```
  5:kirsch resume                # resume the most recent session for this workspace
  6:kirsch --new                 # start a fresh session even when a previous session exists
  11:`kirsch resume` picks up the most recent session for this workspace, including
  13:starts a fresh session instead and says so. `kirsch --new` starts a fresh
  ```

- [ ] The sentence beginning "Nothing consequential happens silently" is gone. The replacement names the exact-match allowlist and session grants (`plan/milestones/milestone-5.md:1264`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n 'Nothing consequential happens silently' doc/usage.md
  ```

  The same check as commands, which print PASS or FAIL (authored here). The control reads the pre-change text at `b2be4ba`:

  ```bash
  # authored here
  test "$(grep -c 'Nothing consequential happens silently' /Volumes/DATA/Github/kirsch/doc/usage.md)" = 0 && echo "PASS real" || echo "FAIL real"
  test "$(git -C /Volumes/DATA/Github/kirsch show b2be4ba:doc/usage.md | grep -c 'Nothing consequential happens silently')" -ge 1 && echo "PASS control" || echo "FAIL control"
  ```

  Expected: the first command prints nothing, and the PASS/FAIL lines read `PASS real` and `PASS control`.

- [ ] The composer paragraph describes the behaviour O10 records (`plan/milestones/milestone-5.md:1265`). Command: below. Review step.

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n 'scrolls up and down\|does not scroll sideways' doc/usage.md
  ```

  Expected output, as printed. The first line is the sentence about scrolling up and down, and the second is the sentence about sideways scrolling:

  ```
  84:height. Past that maximum the composer scrolls up and down to keep the line you
  86:`⋯` marker; the composer does not scroll sideways. While a turn is running it
  ```

- [ ] `doc/usage.md` makes no claim of a missing-`rg` warning (`plan/milestones/milestone-5.md:1266`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n 'missing .rg' doc/usage.md
  ```

  The same check as commands, which print PASS or FAIL (authored here). The control reads the pre-change text at `b2be4ba`:

  ```bash
  # authored here
  test "$(grep -c 'missing .rg' /Volumes/DATA/Github/kirsch/doc/usage.md)" = 0 && echo "PASS real" || echo "FAIL real"
  test "$(git -C /Volumes/DATA/Github/kirsch show b2be4ba:doc/usage.md | grep -c 'missing .rg')" -ge 1 && echo "PASS control" || echo "FAIL control"
  ```

  Expected: the first command prints nothing, and the PASS/FAIL lines read `PASS real` and `PASS control`.

- [ ] The plural gate. The milestone's own pattern, `'milestone|ADR'` (`plan/milestones/milestone-5.md:1267`), misses plurals, and the control below shows it. This check takes the plural forms as section 17 does. The control is a printed line with a plural in it, and the old pattern run on the same input prints only the singular line.

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n -i -w -E 'milestones?|ADRs?' doc/usage.md
  printf 'two milestones\nan ADR\nheadroom\n' | grep -n -i -w -E 'milestones?|ADRs?'
  printf 'two milestones\nan ADR\nheadroom\n' | grep -n -i -w -E 'milestone|ADR'
  ```

  The first command prints nothing. The second prints `1:two milestones` and `2:an ADR`, and not `headroom`. The third, the milestone's own pattern, prints only `2:an ADR`.

- [ ] The source gate (authored here). The guide cites no source path and carries no source label.

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n -E 'internal/|cmd/kirsch/|Sources?:' doc/usage.md
  printf 'see internal/tui\nSource: x\nplain\n' | grep -n -E 'internal/|cmd/kirsch/|Sources?:'
  ```

  The first command prints nothing. The second prints `1:see internal/tui` and `2:Source: x`, and not `plain`.

- [ ] The spec sentence is in the session grants paragraph (`plan/spec/kirsch-plan.md`, authored here). Commands and PASS/FAIL, with the controls reading `b2be4ba`, where the sentence does not exist. The positive control runs first, and proves the old paragraph was read, so the absence control cannot pass on a failed `git show`:

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n 'stays cleared after' plan/spec/kirsch-plan.md
  # authored here
  test "$(grep -c 'stays cleared after' /Volumes/DATA/Github/kirsch/plan/spec/kirsch-plan.md)" -ge 1 && echo "PASS real" || echo "FAIL real"
  test "$(git -C /Volumes/DATA/Github/kirsch show b2be4ba:plan/spec/kirsch-plan.md | grep -c 'Session grants live in memory')" -ge 1 && echo "PASS control" || echo "FAIL control"
  test "$(git -C /Volumes/DATA/Github/kirsch show b2be4ba:plan/spec/kirsch-plan.md | grep -c 'stays cleared after')" = 0 && echo "PASS control absent" || echo "FAIL control absent"
  ```

  Expected: the first command prints `202:` and the session grants line, which contains `stays cleared after` followed by `kirsch resume`. The PASS/FAIL lines read `PASS real`, `PASS control` and `PASS control absent`.

- [ ] The step 2 wording is present: no setting turns the patch approval off (the approval model's first paragraph).

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n 'no setting turns that off' doc/usage.md
  ```

  Expected output, as printed:

  ```
  304:A patch appears as a card and waits, and no setting turns that off. A command
  ```

- [ ] The step 2 wording that was wrong is gone (authored here). The old phrases are in the text at `b2be4ba`, and the draft phrases were only in the draft the review read. The patterns are split, and each has its own control:

  ```bash
  # authored here
  PAT_OLD='/run|Clear all session grants'
  PAT_DRAFT='turned off|model family alone|spinner by itself|pending state|every width|which moves focus to the cards|carries no token figures'
  test "$(git -C /Volumes/DATA/Github/kirsch show b2be4ba:doc/usage.md | grep -c -E "$PAT_OLD")" -ge 1 && echo "PASS control old" || echo "FAIL control old"
  test "$(printf '%s\n' 'turned off' 'model family alone' 'spinner by itself' 'pending state' 'every width' 'which moves focus to the cards' 'carries no token figures' | grep -c -v -E "$PAT_DRAFT")" = 0 && echo "PASS control draft" || echo "FAIL control draft"
  test "$(grep -c -E "$PAT_OLD|$PAT_DRAFT" /Volumes/DATA/Github/kirsch/doc/usage.md)" = 0 && echo "PASS real" || echo "FAIL real"
  ```

  Expected: `PASS control old`, `PASS control draft` and `PASS real`. The first control shows the old phrases match the pre-change text, and the second shows every draft phrase is matched by its pattern, so the empty result in the real file means the phrases are gone and the patterns are not broken.

- [ ] The positive checks for the corrections (authored here). Each phrase prints `=> 1` when its replacement text is present in the guide, and the last phrase, which is not in the guide, prints `=> 0`:

  ```bash
  cd /Volumes/DATA/Github/kirsch && for p in 'the status bar also shortens' 'sit at the right. No narrowing' 'the first moves focus' '^in `/new`' 'cleared after `kirsch resume`' 'no such phrase here'; do printf '%s => ' "$p"; grep -c -- "$p" doc/usage.md; done
  ```

  Expected: `=> 1` after every phrase except the last, and `=> 0` after the last. The first line reads `the status bar also shortens => 1`.

**Hands-on checks**

1. Every control that needs the pre-change text reads it with `git show` from commit `b2be4ba`, the commit before these corrections. The other controls read text printed by the same command, except the last phrase of the positive-check loop, `no such phrase here`, which reads the working copy of the guide and must print `0` whatever the guide says. The copy in `.claude/memory/workspace/` is scratch and is archived when the next mission starts, so no control reads it.

2. Emitter search for the missing-`rg` warning (`plan/milestones/milestone-5.md:1233`). Command:

   ```bash
   cd /Volumes/DATA/Github/kirsch && grep -rn -i 'rg\b\|ripgrep' internal/tui internal/app --include='*.go' | grep -v '_test\.go'
   ```

   Expected result in words: no line in `internal/tui`, where the status bar is drawn, sets, emits or prints a warning about a missing `rg`. In `internal/app/app.go`, the matches are the line that sets `HasRG` from `a.hasRG()`, the `hasRG()` function, which runs `exec.LookPath("rg")`, and the end of ordinary words such as `arg`. The match in `internal/tui` is the end of `org` in a comment. `HasRG` is defined in `internal/agent/prompt/prompt.go` and feeds only the system prompt, where `internal/agent/prompt/system.md` writes the value after `ripgrep (rg) available`. The missing-`rg` claim therefore has no status-bar emitter behind it, and the guide does not make it. Output as printed:

   ```
   internal/tui/styles.go:49:// empty NO_COLOR is treated as unset, the no-color.org convention (operator
   internal/app/app.go:763:		HasRG:         a.hasRG(),
   internal/app/app.go:1470:func (a *App) hasRG() bool {
   internal/app/app.go:1471:	_, err := exec.LookPath("rg")
   internal/app/app.go:2542:			for _, arg := range anySlice {
   internal/app/app.go:2543:				if s, ok := arg.(string); ok {
   ```

**Notes / evidence**

_Empty. Fill at run._

---

## 23 · Front-door and tracker freshness (m5-d15)

**Context:** Terminal: Setup. Binary: none (greps, `ls`). Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

Landed in mission-20261106-02 (2026-11-06). Each claim below is checked by the command named in *Notes / evidence*, by row number.

**Judgement for each "0001–0007" site (DIR-004).** The sites are the ones the repo-wide grep prints, with their live line numbers.

| Site | Before | After | Judgement |
|---|---|---|---|
| `plan/spec/kirsch-plan.md:391` (the *Design docs are already settled* bullet) | `` `plan/adr/` (0001–0007). `` | `` `plan/adr/` (0001–0007, the ADRs in the planning baseline; ADR 0008 onward are later and are indexed in `plan/adr/README.md`). `` | Kept, and qualified. The range names the planning baseline, and the qualifier says where the later ADRs are indexed. |
| `plan/README.md` (the `adr/` row) | No range. The step 1 copy `m5-d15-plan-README.pre.md` has none either. | No range. | No site to judge. The range left this file at commit `4a50a76`, before this mission. |
| `AGENTS.md:21` (*Why was X decided?*) | `ADRs 0001–0007` | `ADRs 0001–0012` | Changed to the range that `ls plan/adr/` lists. |
| `plan/adr/README.md:65` (*Status Update*) | `ADRs 0001–0007 were written during planning, and most still carry the status ...` | Unchanged. | Kept. The sentence is true as written. The status wording is the operator's decision, and it is not edited. |
| `CHANGELOG.md:27` (the planning-baseline entry) | `and ADRs 0001–0007.` | Unchanged. | Kept. It is history. |
| `plan/PROGRESS.md:593` (*Acceptance box on "0001–0007"*, under *Open from m5-d15*) | The open item quotes the range. | Unchanged. | Kept. It is a note about the box, not a claim about the ADR set. |
| `plan/testing/manual/milestone-5.md` (this file) | This section quotes the range as the subject of its checks. | Unchanged in substance. | Kept. It is the check's own text, and the master's copy of the box excludes it. |

**The install section as written.** Lines 23 to 43 of `README.md`, copied verbatim:

    ## Install

    > **v0.1.0 is not published yet.** Until it is, build Kirsch from a clone of the repository. The release-archive route and `go install` by release tag, which [`doc/install.md`](doc/install.md) describes, need a published release, so they do not apply yet.

    Build from a clone. These are the commands from the *Build from source* section of [`doc/install.md`](doc/install.md):

    ```bash
    git clone https://github.com/djm56/kirsch.git
    cd kirsch
    bash scripts/build.sh
    ./bin/kirsch --version
    ```

    You need Go and `git`. [`doc/install.md`](doc/install.md) gives the Go version. The binary is written to `bin/kirsch` inside the checkout. The same page covers what Kirsch needs on the machine, setting up an API key, and checking the install with `kirsch doctor`.

    Developers working on Kirsch itself can run it from the clone without building a binary:

    ```bash
    go run ./cmd/kirsch
    ```

**Waits on:** m5-d1 (X2 wording), m5-d10a and m5-d10b (release tooling the README refers to), m5-d13c (the install page), and O22 (the "multiple providers" wording), from `plan/milestones/milestone-5.md:118`.

**Before any edit.** Save verbatim copies to `.claude/memory/workspace/`: `plan/spec/kirsch-plan.md` as `m5-d15-kirsch-plan.pre.md`, `plan/adr/README.md` as `m5-d15-adr-README.pre.md`, `plan/README.md` as `m5-d15-plan-README.pre.md`, `AGENTS.md` as `m5-d15-AGENTS.pre.md`, `README.md` as `m5-d15-README.pre.md` (`plan/milestones/milestone-5.md:1345`), and `plan/PROGRESS.md` as `m5-d15-PROGRESS.pre.md` (`plan/milestones/milestone-5.md:1353`, DIR-020). The copies are in the workspace, saved by mission-20261106-02 step 1, and the controls below read them. The copies taken before this step's own edits are `m5-d15-s5-*.pre.md`.

**Automated checks**

- [ ] `README.md` has an install section that links `doc/install.md` (`plan/milestones/milestone-5.md:1356`). Command: `grep -n '^## Install' README.md` and `grep -n 'doc/install.md' README.md` (rows 1 and 2). Control: `grep -c '^## Install'` on the step 1 copy prints 0 (row 3), so the check can fail. Result: PASS. The section is quoted under *What landed*.
- [ ] Every line that `grep -n '0001–0007'` prints in the named files is a site the step report records as kept, with its reason (`plan/milestones/milestone-5.md:1357`). The M5 manual files are excluded, because they contain the string themselves. The Milestone 4 manual files do not contain it, so they are not excluded.

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -rln '0001–0007' --include='*.md' . | grep -v '\.claude/memory/' | grep -v 'plan/milestones/' | grep -v -e 'plan/testing/manual/milestone-5.md' -e 'plan/testing/manual/user-testing/milestone-5-donovan.md'
  ```

  Control, on the step 1 copies (authored here). Each copy must print a line that holds the range:

  ```bash
  # authored here
  cd /Volumes/DATA/Github/kirsch && for f in kirsch-plan adr-README plan-README AGENTS; do grep -n '0001–0007' .claude/memory/workspace/m5-d15-$f.pre.md; done
  ```

  Result (row 5): each copy except `m5-d15-plan-README.pre.md` prints its line. The plan README copy prints nothing, because that file has not held the range since commit `4a50a76`. **Cannot pass as written.** The repo-wide grep also prints `plan/testing/manual/milestone-5.md`, which this box excludes but the milestone's copy of the box does not. The sites are judged under *What landed*. See *Open from m5-d15* (the *Acceptance box on "0001–0007"* item) in `plan/PROGRESS.md`.

- [ ] `grep -n 'lint-screens.py' README.md` prints a line, and `grep -n 'every script is one line' README.md` prints nothing (`plan/milestones/milestone-5.md:1358`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n 'lint-screens.py' README.md
  cd /Volumes/DATA/Github/kirsch && grep -n 'every script is one line' README.md
  ```

  Control, on the step 1 copy (authored here). The copy must hold the line, so this prints PASS:

  ```bash
  # authored here
  test "$(grep -c 'every script is one line' .claude/memory/workspace/m5-d15-README.pre.md)" = 1 && echo 'PASS control' || echo 'FAIL control'
  ```

  Result: PASS control (row 7). The live check prints no line and exits 1 (row 6), so the box passes.

- [ ] `CHANGELOG.md:16` and `plan/PROGRESS.md:231` describe the narrow-width behaviour that m5-d1 built (`plan/milestones/milestone-5.md:1359`). Command: `grep -n 'N tok' CHANGELOG.md` and `grep -n 'corrected 2026-11-06, m5-d15' plan/PROGRESS.md` (rows 8). The lines have moved: the CHANGELOG entry is line 22, not 16, and the tracker row is line 232, not 231. Result: the corrected entry says the `N tok` fallback applies at 80 columns and wider only, which is what `plan/spec/ui-spec-v0.1.md:131` says. Whether the code that m5-d1 built matches that is for the review step, which is mission-20261106-02 step 6 (`@apone`) and has not run. The box stays unticked until that review and the operator's own check.
- [ ] `README.md:47` states the configured endpoints, in the wording O22 records (`plan/milestones/milestone-5.md:1360`). Command: `grep -n '^| Providers' README.md` (row 9). The row is now line 69, and it names the *built-in* endpoints, `opencode` and `anthropic`. **Cannot pass as written:** the box asks for *configured* endpoints. See *Open from m5-d15* (the X3 item) in `plan/PROGRESS.md`.
- [ ] C21, C32, C33 and C42 are marked Resolved in `plan/PROGRESS.md`. C39 is marked Resolved with the 2026-10-23 check of `internal/config/config.go:44` cited beside it (`plan/milestones/milestone-5.md:1361`). Command: `grep -n 'Resolved 2026-11-06 (m5-d15)' plan/PROGRESS.md` and `grep -n 'checked on 2026-10-23' plan/PROGRESS.md` (row 10). Result: the marks sit on the rows for C21, C33 and C42, and on the rows for C32 and C39 for part of their scope. The C32 mark covers the settings clause only. The C39 mark covers the policy and config comments, and the grace-poll comments stay open. The C39 row cites the 2026-10-23 check. **Cannot pass as written** for C32 and C39. See *Open from m5-d15* (the C39 item) in `plan/PROGRESS.md`.
- [ ] `plan/adr/README.md:64` status wording is unchanged (`plan/milestones/milestone-5.md:1362`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -c 'Accepted (planning phase' plan/adr/README.md
  cd /Volumes/DATA/Github/kirsch && grep -c 'Accepted (planning phase' .claude/memory/workspace/m5-d15-adr-README.pre.md
  ```

  The printed counts must match. Result: 1 and 1, so PASS (row 11).

**Hands-on checks**

1. List the ADR files, to choose the range the sites must state (`plan/milestones/milestone-5.md:1349`).

   ```bash
   cd /Volumes/DATA/Github/kirsch && ls plan/adr/
   ```

   Result: the listing is in *Notes / evidence* (row 12).

2. Record the judgement for each "0001–0007" site (DIR-004). The milestone lists `plan/spec/kirsch-plan.md:392`, `plan/README.md:25` and `AGENTS.md:21`, plus `plan/adr/README.md:64`, which may stay as written. `CHANGELOG.md:21` is history and stays. Recorded under *What landed*. The spec line is now 391, not 392. `plan/README.md` no longer carries the range, so there is nothing at line 25 to judge. `CHANGELOG.md` is now at line 27, and `plan/adr/README.md` at line 65.

**Notes / evidence**

Run by `@lambert` in mission-20261106-02, step 5, on 2026-11-06, from `/Volumes/DATA/Github/kirsch`. Every command is read-only, and no key or model is used. The operator passed this section of the plain checklist on 2026-11-09.

**Claims and the command that checks each**

| Row | Claim | Command in the executed block | Result |
|---|---|---|---|
| 1 | `README.md` has an Install section. | `grep -n '^## Install' README.md` | Line 23. |
| 2 | The install section links `doc/install.md`, and the target exists. | `grep -n 'doc/install.md' README.md`; `test -f doc/install.md` | Lines 25, 27 and 36 hold the link. The file is present. |
| 3 | The step 1 copy has no Install section, so the control can fail. | `grep -c '^## Install' .claude/memory/workspace/m5-d15-README.pre.md` | 0, exit 1. |
| 4 | The range sits only at the sites in *What landed*. | `grep -rln '0001–0007'` in the milestone's repo-wide form; `grep -n '0001–0007'` on the live files | The files the repo-wide grep prints are the ones in the judgement table. |
| 5 | The step 1 copies hold the range where the table says. | The `for f` loop over the step 1 copies | Each copy except `m5-d15-plan-README.pre.md` prints its line. The plan README copy prints nothing. |
| 6 | The npm sentence names the Python script, and the old wording is gone. | `grep -n 'lint-screens.py' README.md`; `grep -n 'every script is one line' README.md` | Line 168 names the script. The old wording prints nothing and exits 1. |
| 7 | The step 1 copy held the old wording, so the control can pass. | `test "$(grep -c ...)" = 1` | PASS control. |
| 8 | The X2 corrections sit beside the originals. | `grep -n 'N tok' CHANGELOG.md`; `grep -n 'corrected 2026-11-06, m5-d15' plan/PROGRESS.md` | CHANGELOG line 22 is the m4-d9 entry, with the correction after the original. Line 12 is the new bullet that names it. Tracker line 232 is the m4-d9 row. |
| 9 | The Providers row names the built-in endpoints. | `grep -n '^| Providers' README.md` | Line 69. |
| 10 | The Resolved marks are present, and C39 cites the 2026-10-23 check. | `grep -n 'Resolved 2026-11-06 (m5-d15)' plan/PROGRESS.md`; `grep -n 'checked on 2026-10-23' plan/PROGRESS.md` | Rows at lines 129, 151, 152, 158 and 172. Row 158 holds the check. |
| 11 | The ADR status wording is unchanged. | The `grep -c 'Accepted (planning phase'` lines, on the live file and the copy | 1 and 1. |
| 12 | The ADR files on disk run from 0001 to 0012. | `ls plan/adr/` | Listed in the block. |
| 13 | The W1 cite is gone from the milestone, and the W4 sentence is gone from the milestone. The master guide had no such sentence before this step, and it now quotes the phrase only in this section. | `grep -c 'main.go:274' plan/milestones/milestone-5.md`; `grep -c 'unless O4 changes' plan/milestones/milestone-5.md` | 0; 0. |
| 14 | The W3 list at line 476 names `workspace`, `new`, `debug` and `version`. | `sed -n '476p' ... \| grep -o` | The names are printed as the list. |
| 15 | The W2 sentence is dated, not present tense. | `sed -n '5p' ... \| grep -o` | The clause reads "in that step". |
| 16 | No operator tick or verdict was in this section before this step. | The DIR-020 `awk` check | Exit 1, no lines. Nothing was overwritten. |

**Executed block** (the output is verbatim, and each `exit=` line is that command's real exit status):

```text
### DIR-020 check: operator ticks or verdicts already in section 23 of the master guide
$ awk '/^## 23 · /{f=1} /^## 24 · /{f=0} f' plan/testing/manual/milestone-5.md | grep -n -E '\[✅\]|\[❌\]|operator (passed|tested)'
exit=1

### Box: install section
$ grep -n '^## Install' README.md
23:## Install
exit=0

$ grep -n 'doc/install.md' README.md | cut -c1-80
25:> **v0.1.0 is not published yet.** Until it is, build Kirsch from a clone of 
27:Build from a clone. These are the commands from the *Build from source* secti
36:You need Go and `git`. [`doc/install.md`](doc/install.md) gives the Go versio
exit=0

$ test -f doc/install.md && echo 'doc/install.md is present'
doc/install.md is present
exit=0

$ grep -c '^## Install' .claude/memory/workspace/m5-d15-README.pre.md
0
exit=1

### Box: 0001–0007, repo-wide as the milestone box writes it
$ grep -rln '0001–0007' --include='*.md' . | grep -v '\.claude/memory/' | grep -v 'plan/milestones/'
./CHANGELOG.md
./plan/adr/README.md
./plan/spec/kirsch-plan.md
./plan/PROGRESS.md
./plan/testing/manual/milestone-5.md
exit=0

### Box: 0001–0007, live sites
$ grep -n '0001–0007' plan/spec/kirsch-plan.md plan/adr/README.md plan/README.md AGENTS.md CHANGELOG.md | cut -c1-140
plan/spec/kirsch-plan.md:391:- Design docs are already settled, not drafts: `plan/spec/architecture.md`, `plan/spec/ui-spec-v0.1.md`, `plan/
plan/adr/README.md:65:ADRs 0001–0007 were written during planning, and most still carry the status "Accepted (planning phase; no code yet)".
CHANGELOG.md:27:  and ADRs 0001–0007.
exit=0

$ grep -n '0001–0007' plan/PROGRESS.md | cut -c1-100
593:- **Acceptance box on "0001–0007" (m5-d15, cannot be satisfied as written):** the m5-d15 accepta
exit=0

### Box: 0001–0007, control on the step 1 pre-copies
$ for f in kirsch-plan adr-README plan-README AGENTS; do echo "-- $f.pre.md"; grep -n '0001–0007' .claude/memory/workspace/m5-d15-$f.pre.md | cut -c1-140; done
-- kirsch-plan.pre.md
391:- Design docs are already settled, not drafts: `plan/spec/architecture.md`, `plan/spec/ui-spec-v0.1.md`, `plan/spec/kirsch-ui-screens.md
-- adr-README.pre.md
65:ADRs 0001–0007 were written during planning, and most still carry the status "Accepted (planning phase; no code yet)". That parenthetical
-- plan-README.pre.md
-- AGENTS.pre.md
21:| Why was X decided? | [`plan/adr/`](plan/adr/) — ADRs 0001–0007 |
exit=0

### Box: lint-screens and npm sentence
$ grep -n 'lint-screens.py' README.md
168:`python3 scripts/lint-screens.py`.
exit=0

$ grep -n 'every script is one line' README.md
exit=1

$ grep -n 'every script is one line' .claude/memory/workspace/m5-d15-README.pre.md | cut -c1-140
143:`npm` is a front door, not a dependency — every script is one line of `go` or
exit=0

$ test "$(grep -c 'every script is one line' .claude/memory/workspace/m5-d15-README.pre.md)" = 1 && echo 'PASS control' || echo 'FAIL control'
PASS control
exit=0

### Box: X2 narrow width
$ grep -n 'N tok' CHANGELOG.md | cut -c1-120
12:- **The README has an Install section, and the narrow-width `N tok` claim is corrected.** The new *Install* section i
22:- **Milestone 4, m4-d9 — Token and Cost Tracking.** `usage` events → `usage.json` cumulative totals (prompt, completi
exit=0

$ grep -n 'corrected 2026-11-06, m5-d15' plan/PROGRESS.md | cut -c1-80
232:| 2026-10-17 | m4-d9 — token and cost tracking | `mission-20261017-05`. `usa
exit=0

### Box: Providers row
$ grep -n '^| Providers' README.md | cut -c1-150
69:| Providers | One wire format for v0.1 (Anthropic Messages), with the built-in endpoints `opencode` (OpenCode Go, the default) and `anthropic` |
exit=0

### Box: Resolved marks and C39 check
$ grep -n 'Resolved 2026-11-06 (m5-d15)' plan/PROGRESS.md | cut -c1-70
129:- **(routed to m3-d6) `max_turns_exceeded` error-card rendering:**
151:- **Unwired settings and hooks:** `allow_session_scoped_grants`, `
152:- **Configured command timeout ignored:** `run_command` never read
158:- **Stale code comments:** `internal/policy/policy.go:57` and `int
172:- **Deferred review suggestion — `usage.json` cost field:** the `U
exit=0

$ grep -n 'checked on 2026-10-23' plan/PROGRESS.md | cut -c1-70
158:- **Stale code comments:** `internal/policy/policy.go:57` and `int
exit=0

### Box: ADR status wording
$ grep -c 'Accepted (planning phase' plan/adr/README.md
1
exit=0

$ grep -c 'Accepted (planning phase' .claude/memory/workspace/m5-d15-adr-README.pre.md
1
exit=0

### Hands-on 1: ADR files
$ ls plan/adr/
0001-go-bubble-tea.md
0002-jsonl-session-store.md
0003-provider-abstraction.md
0004-workspace-confinement.md
0005-patch-only-editing.md
0006-session-scoped-approvals.md
0007-untrusted-tool-output.md
0008-provider-endpoints.md
0009-planning-layout-from-m6.md
0010-model-catalog-and-roles.md
0011-wire-format-adapters.md
0012-execution-seam.md
README.md
TEMPLATE.md
exit=0

### Carried items in the Landed paragraph
$ grep -c 'main.go:274' plan/milestones/milestone-5.md
0
exit=1

$ grep -c 'unless O4 changes' plan/milestones/milestone-5.md
0
exit=1

$ sed -n '476p' plan/milestones/milestone-5.md | grep -o '(`workspace`, `new`, `debug`, and `version`'
(`workspace`, `new`, `debug`, and `version`
exit=0

$ sed -n '5p' plan/testing/manual/milestone-5.md | grep -o 'goreleaser is not installed, and no install was allowed in that step'
goreleaser is not installed, and no install was allowed in that step
exit=0
```

---

## 24 · Doctor: state directory and lock state (m5-d4b)

**Context:** Terminal: Setup (instance 1), and a second terminal that shares instance 1's `$T_CFG` and `$T_STATE`. The second terminal copies the paths from the first and does not rerun the `mktemp` lines, because a fresh scratch directory has no shared lock. Binary: `"$KIRSCH_BIN"`, built by `npm run build` (the build block in §2). Instance 1 runs `"$KIRSCH_BIN" --workspace "$WORKSPACE"`, and each check runs `"$KIRSCH_BIN" doctor --workspace "$WORKSPACE"`. Working directory: `$T_CFG`. Workspace: `--workspace "$WORKSPACE"` on each kirsch launch. Key state: all key variables unset in both terminals. Config state: Setup, unchanged.

**What landed**

After the `context` line, `kirsch doctor` prints two lines, `state` and then `lock`. They are the last two lines of the report. The test `TestStateAndLockLinesFollowContext` pins that order. Both use the grammar from m5-d3a, `<status>  <check>: <detail>`. The code is `internal/doctor/state.go` (`stateRootFor`, `stateResults`, `stateResult`, `sessionDirResult`, `rootProblem`, `indexProblem`, `lockResult`, `lockLine`, `errText`), `internal/doctor/access_unix.go` and `access_other.go` (the writability checks), and `internal/session/lockprobe.go` (`ProbeLock`), with `lockstate.go` for the three outcomes and `lockprobe_other.go` for platforms that do not enforce the lock. The lines below are as the binary prints them. Each row says whether it was seen in a measured run (*Notes / evidence*) or read from the code.

What doctor does, and does not do:

- It creates no file and no folder. Writability is checked with `access(2)` (write, and search for a folder), which writes nothing. There is no write probe (`internal/doctor/access_unix.go`).
- For the lock it opens an existing `.lock` read-only and makes one non-blocking shared `flock` attempt, then releases it at once (`internal/session/lockprobe.go`). A missing `.lock` reads free. Doctor never waits and never takes the exclusive lock.

The path shown is `<root>/sessions/<hash>`. The root comes from `[session] storage_dir` with a trailing `sessions` removed. That is the same rule as the unexported `App.stateRoot` (`internal/app/app.go:851`, doc comment from `:845`), and the comment on `stateRootFor` (`internal/doctor/state.go:43–54`) names the two as a pair. Dated correction, 2026-10-30 (m5-d4b, step 4c): the earlier cites `app.go:831–855` and `state.go:24–35` were stale. A leading `~` is expanded with the home directory.

| Check | Line as printed | When | Evidence |
|---|---|---|---|
| state | `ok  state: <dir> exists and is writable` | the session folder exists and `access(2)` allows writing (`state.go:161`) | measured, run B (step 4c) |
| state | `ok  state: <dir> does not exist yet; kirsch will create it (<anc> is writable)` | the folder is missing; `<anc>` is its nearest existing ancestor, and that is writable (`state.go:180`) | measured, run A (step 4c) |
| state | `fail  state: <dir> is not writable; kirsch cannot create a new session, so any start that needs one fails until this is fixed` | the session folder exists and is not writable (`state.go:159`) | read from the code; `TestStateLineStatuses`, subtest `non-writable directory is fail` |
| state | `fail  state: <dir> is not a folder; kirsch cannot start until this is fixed` | a file is where the folder should be (`state.go:152`) | read from the code; `TestStateLineStatuses`, subtest `file where the directory should be is fail` |
| state | `fail  state: <dir> does not exist and <anc> is not writable; kirsch cannot start until this is fixed` | the folder is missing and its ancestor is not writable (`state.go:178`) | read from the code; `TestStateLineStatuses`, subtest `missing with non-writable ancestor is fail` |
| state | `fail  state: <dir> does not exist and <anc> is not a folder; kirsch cannot start until this is fixed` | the ancestor is a file (`state.go:174`) | read from the code; defensive; tested through the stat seam (`TestStateResultAncestorNotAFolder`) |
| state | `fail  state: <dir> does not exist and <anc> cannot be read: <error>; kirsch cannot start until this is fixed` | `stat` of the ancestor fails for a reason other than not-exist (`state.go:184`) | read from the code; defensive; tested through the stat seam (`TestStateResultAncestorCannotBeRead`) |
| state | `fail  state: cannot read <dir>: <error>; kirsch cannot start until this is fixed` | `stat` of the folder fails for a reason other than not-exist, for example when the storage root sits under a file (`state.go:163`) | read from the code; `TestStateResultCannotRead` |
| state | `fail  state: <root> is not writable, so index.json cannot be updated; kirsch cannot create a new session, so any start that needs one fails until this is fixed` | the state root (the folder that holds `index.json`) exists and is not writable (`state.go:215`) | measured, run C (step 4c); `TestStateRootNotWritable` |
| state | `fail  state: <root> is not a folder; kirsch cannot start until this is fixed` | the state root is a file (`state.go:208`) | read from the code; defensive; tested through the stat seam (`TestStateRootDefensiveBranches`, subtest `root is not a folder`) |
| state | `fail  state: cannot read <root>: <error>; kirsch cannot start until this is fixed` | `stat` of the state root fails for a reason other than not-exist (`state.go:206`) | read from the code; defensive; tested through the stat seam (`TestStateRootDefensiveBranches`, subtest `root cannot be read`) |
| state | `fail  state: <index> is not readable; kirsch cannot start until this is fixed` | `index.json` fails `access(2)` for read (`state.go:244`) | read from the code; `TestStateIndexNotReadable`, subtest `not readable` |
| state | `fail  state: <index> is a folder, not a file; kirsch cannot start until this is fixed` | `index.json` is a folder (`state.go:237`) | read from the code; `TestStateIndexNotReadable`, subtest `a folder` |
| state | `fail  state: cannot read <index>: <error>; kirsch cannot start until this is fixed` | `stat` of `index.json` fails for a reason other than not-exist (`state.go:235`) | read from the code; defensive; tested through the stat seam (`TestStateRootDefensiveBranches`, subtest `index cannot be read`) |
| state | `warn  state: <dir> exists; writability not checked on this platform` | a platform without the access check (`state.go:157`) | read from the code; not testable here |
| state | `warn  state: <dir> does not exist yet; writability of <anc> not checked on this platform` | the same, for a missing folder (`state.go:176`) | read from the code; not testable here |
| state | `warn  state: <root> exists; writability not checked on this platform` | the same, for the state root (`state.go:213`) | read from the code; not testable here |
| state | `warn  state: <index> exists; readability not checked on this platform` | the same, for `index.json` (`state.go:242`) | read from the code; not testable here |
| state | `warn  state: skipped: config did not load` | the settings did not load (`state.go:96`) | read from the code; `TestStateAndLockSkipped` |
| state | `warn  state: skipped: no workspace` | no workspace was found (`state.go:98`) | read from the code; `TestStateAndLockSkipped` |
| state | `fail  state: cannot work out the state folder: home directory unknown; kirsch cannot start until this is fixed` | there is no home directory (`state.go:105–107`) | read from the code; `TestStateNeedsHome` |
| lock | `ok  lock: free (no lock file yet)` | there is no `.lock` file (`state.go:298`) | measured, runs A to D (step 4c) |
| lock | `ok  lock: free` | the `.lock` file exists and the shared attempt succeeds (`state.go:276`, `:303`) | read from the code; `TestLockLineHeldThenFree`, after the release |
| lock | `warn  lock: held by another kirsch in this workspace; a new start would skip resume and session grants` | another process holds the lock (`state.go:274`) | read from the code; `TestLockLineHeldThenFree`. Live check: hands-on steps 2 and 3 |
| lock | `warn  lock: free, but <path> is not writable; the single-instance guard will be off` | the `.lock` file exists, is free, and the user cannot write it (`state.go:301`) | read from the code; `TestLockLineNotWritable` (skips as root) |
| lock | `warn  lock: could not check the lock: <error>` | the probe fails (`state.go:272`) | read from the code; `TestLockResultProbeError` |
| lock | `warn  lock: not checked on this platform` | a platform without the lock check (`state.go:278`) | read from the code; `TestLockLineNotCheckedHere` |
| lock | `warn  lock: cannot work out the state folder: home directory unknown` | there is no home directory (`state.go:108`) | read from the code; `TestStateNeedsHome` |
| lock | `warn  lock: skipped: config did not load` and `warn  lock: skipped: no workspace` | as for the state line (`state.go:96`, `:98`) | read from the code; `TestStateAndLockSkipped` |

**Status, by the O5 rule.** The state line can be `fail`, and it fails where a start would stop. The start-up path is `startupPostAttach` (`cmd/kirsch/main.go:320`), then `initSession` (the callback from `chooseSessionInit`, `cmd/kirsch/main.go:277`), then `App.StartNewSession` (`internal/app/app.go:909`), `createSessionStore` (`app.go:917`) and `Layout.CreateSession`. Two endings are used. `kirsch cannot start until this is fixed` is used where the resume path breaks too, so every start fails. `kirsch cannot create a new session, so any start that needs one fails until this is fixed` is used for a session folder or state root that is not writable (`state.go:159`, `:215`). A resume can still reopen an existing session file in such a folder, so a start that resumes may work, and the ending says only what is certain. A `fail` makes doctor exit 1. The lock line is only `ok` or `warn`: `App.AcquireLock` returns nil on every path (`internal/app/app.go:777–805`), so no lock problem stops a start. Measured on 2026-10-30: run C (state root not writable) ends `exit=1`, and runs A, B and D end `exit=0` with the key warning only. An `ok` state line does not guarantee that a start works (see *Known limits*, W2 and W3).

**What a held lock means.** A held lock means another kirsch is running in this workspace. A kirsch that starts now would run in concurrent-instance mode: no resume, no session grants, and the status "another Kirsch is running here" (`internal/app/app.go:792–798`).

**The probe and a starting kirsch.** The probe holds the shared lock for a few system calls. A kirsch that starts in that same instant is refused the exclusive lock and starts in concurrent-instance mode, as above. The window is narrow. The probe never waits, and two probes at once share the lock and do not affect each other (`ProbeLock` docblock, `internal/session/lockprobe.go`). Do not start kirsch in the same moment as a doctor run.

**Decisions for the operator.** Both points are recorded as Bishop's *O5 detail (m5-d4b)* in `plan/PROGRESS.md`, *Milestone 5 — decisions recorded*, and open in *Open from m5-d4b*. (1) O5 says doctor takes no lock: it "only checks and never changes anything (no folders created, no lock taken)" (row O5). This probe takes the shared lock for a moment, to tell held from free. The alternative is a lock line that reads "not checked". (2) The state line reads O5's "would stop kirsch starting" to include a start that needs a new session, even where a resume could still reopen the existing session file (the `cannot create a new session` ending). If the rule means every start, those two cases should be `warn`. Neither point is decided.

**Known limits, carried and not fixed in m5-d4b.**

- `stateRootFor` is a hand-kept copy of `App.stateRoot`, because doctor cannot call the unexported function. No test ties the two together. A shared helper would need an edit to `internal/app`, which is outside this deliverable (`internal/doctor/state.go:43–54`; the function is at `:55–81`).
- The `HomeDir` comment (`internal/doctor/probes.go:65–73`) says "a leading ~", but `config.ExpandHome` (`internal/config/config.go:509–517`) expands only `~` and `~/`. So `~user` is not expanded, and the comment is wider than the code. An empty `storage_dir` cannot arrive through `Run`, because the config fills in the default (`internal/config/config.go:113`), so the empty case in that comment is also wider than the code. Optional tightening.
- Windows: `GOOS=windows go build ./internal/session` and `GOOS=windows go vet ./internal/session` both succeed, so `lockprobe_other.go` compiles there. `internal/doctor` does not build for Windows today, because it imports `internal/tool`, and `internal/tool/run_command.go` uses Unix-only calls (`:184`, `:353`, `:361`, `:369`). That file is not part of m5-d4b, so `access_other.go` is not compiled there. Neither non-Unix file has a test.
- **W1 (step 4b review, carried to the operator, not fixed; read from the code, not run).** When the session folder is not writable, the lock line still reads `ok  lock: free (no lock file yet)`, because `lockResult` returns that line before it checks the folder (`internal/doctor/state.go:298`). A real start could not create `.lock` in that folder, so `App.AcquireLock` would log "lock unavailable" and carry on with the single-instance guard off. The state line fails in that case, so the lock line's `ok` is not a claim that a start works.
- **W2 (carried, not fixed; read from the code, not run).** A corrupt `index.json` stops a start, because `ReadIndex` fails inside `Layout.CreateSession` and on the resume path. Doctor checks access only and never parses the file (`internal/doctor/state.go:226–228`), so a corrupt file reads ok on the state line.
- **W3 (carried, not fixed; read from the code, not run).** The root check tests write and search, not read (`internal/doctor/state.go:200–218`). `WriteIndex` opens the state root read-only to fsync it (`internal/session/persist.go:78–82`). A root with mode `-wx` should therefore read ok in doctor and still stop a start.
- **W4 (carried, not fixed).** No test drives the resume path to the two endings. The `cannotStart` and `cannotStartNew` wording for a start is checked against a helper that mirrors the start-up path, and that helper does not include the resume path (`App.ResumeLastSession`, `internal/app/app.go:1026`). The wording on that path rests on review.

**Waited on, both done:** m5-d4a and O5 (side-effect policy and lock method), from `plan/milestones/milestone-5.md:102`. `session.Acquire` creates the session directory and `.lock` (`internal/session/lock.go:34, :42`), so the probe does not call it. It uses `session.ProbeLock` (`internal/session/lockprobe.go`) instead.

**Key-state and repository checks, first.** Authored here. It must print `CONTROL PASS`, then `0`. The first `git` line must exit non-zero, and the last must print `true`.

```bash
# authored here
keycount_control
keycount
git -C "$T_CFG" rev-parse --is-inside-work-tree; echo "exit=$?"
git -C "$WORKSPACE" rev-parse --is-inside-work-tree
```

**Automated checks**

Each box below is ticked only on the run named in it. Box 1 skips as root, because access(2) grants write to root, so run it as an ordinary user. A `--- SKIP` line in the output means the box was not tested.

Count check first (authored here). Its two numbers must be equal. A pattern that selects too few tests would otherwise pass with `ok` and no failure.

```bash
# authored here
cd /Volumes/DATA/Github/kirsch
PAT='^(TestStateLineStatuses|TestLockLineHeldThenFree|TestLockLineFreeLeavesNoLock|TestStatePathPrinted|TestStateAndLockSkipped|TestProbeLock|TestProbeLockFreeLeavesNoLock)$'
printf '%s\n' "$PAT" | grep -o 'Test[A-Za-z0-9_]*' | wc -l
go test -list "$PAT" ./internal/doctor ./internal/session | grep -c '^Test'
```

The two numbers were both `7` on the step 4c run (`.claude/memory/workspace/m20261030-01-step4c-runs.txt`, part 1). If they differ, do not tick the boxes below.

- [✅] A non-writable state directory is flagged, and the test restores the directory mode afterwards (`plan/milestones/milestone-5.md:691`). Test: `TestStateLineStatuses`, subtest `non-writable directory is fail`; the test sets mode 0o500, expects the fail line with the `cannot create a new session` ending, then restores 0o700 and checks that the directory reads ok again. Command: `go test -count=1 -v -run '^TestStateLineStatuses$' ./internal/doctor`. Output: `--- PASS: TestStateLineStatuses/non-writable_directory_is_fail`, no `--- SKIP` line, and `ok` for `./internal/doctor`. Measured in mission-20261030-01, step 4c, with exit 0 (`.claude/memory/workspace/m20261030-01-step4c-runs.txt`, part 2, scratch). The box says the folder is "flagged": it still is, now as `fail` where the step 3 wording read `warn`, so the box is met on the new wording. The step 3 run (`m20261030-01-step3-box1-run.txt`) tested the earlier wording and is superseded.
- [✅] While the test holds the lock with `session.Acquire`, doctor reports held. After the release, doctor reports free (`plan/milestones/milestone-5.md:692`). Test: `TestLockLineHeldThenFree`, one process. Command: `go test -count=1 -v -run '^TestLockLineHeldThenFree$' ./internal/doctor`. Output: `--- PASS: TestLockLineHeldThenFree` and `ok` for `./internal/doctor`.
- [✅] The held lock is still held after the doctor run, so the live instance is not disturbed (`plan/milestones/milestone-5.md:693`). Tests: `TestLockLineHeldThenFree` asserts that a second `session.Acquire` still returns `ErrLockHeld` after the doctor run. In `internal/session`, `TestProbeLock`, subtest `held lock is held, and stays held`, probes twice and then asserts the same. The free side is covered by `TestLockLineFreeLeavesNoLock` and `TestProbeLockFreeLeavesNoLock`, which assert that a free probe leaves no lock behind. Commands: `go test -count=1 -v -run '^(TestLockLineHeldThenFree|TestLockLineFreeLeavesNoLock)$' ./internal/doctor` and `go test -count=1 -v -run '^(TestProbeLock|TestProbeLockFreeLeavesNoLock)$' ./internal/session`. Output: a `--- PASS` line for each name and an `ok` line for each package.
- [✅] The probe path printed is `<root>/sessions/<hash>` for a fixture root (`plan/milestones/milestone-5.md:694`). Test: `TestStatePathPrinted`, with subtests `plain root`, `trailing sessions`, `tilde`, `tilde alone` and `unclean`. Command: `go test -count=1 -v -run '^TestStatePathPrinted$' ./internal/doctor`. Output: a `--- PASS` line for the test and each subtest, and `ok` for `./internal/doctor`.
- [✅] With config not loaded, both lines read `skipped: config did not load` (`plan/milestones/milestone-5.md:695`). Test: `TestStateAndLockSkipped`, which runs doctor with a broken settings file and asserts that both lines are warn with that detail. Command: `go test -count=1 -v -run '^TestStateAndLockSkipped$' ./internal/doctor`. Output: `--- PASS: TestStateAndLockSkipped` and `ok` for `./internal/doctor`.

**Hands-on checks**

**Second terminal attach** (authored here; run it first in the second terminal, before any launch). Replace each `<…>` with the value instance 1 printed for that variable, from `echo "$T_CFG"`, `echo "$T_STATE"` and `echo "$WORKSPACE"`. Do not run the placeholders as written.

```bash
# authored here
export T_CFG="<instance 1 value of T_CFG>"
export T_STATE="<instance 1 value of T_STATE>"
export WORKSPACE="<instance 1 value of WORKSPACE>"
export XDG_CONFIG_HOME="$T_CFG"
export XDG_STATE_HOME="$T_STATE"
unset ANTHROPIC_API_KEY KIRSCH_ANTHROPIC_API_KEY OPENCODE_API_KEY KIRSCH_OPENCODE_API_KEY KIRSCH_LIVE_MODEL
```

Then paste the whole Guard functions block from *Setup* (it defines `storage_ok`, `keycount`, `keycount_control` and `guard`), and run `guard`. It must print `GUARD PASS`.

These steps need two terminals, and both use the scratch paths from Setup. Instance 1 is the Setup terminal. Instance 2 is a second terminal that shares its paths. Do not run Setup again in the second terminal: it would make new folders, and the lock would not be shared. The lock is per workspace, so both launches use `--workspace "$WORKSPACE"`.

1. (operator) **Instance 1 starts, with no key (the Setup terminal).** Working directory: `$T_CFG`. Workspace: `--workspace "$WORKSPACE"`. Key state: unset.

   ```bash
   cd "$T_CFG"
   "$KIRSCH_BIN" --workspace "$WORKSPACE"
   ```

   Expected: kirsch's screen opens and keeps running. With no key set, it shows the no-key notice and does not stop. That is read from the code, not run: `App.CheckOnboarding` sends `NoAPIKey` to the screen (`internal/app/app.go:1318–1333`), and `keyResult` (`internal/doctor/probes.go`) records that kirsch starts. Opening the screen creates `$T_STATE/sessions/<hash>` and its `.lock` (`internal/session/lock.go:34, :42`). Leave this terminal open until step 4.

2. (operator) **Instance 2 reads the lock (the second terminal, after the attach block above).** Working directory: `$T_CFG`. Workspace: `--workspace "$WORKSPACE"`.

   ```bash
   cd "$T_CFG"
   "$KIRSCH_BIN" doctor --workspace "$WORKSPACE"; echo "exit=$?"
   ```

   Expected: a line `warn  lock: held by another kirsch in this workspace; a new start would skip resume and session grants`, a line starting `ok  state:` that names `$T_STATE/sessions/<hash>` and ends `exists and is writable`, and `exit=0`. The held wording is read from the code (`internal/doctor/state.go:274`, corrected at step 4e from the stale `:170`) and tested by `TestLockLineHeldThenFree`. This step is its live check, and it has not been run live by the agent, which cannot start a second live instance.

3. (operator) **Instance 1 is not disturbed (the second terminal, then the first).** Paste the step 2 command again in the second terminal. Expected: the same `warn  lock:` line. Then look at the first terminal: kirsch's screen is still open. Type nothing in the first terminal.

4. (operator) **Quit instance 1, then the lock reads free (the first terminal, then the second).** In the first terminal, type `/quit` and press Enter. kirsch closes. In the second terminal, paste the step 2 command again. Expected: `ok  lock: free` and `exit=0`. The `.lock` file stays on disk after kirsch closes. That is normal: the release unlocks and closes the file (`internal/session/lock.go:63–76`) and does not delete it. That is why the line reads `free` and not `free (no lock file yet)`. The release happens when the app closes (`internal/app/app.go:827–829`).

5. (operator) **The state folder is the test folder (the second terminal).**

   ```bash
   ls "$T_STATE/sessions"
   ```

   Expected: the folder name listed matches the name at the end of the path on the `ok  state:` line from step 2.

6. (operator) **Optional: an unwritable state folder (second terminal, after step 4 has closed instance 1).** This changes the mode of `$T_STATE` only, and the same block puts the mode back. Authored here; no key is used.

   ```bash
   # authored here
   cd "$T_CFG"
   M=$(stat -f %Lp "$T_STATE")
   chmod 500 "$T_STATE"
   "$KIRSCH_BIN" doctor --workspace "$WORKSPACE"; echo "exit=$?"
   chmod "$M" "$T_STATE"
   ls -ld "$T_STATE"
   ```

   Expected: a line `fail  state: <T_STATE> is not writable, so index.json cannot be updated; kirsch cannot create a new session, so any start that needs one fails until this is fixed`, then `doctor: 8 ok, 1 warn, 1 fail` (the key line is the one warning), `exit=1`, and a last line starting `drwx`. Run C in *Notes / evidence* is the same check, on a scratch layout, measured on 2026-10-30.

**Notes / evidence**

Run on 2026-10-30 by mission-20261030-01, step 4c, from the agent's Bash tool, after step 4a changed the state line. The binary is an unstamped build from the current tree: `go build -o .claude/memory/workspace/m20261030-01-step4c-kirsch ./cmd/kirsch`, and its version line reads `kirsch 0.1.0-dev`. The settings are a scratch folder, `.claude/memory/workspace/m20261030-01-step4c-area`, with `XDG_CONFIG_HOME` set to its `cfg` folder, `XDG_STATE_HOME` to its `state` folder, and `storage_dir` to `state/kr`. All four key variables and `KIRSCH_LIVE_MODEL` were unset for each command. The paths below are the agent's, and the operator's will differ. The script is `.claude/memory/workspace/m20261030-01-step4c-run.sh`. Every run, with its command line, `ls -la` listings and `exit=` line, is in `.claude/memory/workspace/m20261030-01-step4c-doctor-run.txt` (scratch, archived at the next mission start). The lines below are quoted from that file.

**Dated correction, 2026-10-30 (step 4c).** The step 3 version of this block called its run 1 "verbatim". The saved step 3 file did not hold that text: its run 1 reads `exists and is writable`, not the `does not exist yet` line quoted here, and it holds no directory listing. The step 3 binary also predates the step 4a fail wording. Those step 3 runs are not relied on below.

Run A, with the session folder absent (run A in the file; the state root `kr` did not exist, and the `ls -la` listings before and after show nothing created):

```
kirsch 0.1.0-dev
ok  git: found at /opt/local/bin/git
ok  rg: found at /opt/homebrew/bin/rg
ok  repository: /Volumes/DATA/Github/kirsch
ok  config: settings valid
warn  key: endpoint opencode: no key found (set KIRSCH_OPENCODE_API_KEY or OPENCODE_API_KEY); kirsch starts but cannot send a prompt
ok  backend: search_code uses ripgrep
ok  model: endpoint opencode, model minimax-m3: context window 128000 tokens
ok  context: AGENTS.md, 6687 bytes
ok  state: /Volumes/DATA/Github/kirsch/.claude/memory/workspace/m20261030-01-step4c-area/state/kr/sessions/70102b6770d0 does not exist yet; kirsch will create it (/Volumes/DATA/Github/kirsch/.claude/memory/workspace/m20261030-01-step4c-area/state is writable)
ok  lock: free (no lock file yet)
doctor: 9 ok, 1 warn, 0 fail
exit=0
```

Run B, with the session folder made by `mkdir -p` and writable: the state line reads `ok  state: …/state/kr/sessions/70102b6770d0 exists and is writable`, the lock line reads `ok  lock: free (no lock file yet)`, the summary reads `doctor: 9 ok, 1 warn, 0 fail`, and the run ends `exit=0`.

Run C, with the state root `kr` (the folder that holds `index.json`) made non-writable by `chmod 500`, the session folder still present and writable:

```
kirsch 0.1.0-dev
ok  git: found at /opt/local/bin/git
ok  rg: found at /opt/homebrew/bin/rg
ok  repository: /Volumes/DATA/Github/kirsch
ok  config: settings valid
warn  key: endpoint opencode: no key found (set KIRSCH_OPENCODE_API_KEY or OPENCODE_API_KEY); kirsch starts but cannot send a prompt
ok  backend: search_code uses ripgrep
ok  model: endpoint opencode, model minimax-m3: context window 128000 tokens
ok  context: AGENTS.md, 6687 bytes
fail  state: /Volumes/DATA/Github/kirsch/.claude/memory/workspace/m20261030-01-step4c-area/state/kr is not writable, so index.json cannot be updated; kirsch cannot create a new session, so any start that needs one fails until this is fixed
ok  lock: free (no lock file yet)
doctor: 8 ok, 1 warn, 1 fail
exit=1
```

The file's `ls -ld` shows `dr-x------` before and after the run. The `chmod 755` restore then shows `drwxr-xr-x`.

Run D, after the restore: the state line reads `ok  state: …/state/kr/sessions/70102b6770d0 exists and is writable`, the summary reads `doctor: 9 ok, 1 warn, 0 fail`, and the run ends `exit=0`.

Run E, outside any repository with no `--workspace`, from a folder under the workspace with `GIT_CEILING_DIRECTORIES` set to the repository root, the same settings:

```
kirsch 0.1.0-dev
ok  git: found at /opt/local/bin/git
ok  rg: found at /opt/homebrew/bin/rg
fail  repository: not in a Git repository
ok  config: settings valid; project file not read (no workspace)
warn  key: endpoint opencode: no key found (set KIRSCH_OPENCODE_API_KEY or OPENCODE_API_KEY); kirsch starts but cannot send a prompt
ok  backend: search_code uses ripgrep
ok  model: endpoint opencode, model minimax-m3: context window 128000 tokens
warn  context: skipped: no workspace
warn  state: skipped: no workspace
warn  lock: skipped: no workspace
doctor: 5 ok, 4 warn, 1 fail
exit=1
```

The automated commands in *Automated checks* were run again at step 4c, on the same date. Their output is `.claude/memory/workspace/m20261030-01-step4c-runs.txt`: part 1 is the count check, which reads `7` and `7`; parts 2 to 6 are the verbose runs of the boxes, each ending `exit=0` with no `--- SKIP` line; part 7 is the doctor and session packages, `ok`, `exit=0`. The step 3 files `m20261030-01-step3-runs.txt` and `m20261030-01-step3-box1-run.txt` are superseded and not relied on. Scratch, archived at the next mission start.

Not run here, and why:

- The held line and the free-with-lock line need a second live kirsch, which the agent cannot start. Both are read from the code and covered by `TestLockLineHeldThenFree` and `TestLockLineFreeLeavesNoLock`. Hands-on steps 2 to 4 are the live check.
- kirsch's screen with no key needs a terminal. It is read from the code (hands-on step 1).
- The Windows build of `internal/doctor` is not possible today (see *Known limits*).
- The optional hands-on step 6 is run here as run C, with the same binary and a scratch folder. The operator's copy of that step, on the test area, has not been run.
- Operator step 8 (the `chmod 500` block, section 24; step 7 before the build step was added at the start of section 24 at step 4g) was not run as written at step 4e: the agent's Bash call for it was denied. An earlier call did run, with the wrong block: it ran section 9's build block (`npm run build`), not the step 8 block. The run file `.claude/memory/workspace/m20261030-01-step4e-step7-run.txt` is that error run, and it is not evidence for this section. Its output for the step 8 block will be cited here once it exists.

---

## 25 · Repository files and golden-regeneration note (m5-d12)

**Context:** Terminal: Setup. Binary: none for the greps; the commands in `CONTRIBUTING.md`, including `npm run ci:local` (which runs `scripts/ci.sh all`), on the clean clone (operator). Working directory: /Volumes/DATA/Github/kirsch for the greps; the clean clone's location is `$T_CFG/clean`. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

- `CONTRIBUTING.md` is new. It covers the requirements, the build and check commands, how the golden screens are regenerated, the principles every change keeps, and what a pull request holds.
- `SECURITY.md` is new. It states the supported versions, the private reporting route, the threat model and the known limits. Each claim in it has a row in the claims table, `.claude/memory/missions/mission-20261103-06/security-claims.md`, which cites the code location or test behind it.
- `NOTICE` is new. It credits each direct require in `go.mod` with its licence, and says where the full licence text is published.
- The new files are `CONTRIBUTING.md`, `SECURITY.md` and `NOTICE`. The check is `ls CONTRIBUTING.md SECURITY.md NOTICE`, which lists all three.
- O21 took the branch "correct the spec", with no product change. Each of `plan/spec/kirsch-plan.md` and `plan/spec/ui-spec-v0.1.md` had one golden-regeneration sentence that named `-update`. `internal/tui/regen_test.go` defines no `-update` flag; the test is skipped unless `KIRSCH_REGEN` is set. Both sentences now name `KIRSCH_REGEN=1`. No code changed.

**Waits on:** m5-d10a and m5-d10b (build commands), O7 (`SECURITY.md` reporting channel), O8 (`NOTICE` contents), O21 (the `-update` flag versus doc), and m5-d8 (file ownership of `ui-spec-v0.1.md`, amend branch), from `plan/milestones/milestone-5.md:112`.

**Before any edit.** Save verbatim copies of `plan/spec/kirsch-plan.md` to `.claude/memory/workspace/m5-d12-kirsch-plan.pre.md` and of `plan/spec/ui-spec-v0.1.md` to `.claude/memory/workspace/m5-d12-ui-spec.pre.md` (`plan/milestones/milestone-5.md:1096`). The acceptance box uses them as its control.

**Automated checks**

- [ ] `CONTRIBUTING.md` names `KIRSCH_REGEN=1` (`plan/milestones/milestone-5.md:1100`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n 'KIRSCH_REGEN=1' CONTRIBUTING.md
  ```

- [ ] No `-update` remains in the named files (`plan/milestones/milestone-5.md:1100`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n -e '-update' CONTRIBUTING.md plan/spec/kirsch-plan.md plan/spec/ui-spec-v0.1.md
  printf 'run with -update\n' | grep -n -e '-update'
  ```

  The first command must print nothing. The second is the control: it must print `1:run with -update`, which shows the grep does match the old flag.

  Control, pinned to commit `0d6bbe4`, the commit before m5-d12. The pre-change text must hold the old wording, so each count must be at least 1:

  ```bash
  git -C /Volumes/DATA/Github/kirsch show 0d6bbe4:plan/spec/kirsch-plan.md | grep -c -e '-update'
  git -C /Volumes/DATA/Github/kirsch show 0d6bbe4:plan/spec/ui-spec-v0.1.md | grep -c -e '-update'
  ```

  Each count prints `1`.

- [ ] If O21 chose `-update` instead: `grep -n -e '-update' internal/tui/regen_test.go` prints the flag definition (`plan/milestones/milestone-5.md:1101`). This box replaces the one above and runs only on that branch. Not run: O21 took the spec-correction branch.
- [ ] `SECURITY.md` names the supported versions and the reporting channel O7 records (`plan/milestones/milestone-5.md:1102`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n 'advisories/new' SECURITY.md
  cd /Volumes/DATA/Github/kirsch && grep -n '7 days' SECURITY.md
  cd /Volumes/DATA/Github/kirsch && grep -n 'latest release' SECURITY.md
  ```

  Each command prints one line: the reporting link, the seven-day reply target, and the supported-versions sentence.

- [ ] Every claim in `SECURITY.md` traces to a code location or a test (`plan/milestones/milestone-5.md:1103`). Command: none; this is a read-through. The claims table is `.claude/memory/missions/mission-20261103-06/security-claims.md`, and each row cites the code location or test behind its claim. Review steps 2, 2b, 4 and 6 of mission-20261103-06 checked the rows against the code.
- [ ] `NOTICE` credits each direct require in `go.mod` (`plan/milestones/milestone-5.md:1104`). Its list is checked by name, one line per require, with a control that proves a missing name prints `MISSING`:

  ```bash
  cd /Volumes/DATA/Github/kirsch && awk '/^require \(/{f++} f==1&&/^\t/{print $1}' go.mod | while read m; do grep -q -F "$m" NOTICE && echo "ok $m" || echo "MISSING $m"; done
  printf 'require (\n\tgithub.com/example/planted v1.0.0\n)\n' > "$T_CFG/planted.mod" && awk '/^require \(/{f++} f==1&&/^\t/{print $1}' "$T_CFG/planted.mod" | while read m; do grep -q -F "$m" NOTICE && echo "ok $m" || echo "MISSING $m"; done
  ```

  Every line of the first command starts with `ok`. The second command is the control: it must print `MISSING github.com/example/planted`.
- [ ] Non-public references: the public files carry no `.go:` file references and no `plan/process` path. Control included:

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n -E '\.go:[0-9]|plan/process' CONTRIBUTING.md SECURITY.md NOTICE
  printf 'internal/x.go:3\n' | grep -n -E '\.go:[0-9]|plan/process'
  ```

  The first command must print nothing. The second must print `1:internal/x.go:3`.

**Hands-on checks**

1. (operator) Each command in `CONTRIBUTING.md` is run once on a clean clone and exits 0 (`plan/milestones/milestone-5.md:1099`). The milestone names `npm run check` and `scripts/ci.sh` as examples (`plan/milestones/milestone-5.md:1093`).

   The commands are the rows of the table in `CONTRIBUTING.md` (*Build, test and check*), in order, with `npm run tools` added from the same file's *Requirements* section, because the table does not list it and the section says to run it once:

   1. `npm run tools` (installs `govulncheck`, `gosec`, `staticcheck`, `golangci-lint` and `osv-scanner` at their latest versions into the Go bin directory; `scripts/install-tools.sh`)
   2. `npm run build`
   3. `npm start` (opens kirsch; leave with `/quit`; sends nothing to a model)
   4. `npm test`
   5. `npm run test:race`
   6. `npm run check`
   7. `npm run security`
   8. `npm run fuzz` (each target runs for about 30 seconds)
   9. `npm run ci:local` (replaces `govulncheck` and `gosec` in the Go bin directory with their latest versions)

   The list is checked against the table with `sed -n '/^| Task/,/^$/p' CONTRIBUTING.md`, which prints the table's rows, and every row's command is on the list above. The clean clone is `$T_CFG/clean`, made from the last commit with `git clone -q /Volumes/DATA/Github/kirsch "$T_CFG/clean"`, so the commit must come first. The operator copy's section 25, step 7, carries the same list with each command's exit code.

**Notes / evidence**

**Crew run, 2026-11-03 (mission-20261103-06).** Raw output of the automated checks in this section, run from `/Volumes/DATA/Github/kirsch` against the working tree. The operator's clean-clone run (hands-on step 1) is still owed; nothing below replaces it.

```
### KIRSCH_REGEN=1 grep (CONTRIBUTING.md)
47:   KIRSCH_REGEN=1 go test ./internal/tui -run TestGenerateGrids
exit=0

### -update grep (CONTRIBUTING.md, both spec files)
exit=1

### -update control
1:run with -update
exit=0

### -update controls at 0d6bbe4 (pre-change text)
1
1

### SECURITY.md greps
15:https://github.com/djm56/kirsch/security/advisories/new
21:We aim to reply within 7 days.
5:Only the latest release of Kirsch receives security fixes. If you run an older

### NOTICE loop (go.mod direct requires)
ok github.com/BurntSushi/toml
ok github.com/charmbracelet/bubbles
ok github.com/charmbracelet/bubbletea
ok github.com/charmbracelet/lipgloss
ok github.com/charmbracelet/x/term
ok github.com/go-git/go-git/v5
ok github.com/mattn/go-runewidth
ok github.com/muesli/cancelreader
ok github.com/muesli/termenv
ok github.com/rivo/uniseg

### NOTICE planted control
MISSING github.com/example/planted

### Non-public references (CONTRIBUTING.md SECURITY.md NOTICE)
exit=1

### Non-public control
1:internal/x.go:3
exit=0
```

---

## 26 · `/new` busy state and turn fencing: C48, C49 (optional) (m5-d17b)

**Context:** Terminal: Setup. Binary: `go test` only; no kirsch launch in this section. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a; the tests use temporary folders. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

This section is automated only. C48 and C49 were each reproduced as a test on the code before the fix, and the fixes are checked by the tests under *Automated checks*. No kirsch launch and no model call is involved, so no key is used.

**Reproductions on the code before the fix.** The blocks below are the raw output kept in `.claude/memory/workspace/m5-d17b-raw/`, copied as written. The line numbers in them (for example `new_pending_test.go:81`) are for the test files as they stood at the reproduction, not the current files. The raw files have no `exit=` line, and none is added here.

C48, from `.claude/memory/workspace/m5-d17b-raw/c48-prefix.txt`:

```
--- FAIL: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages (0.05s)
    --- FAIL: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/TurnCompleteMsg (0.01s)
        new_pending_test.go:81: after tui.TurnCompleteMsg: busy = {false ""}, want {true "new session"}
        new_pending_test.go:91: after tui.TurnCompleteMsg: Enter during /new reached Submit 1 time(s), want 0
        new_pending_test.go:94: after tui.TurnCompleteMsg: Enter during /new left 1 error card(s), want the 0 already there
    --- FAIL: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/TurnErrorMsg (0.00s)
        new_pending_test.go:81: after tui.TurnErrorMsg: busy = {false ""}, want {true "new session"}
        new_pending_test.go:91: after tui.TurnErrorMsg: Enter during /new reached Submit 1 time(s), want 0
        new_pending_test.go:94: after tui.TurnErrorMsg: Enter during /new left 2 error card(s), want the 1 already there
    --- FAIL: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/ErrorMsg (0.00s)
        new_pending_test.go:81: after tui.ErrorMsg: busy = {false ""}, want {true "new session"}
        new_pending_test.go:91: after tui.ErrorMsg: Enter during /new reached Submit 1 time(s), want 0
        new_pending_test.go:94: after tui.ErrorMsg: Enter during /new left 2 error card(s), want the 1 already there
    --- FAIL: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/ManualCompactMsg (0.00s)
        new_pending_test.go:81: after tui.ManualCompactMsg: busy = {false ""}, want {true "new session"}
        new_pending_test.go:91: after tui.ManualCompactMsg: Enter during /new reached Submit 1 time(s), want 0
        new_pending_test.go:94: after tui.ManualCompactMsg: Enter during /new left 1 error card(s), want the 0 already there
    --- FAIL: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/ToolStartedMsg (0.00s)
        new_pending_test.go:81: after tui.ToolStartedMsg: busy = {true "read_file"}, want {true "new session"}
    --- FAIL: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/AssistantTextDeltaMsg (0.00s)
        new_pending_test.go:81: after tui.AssistantTextDeltaMsg: busy = {true "thinking"}, want {true "new session"}
    --- FAIL: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/CompactingMsg (0.00s)
        new_pending_test.go:81: after tui.CompactingMsg: busy = {true "compacting"}, want {true "new session"}
FAIL
FAIL	github.com/djm56/kirsch/internal/tui	0.284s
FAIL
```

The failing subtests are `TurnCompleteMsg`, `TurnErrorMsg`, `ErrorMsg`, `ManualCompactMsg`, `ToolStartedMsg`, `AssistantTextDeltaMsg` and `CompactingMsg`. In each, busy was not `new session` after the late message. For the first four, the raw lines also show Enter during `/new` reaching `Submit`.

C49, from `.claude/memory/workspace/m5-d17b-raw/c49-prefix.txt`:

```
--- FAIL: TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns (0.53s)
    session_fence_test.go:102: 5 old-turn message(s) delivered after NewSession returned: []interface {}{tui.BudgetMsg{EstimatedTokens:4997, ContextWindow:30000}, tui.TurnCompleteMsg{}, tui.SessionUsageMsg{TokensIn:7, TokensOut:3, CacheRead:0, CacheWrite:0, CostUSD:0, CostState:"flat", ModelKnown:true}, tui.UsageMsg{InputTokens:7, OutputTokens:3, CacheReadTokens:0, CacheWriteTokens:0}, tui.AssistantTextDeltaMsg{BlockID:1, Delta:"late reply"}}
FAIL
FAIL	github.com/djm56/kirsch/internal/app	1.019s
FAIL
```

The five messages named in the failure line are `BudgetMsg`, `TurnCompleteMsg`, `SessionUsageMsg`, `UsageMsg` and `AssistantTextDeltaMsg`. They were delivered after `NewSession` returned.

**Both reproductions were run against earlier drafts of their tests.** The C49 draft is recorded in the step 2 review (`.claude/memory/missions/mission-20261105-09/PROGRESS.md`, row 2, item S3), and the C49 test was reworked in steps 2a and 2c. Neither reproduction has been run again against the final tests.

**The fixes.**

- **C48, `internal/tui/update.go`.** `Model.Update` (`:279`) wraps `update` (`:292`). While `/new` is pending, it puts `busy` back to what it was on entry, whatever `update` wrote. The rest of the message's effect (cards, notices, token figures) is kept. `NewSessionDoneMsg` (`:360`) ends the hold: it clears `newSessionPending` and busy.
- **C49, `internal/app/app.go`.** `sessionGen` (`:412`) is an atomic counter of installed sessions. `installSession` advances it at its end (`:1376`, through `bumpGeneration` at `:1914`), under `genMu`. The turn's senders, `send` (`:1946`) and `sendAsyncSequenceFenced` (`:2011`), deliver only through `deliverIfCurrent` (`:1925`). A message whose generation is no longer current is dropped, so turn messages from the old session's work do not arrive after `/new` installs the new session. `Resolve` is not fenced (see S-4 below).
- **The structural pin.** `TestTurnPathsUseOnlyFencedSenders` (`internal/app/session_fence_test.go:839`) fails when a function in package app calls an unfenced sender and is not on the `knownUnfenced` allowlist (`internal/app/session_fence_test.go:608`).

The change went through the Code-Quality Pipeline. @hicks did step 1 and fix rounds 2a and 2c. The round trigger then escalated the work to @vasquez (steps 2e and 2g), and the final review (step 2h, @apone) approved it.

**Mutant record.** The table is taken from the step reports of @hicks (steps 1, 2a and 2c) and @vasquez (steps 2e and 2g), as recorded in `.claude/memory/missions/mission-20261105-09/PROGRESS.md` (rows 1, 2a, 2c, 2e and 2g). It is from the step reports; not re-run in this step. The step 1 mutants M1 to M6 are not itemised in that record.

| Step | Mutant | Result as reported |
|------|--------|--------------------|
| 1 (@hicks) | M1 to M6 | fail |
| 1 | M7 (context generation in `RunTurn`) | survived |
| 2a (@hicks) | M7, M8, M9 and the W2 mutant | fail |
| 2a | Not individually mutated: Submit's refusal sends and its `TryLock` refusal | not mutated |
| 2c (@hicks) | M7, re-run | fail |
| 2e (@vasquez) | M-flush, M-usage, M-arg, M7, M-card, M-fail and M-presence | fail |
| 2g (@vasquez) | hold removed; a new helper; `flushStore` left unfenced | fail |

**Waits on:** m5-d8 (shared `update.go`) and m5-d17a (shared `app.go`), both done, and O13 (scheduling), recorded in `plan/PROGRESS.md`, from `plan/milestones/milestone-5.md:121`.

**Automated checks**

The commands are authored here, not quoted from the milestone. Each output is pasted as Bishop captured it on 2026-11-05, after step 3, with its `exit=` line. Your times will differ a little, and that is normal.

- [ ] A test shows that pressing Enter while `/new` is pending shows no error card (`plan/milestones/milestone-5.md:1494`). Tests: `TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages` (`internal/tui/new_pending_test.go:61`), `TestManualCompactMsgDuringNewAfterBusyClearedMidCompaction` (`:204`) and `TestNewSessionDoneEndsBusyAfterLateTurnMessages` (`:151`). Command and output below.
- [ ] A test shows that a message carrying a cancelled turn's generation is dropped (`plan/milestones/milestone-5.md:1495`). Tests: `TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns` (`internal/app/session_fence_test.go:223`), `TestEverySenderDropsAStaleGenerationAndDeliversACurrentOne` (`:442`) and `TestTurnPathsUseOnlyFencedSenders` (`:839`). Command and output below.
- [ ] The C48 test fails on the pre-fix code, and the failing output is pasted (`plan/milestones/milestone-5.md:1497`). The pasted output is under *Reproductions on the code before the fix*, above. The caveat on the earlier drafts applies there.

```bash
# authored here
go test -count=1 ./internal/tui/ -run 'TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages$|TestManualCompactMsgDuringNewAfterBusyClearedMidCompaction$|TestNewSessionDoneEndsBusyAfterLateTurnMessages$' -v; echo "exit=$?"
```

```
=== RUN   TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages
=== RUN   TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/control:_UsageMsg_touches_no_busy_state
=== RUN   TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/TurnCompleteMsg
=== RUN   TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/TurnCancelledMsg
=== RUN   TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/TurnErrorMsg
=== RUN   TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/ErrorMsg
=== RUN   TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/ManualCompactMsg_(delivered_directly,_no_/compact_run)
=== RUN   TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/ToolStartedMsg
=== RUN   TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/AssistantTextDeltaMsg
=== RUN   TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/CompactingMsg
=== RUN   TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/positive_control:_no_/new_pending,_Enter_reaches_Submit
--- PASS: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages (0.05s)
    --- PASS: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/control:_UsageMsg_touches_no_busy_state (0.01s)
    --- PASS: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/TurnCompleteMsg (0.01s)
    --- PASS: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/TurnCancelledMsg (0.01s)
    --- PASS: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/TurnErrorMsg (0.00s)
    --- PASS: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/ErrorMsg (0.00s)
    --- PASS: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/ManualCompactMsg_(delivered_directly,_no_/compact_run) (0.00s)
    --- PASS: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/ToolStartedMsg (0.00s)
    --- PASS: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/AssistantTextDeltaMsg (0.00s)
    --- PASS: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/CompactingMsg (0.00s)
    --- PASS: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/positive_control:_no_/new_pending,_Enter_reaches_Submit (0.00s)
=== RUN   TestNewSessionDoneEndsBusyAfterLateTurnMessages
=== RUN   TestNewSessionDoneEndsBusyAfterLateTurnMessages/success
=== RUN   TestNewSessionDoneEndsBusyAfterLateTurnMessages/failure
--- PASS: TestNewSessionDoneEndsBusyAfterLateTurnMessages (0.01s)
    --- PASS: TestNewSessionDoneEndsBusyAfterLateTurnMessages/success (0.00s)
    --- PASS: TestNewSessionDoneEndsBusyAfterLateTurnMessages/failure (0.00s)
=== RUN   TestManualCompactMsgDuringNewAfterBusyClearedMidCompaction
--- PASS: TestManualCompactMsgDuringNewAfterBusyClearedMidCompaction (0.00s)
PASS
ok  	github.com/djm56/kirsch/internal/tui	0.689s
exit=0
```

```bash
# authored here
go test -count=1 ./internal/app/ -run 'TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns$|TestEverySenderDropsAStaleGenerationAndDeliversACurrentOne$|TestTurnPathsUseOnlyFencedSenders$' -v; echo "exit=$?"
```

```
=== RUN   TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns
=== RUN   TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns/BudgetMsg
=== RUN   TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns/AssistantTextDeltaMsg
=== RUN   TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns/final_turn_message
=== RUN   TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns/Submit_refusal
--- PASS: TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns (1.66s)
    --- PASS: TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns/BudgetMsg (0.38s)
    --- PASS: TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns/AssistantTextDeltaMsg (0.52s)
    --- PASS: TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns/final_turn_message (0.47s)
    --- PASS: TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns/Submit_refusal (0.29s)
=== RUN   TestEverySenderDropsAStaleGenerationAndDeliversACurrentOne
--- PASS: TestEverySenderDropsAStaleGenerationAndDeliversACurrentOne (0.22s)
=== RUN   TestTurnPathsUseOnlyFencedSenders
=== RUN   TestTurnPathsUseOnlyFencedSenders/package_app
=== RUN   TestTurnPathsUseOnlyFencedSenders/control
--- PASS: TestTurnPathsUseOnlyFencedSenders (0.01s)
    --- PASS: TestTurnPathsUseOnlyFencedSenders/package_app (0.01s)
    --- PASS: TestTurnPathsUseOnlyFencedSenders/control (0.00s)
PASS
ok  	github.com/djm56/kirsch/internal/app	2.406s
exit=0
```

**Notes / evidence**

The boxes above are unticked. The operator ticks them.

Carried items. None of these is fixed by m5-d17b. Each is recorded in `plan/PROGRESS.md` under *Open from m5-d17b*.

- **W-A, for the operator to schedule.** The `ErrorMsg` case clears busy unconditionally (`internal/tui/update.go:379`). During `/compact` that lets typed text reach `Submit`, and the app's compaction refusal can then show as a stray error card. This is the C48 class, in `/compact`. The clearing is shown by the setup assertion of `TestManualCompactMsgDuringNewAfterBusyClearedMidCompaction` (`internal/tui/new_pending_test.go:218`). The Enter path (`internal/tui/update.go:807–810`) and the `/compact` guard (`internal/tui/update.go:1628`) are read from the code, not seen in a run. It needs its own step.
- **W-B.** The comment at `internal/tui/update.go:1635` ("busy stays active until that message arrives") is no longer accurate, because W-A can clear busy first. It is not changed here.
- **S-1.** `internal/app/session_fence_test.go:30–32` has an overlong line in the file header. It needs reflowing, and is not changed here.
- **S-2.** The control subtest of `TestTurnPathsUseOnlyFencedSenders` (`internal/app/session_fence_test.go:851`) runs against the live `knownUnfenced` and `fencedCallers` tables (`:608`, `:626`). The generation argument is matched by its spelling (`gen`, `currentGen()`, `turnGen()`), not by its value.
- **S-3.** The wording of the `sessionGen` comment at `internal/app/app.go:399–400`. It is not changed here.
- **S-4.** `Resolve` (`internal/app/app.go:2660`) sends a `NoticeMsg` (`:2713`) and an `ApprovalResolvedMsg` (`:2720`) through `sendAsyncSequence` (`:2725`), without the fence. @apone's step 2h review called this effectively unreachable after `/new` (in its report text; the review record on disk does not carry that sentence). Reachability after `/new` was not run. But `resetForNewSession` (`internal/tui/update.go:1227`) resets the transcript and not `approvalCards` (`internal/tui/model.go:186`). The comment at `internal/tui/update.go:1387–1396` calls the `!ok` branch unreachable, which is no longer true once the transcript is reset.
- **The C48 and C49 drafts.** Both pre-fix reproductions were run against earlier drafts of their tests (see *Reproductions on the code before the fix*). Neither has been run again against the final tests.

---

## 27 · `doc/troubleshooting.md` and `doc/README.md` (m5-d14b)

**Context:** Terminal: Setup. Binary: `$KIRSCH_BIN` for the reproductions. Working directory: `$WORKSPACE`, with `--workspace "$WORKSPACE"` on each launch. Workspace: `$WORKSPACE`. Key state: all key variables unset, except a deliberately invalid dummy key for the 401 reproduction, and key entered for this live step for the 403 reproduction, if the operator decides that case needs one. Config state: Setup, unchanged.

**What landed**

`doc/troubleshooting.md` explains each line `kirsch doctor` prints and the messages for the common failures. `doc/README.md` lists the docs pages and their state. The doctor line map is under the doctor-line check below.

Product finding, not a page item: the *Not a Git repository* onboarding notice is never shown. Nothing in production code sets `NotGitRepo` to true; only tests do. The guide does not quote that notice.

**Waits on:** m5-d3a, m5-d3b, m5-d4a and m5-d4b (the doctor lines the page explains), m5-d13a and m5-d13c (the pages the index names), and O6, from `plan/milestones/milestone-5.md:117`.

**Key-state check before the live step.** Authored here. It must print `CONTROL PASS`, then `0` before any key is entered.

```bash
# authored here
keycount_control
keycount
```

**Automated checks**

- [ ] `grep -n -i -w -E 'milestones?|ADRs?' doc/troubleshooting.md doc/README.md` prints nothing. The milestone's own pattern, `'milestone|ADR'`, misses plurals, and the control below shows the new pattern matching them. Control: the printf line prints `1:an ADR`, `2:ADRs` and `3:milestones`, and not `headroom`.

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n -i -w -E 'milestones?|ADRs?' doc/troubleshooting.md doc/README.md
  printf 'an ADR\nADRs\nmilestones\nheadroom\n' | grep -n -i -w -E 'milestones?|ADRs?'
  ```

- [ ] `grep -n -E '\.go\b|internal/|cmd/kirsch/|Sources?:' doc/troubleshooting.md doc/README.md` prints nothing. Control: the printf line prints back `1:cmd/kirsch/main.go:85`, `2:internal/doctor/probes.go:138` and `3:*Source:* x`.

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n -E '\.go\b|internal/|cmd/kirsch/|Sources?:' doc/troubleshooting.md doc/README.md
  printf 'cmd/kirsch/main.go:85\ninternal/doctor/probes.go:138\n*Source:* x\n' | grep -n -E '\.go\b|internal/|cmd/kirsch/|Sources?:'
  ```

- [ ] Every check that doctor prints has a section in `doc/troubleshooting.md`. The first command prints `section <name>` for each check name the code declares, and `NO SECTION <name>` for any name the guide has no heading for. Every line must read `section`, and the command must print at least one line. The second command is the control: it runs the same loop against `/dev/null`, so it must print `NO SECTION <name>` for every name the first command printed. The lines under each section are matched to the map below by reading, not by these commands.

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -h -E '^\s+check[A-Za-z]+\s+= "' internal/doctor/probes.go internal/doctor/state.go | sed -E 's/.*"(.*)".*/\1/' | while read c; do grep -q "^## $c\$" doc/troubleshooting.md && echo "section $c" || echo "NO SECTION $c"; done
  cd /Volumes/DATA/Github/kirsch && grep -h -E '^\s+check[A-Za-z]+\s+= "' internal/doctor/probes.go internal/doctor/state.go | sed -E 's/.*"(.*)".*/\1/' | while read c; do grep -q "^## $c\$" /dev/null && echo "section $c" || echo "NO SECTION $c"; done
  ```

  Doctor line map. Paths are under `internal/`. Text in `<...>` is filled in at run time.

  | Line as printed | Source | Page section |
  |---|---|---|
  | `ok  git: found at <path>` | `doctor/probes.go:138` | git |
  | `fail  git: not found on PATH; kirsch needs git to find the repository` | `doctor/probes.go:142` | git |
  | `warn  git: not found on PATH; kirsch starts with --workspace but the Git features are off, ...` | `doctor/probes.go:149` | git |
  | `ok  rg: found at <path>` | `doctor/probes.go:156` | rg |
  | `warn  rg: not found on PATH; search_code uses the pure-Go fallback` | `doctor/probes.go:158` | rg, Missing ripgrep |
  | `ok  repository: <root>` | `doctor/probes.go:364` | repository |
  | `fail  repository: not in a Git repository` | `doctor/probes.go:352`, detail `doctor/probes.go:50` | repository, Not in a repository |
  | `fail  repository: <err>` (`--workspace <path>: <error>`, `... is not a directory`, `resolve workspace root <root>: <error>`) | `doctor/probes.go:354`; `workspace/workspace.go:82`, `:85`, `:108` | repository |
  | `warn  repository: <root> is not a Git repository; kirsch starts, but ...` | `doctor/probes.go:359` | repository |
  | `warn  config: <file>: unknown key "<key>" (ignored)` | `doctor/probes.go:385`; `config/config.go:156` | config |
  | `warn  config: <file>: key "<key>" ignored (a project config may set only [context].project_files)` | `config/config.go:153` | config |
  | `warn  config: <file>: context.project_files must be a list of strings; ignored` | `config/config.go:424` | config |
  | `fail  config: read <path>: <error>` / `fail  config: parse <path>: <error>` | `config/config.go:315`, `:364` (read); `:323`, `:333`, `:372` (parse); emitted at `doctor/probes.go:389` | config |
  | `fail  config: <file>: key "<key>" looks like a credential ...` | `config/config.go:485` | config |
  | `fail  config: invalid settings (<consulted>): <error>` | `doctor/probes.go:394`; `no config files` at `doctor/probes.go:420` | config |
  | `ok  config: settings valid` / `ok  config: settings valid; project file not read (no workspace)` | `doctor/probes.go:407`, `:402` | config |
  | `ok  key: endpoint <endpoint>: key found in <variable>` | `doctor/probes.go:242` | key |
  | `warn  key: endpoint <endpoint>: no key found (set ...); kirsch starts but cannot send a prompt` | `doctor/probes.go:236` | key, No API key |
  | `warn  key: endpoint <endpoint>: api_key_env does not look like an environment variable name ...` | `doctor/probes.go:229` | key |
  | `warn  key: skipped: config did not load` | `doctor/probes.go:198` | key |
  | `ok  backend: search_code uses ripgrep` | `doctor/probes.go:254` | backend |
  | `warn  backend: search_code uses pure-go (rg is not on PATH)` | `doctor/probes.go:256` | backend, Missing ripgrep |
  | `ok  model: endpoint <endpoint>, model <id>: context window <tokens> tokens` | `doctor/probes.go:272` | model |
  | `warn  model: endpoint <endpoint>, model <model>: not in the table; conservative fallback ...` | `doctor/probes.go:266` | model |
  | `warn  model: skipped: config did not load` | `doctor/probes.go:200` | model |
  | `ok  context: <file>, <size> bytes` / `ok  context: none; looked for <files>` / `ok  context: none; no project files configured` | `doctor/probes.go:313` | context |
  | `warn  context: <file>, <size> bytes; project context candidate ...` / `... a candidate could not be read: ...` | `doctor/probes.go:313`; `agent/prompt/loader.go:62`, `:71`, `:74`, `:86` | context |
  | `warn  context: skipped: no workspace` | `doctor/probes.go:289` | context |
  | `warn  context: skipped: config did not load` | `doctor/probes.go:201` | context |
  | `fail  state: cannot work out the state folder: home directory unknown; ...` | `doctor/state.go:106`, `:57` | state |
  | `ok`, `warn` or `fail  state: <dir> ...` (session folder, `does not exist yet`, `is not a folder`, `is not writable`, `cannot read`) | `doctor/state.go:152` to `:184` | state |
  | `warn  lock: cannot work out the state folder: home directory unknown` | `doctor/state.go:108` | state (the lock section does not quote it) |
  | `ok`, `warn` or `fail  state: <root> ...` and `state: <index> ...` (storage root and `index.json`) | `doctor/state.go:206` to `:244` | state |
  | `warn  state: skipped: config did not load` / `skipped: no workspace` | `doctor/state.go:96`, `:98` | state |
  | `ok  lock: free` / `ok  lock: free (no lock file yet)` | `doctor/state.go:276`, `:298` | lock |
  | `warn  lock: held by another kirsch in this workspace; ...` | `doctor/state.go:274` | lock, Lock held |
  | `warn  lock: could not check the lock: <error>` / `not checked on this platform` | `doctor/state.go:272`, `:278` | lock |
  | `warn  lock: free, but <lockPath> is not writable; ...` | `doctor/state.go:301` | lock |
  | `warn  lock: skipped: ...` | `doctor/state.go:96`, `:98` | lock |

  Examples of the program's own messages the guide quotes (not the whole set): `kirsch: Kirsch runs inside a Git repository.` at `workspace/workspace.go:72` (*Not in a repository*), and `no API key for endpoint "<endpoint>" (set ...)` at `app/app.go:1389` (*No API key*).

  The map is kept by hand and its line numbers move with the code. Re-derive the emitting sites each run:

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n 'Result{' internal/doctor/probes.go internal/doctor/state.go | grep -v 'Result{}' | grep -v -E '\[\]Result\{$|\[\]Result\{[a-z]+Result\('
  ```

  The last filter drops the lines that only open a list of results; the results in those lists are on their own lines. Every printed line must fall under a Source cell above (a range counts). A line no row covers is a gap in the map, and the guide needs a look for it.

- [ ] The status lines in `doc/README.md` match the files in `doc/`. Each `doc/*.md` file other than `README.md` has a row, each row names an existing file, and no row reads anything but `Written`.

  ```bash
  cd /Volumes/DATA/Github/kirsch/doc && for f in *.md; do [ "$f" = README.md ] && continue; grep -q "^| \[\`$f\`\]" README.md && echo "row $f" || echo "NO ROW $f"; done
  for f in $(grep -o '`[a-z-]*\.md`' README.md | tr -d '`' | sort -u); do test -f "$f" && echo "file $f" || echo "MISSING $f"; done
  cd /Volumes/DATA/Github/kirsch && sed -n '/^| Document/,/^$/p' doc/README.md | grep '^| \[' | grep -v '| Written |'
  cd /Volumes/DATA/Github/kirsch && sed -n '/^| Document/,/^$/p' doc/README.md | grep '^| \[' | grep -v '| NoSuchState |'
  ```

  The third command prints nothing when every row reads `Written`. The line after it is the control: no row reads `NoSuchState`, so it must print every table row. If it prints nothing, the `sed` range did not find the table and the empty result above proves nothing.

**Hands-on checks**

1. (operator) Each failure the troubleshooting page covers is reproduced, and the advice fixes it (`plan/milestones/milestone-5.md:1309`).

   The 401 reproduction uses a deliberately invalid dummy key, never a real one. Authored here: `export OPENCODE_API_KEY=dummy-invalid-key`, then run the reproduction. It still contacts the provider, which refuses the key. The 403 reproduction needs a key the provider refuses for the model. It takes a real key only if the operator decides it must, entered with the live-key block in *Setup*. Only a step that genuinely needs the model takes a real key. The no-key case needs no key.

   The failures are the subsections under *Common failures* in the guide (`grep -n '^### ' doc/troubleshooting.md` lists them). The operator copy reproduces each one as follows:

   - *No API key*: step 4 of the operator copy (no TUI, the crew runs it), and step 7 (TUI, optional). Do not reproduce this case with a key set.
   - *Not in a repository*: step 5 (no TUI, the crew runs it).
   - *Missing ripgrep*: step 6 (no TUI, the crew runs it).
   - *401 or 403: rejected the API key*: step 8 (TUI, optional). The 401 uses the dummy key above, and the provider refuses it, usually with HTTP 401; the operator copy also accepts HTTP 403 for that key. The 403 needs a key the account cannot use with the model. It stays operator-owned and optional, and the operator copy does not carry it.
   - *Lock held*: section 24 of the operator copy, which starts a second kirsch. This section does not repeat it.

   The crew runs the non-TUI steps only. Steps 7 and 8 need a terminal window and stay with the operator.

   - [ ] (operator, not yet run) Re-check the first claim of the *No API key* paragraph against the screen. In a new Terminal window with no key set, first set this window to 80 columns wide and 24 rows tall. In Terminal, the Window settings for the profile hold the Columns and Rows fields. Then start a fresh session in a new temporary folder and a new temporary home: `FAKE_HOME="$(mktemp -d)" && cd "$(mktemp -d)" && git init -q . && env -i HOME="$FAKE_HOME" PATH="$PATH" TERM=xterm-256color /Volumes/DATA/Github/kirsch/bin/kirsch`. The onboarding should be on screen, with the `no previous session` notice beneath it. Compare with the 80 by 24 sample in section 18 (`plan/testing/manual/milestone-5.md:2302`, `:2308`); under `env -i` the dash in that notice is a plain `-`. A much shorter window shows the ordinary transcript instead of the onboarding. That is expected, not a failure. Quit with two `Ctrl+C`. Send no message. The paragraph's second claim, that a resumed session with messages does not show the onboarding, is not tested by this item, because a run with no key cannot keep a message (`internal/app/app.go:1385–1391` returns with an `auth` card before any session is stored). That half is approximated by the unit test `TestFirstNonNoticeItemEndsOnboarding` in `internal/tui/notice_onboarding_test.go:67`, which shows that a user item appended to a model in memory after a notice ends the onboarding. It does not load a saved session. This item is unticked because no run has been made yet.

   **Key-state check, last** (authored here). Remove the dummy or real key. The unset line clears every key variable. The count after the unset must print `CONTROL PASS`, then `0`.

   ```bash
   # authored here
   unset ANTHROPIC_API_KEY KIRSCH_ANTHROPIC_API_KEY OPENCODE_API_KEY KIRSCH_OPENCODE_API_KEY
   keycount_control
   keycount
   ```

**Notes / evidence**

_Empty. Fill at run._

---

## 28 · Resume totals and compaction % (optional) (m5-d18)

**Context:** Terminal: Setup. Binary: `go test` only; no kirsch launch in this section. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a; the tests use temporary folders. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

This section is automated only. The changes landed under m5-d18. On resume, the status bar's session totals are summed from the session log's `usage` events, not read from `usage.json`, which sits at the kirsch data root and holds the totals of every session that uses the same data root. After a manual `/compact`, `/status` shows the compaction % of the compacted conversation. Both are checked by the tests under *Automated checks*. No kirsch launch and no model call is involved, so no key is used.

**Reproductions on the code before the fix.** The block below is copied from the step 1 report of mission-20261105-10 (@hicks). It is not re-run here. The line numbers in it are for the test files as they stood at the reproduction, not the current files.

```
--- FAIL: TestResumeLastSessionSendsLogTotalsToStatusBar (5.07s)
    resume_usage_test.go:122: got 0 SessionUsageMsg after 5s, want 1: []tui.SessionUsageMsg(nil)
--- FAIL: TestResumeLastSessionSendsZeroTotalsWithoutUsageEvents (5.05s)
    resume_usage_test.go:162: got 0 SessionUsageMsg after 5s, want 1: []tui.SessionUsageMsg(nil)
--- FAIL: TestResumeLastSessionTornUsageLine (5.05s)
    resume_usage_test.go:189: got 0 SessionUsageMsg after 5s, want 1: []tui.SessionUsageMsg(nil)
--- FAIL: TestManualCompactUpdatesStatusCompactionPercent (0.10s)
    resume_usage_test.go:223: compaction % before /compact: 60
    resume_usage_test.go:239: compaction % after /compact: 60 (estimate 9727 tokens)
    resume_usage_test.go:241: compaction % after /compact is still 60: /status is stale
```

**Waits on:** m5-d17a (file ownership only; both own `internal/app/app.go`), and O13 (scheduling), from `plan/milestones/milestone-5.md:122`.

**Automated checks**

Each check below has one command, run as written, with its raw output pasted under it. Each output ends with its `exit=` line. The commands were run on 2026-11-05 by mission-20261105-10 (step 3). Your times will differ a little, and that is normal.

- [ ] A test shows that after resume the status bar shows the session's totals before any new usage (`plan/milestones/milestone-5.md`, m5-d18 *Acceptance*, first box). Command and output below, first block. The test that proves it is `TestResumeLastSessionSendsLogTotalsToStatusBar`.
- [ ] A test shows that the totals equal the sum of the session's `usage` events in a fixture log (`plan/milestones/milestone-5.md`, m5-d18 *Acceptance*, second box). Command and output below, second block. The test that proves it is `TestLoadUsageTotalsSumsUsageEvents`.
- [ ] A test shows the compaction % before `/compact` is 60%, and after it is the value `Compact` returns, which differs from 60%. Both values are pasted (`plan/milestones/milestone-5.md`, m5-d18 *Acceptance*, third box). Command and output below, third block. The test that proves it is `TestManualCompactUpdatesStatusCompactionPercent`.

```bash
# authored here
go test -count=1 -v -run 'TestResumeLastSession|TestResumeCompletesPartialTurn' ./internal/app/; echo exit=$?
```

```
=== RUN   TestResumeCompletesPartialTurn
--- PASS: TestResumeCompletesPartialTurn (0.10s)
=== RUN   TestResumeLastSessionNoSession
--- PASS: TestResumeLastSessionNoSession (0.01s)
=== RUN   TestResumeLastSessionSendsLogTotalsToStatusBar
--- PASS: TestResumeLastSessionSendsLogTotalsToStatusBar (0.17s)
=== RUN   TestResumeLastSessionSendsZeroTotalsWithoutUsageEvents
--- PASS: TestResumeLastSessionSendsZeroTotalsWithoutUsageEvents (0.15s)
=== RUN   TestResumeLastSessionTornUsageLine
--- PASS: TestResumeLastSessionTornUsageLine (0.16s)
=== RUN   TestResumeLastSessionAfterCloseIsRefused
--- PASS: TestResumeLastSessionAfterCloseIsRefused (0.08s)
PASS
ok  	github.com/djm56/kirsch/internal/app	1.189s
exit=0
```

```bash
# authored here
go test -count=1 -v -run 'TestLoadUsageTotals' ./internal/session/; echo exit=$?
```

```
=== RUN   TestLoadUsageTotalsSumsUsageEvents
--- PASS: TestLoadUsageTotalsSumsUsageEvents (0.01s)
=== RUN   TestLoadUsageTotalsNoUsageEvents
--- PASS: TestLoadUsageTotalsNoUsageEvents (0.01s)
=== RUN   TestLoadUsageTotalsStopsAtTornLine
--- PASS: TestLoadUsageTotalsStopsAtTornLine (0.01s)
=== RUN   TestLoadUsageTotalsStopsAtUndecodableUsageEvent
--- PASS: TestLoadUsageTotalsStopsAtUndecodableUsageEvent (0.01s)
=== RUN   TestLoadUsageTotalsMissingFile
--- PASS: TestLoadUsageTotalsMissingFile (0.00s)
PASS
ok  	github.com/djm56/kirsch/internal/session	0.638s
exit=0
```

```bash
# authored here
go test -count=1 -v -run 'TestManualCompactUpdatesStatusCompactionPercent' ./internal/app/; echo exit=$?
```

```
=== RUN   TestManualCompactUpdatesStatusCompactionPercent
    resume_usage_test.go:224: compaction % before /compact: 60
    resume_usage_test.go:240: compaction % after /compact: 37 (estimate 9727 tokens)
--- PASS: TestManualCompactUpdatesStatusCompactionPercent (0.12s)
PASS
ok  	github.com/djm56/kirsch/internal/app	0.384s
exit=0
```

The third block is the compaction check. Its before value is 60, from the line `compaction % before /compact: 60`. Its after value is 37, from the line `compaction % after /compact: 37 (estimate 9727 tokens)`. The after value differs from 60, as the milestone requires. The pre-fix run in the reproduction above showed 60 after `/compact`, which is why the check failed before the change.

**Hands-on checks**

Not authored here. The hands-on run in the *Manual test hook* of m5-d18 (`plan/milestones/milestone-5.md`) needs a model call, so it is not written into this section; it is optional.

**Notes / evidence**

The automated outputs above were captured on 2026-11-05 by mission-20261105-10 (step 3). The operator passed this section of the plain checklist on 2026-11-09.

---

## 29 · Record the operator's decisions (m5-d20)

**Context:** Terminal: none. The operator records decisions in `plan/PROGRESS.md`, and no command runs. Binary: none. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: none needed. Config state: none.

**What landed**

> **TO EXPAND when m5-d20 is recorded:** each decision with the option chosen, in `plan/PROGRESS.md`. This gate is operator-owned.

**Waits on:** none. Records O12, O15 and O16, and the decision list below (`plan/milestones/milestone-5.md:125`). The C29 item of O15 must be recorded before m5-d19b runs in wave 2.

**Automated checks**

- None. The acceptance is a record in `plan/PROGRESS.md`, and no command checks it. The operator confirms each row against the decision list (`plan/milestones/milestone-5.md:1628–1629`).

**Hands-on checks**

1. (operator) Record one decision per ID in `plan/PROGRESS.md`, with the option chosen. The IDs to record are the ones named in the decision list at `plan/milestones/milestone-5.md:1609–1611`. Read that range rather than a copy of it here, and record the open O-decisions named there (`plan/milestones/milestone-5.md:1610–1611`).
2. (operator) If C24 is clip, record the byte bound with the choice. m5-d21c quotes that bound, and no other step defines it (`plan/milestones/milestone-5.md:1625`).
3. (operator) Record each outcome before the m5-d21 deliverable that depends on it runs (`plan/milestones/milestone-5.md:1624`).

**Notes / evidence**

_Empty. Fill at run. Record the decision date, not an approver credential._

---

## 30 · Model table (conditional) (m5-d21a)

**Context:** Terminal: Setup. Binary: `go test` only; no kirsch launch in this section. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

This section is automated only. The changes landed under m5-d21a (mission-20261105-11, 2026-11-05). One figure changed: `claude-sonnet-5-5` `CacheReadPerMTok` went from 0.2 to 0.1 in `internal/provider/models.go`. The line now reads `CacheReadPerMTok:  0.1,`.

The decisions behind it are X5 and C9, quoted from `plan/PROGRESS.md` (*Milestone 5 — decisions recorded*, 2026-11-03). X5: "Operator accepted Bishop's recommendation ("Accept all (Recommended)", m5-d20 decision sheet): Re-read the model-table figures against the vendors' published pages, and update the table and its date (m5-d21a)." C9: "Operator accepted Bishop's recommendation ("Accept all (Recommended)", m5-d20 decision sheet): Check the cache-write prices against the vendors' published price pages and update the model table (m5-d21a)."

The comments in `internal/provider/models.go` now name the pricing page and the overview page, and the date 2026-11-05. The 5-minute cache-write values are the published rates, and the cache-read comment names `claude-sonnet-5-5` at 5%. The other per-token figures and the context windows matched the pages, so they did not change. Haiku 4.5's 200K and 64K were not re-read. No kirsch launch and no model call is involved, so no key is used.

**Waits on:** m5-d20 (the X5 and C9 decisions), from `plan/milestones/milestone-5.md:126`. Runs only if the operator changed the figures or dates (`plan/milestones/milestone-5.md:1661`).

**Automated checks**

Checks one and two each have one command, run as written, with its raw output pasted under it, and each output ends with its `exit=` line. Check three has no command and no pasted output: it is a process check, described under its box. The commands were run on 2026-11-05 by @lambert in mission-20261105-11 (step 3). Your times will differ a little, and that is normal.

- [ ] The table matches the recorded decision (`plan/milestones/milestone-5.md`, m5-d21a *Acceptance*, first box). Command and output below, first block. The test that proves the figures is `TestPublishedPerTokenFigures`. The other names in that `-run` pattern run the cost arithmetic against the same table.
- [ ] `go test ./internal/provider/` passes (`plan/milestones/milestone-5.md`, m5-d21a *Acceptance*, first box). Command and output below, second block.
- [ ] The review step is recorded in `plan/PROGRESS.md` (`plan/milestones/milestone-5.md`, m5-d21a *Acceptance*, second box). No command checks it. This is a process check: the review record is in `.claude/memory/missions/mission-20261105-11/PROGRESS.md`, which is the record of every step and review. `plan/PROGRESS.md` row 300 records only the status and the mission id, not the review steps. Whether that meets this box is the operator's call.

```bash
# authored here
go test -count=1 -v -run 'TestPublishedPerTokenFigures|TestCostPerTokenModel|TestCostAllPerTokenModels' ./internal/provider/; echo exit=$?
```

```
=== RUN   TestCostPerTokenModel
--- PASS: TestCostPerTokenModel (0.00s)
=== RUN   TestCostAllPerTokenModels
=== RUN   TestCostAllPerTokenModels/claude-sonnet-5-5
=== RUN   TestCostAllPerTokenModels/claude-opus-5-5
=== RUN   TestCostAllPerTokenModels/claude-fable-5-1
=== RUN   TestCostAllPerTokenModels/claude-haiku-4-5-20251001
--- PASS: TestCostAllPerTokenModels (0.00s)
    --- PASS: TestCostAllPerTokenModels/claude-sonnet-5-5 (0.00s)
    --- PASS: TestCostAllPerTokenModels/claude-opus-5-5 (0.00s)
    --- PASS: TestCostAllPerTokenModels/claude-fable-5-1 (0.00s)
    --- PASS: TestCostAllPerTokenModels/claude-haiku-4-5-20251001 (0.00s)
=== RUN   TestPublishedPerTokenFigures
=== RUN   TestPublishedPerTokenFigures/claude-sonnet-5-5
=== RUN   TestPublishedPerTokenFigures/claude-opus-5-5
=== RUN   TestPublishedPerTokenFigures/claude-fable-5-1
=== RUN   TestPublishedPerTokenFigures/claude-haiku-4-5-20251001
--- PASS: TestPublishedPerTokenFigures (0.00s)
    --- PASS: TestPublishedPerTokenFigures/claude-sonnet-5-5 (0.00s)
    --- PASS: TestPublishedPerTokenFigures/claude-opus-5-5 (0.00s)
    --- PASS: TestPublishedPerTokenFigures/claude-fable-5-1 (0.00s)
    --- PASS: TestPublishedPerTokenFigures/claude-haiku-4-5-20251001 (0.00s)
PASS
ok  	github.com/djm56/kirsch/internal/provider	0.640s
exit=0
```

```bash
# authored here
cd /Volumes/DATA/Github/kirsch && go test -count=1 ./internal/provider/; echo exit=$?
```

```
ok  	github.com/djm56/kirsch/internal/provider	0.408s
exit=0
```

**Hands-on checks**

Not authored here. No kirsch launch and no model call is involved, so this section has no hands-on checks.

**Notes / evidence**

The automated outputs above were captured on 2026-11-05 by @lambert in mission-20261105-11 (step 3). The operator passed this section of the plain checklist on 2026-11-09.

---

## 31 · Help footer and goldens (conditional) (m5-d21b)

**Context:** Terminal: Setup. Binary: `go test`; `npm run screens`. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

This section is automated, with a hands-on step for the operator. The change landed under m5-d21b (mission-20261105-12, 2026-11-05). The help overlay's footer now names `docs: github.com/djm56/kirsch` where it named `docs: doc/usage.md`, in the footer ladder: the clipped variant at `internal/tui/modal.go:300` and the unclipped variant at `internal/tui/modal.go:308`. The decision is O16, quoted from `plan/PROGRESS.md` (*Milestone 5 — decisions recorded*, 2026-11-03): "Operator accepted Bishop's recommendation ("Accept all (Recommended)", m5-d20 decision sheet): The help footer's `docs: doc/usage.md` becomes `docs: github.com/djm56/kirsch`, because an installed binary has no `doc/` folder (m5-d21b)." X8 records the same decision, and O23 sets how the golden fences move.

The sized fence that moved is `## 06 · Help overlay` (80×28) in `plan/spec/kirsch-ui-screens.md`. Its footer row is line 405 of that file, and it now reads `kirsch v0.1.0-dev · Esc or ? closes`. No other sized fence moved. The unsized footer row of the colour legend, and a bullet under fence 06 that states the width condition, were also edited.

The docs link shows only when the help box is wide enough for the full rung. At 80 columns it does not show, and the golden fence `## 06 · Help overlay` pins that. On terminals wide enough for the full footer it shows: `TestHelpFooterNamesScrollKeysWhenClipped` asserts the link at 120×27 (clipped body) and at 120×30 (whole body), and those cases pass. No terminal run in this section measured where the link starts. Those widths are worked out from the code, with their conditions, in *Open from m5-d21b* in `plan/PROGRESS.md`. A different version string changes them.

The steps and their reviews are recorded in `.claude/memory/missions/mission-20261105-12/PROGRESS.md`. This section does not repeat them.

**Waits on:** m5-d20 (O16, recorded) and O23 (golden handling, recorded), from `plan/milestones/milestone-5.md:127`. The footer rungs that carry the link are at `internal/tui/modal.go:300` and `:308`.

**Before any edit.** Verbatim copies exist, as the golden procedure requires (*Golden mechanism and regeneration procedure*, step 2). `.claude/memory/workspace/m5-d21b-screens.pre.md` is the spec before step 1 changed it, and it is the base of the golden diff below. `.claude/memory/workspace/m5-d21b-docs-pre/plan_spec_kirsch-ui-screens.md` is the spec as step 1 left it, before this docs step. They sit in `workspace/`, so the next mission init archives them and does not delete them.

**Automated checks**

Each command below is run as written, from the kirsch folder, and its raw output is pasted under it. Each output ends with its `exit=` line. The commands were run on 2026-11-05 by @lambert in mission-20261105-12 (step 3), after the spec edit, so the screen check reads the spec as it now stands. Your times will differ a little, and that is normal.

- [ ] The footer matches O16 (`plan/milestones/milestone-5.md`, m5-d21b *Acceptance*, first box). The test that proves it is `TestHelpFooterNamesScrollKeysWhenClipped` (`internal/tui/header_test.go`). Within its loop, the 120×27 case (clipped body) asserts that the footer contains `docs: github.com/djm56/kirsch` and does not contain `docs: doc/usage.md`, and the 120×30 case (whole body) asserts the same. The test's name and docblock describe only the scroll keys, so the docs assertions are not visible from the name. Command and output below, first block.

  ```bash
  # authored here
  go test -count=1 -v -run 'TestHelpFooterNamesScrollKeysWhenClipped' ./internal/tui/; echo exit=$?
  ```

  ```
  === RUN   TestHelpFooterNamesScrollKeysWhenClipped
  --- PASS: TestHelpFooterNamesScrollKeysWhenClipped (0.04s)
  PASS
  ok  	github.com/djm56/kirsch/internal/tui	0.705s
  exit=0
  ```

- [ ] The golden fence `## 06 · Help overlay` (80×28) matches its render, as do the other sized fences the test covers. The test is `TestMatchesScreenReference` (`internal/tui/golden_test.go`), which runs inside the package command. The procedure gives `go test ./internal/tui`; this run adds `-count=1` so the result is not cached. Command and output below, second block.

  ```bash
  # authored here
  go test -count=1 ./internal/tui/; echo exit=$?
  ```

  ```
  ok  	github.com/djm56/kirsch/internal/tui	9.723s
  exit=0
  ```

- [ ] `npm run screens` lints the screens document, its grids and its content (*Golden mechanism and regeneration procedure*, step 5). Command and output below, third block.

  ```bash
  npm run screens; echo exit=$?
  ```

  ```
  > kirsch@0.1.0-dev screens
  > python3 scripts/lint-screens.py

  kirsch-ui-screens.md: grids OK, content OK
  exit=0
  ```

- [ ] Every moved golden is listed in the step report, by fence header (m5-d21b *Acceptance*, first box). No command: this is a process check. Step 1's record in `.claude/memory/missions/mission-20261105-12/PROGRESS.md` names the moved fence, `## 06 · Help overlay` (80×28), and no other.

- [ ] The review step covers the golden diff and is recorded (m5-d21b *Acceptance*, second box). No command: this is a process check. Step 2 of `.claude/memory/missions/mission-20261105-12/PROGRESS.md` records @apone's approval of step 1 with "only fence 06 moved". On 2026-11-11 the operator accepted their 2026-11-09 tick of section 31 step 2 of the plain checklist as the human review of the golden diff that O23 `a` calls for (*Open from m5-d21b* item d in `plan/PROGRESS.md`).

**Hands-on checks**

1. (operator) Review the golden diff (m5-d21b *Mission steps*, step 2: "The operator also reviews the golden diff"). The diff is Bishop's diff of `.claude/memory/workspace/m5-d21b-screens.pre.md` against `plan/spec/kirsch-ui-screens.md`. The change inside the sized fence is these lines, in `## 06 · Help overlay` (80×28), at line 405 of the spec:

   ```diff
   -         │ kirsch v0.1.0-dev · docs: doc/usage.md · Esc or ? closes   │
   +         │ kirsch v0.1.0-dev · Esc or ? closes                        │
   ```

   The same diff also shows edits outside any sized fence: the footer row of the colour legend, and a bullet under fence 06. Neither is a picture. The complete diff for this docs step is `.claude/memory/workspace/m5-d21b-step3.diff`.

   > **Look at:** the docs link is gone from the 80×28 picture, and nothing takes its place, because at 80 columns the link does not show. The link shows on terminals wide enough for the full footer (see *What landed*). Whether the 80-column picture should keep a link is for you to decide. Bishop's recommendation is in *Open from m5-d21b* in `plan/PROGRESS.md`.

**Notes / evidence**

The automated outputs above were captured on 2026-11-05 by @lambert in mission-20261105-12 (step 3). No width threshold in this section was run in a terminal. The thresholds are worked out from the code and recorded in `plan/PROGRESS.md`. The operator passed this section of the plain checklist on 2026-11-09. The operator confirmed the O23 reading on 2026-11-11.

---

## 32 · Debug log clip (conditional) (m5-d21c)

**Context:** Terminal: Setup. Binary: `go test`. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d21c lands, or record "no step":** the byte bound from the m5-d20 decision, and the clip as built.

**Waits on:** m5-d20 (the C24 decision and its byte bound), from `plan/milestones/milestone-5.md:128`. The call site is `internal/provider/anthropic/client.go:453`; `logBadRequest` is at `client.go:506–512`.

**Before any edit.** Save `SECURITY.md` to `.claude/memory/workspace/m5-d21c-SECURITY.pre.md` (`plan/milestones/milestone-5.md:1726`). The control below runs on that copy.

**Automated checks**

- [ ] The 400 log writes at most the bound quoted in the brief. The test asserts presence of the first N bytes and absence of the bytes after N, and it passes (`plan/milestones/milestone-5.md:1729`). Command: `TO EXPAND`.
- [ ] `SECURITY.md` no longer carries the phrase "full request body" (`plan/milestones/milestone-5.md:1730`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -c 'full request body' SECURITY.md
  cd /Volumes/DATA/Github/kirsch && grep -c 'full request body' .claude/memory/workspace/m5-d21c-SECURITY.pre.md
  ```

  The first prints 0. The control on the pre-edit copy prints at least 1.

- [ ] The review step is recorded (`plan/milestones/milestone-5.md:1731`). Process check.

**Hands-on checks**

1. Quote the recorded bound, in bytes, into the brief (`plan/milestones/milestone-5.md:1724`).

**Notes / evidence**

_Empty. Fill at run._

---

## 33 · Debug log disclosure (conditional) (m5-d21d)

**Context:** Terminal: none; Kirsch is not started. Binary: none; `grep`, `awk` and `wc`, each run through `bash -c` from the repository root, and `echo` for the `exit=` lines. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: no key is used. Config state: n/a; no config file is read.

**What landed**

The decision, quoted from `plan/PROGRESS.md` (row C24, 2026-11-03):

> Operator accepted Bishop's recommendation ("Accept all (Recommended)", m5-d20 decision sheet): Disclose in `SECURITY.md` that a 400 writes the full request body to the debug log; do not clip it (m5-d21d runs; m5-d21c does not).

The debug-log bullet under *Known limits* in `SECURITY.md` now states what the debug log can hold, and it states the 400 behaviour as a disclosed limit: the full request body is written, and it is not clipped.

**Automated checks**

**Box 1.** First item of the *Acceptance* list under `## m5-d21d (conditional) — Debug log disclosure (C24)` in `plan/milestones/milestone-5.md`: "`SECURITY.md` states the 400 log behaviour, and each claim traces to a cited line."

The commands to run are in the block below. The table is not for pasting.

| Claim | Command |
|-------|---------|
| 1. Title: messages, paths, commands and, on a 400, the whole conversation | `grep -n -A1 "The debug log can hold your messages" SECURITY.md` |
| 2. Off unless `--debug`, `KIRSCH_DEBUG=1` or `telemetry.debug_log` is set | `grep -n "debug := \*flagDebug" cmd/kirsch/main.go` |
| 3. Default is off | `grep -n "DebugLog: false" internal/config/config.go` |
| 4. A project's configuration cannot turn it on (absence; control in block) | `awk '/^func mergeProjectFile/,/^}/' internal/config/config.go \| grep -n "DebugLog"` |
| 5. Each message's text is logged as a plain attribute | `grep -n "tui submitted" cmd/kirsch/main.go` |
| 6. The message text is not clipped (absence; control in block) | `grep -n "telemetry.Content" cmd/kirsch/main.go` |
| 7. The startup line logs the workspace path | `grep -n "\"workspace\", ws.Root" cmd/kirsch/main.go` |
| 8. A configuration warning names the file and the key | `grep -n "unknown key %q\|key %q ignored" internal/config/config.go` |
| 9. Configuration warnings are logged | `grep -n "log.Warn(\"config\"" cmd/kirsch/main.go` |
| 10. The argv of a refused session grant is logged | `grep -n "\"grant rejected\", \"argv\"" internal/app/app.go` |
| 11. The saved pattern of a restored grant that the policy refuses is logged | `grep -n "restored grant rejected by policy" internal/app/app.go` |
| 12. Refused context-file names are in warnings, and the warnings are logged | `grep -n "candidate %q refused\|\"project context\", \"warning\", w" internal/agent/prompt/loader.go internal/app/app.go` |
| 13. A context file whose read or stat fails is named, and the error is logged | `grep -n "candidate %q: %w\|\"project context\", \"error\"" internal/agent/prompt/loader.go internal/app/app.go` |
| 14. The base name of a usage file moved aside is logged | `grep -n "recovered from corrupt\|recoveredTo = filepath.Base" internal/app/app.go internal/session/usage.go` |
| 15. Session, session index, usage and lock errors are logged | `grep -n "session store error\|failed to update session index\|usage update failed\|lock unavailable" internal/app/app.go` |
| 16. The lock-file open error wraps the `os.OpenFile` error for the lock path | `grep -n "os.OpenFile(lockPath\|open lock file: %w" internal/session/lock.go` |
| 16a. An `os.OpenFile` error names the path it failed on (Go standard library `*os.PathError`; not a Kirsch code line) | none — stdlib behaviour |
| 17. No tool call is logged (absence; control in block) | `grep -rn --include="*.go" --exclude="*_test.go" "\.RunTool(" internal cmd` |
| 18. The executor writes no log line (absence; control in block) | `awk '/^func \(te \*toolExecutor\) Invoke/,/^}/' internal/app/app.go \| grep -n "log\."` |
| 19. No slash command is logged; `internal/tui` has no logger call (absence; control in block) | `grep -rnE --include="*.go" --exclude="*_test.go" "(^\|[^a-zA-Z])s?log\." internal/tui` |
| 20. Redaction needles, matched without regard to case | `grep -n "ToLower\|env_value" internal/telemetry/telemetry.go` |
| 21. Redaction examines attribute names only | `awk '/^func redactArgs/,/^}/' internal/telemetry/telemetry.go` |
| 22. Emit passes its arguments through redaction | `grep -n "redactArgs(args)" internal/telemetry/telemetry.go` |
| 23. A 400 writes the request body to the debug log | `grep -n "logBadRequest(\|provider bad request\|request_body" internal/provider/anthropic/client.go` |
| 24. The request body is not clipped (absence; control in block) | `grep -n "Content(.*reqBody\|Content(.*request" internal/provider/anthropic/client.go` |
| 25. The request holds the system prompt, messages and tool definitions | `grep -n "reqMap\[\"\(system\|messages\|tools\)\"\]" internal/provider/anthropic/encode.go` |
| 26. The request holds thinking blocks, tool calls and tool results | `grep -n "\"type\": *\"\(thinking\|tool_use\|tool_result\)\"" internal/provider/anthropic/encode.go` |
| 27. Tool results carry file contents and command output | `grep -n "content, cut := Truncate\|Truncate(output" internal/tool/readfile.go internal/tool/run_command.go` |
| 28. A MissingSessionID 400 returns before the body is logged | `grep -n -A4 "case 400:" internal/provider/anthropic/client.go` |
| 29. A response body over 256 bytes is cut to 256 bytes and marked | `grep -n "len(body) > MaxContentBytes\|MaxContentBytes = 256\|…(truncated)" internal/telemetry/telemetry.go` |
| 30. Advice to keep the log off (text only; not a code claim) | `grep -n -A1 "Keep the$" SECURITY.md` |

```bash
set -o pipefail
cd /Volumes/DATA/Github/kirsch
# row 1 — title
grep -n -A1 "The debug log can hold your messages" SECURITY.md; echo "exit=$?"
# row 2 — off unless --debug, KIRSCH_DEBUG=1 or telemetry.debug_log
grep -n "debug := \*flagDebug" cmd/kirsch/main.go; echo "exit=$?"
# row 3 — default is off
grep -n "DebugLog: false" internal/config/config.go; echo "exit=$?"
# row 4 — a project's configuration cannot turn it on (absence)
awk '/^func mergeProjectFile/,/^}/' internal/config/config.go | grep -n "DebugLog"; echo "exit=$?"
# row 4, control — the same search finds the project key that is set
awk '/^func mergeProjectFile/,/^}/' internal/config/config.go | grep -n "ProjectFiles"; echo "exit=$?"
# row 5 — each message's text is logged as a plain attribute
grep -n "tui submitted" cmd/kirsch/main.go; echo "exit=$?"
# row 6 — the message text is not clipped (absence)
grep -n "telemetry.Content" cmd/kirsch/main.go; echo "exit=$?"
# row 6, control — the same search finds telemetry.Content in the provider
grep -n "telemetry.Content" internal/provider/anthropic/client.go; echo "exit=$?"
# row 7 — startup line logs the workspace path
grep -n "\"workspace\", ws.Root" cmd/kirsch/main.go; echo "exit=$?"
# row 8 — a configuration warning names the file and the key
grep -n "unknown key %q\|key %q ignored" internal/config/config.go; echo "exit=$?"
# row 9 — configuration warnings are logged
grep -n "log.Warn(\"config\"" cmd/kirsch/main.go; echo "exit=$?"
# row 10 — argv of a refused session grant is logged
grep -n "\"grant rejected\", \"argv\"" internal/app/app.go; echo "exit=$?"
# row 11 — saved pattern of a restored grant that the policy refuses is logged
grep -n "restored grant rejected by policy" internal/app/app.go; echo "exit=$?"
# row 12 — refused context-file names are in warnings, and the warnings are logged
grep -n "candidate %q refused\|\"project context\", \"warning\", w" internal/agent/prompt/loader.go internal/app/app.go; echo "exit=$?"
# row 13 — a context file whose read or stat fails is named, and the error is logged
grep -n "candidate %q: %w\|\"project context\", \"error\"" internal/agent/prompt/loader.go internal/app/app.go; echo "exit=$?"
# row 14 — base name of a usage file moved aside is logged
grep -n "recovered from corrupt\|recoveredTo = filepath.Base" internal/app/app.go internal/session/usage.go; echo "exit=$?"
# row 15 — session, session index, usage and lock errors are logged
grep -n "session store error\|failed to update session index\|usage update failed\|lock unavailable" internal/app/app.go; echo "exit=$?"
# row 16 — the lock-file open error wraps the os.OpenFile error for the lock path
grep -n "os.OpenFile(lockPath\|open lock file: %w" internal/session/lock.go; echo "exit=$?"
# row 17 — no tool call is logged (absence)
grep -rn --include="*.go" --exclude="*_test.go" "\.RunTool(" internal cmd; echo "exit=$?"
# row 17, control — the same path finds the executor's calls
grep -rn --include="*.go" --exclude="*_test.go" "\.Invoke(" internal cmd; echo "exit=$?"
# row 18 — the executor writes no log line (absence)
awk '/^func \(te \*toolExecutor\) Invoke/,/^}/' internal/app/app.go | grep -n "log\."; echo "exit=$?"
# row 18, control — the same window finds the call to the registry
awk '/^func \(te \*toolExecutor\) Invoke/,/^}/' internal/app/app.go | grep -n "reg.Invoke"; echo "exit=$?"
# row 19 — no slash command is logged: internal/tui has no logger call (absence)
grep -rnE --include="*.go" --exclude="*_test.go" "(^|[^a-zA-Z])s?log\." internal/tui; echo "exit=$?"
# row 19, control — the same search finds logger calls in internal/app
grep -rnE --include="*.go" --exclude="*_test.go" "(^|[^a-zA-Z])s?log\." internal/app | wc -l; echo "exit=$?"
# row 20 — redaction needles, matched without regard to case
grep -n "ToLower\|env_value" internal/telemetry/telemetry.go; echo "exit=$?"
# row 21 — redaction examines attribute names only
awk '/^func redactArgs/,/^}/' internal/telemetry/telemetry.go; echo "exit=$?"
# row 22 — emit passes its arguments through redactArgs
grep -n "redactArgs(args)" internal/telemetry/telemetry.go; echo "exit=$?"
# row 23 — a 400 writes the request body to the debug log
grep -n "logBadRequest(\|provider bad request\|request_body" internal/provider/anthropic/client.go; echo "exit=$?"
# row 24 — the request body is not clipped (absence)
grep -n "Content(.*reqBody\|Content(.*request" internal/provider/anthropic/client.go; echo "exit=$?"
# row 24, control — the same search finds the response body's Content call
grep -n "Content(.*respBody" internal/provider/anthropic/client.go; echo "exit=$?"
# row 25 — the request holds the system prompt, messages and tool definitions
grep -n "reqMap\[\"\(system\|messages\|tools\)\"\]" internal/provider/anthropic/encode.go; echo "exit=$?"
# row 26 — the request holds thinking blocks, tool calls and tool results
grep -n "\"type\": *\"\(thinking\|tool_use\|tool_result\)\"" internal/provider/anthropic/encode.go; echo "exit=$?"
# row 27 — tool results carry file contents and command output
grep -n "content, cut := Truncate\|Truncate(output" internal/tool/readfile.go internal/tool/run_command.go; echo "exit=$?"
# row 28 — a MissingSessionID 400 returns before the body is logged
grep -n -A4 "case 400:" internal/provider/anthropic/client.go; echo "exit=$?"
# row 29 — a response body over 256 bytes is cut to 256 bytes and marked
grep -n "len(body) > MaxContentBytes\|MaxContentBytes = 256\|…(truncated)" internal/telemetry/telemetry.go; echo "exit=$?"
# row 30 — advice, not a code claim: keep the log off
grep -n -A1 "Keep the$" SECURITY.md; echo "exit=$?"
```

```text
80:- **The debug log can hold your messages, paths, commands and, on a 400, the
81-  whole conversation.** The debug log is off unless you turn it on with
exit=0
211:	debug := *flagDebug || os.Getenv("KIRSCH_DEBUG") == "1" || cfg.Telemetry.DebugLog
exit=0
116:		Telemetry: TelemetryConfig{DebugLog: false},
exit=0
exit=1
49:					if projectFilesVal, hasProjectFiles := contextTable["project_files"]; hasProjectFiles {
62:								cfg.Context.ProjectFiles = projectFiles
exit=0
370:		log.Debug("tui submitted", "text", text)
exit=0
exit=1
511:		slog.String("request_body", reqBody), telemetry.Content("response_body", respBody))
exit=0
256:		"workspace", ws.Root,
exit=0
153:		return fmt.Sprintf("%s: key %q ignored (a project config may set only [context].project_files)", w.File, w.Key)
156:			return fmt.Sprintf("%s: unknown key %q (ignored)", w.File, w.Key)
exit=0
252:		log.Warn("config", "detail", w.String())
exit=0
2728:			a.log.Debug("grant rejected", "argv", app.argv, "err", err)
exit=0
1510:			a.log.Debug("restored grant rejected by policy", "action", g.Action, "pattern", g.Pattern, "error", err)
exit=0
internal/agent/prompt/loader.go:62:			warnings = append(warnings, fmt.Sprintf("project context candidate %q refused: %s", candidate, resolveErr))
internal/app/app.go:842:		a.log.Warn("project context", "warning", w)
exit=0
internal/agent/prompt/loader.go:71:			return "", "", 0, warnings, fmt.Errorf("stat project context candidate %q: %w", candidate, statErr)
internal/agent/prompt/loader.go:86:			return "", "", 0, warnings, fmt.Errorf("read project context candidate %q: %w", candidate, readErr)
internal/app/app.go:845:		a.log.Error("project context", "error", err)
exit=0
internal/app/app.go:2412:			a.log.Warn("recovered from corrupt usage.json", "moved_to", recovered)
internal/session/usage.go:191:			recoveredTo = filepath.Base(corruptPath)
exit=0
893:	a.log.Warn("lock unavailable — single-instance guard off", "error", err)
894:	a.SendStatusWarning("lock unavailable — single-instance guard off")
1202:			a.log.Debug("session store error", "error", err)
1433:			a.log.Debug("session store error", "error", err)
1441:		a.log.Debug("failed to update session index", "error", err)
2415:			a.log.Warn("usage update failed", "error", err)
exit=0
42:	file, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE, 0o600) // #nosec G304 -- fixed ".lock" basename in a session dir named by a sha256 hex of the workspace, created 0700
44:		return nil, fmt.Errorf("open lock file: %w", err)
exit=0
exit=1
internal/app/app.go:313:	res := te.app.reg.Invoke(ctx, c.Name, c.Input)
internal/app/app.go:2576:		res := a.reg.Invoke(ctx, name, raw)
internal/agent/agent.go:564:			out <- a.Tools.Invoke(ctx, copyToolCall(call))
internal/tool/tool.go:146:	return t.Invoke(ctx, raw)
exit=0
exit=1
13:	res := te.app.reg.Invoke(ctx, c.Name, c.Input)
exit=0
exit=1
      25
exit=0
162:	k := strings.ToLower(key)
163:	for _, needle := range []string{"api_key", "apikey", "token", "secret", "password", "env_value"} {
exit=0
func redactArgs(args []any) []any {
	out := make([]any, len(args))
	copy(out, args)
	for i := 0; i+1 < len(out); i += 2 {
		key, ok := out[i].(string)
		if !ok {
			continue
		}
		if looksSecret(key) {
			out[i+1] = "<redacted>"
		}
	}
	for i, a := range out {
		if attr, ok := a.(slog.Attr); ok && looksSecret(attr.Key) {
			out[i] = slog.String(attr.Key, "<redacted>")
		}
	}
	return out
}
exit=0
116:	l.log.Log(nil, level, msg, redactArgs(args)...) //nolint:staticcheck // nil ctx is fine here
exit=0
453:		c.logBadRequest(string(reqBody), string(bodyBytes))
506:func (c *Client) logBadRequest(reqBody, respBody string) {
510:	c.log.Debug("provider bad request", slog.String("endpoint", c.endpoint), slog.Int("status", 400),
511:		slog.String("request_body", reqBody), telemetry.Content("response_body", respBody))
exit=0
exit=1
511:		slog.String("request_body", reqBody), telemetry.Content("response_body", respBody))
exit=0
113:		reqMap["system"] = system
129:	reqMap["messages"] = messages
142:		reqMap["tools"] = tools
exit=0
260:			"type":      "thinking",
271:			"type":  "tool_use",
279:			"type":        "tool_result",
exit=0
internal/tool/readfile.go:127:	content, cut := Truncate(b.String(), MaxFileBytes)
internal/tool/run_command.go:260:			content, truncated := Truncate(output, maxOutputBytes)
internal/tool/run_command.go:274:	content, truncated := Truncate(output, maxOutputBytes)
exit=0
449:	case 400:
450-		if c.isMissingSessionID(bodyBytes) {
451-			return &Error{Kind: KindConfig, Status: status, Attempts: attempt, Message: fmt.Sprintf("provider %q refused the request: MissingSessionID (configuration error, not retried)", c.endpoint)}
452-		}
453-		c.logBadRequest(string(reqBody), string(bodyBytes))
exit=0
25:const MaxContentBytes = 256
126:	if len(body) > MaxContentBytes {
127:		body = body[:MaxContentBytes] + "…(truncated)"
exit=0
107:  bytes is cut to its first 256 bytes, followed by `…(truncated)`. Keep the
108-  debug log off when you work on material you cannot share.
exit=0
```

**Box 2.** Second item of the same *Acceptance* list: "The sentence in `SECURITY.md` that says a 400 writes the full request body now says that this is a disclosed limit. `grep -n 'full request body' SECURITY.md` prints a line, and that line contains `disclosed`. On the file that m5-d12 leaves, the sentence has no `disclosed`, so this box fails until the update is made."

```bash
set -o pipefail
cd /Volumes/DATA/Github/kirsch
grep -n 'full request body' SECURITY.md; echo "exit=$?"
grep -n 'full request body' SECURITY.md | grep -c disclosed; echo "exit=$?"
# control — the pre-mission copy: the first command must print the line, the second must print 0
grep -n 'full request body' .claude/memory/workspace/m5-d21d-pre/SECURITY.md; echo "exit=$?"
grep -n 'full request body' .claude/memory/workspace/m5-d21d-pre/SECURITY.md | grep -c disclosed; echo "exit=$?"
```

```text
100:  full request body. This is a disclosed limit, and the body is not clipped. It
exit=0
1
exit=0
85:  provider writes the full request body to the debug log. That body includes
exit=0
0
exit=1
```

The control runs the same command on the pre-mission copy, `.claude/memory/workspace/m5-d21d-pre/SECURITY.md`, and prints `0` with `exit=1`, so the box fails there, as it should. The pre-mission copy moves to `.claude/memory/workspace/archive-mission-<next id>/` at the next mission init; after that the control commands report a missing file instead of line 85.

For reference, line 85 of the pre-mission copy, verbatim:

```text
  provider writes the full request body to the debug log. That body includes
```

The quote has no `disclosed`, so a check against it fails, as it should.

**Operator.** Run only if C24 disclosed, which it did (row C24, 2026-11-03). The operator reads the debug-log bullet under *Known limits* in `SECURITY.md` and runs the plain checks in `plan/testing/manual/user-testing/milestone-5-donovan.md` § 33.

**Notes / evidence.** Outputs pasted from the run in mission-20261105-13 on 2026-11-06 (UTC). `SECURITY.md` was not changed by this section's steps.

---

## 34 · ULID dependency (conditional) (m5-d21e)

**Context:** Terminal: Setup. Binary: `go test`. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d21e lands, or record "no step":** the stdlib ULID replaced by `oklog/ulid`, or the operator's decision to keep it.

**Waits on:** m5-d20 (the C40 decision), from `plan/milestones/milestone-5.md:130`. The code is at `internal/session/layout.go:96–140`.

**Automated checks**

- [ ] `go test ./internal/session/` passes, and the review step is recorded (`plan/milestones/milestone-5.md:1788`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test ./internal/session/
  ```

**Hands-on checks**

1. Run only if the operator adds `oklog/ulid`. If the operator keeps the stdlib ULID, record "no step" (`plan/milestones/milestone-5.md:1785`).

**Notes / evidence**

_Empty. Fill at run._

---

## 35 · §7 amendment (conditional) (m5-d21f)

**Context:** Terminal: none; Kirsch is not started. Binary: none; `bash`, `grep`, `awk`, `sed` and `eval`, each run through `bash -c` from the repository root. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: no key is used. Config state: n/a; no config file is read.

**What landed**

Section 7 of `plan/spec/kirsch-plan.md` now lists only the session event types the code writes, states the flattened `assistant.message` that decision C43 accepts ("Accept the flattening, and amend §7 to state it"), and leaves out `session.started` and `session.resumed`, as decision C46 keeps them ("Keep the code, which does not write `session.started` or `session.resumed`, and amend §7 to match").

**Automated checks**

**Box.** The acceptance box, quoted verbatim from the *Acceptance* list of `## m5-d21f (conditional) — §7 amendment (C43, C46)` in `plan/milestones/milestone-5.md`, and found with `grep -F -e` (section J of the block below): "§7 matches the events the code writes, and the review step is recorded." Its review step is step 2 of mission-20261106-01, which had not run when this section was written. The box is the operator's, and it is not ticked here.

The block below is one bash script with every command in it. It was run from the repository root as `bash -c 'bash <file>'`, where the file holds the block's text. Its raw output, `exit=` lines included, is in the block after it.

| Claim | Verdict | Where in the block |
|-------|---------|--------------------|
| `user.message` is written | written | B, `eventTypeUserMessage`; C; E (kept) |
| `assistant.message` is written | written | B, `eventTypeAssistantMessage`; C; E (kept) |
| `assistant.thinking` is written | written | B, `eventTypeAssistantThinking`; C; E (kept) |
| `tool.completed` is written | written | B, `eventTypeToolCompleted`; C; E (kept) |
| `turn.started` is written | written | B, `eventTypeTurnStarted`; D; E (kept) |
| `turn.completed` is written | written | B, `eventTypeTurnCompleted`; E (kept) |
| `usage` is written | written | B, `eventTypeUsage`; E (kept) |
| `approval.scope_granted` is written | written | B, `eventTypeApprovalScopeGranted`; E (kept) |
| `approval.scopes_cleared` is written | written | B, `eventTypeApprovalScopesCleared`; E (kept) |
| `compaction.applied` is written | written | B, `eventTypeCompactionApplied`; E (kept) |
| `session.started` is not written | not written | E (absent from §7, with a control on the pre-edit copy); F (absent from the code, with a control) |
| `session.resumed` is not written | not written | E; F |
| `tool.requested` is not written | not written | E; F |
| `approval.requested` is not written | not written | E; F |
| `approval.resolved` is not written | not written | E; F |
| `error` is not written | not written | E; G (no literal `Type` value in the session package, with a control); H |
| `assistant.delta` is not written | not written | B (no marshal site, with a control on the same grep shape); E |

```bash
set -o pipefail
cd /Volumes/DATA/Github/kirsch || exit 2
SPEC=plan/spec/kirsch-plan.md
PRE=.claude/memory/workspace/m5-d21f-pre/plan-spec-kirsch-plan.md
ST=internal/session/event.go
SEC7='awk "/^## 7\. Session Events/{p=1} /^## 8\. /{p=0} p"'

echo "### A. Timestamp"
date -u +"%Y-%m-%d %H:%M UTC"; echo "exit=$?"

echo "### B. Write check per event constant: (a) marshal site, (b) enclosing func, (c) caller of that func, (d) Store Record method called outside internal/session"
for pair in UserMessage:Record AssistantMessage:Record AssistantThinking:Record ToolCompleted:Record TurnStarted:RecordTurnStarted TurnCompleted:RecordTurnCompleted Usage:RecordUsage ApprovalScopeGranted:RecordScopeGranted ApprovalScopesCleared:RecordScopesCleared CompactionApplied:RecordCompaction AssistantDelta:NONE; do
  c=${pair%%:*}; m=${pair#*:}
  echo "## eventType$c (Store method: $m)"
  L=$(grep -n -e "Type: *eventType$c}" $ST | cut -d: -f1 | head -1)
  grep -n -e "Type: *eventType$c}" $ST; echo "(a) exit=$?"
  if [ -n "$L" ]; then
    F=$(awk -v L="$L" 'NR<=L && /^func /{f=$0} NR==L{print f; exit}' $ST)
    echo "(b) $F"
    FN=$(echo "$F" | sed -E 's/^func ([A-Za-z]+)\(.*/\1/')
    grep -rn -F -e "$FN(" internal/session --include="*.go" | grep -v "_test\.go" | grep -v -F -e "func $FN("; echo "(c) exit=$?"
    grep -rn -F -e ".$m(" internal/ cmd/ --include="*.go" | grep -v "_test\.go" | grep -v "^internal/session/"; echo "(d) exit=$?"
  else
    echo "(a) no marshal site. Control, same grep shape on a written type:"
    grep -n -e "Type: *eventTypeTurnStarted}" $ST; echo "control exit=$?"
    echo "read sites only:"
    grep -rn -F -e "case eventType$c:" internal/session --include="*.go" | grep -v "_test\.go"; echo "read exit=$?"
  fi
  echo
done

echo "### C. Chain above the encoders"
grep -n -e "encode(kind" internal/session/store.go; echo "exit=$?"
grep -n -e "encodeAppend(m)" -e "encodeMerge(m)" -e "encodeUserAppend(m)" -e "encodeAssistantAppend(m)" $ST; echo "exit=$?"

echo "### D. Reachability: function-level callers. Store methods called from internal/app/app.go (control: RecordTurnCompleted, called from the turn-status switch)"
grep -n -F -e "RecordTurnStarted(" -e "RecordUsage(" -e "RecordScopesCleared(" -e "RecordScopeGranted(" -e "RecordCompaction(" -e "RecordTurnCompleted(" internal/app/app.go; echo "app.go exit=$?"
echo "App methods wired from cmd/kirsch/main.go (the wiring the TUI reaches the app through)"
grep -n -e "Submit" -e "ClearGrants" -e "ManualCompact" cmd/kirsch/main.go; echo "main.go exit=$?"

echo "### E. Section 7 of the spec, example form. Removed names must not match in the current file. The same command on the pre-edit copy must match (the corrupted input the check has to reject)."
for n in session.started session.resumed tool.requested approval.requested approval.resolved error assistant.delta; do
  eval "$SEC7 $SPEC" | grep -n -F -e "\"type\":\"$n\""; echo "removed $n, current exit=$?"
  eval "$SEC7 $PRE" | grep -n -F -e "\"type\":\"$n\""; echo "removed $n, pre-edit copy exit=$?"
done
for n in turn.started user.message assistant.thinking assistant.message approval.scope_granted approval.scopes_cleared tool.completed compaction.applied usage turn.completed; do
  eval "$SEC7 $SPEC" | grep -n -F -e "\"type\":\"$n\""; echo "kept $n, current exit=$?"
done

echo "### F. Code: literal JSON type names in non-test Go under internal and cmd (control: turn.started)"
for n in session.started session.resumed tool.requested approval.requested approval.resolved; do
  grep -rn -F -e "\"$n\"" internal/ cmd/ --include="*.go" | grep -v "_test\.go"; echo "code $n exit=$?"
done
grep -rn -F -e '"turn.started"' internal/ cmd/ --include="*.go" | grep -v "_test\.go"; echo "control turn.started exit=$?"

echo "### G. Literal Type values in the session package (error is not a written type; control: constant Type values)"
grep -rn -e 'Type: *"' internal/session --include="*.go" | grep -v "_test\.go"; echo "literal Type exit=$?"
grep -rn -e 'Type: *eventType' internal/session --include="*.go" | grep -v "_test\.go"; echo "control constant Type exit=$?"

echo "### H. JSON type tags in session and cmd/kirsch (event.go:51 is the only tag on a written line; the two others decode on read; control: the v tag)"
grep -rn -F -e 'json:"type"' internal/session cmd/kirsch --include="*.go" | grep -v "_test\.go"; echo "type tags exit=$?"
grep -rn -F -e 'json:"v"' internal/session --include="*.go" | grep -v "_test\.go"; echo "control v tag exit=$?"

echo "### I. Workspace path in the event structs (control: the Version field)"
grep -n -i -e 'workspace' $ST; echo "workspace exit=$?"
grep -n -e 'Version int' $ST; echo "control Version exit=$?"

echo "### J. Quotes against their sources (controls: a near-miss must not match)"
grep -n -F -e "Accept the flattening, and amend §7 to state it" plan/PROGRESS.md; echo "C43 in PROGRESS exit=$?"
grep -n -F -e "Accept the flattening, and amend §7 to state it" $SPEC; echo "C43 in spec exit=$?"
grep -n -F -e "Accept the flattening, and amend §7 to state this" plan/PROGRESS.md; echo "C43 near-miss exit=$?"
grep -n -F -e 'Keep the code, which does not write `session.started` or `session.resumed`, and amend §7 to match' plan/PROGRESS.md; echo "C46 in PROGRESS exit=$?"
grep -n -F -e 'Keep the code, which does not write `session.started` or `session.resumed`, and amend §7 to match' $SPEC; echo "C46 in spec exit=$?"
grep -n -F -e "§7 matches the events the code writes, and the review step is recorded." plan/milestones/milestone-5.md; echo "acceptance box exit=$?"

echo "### K. Planning cites that point at the pre-amendment lines of the spec"
P=$(printf 'kirsch-plan.md:%s' 34)
grep -n -F -e "$P" plan/milestones/milestone-5.md; echo "exit=$?"
```

Raw output of the block above, run at 06:24 UTC on 2026-11-06 (the timestamp is the run's own, from `date -u`):

```text
### A. Timestamp
2026-11-06 06:24 UTC
exit=0
### B. Write check per event constant: (a) marshal site, (b) enclosing func, (c) caller of that func, (d) Store Record method called outside internal/session
## eventTypeUserMessage (Store method: Record)
181:		baseEvent: baseEvent{Version: eventVersion, Type: eventTypeUserMessage},
211:			baseEvent: baseEvent{Version: eventVersion, Type: eventTypeUserMessage},
(a) exit=0
(b) func encodeMerge(m agent.Message) ([][]byte, error) {
internal/session/event.go:153:		return encodeMerge(m)
(c) exit=0
internal/agent/agent.go:497:		a.Recorder.Record(kind, copyMessage(m))
(d) exit=0

## eventTypeAssistantMessage (Store method: Record)
284:			baseEvent: baseEvent{Version: eventVersion, Type: eventTypeAssistantMessage},
(a) exit=0
(b) func encodeAssistantAppend(m agent.Message) ([][]byte, error) {
internal/session/event.go:165:		return encodeAssistantAppend(m)
(c) exit=0
internal/agent/agent.go:497:		a.Recorder.Record(kind, copyMessage(m))
(d) exit=0

## eventTypeAssistantThinking (Store method: Record)
261:				baseEvent: baseEvent{Version: eventVersion, Type: eventTypeAssistantThinking},
(a) exit=0
(b) func encodeAssistantAppend(m agent.Message) ([][]byte, error) {
internal/session/event.go:165:		return encodeAssistantAppend(m)
(c) exit=0
internal/agent/agent.go:497:		a.Recorder.Record(kind, copyMessage(m))
(d) exit=0

## eventTypeToolCompleted (Store method: Record)
225:				baseEvent: baseEvent{Version: eventVersion, Type: eventTypeToolCompleted},
(a) exit=0
(b) func encodeUserAppend(m agent.Message) ([][]byte, error) {
internal/session/event.go:163:		return encodeUserAppend(m)
(c) exit=0
internal/agent/agent.go:497:		a.Recorder.Record(kind, copyMessage(m))
(d) exit=0

## eventTypeTurnStarted (Store method: RecordTurnStarted)
308:		baseEvent: baseEvent{Version: eventVersion, Type: eventTypeTurnStarted},
(a) exit=0
(b) func marshalTurnStarted(turn int) ([]byte, error) {
internal/session/store.go:189:	line, err := marshalTurnStarted(turn)
(c) exit=0
internal/app/app.go:2257:			store.RecordTurnStarted(turn)
(d) exit=0

## eventTypeTurnCompleted (Store method: RecordTurnCompleted)
316:		baseEvent: baseEvent{Version: eventVersion, Type: eventTypeTurnCompleted},
(a) exit=0
(b) func marshalTurnCompleted(status string) ([]byte, error) {
internal/session/store.go:201:	line, err := marshalTurnCompleted(status)
(c) exit=0
internal/app/app.go:2286:			store.RecordTurnCompleted(status)
(d) exit=0

## eventTypeUsage (Store method: RecordUsage)
324:		baseEvent:    baseEvent{Version: eventVersion, Type: eventTypeUsage},
(a) exit=0
(b) func marshalUsage(u agent.Usage) ([]byte, error) {
internal/session/store.go:214:	line, err := marshalUsage(u)
(c) exit=0
internal/app/app.go:2398:		store.RecordUsage(usage)
(d) exit=0

## eventTypeApprovalScopeGranted (Store method: RecordScopeGranted)
335:		baseEvent: baseEvent{Version: eventVersion, Type: eventTypeApprovalScopeGranted},
(a) exit=0
(b) func marshalApprovalScopeGranted(action, pattern string) ([]byte, error) {
internal/session/store.go:227:	line, err := marshalApprovalScopeGranted(action, pattern)
(c) exit=0
internal/app/app.go:2742:				store.RecordScopeGranted(action, strings.Join(app.argv, " "))
(d) exit=0

## eventTypeApprovalScopesCleared (Store method: RecordScopesCleared)
344:		baseEvent: baseEvent{Version: eventVersion, Type: eventTypeApprovalScopesCleared},
(a) exit=0
(b) func marshalApprovalScopesCleared() ([]byte, error) {
internal/session/store.go:241:	line, err := marshalApprovalScopesCleared()
(c) exit=0
internal/app/app.go:633:		store.RecordScopesCleared()
(d) exit=0

## eventTypeCompactionApplied (Store method: RecordCompaction)
351:		baseEvent:    baseEvent{Version: eventVersion, Type: eventTypeCompactionApplied},
(a) exit=0
(b) func marshalCompactionApplied(from, to int, summary string) ([]byte, error) {
internal/session/store.go:254:	line, err := marshalCompactionApplied(from, to, summary)
(c) exit=0
internal/app/app.go:2893:		store.RecordCompaction(comp.From, comp.To, comp.Summary)
(d) exit=0

## eventTypeAssistantDelta (Store method: NONE)
(a) exit=1
(a) no marshal site. Control, same grep shape on a written type:
308:		baseEvent: baseEvent{Version: eventVersion, Type: eventTypeTurnStarted},
control exit=0
read sites only:
internal/session/resume.go:157:		case eventTypeAssistantDelta:
read exit=0

### C. Chain above the encoders
162:	lines, err := encode(kind, m)
exit=0
151:		return encodeAppend(m)
153:		return encodeMerge(m)
163:		return encodeUserAppend(m)
165:		return encodeAssistantAppend(m)
exit=0
### D. Reachability: function-level callers. Store methods called from internal/app/app.go (control: RecordTurnCompleted, called from the turn-status switch)
633:		store.RecordScopesCleared()
2257:			store.RecordTurnStarted(turn)
2286:			store.RecordTurnCompleted(status)
2398:		store.RecordUsage(usage)
2742:				store.RecordScopeGranted(action, strings.Join(app.argv, " "))
2893:		store.RecordCompaction(comp.From, comp.To, comp.Summary)
app.go exit=0
App methods wired from cmd/kirsch/main.go (the wiring the TUI reaches the app through)
367:	// Submit hands ordinary composer text to the app, which starts a real
369:	m.Submit = func(text string) {
371:		a.Submit(text, os.Getenv)
397:	// ClearGrants wires the TUI's grant clearing to the app's policy.
398:	m.ClearGrants = func() {
399:		a.ClearGrants()
417:	// ManualCompact wires the /compact command. The app method blocks until
420:	m.ManualCompact = func() tui.ManualCompactMsg {
421:		return a.ManualCompact(os.Getenv)
main.go exit=0
### E. Section 7 of the spec, example form. Removed names must not match in the current file. The same command on the pre-edit copy must match (the corrupted input the check has to reject).
removed session.started, current exit=1
6:{"v":1,"type":"session.started","session_id":"...","workspace":"...","model":"...","project_type":"go"}
removed session.started, pre-edit copy exit=0
removed session.resumed, current exit=1
7:{"v":1,"type":"session.resumed","session_id":"...","from_event":142}
removed session.resumed, pre-edit copy exit=0
removed tool.requested, current exit=1
13:{"v":1,"type":"tool.requested","id":"toolu_01...","tool":"read_file","input":{}}
removed tool.requested, pre-edit copy exit=0
removed approval.requested, current exit=1
14:{"v":1,"type":"approval.requested","id":"toolu_01...","action":"apply_patch","detail":{}}
removed approval.requested, pre-edit copy exit=0
removed approval.resolved, current exit=1
15:{"v":1,"type":"approval.resolved","id":"toolu_01...","granted":true,"scope":"once"}
removed approval.resolved, pre-edit copy exit=0
removed error, current exit=1
22:{"v":1,"type":"error","kind":"command_timeout","detail":"..."}
removed error, pre-edit copy exit=0
removed assistant.delta, current exit=1
10:{"v":1,"type":"assistant.delta","text":"..."}
removed assistant.delta, pre-edit copy exit=0
6:{"v":1,"type":"turn.started","turn":3}
kept turn.started, current exit=0
7:{"v":1,"type":"user.message","content":"..."}
kept user.message, current exit=0
8:{"v":1,"type":"assistant.thinking","content":"...","signature":"..."}
kept assistant.thinking, current exit=0
9:{"v":1,"type":"assistant.message","content":"...","tool_calls":[{"id":"toolu_01...","tool":"read_file","input":{}}]}
kept assistant.message, current exit=0
10:{"v":1,"type":"approval.scope_granted","action":"run_command","pattern":"go test"}
kept approval.scope_granted, current exit=0
11:{"v":1,"type":"approval.scopes_cleared"}
kept approval.scopes_cleared, current exit=0
12:{"v":1,"type":"tool.completed","id":"toolu_01...","ok":true,"content":"...","truncated":false}
kept tool.completed, current exit=0
13:{"v":1,"type":"compaction.applied","replaced_from":3,"replaced_to":98,"summary":"..."}
kept compaction.applied, current exit=0
14:{"v":1,"type":"usage","input_tokens":1234,"output_tokens":567,"cache_read":8900,"cache_write":0}
kept usage, current exit=0
15:{"v":1,"type":"turn.completed","status":"success"}
kept turn.completed, current exit=0
### F. Code: literal JSON type names in non-test Go under internal and cmd (control: turn.started)
code session.started exit=1
code session.resumed exit=1
code tool.requested exit=1
code approval.requested exit=1
code approval.resolved exit=1
internal/session/event.go:29:	eventTypeTurnStarted = "turn.started"
control turn.started exit=0
### G. Literal Type values in the session package (error is not a written type; control: constant Type values)
literal Type exit=1
internal/session/event.go:181:		baseEvent: baseEvent{Version: eventVersion, Type: eventTypeUserMessage},
internal/session/event.go:211:			baseEvent: baseEvent{Version: eventVersion, Type: eventTypeUserMessage},
internal/session/event.go:225:				baseEvent: baseEvent{Version: eventVersion, Type: eventTypeToolCompleted},
internal/session/event.go:261:				baseEvent: baseEvent{Version: eventVersion, Type: eventTypeAssistantThinking},
internal/session/event.go:284:			baseEvent: baseEvent{Version: eventVersion, Type: eventTypeAssistantMessage},
internal/session/event.go:308:		baseEvent: baseEvent{Version: eventVersion, Type: eventTypeTurnStarted},
internal/session/event.go:316:		baseEvent: baseEvent{Version: eventVersion, Type: eventTypeTurnCompleted},
internal/session/event.go:324:		baseEvent:    baseEvent{Version: eventVersion, Type: eventTypeUsage},
internal/session/event.go:335:		baseEvent: baseEvent{Version: eventVersion, Type: eventTypeApprovalScopeGranted},
internal/session/event.go:344:		baseEvent: baseEvent{Version: eventVersion, Type: eventTypeApprovalScopesCleared},
internal/session/event.go:351:		baseEvent:    baseEvent{Version: eventVersion, Type: eventTypeCompactionApplied},
control constant Type exit=0
### H. JSON type tags in session and cmd/kirsch (event.go:51 is the only tag on a written line; the two others decode on read; control: the v tag)
internal/session/event.go:51:	Type string `json:"type"`
internal/session/grants.go:46:			Type string `json:"type"`
internal/session/usage_totals.go:53:			Type string `json:"type"`
type tags exit=0
internal/session/event.go:49:	Version int `json:"v"`
control v tag exit=0
### I. Workspace path in the event structs (control: the Version field)
workspace exit=1
49:	Version int `json:"v"`
control Version exit=0
### J. Quotes against their sources (controls: a near-miss must not match)
408:| C43 | Operator accepted Bishop's recommendation ("Accept all (Recommended)", m5-d20 decision sheet): Accept the flattening, and amend §7 to state it (m5-d21f). | 2026-11-03 |
C43 in PROGRESS exit=0
358:**Flattened assistant message.** Decision C43 ("Accept the flattening, and amend §7 to state it") is recorded as follows: each assistant message is written as one `assistant.message` line, in which every text block is joined into `content` and every tool call into `tool_calls`, so the order of text between tool calls is lost. Each `assistant.thinking` line is written before the message's `assistant.message` line, whatever the thinking block's original position in the message (`internal/session/event.go`). The relative order of thinking, text and tool calls is therefore lost, not only the order of text between tool calls.
C43 in spec exit=0
C43 near-miss exit=1
409:| C46 | Operator accepted Bishop's recommendation ("Accept all (Recommended)", m5-d20 decision sheet): Keep the code, which does not write `session.started` or `session.resumed`, and amend §7 to match (m5-d21f). | 2026-11-03 |
C46 in PROGRESS exit=0
360:**Types this section does not list.** `session.started` and `session.resumed` are absent because the code does not write them. Decision C46 reads: "Keep the code, which does not write `session.started` or `session.resumed`, and amend §7 to match." The types `tool.requested`, `approval.requested`, `approval.resolved`, `error` and `assistant.delta` are also absent, because the code does not write them.
C46 in spec exit=0
1846:- [ ] §7 matches the events the code writes, and the review step is recorded.
acceptance box exit=0
### K. Planning cites that point at the pre-amendment lines of the spec
73:| O11 | D3: name of the new session event (for example `approval.scopes_cleared`) and the §7 amendment. Whether `/new` also records one (it starts a new file, so no). | — | — | m5-d5 | `plan/adr/0006-session-scoped-approvals.md:45, :67–69`; `plan/spec/kirsch-plan.md:340–359` |
285:| C43 | §7 flattening of interleaved text and tool calls | m5-d20 (decision); m5-d21f (§7 amendment) | `plan/PROGRESS.md:186`; `plan/spec/kirsch-plan.md:340–359` |
288:| C46 | `session.started` / `session.resumed` not written | m5-d20 (decision); m5-d21f (§7 amendment) | `plan/PROGRESS.md:194`; `plan/spec/kirsch-plan.md:342–343` |
720:3. `@lambert` — docs: the §7 amendment (`plan/spec/kirsch-plan.md:340–359`) listing the event the code writes.
1829:- C43 — §7 flattening of interleaved text and tool calls (`plan/spec/kirsch-plan.md:340–359`).
1830:- C46 — the session events `session.started` and `session.resumed` (`plan/spec/kirsch-plan.md:342–343`).
1853:**Landed (mission-20261106-01, 2026-11-06).** The example in §7 of `plan/spec/kirsch-plan.md` now lists only the session event types the code writes, with the JSON fields their marshal structs write. The reconstruction rule no longer says `assistant.delta` exists. A note under §7 states the flattened `assistant.message` (C43). C46 covers `session.started` and `session.resumed`; the other removed types had no decision and are under *Open from m5-d21f* in `plan/PROGRESS.md`. The storage-layout sentence that said each session file holds a `session.started` event is corrected. No code changed. The steps and their records are in `.claude/memory/missions/mission-20261106-01/PROGRESS.md`: step 1 is this amendment, and step 2 is the review by `@apone`. The acceptance box above is the operator's, and none is ticked. Historical note: the cites `plan/spec/kirsch-plan.md:340–359` and `:342–343` in this section, and the same cites at `:73`, `:285`, `:288` and `:720` of this file, give the pre-amendment location, as the plan was written. They are not rewritten here.
exit=0
```

**Notes.** The `:73`, `:285`, `:288` and `:720` cites above are in the planning tables and the m5-d5 step list, and they point at the pre-amendment lines, as the historical note in the m5-d21f section says.

---

## 36 · Lock behaviour (conditional) (m5-d21g)

**Context:** Terminal: Setup. Binary: `go test`. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d21g lands, or record "no step":** the usage-lock behaviour as the operator decided it.

**Waits on:** m5-d20 (the C8 decision), from `plan/milestones/milestone-5.md:132`. The code is at `internal/session/usage.go:28, :152`.

**Automated checks**

- [ ] The lock behaviour matches the recorded decision, and `go test ./internal/session/` passes (`plan/milestones/milestone-5.md:1845`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test ./internal/session/
  ```

**Hands-on checks**

1. Run only if the operator changes the lock behaviour. If the operator accepts it, record "no step" (`plan/milestones/milestone-5.md:1842`).

**Notes / evidence**

_Empty. Fill at run._

---

## 37 · Demo, release smoke run, tag (m5-d22)

**Context:** Terminal: a real terminal, run by the operator, with Setup rerun in it. Binary: `bash scripts/smoke-test.sh` on the release build (live; spends tokens); the demo launch (O12, TO EXPAND). Working directory: /Volumes/DATA/Github/kirsch. Workspace: `--workspace "$WORKSPACE"` on the demo launch (TO EXPAND). Key state: key entered for this live step only, for the smoke run. Config state: Setup, holding only `[session]`; confirm with the model check in Setup before the live step.

**What landed**

> **TO EXPAND when m5-d22 runs:** the demo location (O12), the release smoke transcript, and the tag.

**Waits on:** the m5-d21 deliverables that ran, and O12, O15 and O16 recorded, from `plan/milestones/milestone-5.md:133`. No agent runs this deliverable. Its steps are operator-only (`plan/milestones/milestone-5.md:1876`).

**Key-state check before the live step.** Authored here. It must print `CONTROL PASS`, then `0` before any key is entered.

```bash
# authored here
keycount_control
keycount
```

**Automated checks**

- [ ] `git tag -l v0.1.0` prints `v0.1.0`, and the release page lists the artefacts (`plan/milestones/milestone-5.md:1891`). Run after the tag is pushed.

  ```bash
  cd /Volumes/DATA/Github/kirsch && git tag -l v0.1.0
  ```

- [ ] `ls opencode.json.bak .claude/memory/state/FLIGHT-RECORDER.md.bak` reports both files missing (`No such file or directory`). Control: `ls scripts/smoke-test.sh` lists its file, so the error is not a wrong path. Paste the output (`plan/milestones/milestone-5.md:1892`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && ls opencode.json.bak .claude/memory/state/FLIGHT-RECORDER.md.bak
  cd /Volumes/DATA/Github/kirsch && ls scripts/smoke-test.sh
  ```

  Run this after the C14 deletion, not before.

- [ ] The model table and its dates match the vendor pages as read on the day of tagging (`plan/milestones/milestone-5.md:1893`). Command: `TO EXPAND`.

**Live checks**

> Spends tokens — run only with a key you intend to use. Confirm the budget first (`plan/milestones/milestone-5.md:1901`). Enter the key with the secure pattern below, and not otherwise.
>
> **ONE-WAY:** step 6 of *Hands-on checks* tags `v0.1.0`, and the tag publishes a public release through the release workflow (m5-d11). That is one-way. Do not push the tag until the smoke run has passed and the operator has confirmed it.

Key entry, Milestone 4 pattern (`plan/testing/manual/milestone-4.md:63–73`):

```bash
read -rs OPENCODE_API_KEY && export OPENCODE_API_KEY
```

- [ ] `bash scripts/smoke-test.sh` on the release build prints PASS, and the transcript is pasted (`plan/milestones/milestone-5.md:1890`). Remove any scratch config override first (`plan/testing/manual/milestone-4.md:1331`), which the model check in Setup shows.

  ```bash
  cd /Volumes/DATA/Github/kirsch && bash scripts/smoke-test.sh
  ```

When the run is done, the block below unsets every key variable. Authored here. The count after the unset must print `CONTROL PASS`, then `0`.

```bash
# authored here
unset ANTHROPIC_API_KEY KIRSCH_ANTHROPIC_API_KEY OPENCODE_API_KEY KIRSCH_OPENCODE_API_KEY
keycount_control
keycount
```

**Hands-on checks**

1. (operator) Make the demo (O12) on a real terminal. Store or link it, and embed or link it in `README.md` (`plan/milestones/milestone-5.md:1879`).

   > **TO EXPAND when O12 is recorded:** the demo format and where it is stored.

2. (operator) Delete the `.bak` files `opencode.json.bak` and `.claude/memory/state/FLIGHT-RECORDER.md.bak` (C14, `plan/milestones/milestone-5.md:1881`). This is a deletion. Confirm before it runs.
3. (operator) Re-read the vendor price pages and confirm the model table (C9, X5). Record the date read (`plan/milestones/milestone-5.md:1882`).
4. (operator) Run the walkthrough retests (C38) and tick the outstanding verification boxes (C61) (`plan/milestones/milestone-5.md:1883`).
5. (operator) Update the `README.md` banner and "What works today" to name v0.1.0 (S31, `plan/milestones/milestone-5.md:1884`).
6. (operator) **ONE-WAY.** Tag `v0.1.0` (S33, `plan/milestones/milestone-5.md:1885`). The tag publishes a public release through the release workflow (m5-d11), which is one-way. Run this step only after the smoke run in the live check has passed.

**Notes / evidence**

_Empty. Fill at run. Paste the smoke transcript and the tag output. No key appears in this file or in any report._

---

## Known limitations

> **TO EXPAND as items land:** any limit found while a deliverable runs that is not already listed below. The M4 narrow-width item is not a limit. It was m5-d1 (`plan/milestones/milestone-5.md:7`), resolved on 2026-10-27 as a wrong expectation, not a code defect.

The carried items below are the Backlog in `plan/milestones/milestone-5.md:1906–1943`. The milestone records them as recorded and not scheduled for v0.1. Check (c) in *Mechanical checks* compares this table's IDs against that Backlog.

| ID | Item | Source |
|----|------|--------|
| C2 | Absence-check tests use fixed sleeps | plan/PROGRESS.md:78 |
| C3 | Out-of-order `SessionUsageMsg` can leave stale totals | plan/PROGRESS.md:99 |
| C4 | Mid-session pricing change can understate or hide cost | plan/PROGRESS.md:99 |
| C6 | `applyCompaction` error arm in `Compact` untested | plan/PROGRESS.md:99 |
| C7 | Rename-aside-then-write-failure consequence undocumented | plan/PROGRESS.md:99 |
| C10 | Race mutant for the resume test killed in 1 of 20 runs | plan/PROGRESS.md:99 |
| C12 | "Limitations" pinned as a bare word | plan/PROGRESS.md:106 |
| C13 | `system.md` mutants not independently rerun | plan/PROGRESS.md:107 |
| C15 | Smoke test approval and TestDivide-baseline branches lack an offline negative control | plan/PROGRESS.md:114 |
| C16 | Comment in `smoke_live_test.go` overstates containment | plan/PROGRESS.md:115 |
| C17 | Zero-divisor check is a source regex | plan/PROGRESS.md:116 |
| C18 | `stop_reason`, `tool`, `duration_ms` omitted from session events | plan/PROGRESS.md:126, :178, :216 |
| C19 | Other agent edge cases | plan/PROGRESS.md:127 |
| C20 | Calibration warnings | plan/PROGRESS.md:128 |
| C22 | Cancellation goroutine not joined | plan/PROGRESS.md:130 |
| C23 (code change) | Every 401/403 reads "rejected the API key". The documentation remedy is in m5-d14b | plan/PROGRESS.md:137; `internal/provider/anthropic/client.go:447` |
| C25 | Config review items | plan/PROGRESS.md:139 |
| C26 | Wire-format review items | plan/PROGRESS.md:140 |
| C27 | Live-test review items | plan/PROGRESS.md:141 |
| C28 | Default-model review items | plan/PROGRESS.md:142 |
| C31 | Suspected trailing-newline defect in `internal/patch`, not yet reproduced | plan/PROGRESS.md:150 |
| C41 | Fuzz seeds with `..` for `FuzzSessionPathContained` | plan/PROGRESS.md:171 |
| C44 | Pending-map leniency | plan/PROGRESS.md:187, :188 |
| C45 | Per-event fuzz seeds; deep-copy comment on `cloneBlocks`/`cloneRaw` | plan/PROGRESS.md:188 |
| C47 | No auto-resume happy-path test in `main_test.go` | plan/PROGRESS.md:195 |
| C53 | `/diff` shows unstaged changes only | plan/PROGRESS.md:206 |
| C54 | `/files` resets on resume. If built, D4 changes (m5-d14a) | plan/PROGRESS.md:207 |
| C55 | `update_test.go` and `status_test.go` overlap | plan/PROGRESS.md:208 |
| C56 | Flush-ordering edge, 500 ms window | plan/PROGRESS.md:214 |
| C57 | 100 ms settle window | plan/PROGRESS.md:215 |
| C58 | Per-field mutant evidence shown for Signature only | plan/PROGRESS.md:217 |
| C59 | `held_lock_no_warning` fixed 200 ms settle | plan/PROGRESS.md:223 |
| C60 | Lock-unavailable fallback has no automated test | plan/PROGRESS.md:224 |
| X6 (wiring) | §5 command-line flags precedence layer unwired. v0.1 documents no config flags | `internal/config/config.go:496–506`; `cmd/kirsch/main.go:71–75` |

Out of scope for v0.1, quoted from `plan/milestones/milestone-5.md:1951`:

> Multiple providers, `git commit`/`git push` tools, MCP, LSP, subagents, plugins, OS keychain, web UI, session branching/search, automatic compaction UI tuning. Do not build these early.

The spec's list at `plan/spec/kirsch-plan.md:32–34` differs at the provider item. The difference is open for O22 (`plan/milestones/milestone-5.md:1953`).

---

## Reporting

For any check marked ❌:

1. The exact command you ran.
2. Terminal size (`echo $COLUMNS $LINES`) and whether you are in tmux.
3. Section number, check name, and the `m5-dX` anchor.
4. The mission ID and the step number, if the run belongs to a mission step.
5. Config file content if relevant (`$T_CFG/kirsch/config.toml`).
6. Debug log if relevant: `cat "$XDG_STATE_HOME/kirsch/debug.log"` (`plan/testing/manual/milestone-4.md:1343`).

Never paste API keys into a report. Keys are entered with the secure pattern in Setup and in m5-d22, and they are unset when the run ends.
