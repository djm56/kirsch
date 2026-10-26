# Manual test — Milestone 5

Kirsch Milestone 5 is polish and release: the narrow-width status bar, the `kirsch doctor` command, `--version` and `--help`, the release tooling (build wrapper, goreleaser configuration, release workflow), the repository files, the user documentation under `doc/`, and the operator decisions that gate the conditional follow-through. The deliverables are defined in `plan/milestones/milestone-5.md`.

> **Status: skeleton.** This document was written at planning time (mission-20261023-06, 2026-10-23). It is expanded as each deliverable lands. No command in it has been run. Every block marked `TO EXPAND` is a placeholder, and the expansion belongs to the step that lands the deliverable. To run one section without reading the whole document, follow *Testing a deliverable — the short path* in [`../../milestones/milestone-5-start-here.md`](../../milestones/milestone-5-start-here.md).

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
- **Where expansions land.** Expansions land in this master file first. The operator's working copy, `plan/testing/manual/user-testing/milestone-5-donovan.md`, is then copied from it under DIR-020. The working copy keeps this file's wording.

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

The unset line clears the key variables, as the m5-d16 box does (`plan/milestones/milestone-5.md:1408`), and clears `MY_TOKEN` and `T_BROKEN`, which earlier sections export. `KIRSCH_LIVE_MODEL` is unset as Milestone 4 does (`plan/testing/manual/milestone-4.md:40`).

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

**Cleanup.** If you ran section 7, first run `rm -rf -- "$T_BROKEN"`, because the block below unsets `T_BROKEN`. Then unset the variables when a session ends, and check the count. Authored here. It must print `CONTROL PASS`, then `0`.

```bash
# authored here
unset ANTHROPIC_API_KEY KIRSCH_ANTHROPIC_API_KEY OPENCODE_API_KEY KIRSCH_OPENCODE_API_KEY KIRSCH_LIVE_MODEL MY_TOKEN T_BROKEN
unset XDG_CONFIG_HOME XDG_STATE_HOME
keycount_control
keycount
```

Removing the scratch directories is the operator's step, and agents do not run it. If you ran section 7, first run `rm -rf -- "$T_BROKEN"` before the Cleanup block, because Cleanup unsets `T_BROKEN`. After Cleanup, copy the removal line from `plan/testing/manual/milestone-4.md:60`: a recursive `rm` of `$T_CFG`, `$T_STATE` and `$WORKSPACE`. It does not remove `$T_BROKEN`. The variable `MY_TOKEN` is unset above.

---

## 1 · Narrow-width status-bar fallback (m5-d1)

**Context:** Terminal: Setup. Binary: none for the automated checks; `go run ./cmd/kirsch --workspace "$WORKSPACE"` for hands-on step 2 (TO EXPAND; the launch line is authored here). Working directory: /Volumes/DATA/Github/kirsch. Workspace: `--workspace "$WORKSPACE"` on the hands-on launch; n/a for the checks. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d1 lands:** the band rule and the ladder that O1 records, and the sized fences that moved.

**Waits on:** O1 (what the narrow bar shows) and O23 (golden handling), from `plan/milestones/milestone-5.md:97`. Golden procedure: `plan/milestones/milestone-5.md:39–53`. This is the defect the Milestone 4 walkthrough marked ❌ (`plan/testing/manual/user-testing/milestone-4-donovan.md:1042`).

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

- [ ] Sized fences at 80 columns and wider are unchanged. The command extracts them from the saved copy and from the file, and compares them. Empty diff output means pass (`plan/milestones/milestone-5.md:441–458`). The command needs `$PRE`, the saved copy that Hands-on step 1 creates. Run that step first.

  ```bash
  # authored here
  PRE=/Volumes/DATA/Github/kirsch/.claude/memory/workspace/m5-d1-kirsch-ui-screens.pre.md
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
  rm -f "$CTRL"
  ```

  The first output line is the count of wide fences. The control `diff` must print a difference. Without that, an empty diff proves nothing.

- [ ] At 60–79 columns and below 60, the status bar shows exactly what O1 specifies. `internal/tui/turn_messages_test.go:619–627` pins that behaviour and passes (`plan/milestones/milestone-5.md:438–439`). Command: `TO EXPAND`. The milestone gives no command line for this test.

- [ ] ui-spec §2.2 and §2.3 agree with the code (`plan/milestones/milestone-5.md:440`). Command: `TO EXPAND`. Review step, not a command.

**Hands-on checks**

