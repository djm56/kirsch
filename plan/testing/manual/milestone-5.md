# Manual test — Milestone 5

Kirsch Milestone 5 is polish and release: the narrow-width status bar, the `kirsch doctor` command, `--version` and `--help`, the release tooling (build wrapper, goreleaser configuration, release workflow), the repository files, the user documentation under `doc/`, and the operator decisions that gate the conditional follow-through. The deliverables are defined in `plan/milestones/milestone-5.md`.

> **Status: skeleton.** This document was written at planning time (mission-20261023-06, 2026-10-23). It is expanded as each deliverable lands. Section 1's commands were run in mission-20261027-03, and the operator passed section 1 of the plain checklist on 2026-10-27. Section 2's commands were run in mission-20261026-01, and the operator passed section 2 of the plain checklist on 2026-10-27. Section 4's commands were run in mission-20261027-04, and the operator passed section 4 of the plain checklist on 2026-10-27, with a note (see the carried item under *Carried from Milestone 5 planning* in `plan/PROGRESS.md`). Section 5's commands were run in mission-20261027-06 (step 7) on 2026-10-27, and the operator passed section 5 of the plain checklist on 2026-10-28. Section 6's automated commands were run in mission-20261027-05 (step 5) on 2026-10-27, and the operator passed section 6 of the plain checklist on 2026-10-28; its hands-on steps need a live model turn and have not been run, and the operator checklist's section 6 has only automated steps. Section 9's commands were run in mission-20261027-08 on 2026-10-27, and the operator passed section 9 of the plain checklist on 2026-10-28. Section 16's automated check was run in mission-20261027-07 (step 3) on 2026-10-27; its removal control is recorded in mission-20261027-07 step 1, and the operator passed section 16 of the plain checklist on 2026-10-28. Section 3's automated checks were run in mission-20261028-03 (step 5) on 2026-10-28, from zsh after `source scripts/m5-test-env.sh`, and the output is in `.claude/memory/workspace/m5d5-step5-evidence.txt`; its hands-on step needs a live model turn and an API key. No key-free route was found in a 25-minute attempt (mission-20261028-03 step 6a), so the step has not been run; the operator tested section 3 of the plain checklist on 2026-10-29, and that checklist's quit-and-reopen check is checked at the final release check (section 37 of the checklist). Section 7's automated checks and hands-on commands were run in mission-20261028-03 (step 9) on 2026-10-28, from zsh, and the output is in `.claude/memory/workspace/m7d-step9-evidence.txt`; the operator tested section 7 of the plain checklist on 2026-10-29. Section 10's automated commands were run in mission-20261028-03 (step 13) on 2026-10-28, from zsh, and the output is in `.claude/memory/workspace/m10b-step13-evidence.txt`; the goreleaser snapshot was not run (goreleaser is not installed, and no install was allowed in that step), and the operator tested section 10 of the plain checklist on 2026-10-29. Section 8's automated checks were run in mission-20261028-03 (step 17) on 2026-10-28, from zsh, and the output is in `.claude/memory/workspace/m6d-step17-auto.txt`; the operator tested section 8 of the plain checklist on 2026-10-29, and its hands-on paste step has not been pre-run. Section 13's automated commands and golden check were run in mission-20261029-02 (step 4) on 2026-10-29, from zsh, and the output is in `.claude/memory/workspace/m7a-step4-runs.txt`. The fix round after the step 5 review (mission-20261029-02, step 4a) reran them, and its output is in `.claude/memory/workspace/m7a-step4a-runs.txt`; the operator has not yet tested section 13 of the plain checklist. Section 12's automated and hands-on commands were run in mission-20261029-03 (step 3) on 2026-10-29, from zsh, and the output is in `.claude/memory/workspace/m3b-step3-runs.txt`; the operator has not yet tested section 12 of the plain checklist. Every block marked `TO EXPAND` is a placeholder, and the expansion belongs to the step that lands the deliverable. The operator runs tests from the plain checklist `plan/testing/manual/user-testing/milestone-5-donovan.md`, set up with `source scripts/m5-test-env.sh`; see [`milestone-5-start-here.md`](../../milestones/milestone-5-start-here.md).

## How to read this document