1. Before any change, record the sized fences below 80 columns by fence header. Save the pre-change copy of `plan/spec/kirsch-ui-screens.md` to `.claude/memory/workspace/m5-d1-kirsch-ui-screens.pre.md` first (`plan/milestones/milestone-5.md:428`).

   ```bash
   grep -n '^```text [0-9]*×' /Volumes/DATA/Github/kirsch/plan/spec/kirsch-ui-screens.md | awk -F'[ ×]' '$2+0 < 80'
   ```

   If the list is empty, record "no golden below 80 columns" and change no fence.

2. Run the status bar at each width band that O1 names, and compare the bar with the O1 text.

   > **TO EXPAND when O1 is recorded:** the terminal widths to walk (60–79 and below 60), the expected bar text at each, and the terminal size command (`echo $COLUMNS $LINES`).

**Notes / evidence**

_Empty. Fill at run: command, output, step report reference._

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

  Expected, in order: the help text in *What landed*, then `0`. Then `0` (nothing on stderr for `--help`). Then a non-zero count (`517` on 2026-10-26), the control: it proves the same capture does see stderr. Then `0` (`resume --help` exits 0). Then `0` (`doctor` is not in the help yet; m5-d3a adds it later). Then `2`, the control: it proves the same grep finds a word that is there. `grep -c` exits 1 when the count is 0; the printed number is what counts.

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

  The error appears twice, once from the flag parser and once from `kirsch:` in `main` (`cmd/kirsch/main.go:74`), the same as before this change.

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

**Context:** Terminal: Setup. Binary: `go test` for the automated checks; `kirsch resume` for hands-on step 1 (TO EXPAND). Working directory: /Volumes/DATA/Github/kirsch. Workspace: `--workspace "$WORKSPACE"` on the hands-on launches, the first and the resume. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d5 lands:** the new session event (O11), the §7 amendment, and whether `/new` records anything (it does not, unless O11 says so).

**Waits on:** O11 (event name and §7 amendment), from `plan/milestones/milestone-5.md:103`. ADR 0006 makes revocability a requirement (`plan/adr/0006-session-scoped-approvals.md:45, :67–69`).

**Key-state check, first.** Authored here. Run it before any kirsch launch in this section. It must print `CONTROL PASS`, then `0`.

```bash
# authored here
keycount_control
keycount
```

**Automated checks**

- [ ] `go test -race ./internal/session/ ./internal/app/` passes in full, with no exclusion. `TestRunTurnApprovalToolUsesApprovalPath` is included (`plan/milestones/milestone-5.md:732`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test -race ./internal/session/ ./internal/app/
  ```

- [ ] A round-trip test grants a command, clears the grants, reopens the session log, and asserts that no grants are returned (`plan/milestones/milestone-5.md:727`). Command: `TO EXPAND`.
- [ ] A clear writes the new event into the session log, and the test reads the file (`plan/milestones/milestone-5.md:728`). Command: `TO EXPAND`.
- [ ] `LoadGrants` returns the grants before the clear event and none after it (`plan/milestones/milestone-5.md:729`). Command: `TO EXPAND`.
- [ ] A session log with no clear event restores its grants as before (`plan/milestones/milestone-5.md:730`). Command: `TO EXPAND`.
- [ ] The §7 amendment lists the new event type (`plan/milestones/milestone-5.md:731`). Command: `TO EXPAND`. Review step.

**Hands-on checks**

1. Launch `kirsch --workspace "$WORKSPACE"`. The first launch needs the flag: it is where the grant is made, and without the flag the checkout becomes the workspace. Grant a command with `a`. Run `/approvals` and confirm the grant is listed. That is the positive control, and it shows the grant exists before the clear. If nothing is listed, stop: the run would prove nothing. Clear with `/approvals`, quit, run `kirsch resume --workspace "$WORKSPACE"`, then run `/approvals`. Expected after the clear and the resume: no grants listed (`plan/milestones/milestone-5.md:733`, operator). The session directory is keyed by the workspace hash (`internal/session/layout.go:78–79`), so the launches must name `$WORKSPACE`. A resume without the flag reads a different directory and lists nothing, which would pass for the wrong reason.

   > **TO EXPAND when m5-d5 lands:** how a grant is created without a live model turn. `milestone-5.md:733` does not say. Until this is expanded, do not run this step with any key variable set.

**Notes / evidence**

_Empty. Fill at run._

---

## 4 · Terminal capability and resize tests (m5-d9)

**Context:** Terminal: Setup. Binary: `go test`. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d9 lands:** the `DetectCaps` cases written, the O18 decision recorded for empty `NO_COLOR`, and whether `view_smoke_test.go` was extended.

**Waits on:** O18 (empty `NO_COLOR`: keep the code, or treat empty as unset), from `plan/milestones/milestone-5.md:108`.

**Automated checks**

- [ ] `internal/tui/styles_test.go` covers each case in `plan/milestones/milestone-5.md:931–937`, and the positive control returns colour on (`plan/milestones/milestone-5.md:942`). Command: `TO EXPAND`. The milestone names the file and the cases but no command line.
- [ ] The truly-unset `NO_COLOR` case passes, and the test asserts that the prior value is restored (`plan/milestones/milestone-5.md:943`). Command: `TO EXPAND`.
- [ ] The O18 behaviour is pinned by the empty-`NO_COLOR` case (`plan/milestones/milestone-5.md:944`). Command: `TO EXPAND`.
- [ ] (optional) The invariants sweep also runs with a modal and a confirm prompt open (`plan/milestones/milestone-5.md:945`). Command: `TO EXPAND`.

**Hands-on checks**

1. (operator) A real resize with a modal open (`plan/milestones/milestone-5.md:946`).

   > **TO EXPAND when m5-d9 lands:** the terminal, the modal to open, and the expected result.

**Notes / evidence**

_Empty. Fill at run._

---

## 5 · `doc/configuration.md` (m5-d13a)

**Context:** Terminal: Setup. Binary: none (greps on `doc/`). Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d13a lands:** the key table as written, with the scope of each key, and the one sentence stating that v0.1 reads no configuration from command-line flags (`plan/milestones/milestone-5.md:1139`).

**Waits on:** O6 (who writes `doc/`; recommendation: `@lambert` writes, `@apone` reviews the claims against the code), from `plan/milestones/milestone-5.md:113`.

**Automated checks**

- [ ] Every key in `internal/config/config.go:57–80` and every endpoint field in `config.go:219–259` is listed, with its default and valid values (`plan/milestones/milestone-5.md:1143`). The mechanical check is m5-d13b (its section below). Command: `TO EXPAND`.
- [ ] `grep -n -i 'flag' doc/configuration.md` prints at least one line. Every line it prints is a sentence that promises no config flag (`plan/milestones/milestone-5.md:1144`). Paste the output.

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n -i 'flag' doc/configuration.md
  ```

- [ ] `grep -n -i -w -E 'milestone|ADR' doc/configuration.md` prints nothing (`plan/milestones/milestone-5.md:1145`). Control: the printf line must print `an ADR` and not `headroom`.

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -n -i -w -E 'milestone|ADR' doc/configuration.md
  printf 'an ADR\nheadroom\n' | grep -i -w -E 'milestone|ADR'
  ```

**Hands-on checks**

1. Read the page as a user, not a builder. Confirm that `KIRSCH_DEBUG` and the config path are present (`plan/milestones/milestone-5.md:1136`), and that API keys are read from the variables named by `api_key_env` (`plan/milestones/milestone-5.md:1138`).
2. > **TO EXPAND when m5-d13a lands:** the review of each documented default against `Defaults()` (`internal/config/config.go:84–118`), and the reviewer's citations.

**Notes / evidence**

_Empty. Fill at run._

---

## 6 · Safety-test repair for `run_command`: C30, C35, C36 (m5-d19a)

**Context:** Terminal: Setup. Binary: `go test`; `kirsch` running `run_command` for hands-on step 2 (TO EXPAND). Working directory: /Volumes/DATA/Github/kirsch. Workspace: `--workspace "$WORKSPACE"` on the hands-on launch. Key state: all key variables unset. Config state: Setup, plus the `[policy]` block that hands-on step 1 appends, which step 3 removes again, so the Setup config is restored before the section ends. The guard reruns after the append and after the restore.

**What landed**

> **TO EXPAND when m5-d19a lands:** the mutant named for each rewritten C30 test, the test that fails on it, and the failing output.

**Waits on:** O13 (scheduling of the safety group). No other dependency (`plan/milestones/milestone-5.md:123`).

**Key-state check, first.** Authored here. Run it before any kirsch launch in this section. It must print `CONTROL PASS`, then `0`.

```bash
# authored here
keycount_control
keycount
```

**Automated checks**