- **Sections follow the order of work below.** Each section is one deliverable, titled `## N · <Title> (m5-dX)`. Split deliverables (m5-d3a and m5-d3b, for example) have their own sections.
- **Anchors.** The `§ m5-dX` hooks in `plan/milestones/milestone-5.md` resolve to the parenthesised ID in each heading. `§ m5-d3a` is the heading ending `(m5-d3a)`. The check block below proves this, and proves the order-of-work table and the known-limitations table against their sources.
- **Context.** The first line under each heading is a `**Context:**` line. Its label format is fixed by check (d) in *Mechanical checks*, which fails any Context line missing one of its labels. Run a section only in the state its Context line names. *Setup* in a Context line means the Setup terminal, or a new terminal or window that has rerun the whole Setup block, then the Guard functions block (its definitions, not only the `guard` call), then `guard`.
- **Automated checks** are copied from the acceptance boxes in `plan/milestones/milestone-5.md`, with the line they came from. A command is quoted only where the milestone gives one. A test box with no command says `TO EXPAND`. A command the document writes itself, and does not quote from the milestone, is marked by a first-line comment `# authored here` inside its block.
- **Hands-on checks** are numbered steps where the milestone gives enough detail. Otherwise a `TO EXPAND` block says what the operator will observe.
- **Live checks** appear only in sections whose deliverable has a live step. Each one is marked. A provider check contacts the provider and spends tokens. A GitHub release run publishes a public release, which is one-way. Run a live check only with the key, the fork or the tag its marking names.
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
| 19 | m5-d17a `/new` lifecycle: C51, C52 (optional) | 3 | Automated + Hands-on | m5-d5, O13 |
| 20 | m5-d4a Doctor: key, backend, model and project-context lines | 4 | Automated + Hands-on | m5-d3b, O5 |
| 21 | m5-d7b Soft-wrap fidelity: composer cut marker | 4 | Automated + Hands-on + Operator | m5-d6, m5-d7a, O17, O23 |
| 22 | m5-d14a `doc/usage.md` corrections | 4 | Automated + Hands-on | m5-d1, m5-d5, m5-d6, m5-d7a, O6, O10 |
| 23 | m5-d15 Front-door and tracker freshness | 4 | Automated + Hands-on | m5-d1, m5-d10a, m5-d10b, m5-d13c, O22 |
| 24 | m5-d4b Doctor: state directory and lock state | 5 | Automated + Operator | m5-d4a, O5 |
| 25 | m5-d12 Repository files and golden-regeneration note | 5 | Automated + Operator | m5-d10a, m5-d10b, m5-d8, O7, O8, O21 |
| 26 | m5-d17b `/new` busy state and turn fencing: C48, C49 (optional) | 5 | Automated + Hands-on + Live | m5-d8, m5-d17a, O13 |
| 27 | m5-d14b `doc/troubleshooting.md` and `doc/README.md` | 6 | Automated + Live + Operator | m5-d3a, m5-d3b, m5-d4a, m5-d4b, m5-d13a, m5-d13c, O6 |
| 28 | m5-d18 Resume totals and compaction % (optional) | 6 | Automated + Hands-on + Live | m5-d17a (file ownership only), O13 |
| 29 | m5-d20 Record the operator's decisions | gate | Operator | O12, O15, O16, and the decision list |
| 30 | m5-d21a Model table (conditional) | 7 | Automated + Hands-on | m5-d20 (X5, C9) |
| 31 | m5-d21b Help footer and goldens (conditional) | 7 | Automated + Operator | m5-d20 (X8, O16), O23 |
| 32 | m5-d21c Debug log clip (conditional) | 7 | Automated + Hands-on | m5-d20 (C24) |
| 33 | m5-d21d Debug log disclosure (conditional) | 7 | Automated + Hands-on + Operator | m5-d20 (C24), m5-d12 |
| 34 | m5-d21e ULID dependency (conditional) | 7 | Automated + Hands-on | m5-d20 (C40) |
| 35 | m5-d21f §7 amendment (conditional) | 7 | Automated + Hands-on + Operator | m5-d20 (C43, C46), m5-d5 |
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

The unset line clears the key variables, as the m5-d16 box does (`plan/milestones/milestone-5.md:1408`), and clears `MY_TOKEN` and `T_BROKEN`. Nothing in this document sets `T_BROKEN` now: section 7 uses a throwaway folder inside a subshell and leaves nothing behind. The name stays in the unset line because `scripts/m5-test-env.sh` still names it. That is a known leftover, carried in `plan/PROGRESS.md`. `KIRSCH_LIVE_MODEL` is unset as Milestone 4 does (`plan/testing/manual/milestone-4.md:40`).

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

`--version` prints `kirsch <version>` only. It prints no commit and no date. A version stamped with `-ldflags "-X main.version=…"` is printed as given. With no stamp, a `go install …@vX.Y.Z` build reports that module version with the leading `v` removed. A plain `go build` in the checkout keeps `0.1.0-dev`, because Go stamps a pseudo-version or `+dirty` there, and those are ignored (`cmd/kirsch/main.go:50–59`, `resolveVersion`). That `go install` path is covered by `TestResolveVersion`, which feeds version strings to the function; no real `go install` from a tag has been run.

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

Hands-on step 1 was not run, and no no-model substitute was found. `/approvals` is a slash command inside the TUI (`internal/tui/update.go:1471`). `kirsch resume` runs `ResumeLastSession` as the app's startup hook (`cmd/kirsch/main.go:232–238`), and the app is the TUI. Nothing outside the TUI prints the grants, so no command-line check can show the grants a resume restores. Nothing in this section was run against a live provider.

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
- **Warnings.** Config warnings go to `log.Warn` (`cmd/kirsch/main.go:206`). That logger is disabled unless the debug log is on (`main.go:166–172`), so a warning about an ignored project key or a refused context file is shown nowhere by default. The page says so (`doc/configuration.md:208–211`).
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

**Changed 2026-10-29 by m5-d3b.** Any expectation below that quotes the m5-d3a report is stale, and the current results are in §12. That covers the summary counts, the exit statuses, the usage line and the `--workspace` run. Hands-on 1 now prints a `fail` line for config and exits 1 for the broken settings file, where it expected exit 0; the plain run still exits 1. Hands-on 1 also prints `fail  repository: not in a Git repository`, because `$T_CFG` is outside any repository, and its summary reads `doctor: 2 ok, 0 warn, 2 fail` (run 9 of step 3a in `.claude/memory/workspace/m3b-step3a-runs.txt`, a scratch file archived at the next mission start). The operator's section 7 note shows the same lines. Hands-on 3 now exits 1 outside a repository, because the repository check fails there. Hands-on 4 now reads `4:  kirsch doctor [flags]`. Hands-on 5 expected a `--workspace` value not to change the report; with m5-d3b it does. The steps and their expected text are kept as they were at m5-d3a. The carried item (d) in the Notes below is the m5-d3a finding. **Dated note, 2026-10-29 (mission-20261029-06, step 3b):** the summary counts quoted in this section are out of date since m5-d4a as well, because the summary counts the key, backend, model and context lines. Measured on 2026-10-29, from the kirsch folder with no key set and an empty settings folder, the summary reads `doctor: 7 ok, 1 warn, 0 fail` (`.claude/memory/workspace/m20261029-06-step3b-runs.txt`, part 4, scratch). The count outside a repository with no flag is `doctor: 5 ok, 2 warn, 1 fail` (part 6). The same applies to §11 hands-on 1 and 2, whose summary lines are older still.

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

   **Dated note, 2026-10-29 (mission-20261029-06, step 3b):** the `doctor: 0 ok` line in the block above is the output of an earlier doctor that ran no checks, the version that operator section 7 describes. It is kept as recorded. The summary has counted real checks since m5-d3b, so it is no longer the current result.