- [ ] Each C30 test fails under its named mutant and passes on the real code (`plan/milestones/milestone-5.md:1557`). The tests are `TestRunCommandTimeoutProcessGroupDead`, `TestRunCommandCancellationProcessGroupDead` and `TestRunCommandStdinIsDevNull`. Command and mutant names: `TO EXPAND`.
- [ ] The lower-case cases assert that `my_token`, `api_key` and `db_secret` are not stripped, and the suite passes (`plan/milestones/milestone-5.md:1558`). Command: `TO EXPAND`.
- [ ] The C36 test asserts that the blank approval card for an unspecified operation shows the operation name and no empty field (`plan/milestones/milestone-5.md:1559`). Command: `TO EXPAND`.
- [ ] `go test ./internal/tool/ ./internal/tui/` passes (`plan/milestones/milestone-5.md:1560`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test ./internal/tool/ ./internal/tui/
  ```

**Hands-on checks**

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

   > **TO EXPAND when m5-d19a lands:** how the child command is invoked without a live model turn. `milestone-5.md:1562` does not say. Until this is expanded, do not run this step with any key variable set.

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

_Empty. Fill at run._

---

## 7 · Doctor: dispatch, report grammar, version line, exit codes, arch entry (m5-d3a)

**Context:** Terminal: Setup. Binary: `kirsch doctor` (launch command TO EXPAND). Working directory: `$T_CFG`, outside any Git repository, for the outside-repository run; /Volumes/DATA/Github/kirsch for the `go test` checks. Workspace: n/a for the outside-repository run. Key state: all key variables unset. Config state: Setup, unchanged. The broken-config run uses its own directory, set up in hands-on step 1.

**What landed**

> **TO EXPAND when m5-d3a lands:** the report grammar and exit codes as O5 records them, and the `doctor` help line.

**Waits on:** m5-d2 (help text and flag dispatch order) and O5 (doctor design: package placement, output grammar, fail versus warn, exit codes, side-effect policy), from `plan/milestones/milestone-5.md:99`.

**Key-state and repository checks, first.** Authored here. It must print `CONTROL PASS`, then `0`. The first `git` line must exit non-zero, and the last must print `true`.

```bash
# authored here
keycount_control
keycount
git -C "$T_CFG" rev-parse --is-inside-work-tree; echo "exit=$?"
git -C "$WORKSPACE" rev-parse --is-inside-work-tree
```

**Automated checks**

- [ ] `kirsch doctor` is dispatched ahead of `workspace.Detect` and config load (`plan/milestones/milestone-5.md:544`). Command: `TO EXPAND`.
- [ ] The first doctor line shows the version (`plan/milestones/milestone-5.md:545`). Command: `TO EXPAND`.
- [ ] The help output names `doctor` (`plan/milestones/milestone-5.md:546`). Command: `TO EXPAND`.
- [ ] Exit status matches O5 for each status O5 defines (`plan/milestones/milestone-5.md:547`). Command: `TO EXPAND`.
- [ ] The `internal/arch` test passes with the new package entry (`plan/milestones/milestone-5.md:548`). The milestone names `internal/arch/imports_test.go` but gives no command line. Command: `TO EXPAND`.
- [ ] `plan/spec/architecture.md:31–49` names the new package (`plan/milestones/milestone-5.md:549`). Review step.

**Hands-on checks**

1. Run `kirsch doctor` outside a repository and with a broken `config.toml`. Expected: it reports both, and it does not exit before reporting (`plan/milestones/milestone-5.md:536`). The broken config goes in its own directory, so the Setup config is untouched:

   ```bash
   # authored here
   export T_BROKEN=$(mktemp -d)
   mkdir -p "$T_BROKEN/kirsch"
   printf '[session\nstorage_dir = "%s"\n' "$T_STATE/sessions" > "$T_BROKEN/kirsch/config.toml"
   cd "$T_CFG"
   ```

   Launch doctor with `XDG_CONFIG_HOME="$T_BROKEN"` set on that same command. The launch command is TO EXPAND. Run the guard afterwards.

   > **TO EXPAND when O5 is recorded:** the commands to run, the expected lines, and the exit status for each.

**Notes / evidence**

_Empty. Fill at run._

---

## 8 · Composer paste path and caret rows (m5-d6)

**Context:** Terminal: Terminal.app and iTerm2, each after rerunning Setup in that window. Binary: `go test` for the automated checks; `kirsch` for the paste check (TO EXPAND). Working directory: /Volumes/DATA/Github/kirsch. Workspace: `--workspace "$WORKSPACE"` on the paste launch. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d6 lands:** the O10 choice (scroll to the caret, or wrap) and the clamp changes it required.

**Waits on:** O10 (composer past five rows: scroll to the caret, or wrap) and m5-d1 for file ownership of `view.go` only if O10 is wrap, from `plan/milestones/milestone-5.md:104`.

**Key-state check, first.** Authored here. Run it before any kirsch launch in this section. It must print `CONTROL PASS`, then `0`.

```bash
# authored here
keycount_control
keycount
```

**Automated checks**

- [ ] A multi-line paste with indentation lands verbatim (`plan/milestones/milestone-5.md:772`). Command: `TO EXPAND`.
- [ ] A paste over 8 KB shows the y/n prompt. Accept and decline are both tested (`plan/milestones/milestone-5.md:773`). Command: `TO EXPAND`.
- [ ] A pasted multi-line text starting with `/` is not a command (`plan/milestones/milestone-5.md:774`). Command: `TO EXPAND`.
- [ ] A 12-line paste leaves the caret row visible (`plan/milestones/milestone-5.md:775`). Command: `TO EXPAND`.
- [ ] Existing composer tests pass (`plan/milestones/milestone-5.md:776`). Command: `TO EXPAND`.

**Hands-on checks**

1. (operator) Paste a 20-line trace in Terminal.app and in iTerm2. Expected: newlines are kept and nothing acts as a key (`plan/milestones/milestone-5.md:777`). Run Setup in each window first.

   > **TO EXPAND when m5-d6 lands:** the trace to paste, and what the operator records for each terminal.

**Notes / evidence**

_Empty. Fill at run._

---

## 9 · Build wrapper and version proof (m5-d10a)

**Context:** Terminal: Setup. Binary: `bin/kirsch`, built by `scripts/build.sh` into /Volumes/DATA/Github/kirsch/bin. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d10a lands:** the wrapper's behaviour with no argument, as O4 records it.

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

- [ ] With no argument, the version falls back the way O4 records (`plan/milestones/milestone-5.md:982`). Command: `TO EXPAND` (no-argument run, once O4 is recorded).

**Hands-on checks**

1. No hands-on steps are given in the milestone. Milestone text: "Prove the version with a known value" (`plan/milestones/milestone-5.md:977`). The `bin/kirsch` run in the automated check starts from the checkout. This run takes the absolute path from `$T_CFG`, outside the checkout. Authored here:

   ```bash
   # authored here
   cd "$T_CFG" && /Volumes/DATA/Github/kirsch/bin/kirsch --version; echo "exit=$?"
   ```

**Notes / evidence**

_Empty. Fill at run._

---

## 10 · goreleaser config and target-compile check (m5-d10b)

**Context:** Terminal: Setup. Binary: none for the automated checks; `goreleaser` for the snapshot run (operator). Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d10b lands:** the `.goreleaser.yaml` as written, the CI target-check method O20 chose, and each target's compile result.

**Waits on:** m5-d2 (ldflags variable), O2 (goreleaser shape), O4 (version string), O20 (target-compile method), from `plan/milestones/milestone-5.md:110`. The target list is at `plan/spec/kirsch-plan.md:481`. Compilation is unverified until the check runs.

**Automated checks**

- [ ] `.goreleaser.yaml` builds only `./cmd/kirsch`, with `CGO_ENABLED=0` and `-X main.version={{.Version}}` (`plan/milestones/milestone-5.md:1017`). Command: `TO EXPAND`.
- [ ] Every target listed at `plan/spec/kirsch-plan.md:481` compiles. The step report lists each target and its result (`plan/milestones/milestone-5.md:1019`). Command: `TO EXPAND`, per O20. The O20 option is a loop of the form `GOOS=… GOARCH=… go build` (`plan/milestones/milestone-5.md:82`), or `goreleaser build --snapshot`.
- [ ] A forced failure in one target makes the CI check exit non-zero. The forced failure is reverted and recorded (`plan/milestones/milestone-5.md:1018`). Command: `TO EXPAND`.

**Hands-on checks**

1. Record each listed target and its result in the step report (`plan/milestones/milestone-5.md:1013`).

   > **TO EXPAND when O20 is recorded:** the target list in the form the check uses, and the forced-failure method.

2. (operator) `goreleaser release --snapshot --clean` produces `dist/` with archives and a checksums file, and the native binary reports its version (`plan/milestones/milestone-5.md:1020`). `dist/` is gitignored (`.gitignore:10`, `milestone-5.md:996`).

   ```bash
   cd /Volumes/DATA/Github/kirsch && goreleaser release --snapshot --clean
   ```

**Notes / evidence**

_Empty. Fill at run._

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

**Context:** Terminal: Setup. Binary: `kirsch doctor` (launch command TO EXPAND). Working directory: `$T_CFG` for the outside-repository run; /Volumes/DATA/Github/kirsch for the PATH shim. Workspace: `--workspace "$WORKSPACE"` on the `--workspace` run. Key state: all key variables unset. Config state: Setup, unchanged. A broken config uses its own directory, as in m5-d3a.

**What landed**

> **TO EXPAND when m5-d3b lands:** the probe lines as emitted, and the PATH shim used for the missing-tool tests.

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

- [ ] With the PATH shim lacking `rg`, doctor prints a warn line that names the pure-Go fallback (`plan/milestones/milestone-5.md:588`). Command: `TO EXPAND`.
- [ ] With the PATH shim lacking `git`, doctor prints a git-missing line and not "not in a Git repository" (`plan/milestones/milestone-5.md:589`). Command: `TO EXPAND`.
- [ ] Outside a repository, doctor prints "not in a Git repository" and every check that does not need one (`plan/milestones/milestone-5.md:590`). Command: `TO EXPAND`.
- [ ] A broken `config.toml` produces a line naming the file and the parse error, and doctor does not exit on it (`plan/milestones/milestone-5.md:591`). Command: `TO EXPAND`.
- [ ] An unknown config key produces a warn line (`plan/milestones/milestone-5.md:592`). Command: `TO EXPAND`.
- [ ] `--workspace` is honoured (`plan/milestones/milestone-5.md:593`). Command: `TO EXPAND`.

**Hands-on checks**

1. The missing-tool tests use a PATH shim, not an emptied PATH (DIR-009, `plan/milestones/milestone-5.md:585`). The shim holds symlinks to every executable on the original PATH except the tool under test. Its construction is `TO EXPAND`. The control, quoted from `plan/milestones/milestone-5.md:585`: under the same shim PATH, `LookPath("git")` succeeds in the `rg` test, and `LookPath("rg")` still fails. A missing-`rg` result does not count until this control holds.
2. > **TO EXPAND when m5-d3b lands:** the commands for the outside-repository run and the broken-config run, and the expected lines.

**Notes / evidence**

_Empty. Fill at run._

---

## 13 · Soft-wrap fidelity: `wrap()` (m5-d7a)

**Context:** Terminal: Setup. Binary: `go test`; `KIRSCH_REGEN=1 go test` for the grids. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d7a lands:** the callers whose output moved, and the golden fences that moved, listed by header.

**Waits on:** O23 (golden handling, recorded first), m5-d1 (both touch `view.go`), and m5-d6 (`composer.go`, `update.go` order), from `plan/milestones/milestone-5.md:105`.

**Before any edit.** Save the pre-change copy of `plan/spec/kirsch-ui-screens.md` to `.claude/memory/workspace/m5-d7a-kirsch-ui-screens.pre.md`, after m5-d1 has landed and before this edit (`plan/milestones/milestone-5.md:809`). The milestone requires the save but gives no name. This name is chosen here.

**Automated checks**

- [ ] A pasted 200-column indented trace keeps its indentation in the transcript (`plan/milestones/milestone-5.md:812`). Command: `TO EXPAND`.
- [ ] Runs of interior spaces survive in the transcript (`plan/milestones/milestone-5.md:813`). Command: `TO EXPAND`.
- [ ] `wrap` has direct tests in `internal/tui/wrap_test.go`, and they pass (`plan/milestones/milestone-5.md:814`). Command: `TO EXPAND`.
- [ ] Golden check, procedure step 5 (`plan/milestones/milestone-5.md:53`):

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test ./internal/tui
  cd /Volumes/DATA/Github/kirsch && npm run screens
  ```

**Hands-on checks**

1. Record the sized fences that move by header (`plan/milestones/milestone-5.md:809`). The pre-change copy is named in *Before any edit* above.

2. Regenerate and reconcile, by the procedure at `plan/milestones/milestone-5.md:49–53`:

   ```bash
   cd /Volumes/DATA/Github/kirsch && KIRSCH_REGEN=1 go test ./internal/tui -run TestGenerateGrids
   ```

   The grids go to `/tmp/kirsch-grids.txt`. A person reconciles each affected fence in `plan/spec/kirsch-ui-screens.md`.

3. (operator) A person reviews the golden diff before the step closes (`plan/milestones/milestone-5.md:816`).

**Notes / evidence**

_Empty. Fill at run. List every moved golden by header._

---

## 14 · Paste debounce (m5-d8)

**Context:** Terminal: Setup. Binary: `go test` for the checks; `kirsch` for the hands-on paste check (TO EXPAND). Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a for the checks; `--workspace "$WORKSPACE"` on the hands-on launch. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d8 lands:** the O9 branch taken (implement the 20 ms coalescing, or amend the spec and delete `PasteDebounce`).

**Waits on:** m5-d6 (the paste path) and O9 (paste debounce), from `plan/milestones/milestone-5.md:107`. Bracketed paste already gives one message in the normal case (`github.com/charmbracelet/bubbletea@v1.3.10/key_sequences.go:106–118`, the Bubble Tea module pinned at `go.mod:8`). The milestone now cites `github.com/charmbracelet/bubbletea@v1.3.10/key_sequences.go:106–118` (`plan/milestones/milestone-5.md:71`, `:868`, `:905`), the same file this section cites.

**Before any edit.** Save verbatim copies of `internal/tui/styles.go` to `.claude/memory/workspace/m5-d8-styles.pre.go` and of `plan/spec/ui-spec-v0.1.md` to `.claude/memory/workspace/m5-d8-ui-spec.pre.md` (`plan/milestones/milestone-5.md:880`). The controls below run on these copies.

**Key-state check, first.** Authored here. Run it before any kirsch launch in this section. It must print `CONTROL PASS`, then `0`.

```bash
# authored here
keycount_control
keycount
```

**Automated checks**

- [ ] Implement branch: a test with runes arriving within 20 ms inserts once (`plan/milestones/milestone-5.md:890`). Command: `TO EXPAND`.
- [ ] Amend branch: `grep -rn PasteDebounce internal/` prints nothing, and `grep -n 'Paste debounce' plan/spec/ui-spec-v0.1.md` prints nothing (`plan/milestones/milestone-5.md:890`).

  ```bash
  cd /Volumes/DATA/Github/kirsch && grep -rn PasteDebounce internal/
  cd /Volumes/DATA/Github/kirsch && grep -n 'Paste debounce' plan/spec/ui-spec-v0.1.md
  ```

  The pattern is case-sensitive on purpose. `grep -n -i debounce` also matches the unrelated "Never debounced" row.

- [ ] Controls, run on the saved copies before the edit (`plan/milestones/milestone-5.md:892–898`). The first prints one line, the declaration. The second prints two lines, one in §7.2 and one in §11.

  ```bash
  # authored here
  PRE=/Volumes/DATA/Github/kirsch/.claude/memory/workspace
  grep -n PasteDebounce "$PRE/m5-d8-styles.pre.go"
  grep -n 'Paste debounce' "$PRE/m5-d8-ui-spec.pre.md"
  ```

- [ ] With bracketed paste off, a 5 KB paste lands whole (`plan/milestones/milestone-5.md:899`). Command: `TO EXPAND`.

**Hands-on checks**

1. (operator) Bracketed paste off, paste 5 KB, and the UI stays responsive (`plan/milestones/milestone-5.md:901`).

   > **TO EXPAND when O9 is recorded:** how bracketed paste is turned off in each terminal, and the expected result.

**Notes / evidence**

_Empty. Fill at run._

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

**Context:** Terminal: Setup. Binary: `go test`. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged. The removal-and-revert edits `doc/configuration.md`, not the config.

**What landed**

> **TO EXPAND when m5-d13b lands:** the test name, and the key list it derives from `config.go:57–80` and `config.go:219–259`.

**Waits on:** m5-d13a (the page the test reads), from `plan/milestones/milestone-5.md:114`.

**Automated checks**

- [ ] The test passes against the current page. The output is pasted into the step report (`plan/milestones/milestone-5.md:1178`). Command: `TO EXPAND`. The milestone names `internal/config/key_coverage_test.go` and gives no command line.
- [ ] Removing one key from `doc/configuration.md` makes the test fail. The removal is demonstrated once and reverted (`plan/milestones/milestone-5.md:1179`). Command: `TO EXPAND`.

**Hands-on checks**

1. The removal-and-revert control (`plan/milestones/milestone-5.md:1175`).

   > **TO EXPAND when m5-d13b lands:** which key is removed, the exact edit, and the revert. Record both runs.

**Notes / evidence**

_Empty. Fill at run._

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

**Context:** Terminal: Setup. Binary: `go test` for the checks; `kirsch doctor` for hands-on step 1 (TO EXPAND). Working directory: /Volumes/DATA/Github/kirsch for the checks; `$T_CFG`, outside any repository, for the doctor run. Workspace: n/a for the doctor run. Key state: all key variables unset; hands-on step 1 sets one dummy value, never a real key. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d4a lands:** the lines as emitted, and the skipped-line text when config did not load.

**Waits on:** m5-d3b (the config-load result) and O5, from `plan/milestones/milestone-5.md:101`.

**Before any edit.** Save verbatim copies of `internal/tool/searchcode.go` to `.claude/memory/workspace/m5-d4a-searchcode.pre.go` and of `internal/app/app.go` to `.claude/memory/workspace/m5-d4a-app.pre.go` (`plan/milestones/milestone-5.md:624`). The acceptance box compares against these copies, not against HEAD. m5-d5 (wave 1) also edits `app.go`, so a diff against HEAD would show its code.

**Key-state and repository checks, first.** Authored here. It must print `CONTROL PASS`, then `0`. The first `git` line must exit non-zero, and the last must print `true`.

```bash
# authored here
keycount_control
keycount
git -C "$T_CFG" rev-parse --is-inside-work-tree; echo "exit=$?"
git -C "$WORKSPACE" rev-parse --is-inside-work-tree
```

**Automated checks**

- [ ] The key line names the winning variable or says none, and never prints the key value. A test sets a known value and asserts the output does not contain it (`plan/milestones/milestone-5.md:633`). Command: `TO EXPAND`.
- [ ] With `KIRSCH_OPENCODE_API_KEY` and `OPENCODE_API_KEY` each set and unset, the key line names the winner or says none (`plan/milestones/milestone-5.md:634`). Command: `TO EXPAND`.
- [ ] A model id not in the table prints `not in the table` and the conservative fallback (`plan/milestones/milestone-5.md:635`). Command: `TO EXPAND`.
- [ ] With `AGENTS.md` present, the context line names it and its size. With it absent, the line says `none` (`plan/milestones/milestone-5.md:636`). Command: `TO EXPAND`.
- [ ] The backend line reads `ripgrep` with `rg` on PATH and `pure-go` without, under the PATH shim (`plan/milestones/milestone-5.md:637`). Command: `TO EXPAND`.
- [ ] With config not loaded, the key, model and context lines each read `skipped: config did not load` (`plan/milestones/milestone-5.md:638`). Command: `TO EXPAND`.
- [ ] The comments describe doctor lines that exist, and the files change only in comments (`plan/milestones/milestone-5.md:639–653`).

  ```bash
  PRE=/Volumes/DATA/Github/kirsch/.claude/memory/workspace
  check_comments() { diff -U0 "$1" "$2" | grep -E '^[-+]' | grep -vE '^(---|\+\+\+) ' | grep -vE '^[-+][[:space:]]*//'; }
  changed_lines() { diff -U0 "$1" "$2" | grep -E '^[-+]' | grep -vcE '^(---|\+\+\+) '; }
  check_comments "$PRE/m5-d4a-searchcode.pre.go" /Volumes/DATA/Github/kirsch/internal/tool/searchcode.go
  check_comments "$PRE/m5-d4a-app.pre.go" /Volumes/DATA/Github/kirsch/internal/app/app.go
  changed_lines "$PRE/m5-d4a-searchcode.pre.go" /Volumes/DATA/Github/kirsch/internal/tool/searchcode.go
  changed_lines "$PRE/m5-d4a-app.pre.go" /Volumes/DATA/Github/kirsch/internal/app/app.go
  # control: a scratch copy of the saved app.go with one non-comment line appended
  CTRL=$(mktemp); cp "$PRE/m5-d4a-app.pre.go" "$CTRL"; printf 'var controlProbe = 1\n' >> "$CTRL"; check_comments "$PRE/m5-d4a-app.pre.go" "$CTRL"; rm -f "$CTRL"
  ```

  The `check_comments` commands print nothing on a pass. The `changed_lines` commands print a count of at least 1. The control must print `+var controlProbe = 1`.

**Hands-on checks**

1. Run doctor with no key variable set, then with one set to a dummy value (never a real key). Confirm the key line names the variable and never prints the value (`plan/milestones/milestone-5.md:627`).

   This run is safe only if O5 records that doctor makes no network call. The milestone's O5 (`plan/milestones/milestone-5.md:67`) says "no network calls (the spec is silent)", so O5 does not yet record it. Until O5 records it, do not run this step with any real key set.

   > **TO EXPAND when O5 is recorded:** the output grammar for the lines, and the commands to run.

   **Key-state check, last** (authored here). The dummy value is removed. The unset line clears every key variable, and the count after it must print `CONTROL PASS`, then `0`. The next section that checks keys runs the count again before any key is entered.

   ```bash
   # authored here
   unset ANTHROPIC_API_KEY KIRSCH_ANTHROPIC_API_KEY OPENCODE_API_KEY KIRSCH_OPENCODE_API_KEY
   keycount_control
   keycount
   ```

**Notes / evidence**

_Empty. Fill at run._

---

## 21 · Soft-wrap fidelity: composer cut marker (m5-d7b)

**Context:** Terminal: Setup. Binary: `go test`; `npm run screens`. Working directory: /Volumes/DATA/Github/kirsch. Workspace: n/a. Key state: all key variables unset. Config state: Setup, unchanged.

**What landed**

> **TO EXPAND when m5-d7b lands:** the cut marker O17 records (or the wrap choice), and the fences that moved.

**Waits on:** m5-d6 (composer work), m5-d7a (both touch `view.go`), O17 (the marker), and O23 (golden handling), from `plan/milestones/milestone-5.md:106`.

**Automated checks**

- [ ] At 40 columns no composer row is cut without the marker O17 records (`plan/milestones/milestone-5.md:851`). Command: `TO EXPAND`.
- [ ] Golden check, procedure step 5 (`plan/milestones/milestone-5.md:53`):

  ```bash
  cd /Volumes/DATA/Github/kirsch && go test ./internal/tui
  cd /Volumes/DATA/Github/kirsch && npm run screens
  ```

**Hands-on checks**

1. Regenerate and reconcile the moved fences by the procedure at `plan/milestones/milestone-5.md:49–53`. Record which fences moved. The regeneration command is `cd /Volumes/DATA/Github/kirsch && KIRSCH_REGEN=1 go test ./internal/tui -run TestGenerateGrids`, which writes `/tmp/kirsch-grids.txt`.
2. (operator) A person reviews the golden diff before the step closes (`plan/milestones/milestone-5.md:853`).

**Notes / evidence**

_Empty. Fill at run. List every moved golden by header._

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

2. Record the judgement for each "0001–0007" site (DIR-004). The milestone lists `plan/spec/kirsch-plan.md:391`, `plan/README.md:25` and `AGENTS.md:21`, plus `plan/adr/README.md:64`, which may stay as written. `CHANGELOG.md:21` is history and stays.

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

> **TO EXPAND when m5-d21f lands, or record "no step":** the §7 lines amended in `plan/spec/kirsch-plan.md` (`:340–357`), and the session events listed.

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

> **TO EXPAND as items land:** any limit found while a deliverable runs that is not already listed below. The M4 narrow-width defect is not a limit. It is m5-d1 (`plan/milestones/milestone-5.md:7`).

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