2. From the kirsch folder, which is inside a repository (authored here):

   ```bash
   # authored here
   cd /Volumes/DATA/Github/kirsch
   "$KIRSCH_BIN" doctor
   echo "exit=$?"
   ```

   Expected: `kirsch 0.1.0-dev`, `doctor: 0 ok, 0 warn, 0 fail`, then `exit=0`. **Dated note, 2026-10-29 (mission-20261029-06, step 3b):** this count is out of date. Since m5-d3b and m5-d4a the summary counts every check line, and the current run from this folder with no key set reads `doctor: 7 ok, 1 warn, 0 fail` (`.claude/memory/workspace/m20261029-06-step3b-runs.txt`, part 4).

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
- (b) Closed in `cmd/kirsch/main.go`: the comments on `doctorProbes` and in `run` now describe the probes. Open in the `internal/doctor` entry of `internal/arch/imports_test.go`, whose comment still says the probes are still to come. That stays true until m5-d4a and m5-d4b land.
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

> **TO EXPAND when m5-d19b lands:** the O15 C29 branch taken (tighten or relax), and the output of the branch's check.

**Waits on:** m5-d19a (same test file), O15 (the C29 decision, recorded before this step runs), O13, from `plan/milestones/milestone-5.md:124`.

**Before any edit.** On the relax branch only, save a verbatim copy of `plan/milestones/milestone-2.md` to `.claude/memory/workspace/m5-d19b-milestone-2.pre.md` (DIR-020; `plan/milestones/milestone-5.md:1591`). The milestone gives no name for this copy. The name here is chosen, and the milestone's relax box is the one to edit.

**Automated checks**

- [ ] If O15 tightens: a test with a child that ignores SIGTERM passes within 500 ms. The test output is pasted (`plan/milestones/milestone-5.md:1594`). Command: `TO EXPAND`.
- [ ] If O15 relaxes: `plan/milestones/milestone-2.md:418–419` states the value O15 records, and no test asserts 500 ms for that case (`plan/milestones/milestone-5.md:1595`). Command: `TO EXPAND`.
- [ ] O15's C29 decision is recorded in `plan/PROGRESS.md` before step 1 runs (`plan/milestones/milestone-5.md:1596`). Process check; reviewed, not run.

**Hands-on checks**

1. > **TO EXPAND when O15 is recorded:** which branch was taken, and the file each branch touches (`internal/tool/run_command.go` and its test for tighten; `plan/milestones/milestone-2.md` for relax).

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

1. (operator, optional, not run) Bracketed paste off, paste 5 KB, and the UI stays responsive.

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

**Context:** Terminal: Setup. Binary: none (the workflow runs on GitHub; no kirsch launch here). Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset; this block makes no model call. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d11 lands:** the trigger, permissions and pinning as O3 records them, and the workflow file's `on:` block.

**Waits on:** m5-d10b (goreleaser config) and O3 (release workflow: trigger, permissions, pinning, dispatch), from `plan/milestones/milestone-5.md:111`. Repository Actions settings must allow write. That is an operator setting (`plan/milestones/milestone-5.md:1066`).

**Automated checks**

- [ ] `release.yml` triggers on `v*` tags, and on dispatch only if O3 says so (`plan/milestones/milestone-5.md:1057`). Command: `TO EXPAND`. The milestone says "grep the `on:` block" and gives no command line.
- [ ] The write permission is scoped as O3 records (`plan/milestones/milestone-5.md:1058`). Command: `TO EXPAND`. The milestone says "grep `permissions:`" and gives no command line.
- [ ] Action pinning matches O3 (`plan/milestones/milestone-5.md:1059`). Command: `TO EXPAND`.

**Live checks**

> Live GitHub release run. Spends no model tokens. **ONE-WAY:** a `v*` tag or a dispatch run publishes a public release, which is one-way. Run this check only on a fork, or with a throwaway pre-release tag on a fork.

- [ ] (operator) A dispatch run, or a throwaway pre-release tag on a fork, produces a release with archives and checksums (`plan/milestones/milestone-5.md:1060`). Command: `TO EXPAND`.

**Notes / evidence**

_Empty. Fill at run. Record the run URL and the release page, not any credential._

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

**Context:** Terminal: a clean machine or container, not the Setup terminal; Setup does not apply there. Binary: `go install` build, or `bin/kirsch` (TO EXPAND). Working directory: /Volumes/DATA/Github/kirsch for the grep checks; the clean machine for hands-on step 1. Workspace: n/a. Key state: all key variables unset on the clean machine. Config state: none in the Setup terminal.

**What landed**

> **TO EXPAND when m5-d13c lands:** the install page as written, and the `go install` version text as O4 records it.

**Waits on:** m5-d2 (the `--version` text), m5-d10a and m5-d10b (archive names and build wrapper), O4, and O6, from `plan/milestones/milestone-5.md:115`. A `go install` build reports `0.1.0-dev` (`cmd/kirsch/main.go:27–28`) unless O4 changes it.

**Key-state check, first.** Authored here. Run it before any kirsch launch in this section. It must print `CONTROL PASS`, then `0`.

```bash
# authored here
keycount_control
keycount
```

**Automated checks**

- [ ] `doc/install.md` states what a `go install` build reports, as O4 records it (`plan/milestones/milestone-5.md:1211`). Command: `TO EXPAND`. Review step against the code.
- [ ] `grep -n -i -w -E 'milestone|ADR' doc/install.md` prints nothing. Control: the printf line prints `an ADR` and not `headroom` (`plan/milestones/milestone-5.md:1212`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n -i -w -E 'milestone|ADR' doc/install.md
  printf 'an ADR\nheadroom\n' | grep -i -w -E 'milestone|ADR'
  ```

**Hands-on checks**

1. (operator) Follow `doc/install.md` on a clean machine or in a container, and confirm it works (`plan/milestones/milestone-5.md:1213`).

   > **TO EXPAND when m5-d13c lands:** the container or machine used, and the commands the page gives.

**Notes / evidence**

_Empty. Fill at run._

---

## 18 · Fresh-machine no-key check (m5-d16)

**Context:** Terminal: Setup for the `cmp` and the `env -u` run; a clean environment for the onboarding run (O19). Binary: `/Volumes/DATA/Github/kirsch/bin/kirsch` from `scripts/build.sh` (m5-d10a), run under `env -i` by its absolute path from outside the checkout. Working directory: /Volumes/DATA/Github/kirsch for `cmp`; the temporary Git repository the script creates for onboarding. Workspace: n/a. Key state: all key variables unset. The `env -u` command below enforces it. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d16 lands:** the script name, the pseudo-terminal method O19 records, and the onboarding text it captured.

**Waits on:** O14 (whether "not in a repo" is included, and where the check lives), O19 (how the interactive TUI is driven), and m5-d10a (builds the binary), from `plan/milestones/milestone-5.md:119`. This does not depend on m5-d4 (`plan/milestones/milestone-5.md:1381`).

**Before any edit.** Save `scripts/smoke-test.sh` verbatim to `.claude/memory/workspace/m5-d16-smoke-test.pre.sh` (`plan/milestones/milestone-5.md:1391`). The acceptance box compares against it.

**Key-state check, first.** Authored here. Run it before any kirsch launch in this section. It must print `CONTROL PASS`, then `0`.

```bash
# authored here
keycount_control
keycount
```

**Automated checks**

- [ ] A no-key run in a clean environment shows onboarding that names `KIRSCH_OPENCODE_API_KEY` and `OPENCODE_API_KEY`. The output is pasted into the step report (`plan/milestones/milestone-5.md:1402`). Command: `TO EXPAND`, per O19.
- [ ] The check runs in a temporary Git repository, which it creates and removes (`plan/milestones/milestone-5.md:1403`). Command: `TO EXPAND`.
- [ ] `scripts/smoke-test.sh` is unchanged against its saved copy. The saved copy is `.claude/memory/workspace/m5-d16-smoke-test.pre.sh` (`plan/milestones/milestone-5.md:1404`).

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

- [ ] `scripts/smoke-test.sh` still exits 2 without a key. The command strips every key variable from the child with `env -u`, so the script sees no key. It then exits 2 at `scripts/smoke-test.sh:16–22`, whatever the shell holds. With any one key variable set in the child, the script runs `TestSmokeLive` against the live provider and spends tokens (`plan/milestones/milestone-5.md:1405–1411`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && env -u ANTHROPIC_API_KEY -u KIRSCH_ANTHROPIC_API_KEY -u OPENCODE_API_KEY -u KIRSCH_OPENCODE_API_KEY bash scripts/smoke-test.sh; echo "exit=$?"
  ```

  The `; echo "exit=$?"` is as the milestone gives it (`plan/milestones/milestone-5.md:1408`). The printed `exit=` line is pasted.

- [ ] The "not in a repo" case is handled as O14 records (`plan/milestones/milestone-5.md:1412`). Command: `TO EXPAND`.

**Hands-on checks**

1. (operator) Run the no-key check on a clean machine (`plan/milestones/milestone-5.md:1413`).

   > **TO EXPAND when O19 is recorded:** the pseudo-terminal method and the terminal size it uses.

**Notes / evidence**

_Empty. Fill at run._

---

## 19 · `/new` lifecycle: C51, C52 (optional) (m5-d17a)

**Context:** Terminal: Setup. Binary: `go test`; `kirsch` for hands-on step 1 (TO EXPAND). Working directory: /Volumes/DATA/Github/kirsch. Workspace: `--workspace "$WORKSPACE"` on the hands-on launch. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d17a lands:** whether the session file is created lazily or removed on cancel (C51), and how the provider session id rotates (C52).

**Waits on:** m5-d5 (the session layer) and O13 (scheduling of the optional group), from `plan/milestones/milestone-5.md:120`.

**Key-state check, first.** Authored here. Run it before any kirsch launch in this section. It must print `CONTROL PASS`, then `0`.

```bash
# authored here
keycount_control
keycount
```

**Automated checks**

- [ ] A test shows that quitting during `/new` leaves no empty session file (`plan/milestones/milestone-5.md:1446`). Command: `TO EXPAND`.
- [ ] A test shows that after `NewSession` the provider session id differs from the id before it (`plan/milestones/milestone-5.md:1447`). Command: `TO EXPAND`.
- [ ] `go test ./internal/app/` passes (`plan/milestones/milestone-5.md:1448`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test ./internal/app/
  ```

**Hands-on checks**

1. > **TO EXPAND when m5-d17a lands:** the manual steps for quit-during-`/new`, and the session directory listing before and after.

**Notes / evidence**

_Empty. Fill at run._

---

## 20 · Doctor: key, backend, model and project-context lines (m5-d4a)

**Context:** Terminal: Setup. Binary: `"$KIRSCH_BIN" doctor`, built by `npm run build` (the build block in §2); `go test` for the checks. Working directory and workspace are stated inside each hands-on step, because they differ from step to step. Settings: Setup, unchanged, except where a step names its own settings folder for that one command, or reads the operator's own settings (`env -u XDG_CONFIG_HOME`). Key state: all key variables unset; hands-on 1 sets one dummy value for one command at a time, never a real key.
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
  - Hands-on 1 shape, from the kirsch folder, no key set (part 4): `doctor: 7 ok, 1 warn, 0 fail`, and `exit=0`. The one warning is the `key` line. The `context` line reads `AGENTS.md, 6687 bytes`, the size of that file on the day of the run.
  - Outside a repository with no flag, in a folder under the workspace with the git ceiling set to the workspace root (part 6): `doctor: 5 ok, 2 warn, 1 fail`, and `exit=1`. The fail is the repository line. The warnings are the `key` line and `context: skipped: no workspace`.
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

**Context:** Terminal: Setup. Binary: none (greps on `doc/usage.md` and the saved copy). Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d14a lands:** the audit headings recorded, and each correction made against the code.

**Waits on:** m5-d1 (narrow bar), m5-d5 (grant clearing), m5-d6 (composer text), m5-d7a (wrap), O6, and O10, from `plan/milestones/milestone-5.md:116`.

**Automated checks**

- [ ] The audit names a section covering key bindings, one covering slash commands, and one covering the approval model (`plan/milestones/milestone-5.md:1259`). Command: `TO EXPAND`. Record each heading in the step report.
- [ ] The status-bar mock at `doc/usage.md:54` shows cache read and write (`plan/milestones/milestone-5.md:1260`). Command: `TO EXPAND`.
- [ ] The `/files` row at `doc/usage.md:247` says a resumed session starts empty, or C54 has been built and D4 is removed (`plan/milestones/milestone-5.md:1261`). Command: `TO EXPAND`.
- [ ] The section headed `### Debug commands — temporary` is gone, and so is its table. No `/read <path>`, `/ls [path]`, `/search <query>`, `/gitstatus` or `/gitdiff` line remains (`plan/milestones/milestone-5.md:1262`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n '/read <path>\|/ls \[path\]\|/search <query>\|/gitstatus\|/gitdiff' doc/usage.md
  printf '/read <path>\n' | grep -n '/read <path>\|/ls \[path\]\|/search <query>\|/gitstatus\|/gitdiff'
  ```

  The same check as commands, which print PASS or FAIL (authored here). The real file must hold none of the lines, and the control must match the printed line once:

  ```bash
  # authored here
  test "$(grep -c '/read <path>\|/ls \[path\]\|/search <query>\|/gitstatus\|/gitdiff' /Volumes/DATA/Github/kirsch/doc/usage.md)" = 0 && echo "PASS real" || echo "FAIL real"
  test "$(printf '/read <path>\n' | grep -c '/read <path>\|/ls \[path\]\|/search <query>\|/gitstatus\|/gitdiff')" = 1 && echo "PASS control" || echo "FAIL control"
  ```

- [ ] "Starting it" names `kirsch resume` and `kirsch --new` (`plan/milestones/milestone-5.md:1263`). Command: `TO EXPAND`.
- [ ] The sentence "Nothing consequential happens silently…" is gone. The replacement names the exact-match allowlist and session grants (`plan/milestones/milestone-5.md:1264`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n 'Nothing consequential happens silently' doc/usage.md
  cd /Volumes/DATA/Github/kirsch && grep -n 'Nothing consequential happens silently' .claude/memory/workspace/m5-d14a-usage.pre.md
  ```

  As commands, which print PASS or FAIL (authored here). The control reads the pre-change text, which is `HEAD`'s copy, because the saved copy is not in the workspace:

  ```bash
  # authored here
  test "$(grep -c 'Nothing consequential happens silently' /Volumes/DATA/Github/kirsch/doc/usage.md)" = 0 && echo "PASS real" || echo "FAIL real"
  test "$(git -C /Volumes/DATA/Github/kirsch show HEAD:doc/usage.md | grep -c 'Nothing consequential happens silently')" = 1 && echo "PASS control" || echo "FAIL control"
  ```

- [ ] The composer paragraph at `doc/usage.md:73–75` describes the behaviour O10 records (`plan/milestones/milestone-5.md:1265`). Command: `TO EXPAND`. Review step.
- [ ] `doc/usage.md` makes no claim of a missing-`rg` warning (`plan/milestones/milestone-5.md:1266`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n 'missing .rg' doc/usage.md
  cd /Volumes/DATA/Github/kirsch && grep -n 'missing .rg' .claude/memory/workspace/m5-d14a-usage.pre.md
  ```

  As commands, which print PASS or FAIL (authored here). The control reads `HEAD`'s copy, as above:

  ```bash
  # authored here
  test "$(grep -c 'missing .rg' /Volumes/DATA/Github/kirsch/doc/usage.md)" = 0 && echo "PASS real" || echo "FAIL real"
  test "$(git -C /Volumes/DATA/Github/kirsch show HEAD:doc/usage.md | grep -c 'missing .rg')" -ge 1 && echo "PASS control" || echo "FAIL control"
  ```

- [ ] `grep -n -i -w -E 'milestone|ADR' doc/usage.md` prints nothing. Control: the printf line prints `an ADR` and not `headroom` (`plan/milestones/milestone-5.md:1267`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n -i -w -E 'milestone|ADR' doc/usage.md
  printf 'an ADR\nheadroom\n' | grep -i -w -E 'milestone|ADR'
  ```

**Hands-on checks**

1. Save a verbatim copy of `doc/usage.md` to `.claude/memory/workspace/m5-d14a-usage.pre.md` before any edit (`plan/milestones/milestone-5.md:1245`). The controls above run on that copy.
2. Check the `.go` files for an emitter of the missing-`rg` warning before removing the claim (`plan/milestones/milestone-5.md:1233`). Command: `TO EXPAND`.

**Notes / evidence**

_Empty. Fill at run._

---

## 23 · Front-door and tracker freshness (m5-d15)

**Context:** Terminal: Setup. Binary: none (greps, `ls`). Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d15 lands:** the judgement recorded for each "0001–0007" site (DIR-004), and the install section as written.

**Waits on:** m5-d1 (X2 wording), m5-d10a and m5-d10b (release tooling the README refers to), m5-d13c (the install page), and O22 (the "multiple providers" wording), from `plan/milestones/milestone-5.md:118`.

**Before any edit.** Save verbatim copies to `.claude/memory/workspace/`: `plan/spec/kirsch-plan.md` as `m5-d15-kirsch-plan.pre.md`, `plan/adr/README.md` as `m5-d15-adr-README.pre.md`, `plan/README.md` as `m5-d15-plan-README.pre.md`, `AGENTS.md` as `m5-d15-AGENTS.pre.md`, `README.md` as `m5-d15-README.pre.md` (`plan/milestones/milestone-5.md:1345`), and `plan/PROGRESS.md` as `m5-d15-PROGRESS.pre.md` (`plan/milestones/milestone-5.md:1353`, DIR-020). The controls below read `HEAD`'s copy of each file, which is the pre-change text while the file is unedited, because the saved copies are not in the workspace (see the open question in the step report).

**Automated checks**

- [ ] `README.md` has an install section that links `doc/install.md` (`plan/milestones/milestone-5.md:1356`). Command: `TO EXPAND`.
- [ ] Every line that `grep -n '0001–0007'` prints in the named files is a site the step report records as kept, with its reason (`plan/milestones/milestone-5.md:1357`). The M5 manual files are excluded, because they contain the string themselves. The Milestone 4 manual files do not contain it, so they are not excluded.

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -rln '0001–0007' --include='*.md' . | grep -v '\.claude/memory/' | grep -v 'plan/milestones/' | grep -v -e 'plan/testing/manual/milestone-5.md' -e 'plan/testing/manual/user-testing/milestone-5-donovan.md'
  ```

  Control, as commands (authored here). Each tracked file must hold the string at its pre-change text, so each line prints PASS:

  ```bash
  # authored here
  for f in plan/spec/kirsch-plan.md plan/adr/README.md plan/README.md AGENTS.md; do test "$(git -C /Volumes/DATA/Github/kirsch show HEAD:$f | grep -c '0001–0007')" -ge 1 && echo "PASS $f" || echo "FAIL $f"; done
  ```

- [ ] `grep -n 'lint-screens.py' README.md` prints a line, and `grep -n 'every script is one line' README.md` prints nothing (`plan/milestones/milestone-5.md:1358`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n 'lint-screens.py' README.md
  cd /Volumes/DATA/Github/kirsch && grep -n 'every script is one line' README.md
  ```

  Control, as a command (authored here). The pre-change `README.md` must hold the line, so this prints PASS:

  ```bash
  # authored here
  test "$(git -C /Volumes/DATA/Github/kirsch show HEAD:README.md | grep -c 'every script is one line')" = 1 && echo "PASS control" || echo "FAIL control"
  ```

- [ ] `CHANGELOG.md:16` and `plan/PROGRESS.md:231` describe the narrow-width behaviour that m5-d1 built (`plan/milestones/milestone-5.md:1359`). Command: `TO EXPAND`. Review step.
- [ ] `README.md:47` states the configured endpoints, in the wording O22 records (`plan/milestones/milestone-5.md:1360`). Command: `TO EXPAND`.
- [ ] C21, C32, C33 and C42 are marked Resolved in `plan/PROGRESS.md`. C39 is marked Resolved with the 2026-10-23 check of `internal/config/config.go:44` cited beside it (`plan/milestones/milestone-5.md:1361`). Command: `TO EXPAND`.
- [ ] `plan/adr/README.md:64` status wording is unchanged (`plan/milestones/milestone-5.md:1362`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -c 'Accepted (planning phase' plan/adr/README.md
  cd /Volumes/DATA/Github/kirsch && grep -c 'Accepted (planning phase' .claude/memory/workspace/m5-d15-adr-README.pre.md
  ```

  The printed counts must match.

**Hands-on checks**

1. List the ADR files, to choose the range the sites must state (`plan/milestones/milestone-5.md:1349`).

   ```bash
   cd /Volumes/DATA/Github/kirsch && ls plan/adr/
   ```

2. Record the judgement for each "0001–0007" site (DIR-004). The milestone lists `plan/spec/kirsch-plan.md:392`, `plan/README.md:25` and `AGENTS.md:21`, plus `plan/adr/README.md:64`, which may stay as written. `CHANGELOG.md:21` is history and stays.

**Notes / evidence**

_Empty. Fill at run._

---

## 24 · Doctor: state directory and lock state (m5-d4b)

**Context:** Terminal: Setup (instance 1), and a second terminal that shares instance 1's `$T_CFG` and `$T_STATE`. The second terminal copies the paths from the first and does not rerun the `mktemp` lines, because a fresh scratch directory has no shared lock. Binary: `kirsch doctor` (launch command TO EXPAND). Working directory: `$T_CFG`. Workspace: `--workspace "$WORKSPACE"` on each kirsch launch. Key state: all key variables unset in both terminals. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d4b lands:** the state-directory probe and the lock report, as O5 allows them to run.

**Waits on:** m5-d4a and O5 (side-effect policy and lock method), from `plan/milestones/milestone-5.md:102`. `session.Acquire` creates the session directory and `.lock` (`internal/session/lock.go:34, :42`), so calling it has side effects.

**Key-state and repository checks, first.** Authored here. It must print `CONTROL PASS`, then `0`. The first `git` line must exit non-zero, and the last must print `true`.

```bash
# authored here
keycount_control
keycount
git -C "$T_CFG" rev-parse --is-inside-work-tree; echo "exit=$?"
git -C "$WORKSPACE" rev-parse --is-inside-work-tree
```

**Automated checks**

- [ ] A non-writable state directory is flagged, and the test restores the directory mode afterwards (`plan/milestones/milestone-5.md:687`). Command: `TO EXPAND`.
- [ ] While the test holds the lock with `session.Acquire`, doctor reports held. After the release, doctor reports free (`plan/milestones/milestone-5.md:688`). Command: `TO EXPAND`.
- [ ] The held lock is still held after the doctor run, so the live instance is not disturbed (`plan/milestones/milestone-5.md:689`). Command: `TO EXPAND`.
- [ ] The probe path printed is `<root>/sessions/<hash>` for a fixture root (`plan/milestones/milestone-5.md:690`). Command: `TO EXPAND`.
- [ ] With config not loaded, both lines read `skipped: config did not load` (`plan/milestones/milestone-5.md:691`). Command: `TO EXPAND`.

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

1. (operator) A second live instance is run by the operator (`plan/milestones/milestone-5.md:693`).

   > **TO EXPAND when m5-d4b lands:** the terminal steps, the expected doctor output, and the state of the first instance's lock.

**Notes / evidence**

_Empty. Fill at run._

---

## 25 · Repository files and golden-regeneration note (m5-d12)

**Context:** Terminal: Setup. Binary: none for the greps; `npm run check` and `scripts/ci.sh` on the clean clone (operator). Working directory: /Volumes/DATA/Github/kirsch for the greps; the clean clone's location is TO EXPAND. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d12 lands:** the new files (`CONTRIBUTING.md`, `SECURITY.md`, `NOTICE`), and which branch O21 took for the `-update` sentences.

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
  ```

  Control, as commands (authored here). The pre-change text, which is `HEAD`'s copy, must hold the old wording, so each count must be at least 1:

  ```bash
  # authored here
  git -C /Volumes/DATA/Github/kirsch show HEAD:plan/spec/kirsch-plan.md | grep -c -e '-update'
  git -C /Volumes/DATA/Github/kirsch show HEAD:plan/spec/ui-spec-v0.1.md | grep -c -e '-update'
  ```

- [ ] If O21 chose `-update` instead: `grep -n -e '-update' internal/tui/regen_test.go` prints the flag definition (`plan/milestones/milestone-5.md:1101`). This box replaces the one above and runs only on that branch.
- [ ] `SECURITY.md` names the supported versions and the reporting channel O7 records (`plan/milestones/milestone-5.md:1102`). Command: `TO EXPAND`.
- [ ] Every claim in `SECURITY.md` traces to a code location or a test (`plan/milestones/milestone-5.md:1103`). Command: `TO EXPAND`. The review step checks each one.
- [ ] `NOTICE` credits Bubble Tea, Lip Gloss and Bubbles, and its list matches the direct requires in `go.mod` (`plan/milestones/milestone-5.md:1104`). Command: `TO EXPAND`.

**Hands-on checks**

1. (operator) Each command in `CONTRIBUTING.md` is run once on a clean clone and exits 0 (`plan/milestones/milestone-5.md:1099`). The milestone names `npm run check` and `scripts/ci.sh` as examples (`plan/milestones/milestone-5.md:1093`).

   > **TO EXPAND when m5-d12 lands:** the full command list in `CONTRIBUTING.md`, and the clean-clone location.

**Notes / evidence**

_Empty. Fill at run._

---

## 26 · `/new` busy state and turn fencing: C48, C49 (optional) (m5-d17b)

**Context:** Terminal: Setup. Binary: `go test` for the checks; `kirsch` for the C49 reproduction (TO EXPAND). Working directory: /Volumes/DATA/Github/kirsch. Workspace: `--workspace "$WORKSPACE"` on each reproduction launch. Key state: all key variables unset for the checks; key entered for this live step for the C49 reproduction only. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d17b lands:** the reproductions of C48 and C49 on the pre-fix code, and the generation counter as built.

**Waits on:** m5-d8 (shared `update.go`), m5-d17a (shared `app.go`), and O13 (scheduling), from `plan/milestones/milestone-5.md:121`.

**Key-state check before the live step.** Authored here. It must print `CONTROL PASS`, then `0` before any key is entered.

```bash
# authored here
keycount_control
keycount
```

**Automated checks**

- [ ] A test shows that pressing Enter while `/new` is pending shows no error card (`plan/milestones/milestone-5.md:1482`). Command: `TO EXPAND`.
- [ ] A test shows that a message carrying a cancelled turn's generation is dropped (`plan/milestones/milestone-5.md:1483`). Command: `TO EXPAND`.
- [ ] The C48 test fails on the pre-fix code, and the failing output is pasted (`plan/milestones/milestone-5.md:1485`). Command: `TO EXPAND`.

**Hands-on checks**

1. Reproduce C48 and C49 on the current code before any change. Record the command or test run and its output for each (`plan/milestones/milestone-5.md:1477`).

   The C49 reproduction contacts the provider: it is live, it needs a key, and it spends tokens. It needs turns, and there is no offline path yet. Enter the key only for that run, with the live-key block in *Setup* (`read -rs OPENCODE_API_KEY && export OPENCODE_API_KEY`), and only for the C49 reproduction command.

   > **TO EXPAND when m5-d17b lands:** the reproduction commands for each.

   **Key-state check, last** (authored here). The key does not outlive the live step. The count after the unset must print `CONTROL PASS`, then `0`.

   ```bash
   # authored here
   unset ANTHROPIC_API_KEY KIRSCH_ANTHROPIC_API_KEY OPENCODE_API_KEY KIRSCH_OPENCODE_API_KEY
   keycount_control
   keycount
   ```

**Notes / evidence**

_Empty. Fill at run._

---

## 27 · `doc/troubleshooting.md` and `doc/README.md` (m5-d14b)

**Context:** Terminal: Setup. Binary: `kirsch` for the reproductions (TO EXPAND). Working directory: `$WORKSPACE`, with `--workspace "$WORKSPACE"` on each launch. Workspace: `$WORKSPACE`. Key state: all key variables unset, except a deliberately invalid dummy key for the 401 reproduction, and key entered for this live step for the 403 reproduction, if the operator decides that case needs one. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d14b lands:** the doctor lines as emitted, each with its troubleshooting section.

**Waits on:** m5-d3a, m5-d3b, m5-d4a and m5-d4b (the doctor lines the page explains), m5-d13a and m5-d13c (the pages the index names), and O6, from `plan/milestones/milestone-5.md:117`.

**Key-state check before the live step.** Authored here. It must print `CONTROL PASS`, then `0` before any key is entered.

```bash
# authored here
keycount_control
keycount
```

**Automated checks**

- [ ] `grep -n -i -w -E 'milestone|ADR' doc/troubleshooting.md doc/README.md` prints nothing. Control: the printf line prints `an ADR` and not `headroom` (`plan/milestones/milestone-5.md:1303`, `:1308`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n -i -w -E 'milestone|ADR' doc/troubleshooting.md doc/README.md
  printf 'an ADR\nheadroom\n' | grep -i -w -E 'milestone|ADR'
  ```

- [ ] Every doctor line that m5-d3a, m5-d3b, m5-d4a and m5-d4b emit is explained in `doc/troubleshooting.md`. The step report lists the emitted lines and the page section for each (`plan/milestones/milestone-5.md:1306`). Command: `TO EXPAND`.
- [ ] The status lines at `doc/README.md:13`, `:14` and `:16` match the files that exist (`plan/milestones/milestone-5.md:1307`). Command: `TO EXPAND`.

**Hands-on checks**

1. (operator) Each failure the troubleshooting page covers is reproduced, and the advice fixes it (`plan/milestones/milestone-5.md:1309`).

   The 401 reproduction uses a deliberately invalid dummy key, never a real one. Authored here: `export OPENCODE_API_KEY=dummy-invalid-key`, then run the reproduction. It still contacts the provider, which refuses the key. The 403 reproduction needs a key the provider refuses for the model. It takes a real key only if the operator decides it must, entered with the live-key block in *Setup*. Only a step that genuinely needs the model takes a real key. The no-key case needs no key.

   > **TO EXPAND when m5-d14b lands:** the failures the troubleshooting page covers, as the step report lists them, and how each is reproduced. Do not reproduce the no-key case with a key set.

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

**Context:** Terminal: Setup. Binary: `kirsch resume` and `/compact` (TO EXPAND). Working directory: `$WORKSPACE`, with `--workspace "$WORKSPACE"` on each launch. Workspace: `$WORKSPACE`. Key state: all key variables unset until the TO EXPAND below is resolved; then key entered for this live step. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d18 lands:** the totals loader, and the compaction % values before and after `/compact` for the fixture.

**Waits on:** m5-d17a (file ownership only; both own `internal/app/app.go`), and O13 (scheduling), from `plan/milestones/milestone-5.md:122`.

**Key-state check before the live step.** Authored here. It must print `CONTROL PASS`, then `0` before any key is entered.

```bash
# authored here
keycount_control
keycount
```

**Automated checks**

- [ ] A test shows that after resume the status bar shows the session's totals before any new usage (`plan/milestones/milestone-5.md:1520`). Command: `TO EXPAND`.
- [ ] A test shows that the totals equal the sum of the session's `usage` events in a fixture log (`plan/milestones/milestone-5.md:1521`). Command: `TO EXPAND`.
- [ ] A test shows the compaction % before `/compact` is 60%, and after it is the value `Compact` returns, which differs from 60%. Both values are pasted (`plan/milestones/milestone-5.md:1522`). Command: `TO EXPAND`.

**Hands-on checks**

1. Run a turn, quit, run `kirsch resume`, then `/compact` and `/status` (`plan/milestones/milestone-5.md:1524`).

   > **TO EXPAND when m5-d18 lands:** whether a turn here reaches a provider. The milestone does not say whether a fake provider is used. Until this is expanded, do not run this step with a key set.

   **Key-state check, last** (authored here). The key does not outlive the live step. The count after the unset must print `CONTROL PASS`, then `0`.

   ```bash
   # authored here
   unset ANTHROPIC_API_KEY KIRSCH_ANTHROPIC_API_KEY OPENCODE_API_KEY KIRSCH_OPENCODE_API_KEY
   keycount_control
   keycount
   ```

**Notes / evidence**

_Empty. Fill at run. Record both compaction % values._

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

**Context:** Terminal: Setup. Binary: `go test`. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d21a lands, or record "no step":** the figures, dates and cache-write prices changed, from the m5-d20 decision.

**Waits on:** m5-d20 (the X5 and C9 decisions), from `plan/milestones/milestone-5.md:126`. Runs only if the operator changed the figures or dates (`plan/milestones/milestone-5.md:1661`).

**Automated checks**

- [ ] The table matches the recorded decision, and `go test ./internal/provider/` passes (`plan/milestones/milestone-5.md:1665`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test ./internal/provider/
  ```

- [ ] The review step is recorded in `plan/PROGRESS.md` (`plan/milestones/milestone-5.md:1666`). Process check.

**Hands-on checks**

1. Quote the recorded decision into the brief (`plan/milestones/milestone-5.md:1662`).

**Notes / evidence**

_Empty. Fill at run, or write "no step" with the m5-d20 decision it rests on._

---

## 31 · Help footer and goldens (conditional) (m5-d21b)

**Context:** Terminal: Setup. Binary: `go test`; `npm run screens`. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d21b lands, or record "no step":** the footer text O16 records, and the fences that moved.

**Waits on:** m5-d20 (O16 recorded), and O23 (golden handling), from `plan/milestones/milestone-5.md:127`. The footer is at `internal/tui/modal.go:308`.

**Before any edit.** Save a verbatim copy of `plan/spec/kirsch-ui-screens.md` to `.claude/memory/workspace/` before the edit, as the golden procedure requires (`plan/milestones/milestone-5.md:50`). Name it with the m5-d21b prefix, for example `m5-d21b-kirsch-ui-screens.pre.md`. This name is chosen here.

**Automated checks**

- [ ] Golden check, procedure step 5 (`plan/milestones/milestone-5.md:53`):

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test ./internal/tui
  cd /Volumes/DATA/Github/kirsch && npm run screens
  ```

- [ ] The footer matches O16, and every moved golden is listed in the step report (`plan/milestones/milestone-5.md:1695`). Command: `TO EXPAND`.

**Hands-on checks**

1. (operator) The golden diff is reviewed by the operator (`plan/milestones/milestone-5.md:1688`).

   > **TO EXPAND when m5-d21b lands:** the moved fences, by header, and the review record.

**Notes / evidence**

_Empty. Fill at run._

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

**Context:** Terminal: Setup. Binary: none (grep). Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d21d lands, or record "no step":** the disclosure line in `SECURITY.md`, traced to the cited lines.

**Waits on:** m5-d20 (the C24 decision), and m5-d12 (file ownership of `SECURITY.md`), from `plan/milestones/milestone-5.md:129`.

**Automated checks**

- [ ] `SECURITY.md` states the 400 log behaviour, and each claim traces to a cited line (`plan/milestones/milestone-5.md:1759`). Command: `TO EXPAND`.
- [ ] The sentence that says a 400 writes the full request body now says it is a disclosed limit (`plan/milestones/milestone-5.md:1760`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n 'full request body' SECURITY.md
  ```

  The line printed must contain `disclosed`. On the file m5-d12 leaves, it does not, so this box fails until the update is made.

**Hands-on checks**

1. Run only if the operator disclosed under m5-d20. Otherwise record "no step" (`plan/milestones/milestone-5.md:1756`).

**Notes / evidence**

_Empty. Fill at run._

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

**Context:** Terminal: Setup. Binary: none (grep, review). Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d21f lands, or record "no step":** the §7 lines amended in `plan/spec/kirsch-plan.md` (`:340–359`), and the session events listed.

**Waits on:** m5-d20 (the C43 and C46 decisions), and m5-d5 (file ownership of §7), from `plan/milestones/milestone-5.md:131`.

**Automated checks**

- [ ] §7 matches the events the code writes, and the review step is recorded (`plan/milestones/milestone-5.md:1817`). Command: `TO EXPAND`.

**Hands-on checks**

1. Run only if the operator amends §7. If not, record "no step" (`plan/milestones/milestone-5.md:1814`).

**Notes / evidence**

_Empty. Fill at run._

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
