# Manual test — Milestone 1 (workspace and read-only tools)

Checking that Kirsch reads a real repository correctly, and refuses what it
should. Milestone 1 is where the security boundary went in, so roughly half of
this walkthrough is trying to get past it.

**Time:** about 20 minutes.
**Prerequisite:** `npm test` and `npm run security` both green.

```bash
cd /path/to/some/real/repository
npm --prefix /path/to/kirsch run start
# or, from the Kirsch checkout, against Kirsch itself:
npm start
```

The debug slash commands (`/read`, `/ls`, `/search`, `/gitstatus`, `/gitdiff`)
are Milestone 1 scaffolding and are removed in Milestone 3. They exist so a real
repository can be driven before a model can drive it.

---

## 1 · Does it know where it is

- [✅] The header shows the repository's **actual** directory name
- [✅] It shows the **actual** current branch
- [✅] A `●` appears if and only if there are uncommitted changes

Test the override and the failure path:

```bash
cd /tmp && npm --prefix /path/to/kirsch run start
```
- [ ] Outside a repository it refuses with a message naming `--workspace`,
      rather than a panic or a bare exit code

```bash
go run ./cmd/kirsch --workspace /tmp
```
- [✅] A plain directory works, and the header simply omits the branch

## 2 · Reading

- [✅] `/read README.md` renders a tool card reading `read_file`
- [✅] `Enter` on the card expands it, and the content is **line-numbered**
- [✅] The line numbers match the real file (`sed -n '10p' README.md`)
- [✅] `/read` a file over 500 lines shows a truncation note
- [✅] `/read nonexistent.txt` reports `file_not_found`, not a crash
- [✅] `/read` a directory suggests `list_files` instead

## 3 · Listing and searching

- [✅] `/ls` lists the repository root
- [✅] `/ls src` (or any real subdirectory) lists that directory
- [✅] `/search <something you know is there>` finds it
- [✅] Results read `path:line: text`
- [✅] Searching something absent reports `no matches` rather than failing

**Ignore rules.** In a repository with a `node_modules`, `vendor`, or a
populated `.gitignore`:

- [✅] `/ls` does not show `node_modules` or `vendor`
- [✅] `/ls` does not show files your `.gitignore` excludes
- [✅] `/search` does not return hits from inside them
- [✅] `.git` never appears

## 4 · The security boundary

This is the part that matters. Each of these must be **refused**, and the
refusal must render as a card explaining why — not a panic, not an empty card,
not silence.

```
/read .env
/read ../../../etc/passwd
/read /etc/passwd
/read .git/config
/read ../some-sibling-directory/file.txt
```

- [✅] `.env` is refused
- [✅] `.ENV` is **also** refused — uppercase is not a bypass

  Worth doing explicitly. This was a real hole: on macOS the filesystem is
  case-insensitive, so `.ENV` opened the real `.env`. Fuzzing found it, not
  review (plan §11 amendment 51).

- [✅] `.env.local`, `.env.production` are refused
- [✅] A `.pem` or `.key` file is refused
- [✅] `.git/config` is refused
- [✅] `../` traversal is refused
- [✅] An absolute path is refused, and reports *invalid input* rather than a
      violation — it is a malformed request, not an attack

If you have a `.env` in the repository, confirm the content never appears
anywhere on screen, including in the error card.

**Symlinks.** In a scratch directory:

```bash
mkdir -p /tmp/wstest && cd /tmp/wstest
echo "inside" > inside.txt
ln -s inside.txt ok-link
ln -s /etc/passwd escape-link
ln -s nowhere.txt dangling-link
ln -s loop loop
go run /path/to/kirsch/cmd/kirsch --workspace /tmp/wstest
```

- [✅] `/read ok-link` works — an internal symlink is legitimate and must not be
      refused
- [✅] `/read escape-link` is refused
- [✅] `/read dangling-link` reports *not found*, not a violation — it is missing,
      not forbidden
- [✅] `/read loop` returns promptly with a refusal and **does not hang**

## 5 · Git tools

- [✅] `/gitstatus` shows the real working-tree status
- [✅] Its summary matches `git status --porcelain | wc -l`
- [✅] `/gitdiff` shows a real unified diff when there are changes
- [✅] In a clean tree it says so rather than showing nothing
- [✅] Both report clearly in a non-Git directory rather than crashing

## 6 · Cancellation

Search something expensive in a large repository — a single letter across a big
tree — and press `Esc` while it runs.

- [✅] Control returns to the composer within about a second
- [✅] The card shows `⊘`
- [✅] The spinner stops
- [✅] Kirsch stays usable afterwards

## 7 · The debug log

```bash
npm run start:debug
# run a few commands, quit, then:
cat ~/.local/state/kirsch/debug.log
```

- [✅] Every tool run appears with a duration and an outcome
- [✅] No file **contents** appear — only summaries
- [✅] No environment variable **values** appear
- [✅] Nothing was printed to the terminal while it ran

  That last one is load-bearing. A stray write to stdout corrupts the alternate
  screen, and the failure presents as a rendering bug.

- [✅] The directory is `0700` (`ls -ld ~/.local/state/kirsch`)

## 8 · Configuration

```bash
mkdir -p .kirsch
printf '[provider.anthropic]\nmodel = "test-model"\n' > .kirsch/config.toml
npm run start:debug
```
- [✅] It starts normally

```bash
printf '[provider.anthropic]\napi_key = "sk-ant-nope"\n' > .kirsch/config.toml
npm start
```
- [✅] It **refuses to start**, names the file and the key, and explains that
      secrets belong in an environment variable

```bash
printf '[provider.anthropic]\nfuture_option = true\n' > .kirsch/config.toml
npm run start:debug
```
- [✅] It starts anyway, and the debug log warns about the unknown key

  A config written for a newer Kirsch must still boot an older one.

```bash
rm -rf .kirsch
```

## 9 · Scale

Point Kirsch at the largest repository you have.

- [✅] It opens without a noticeable pause
- [✅] `/ls` returns promptly
- [✅] `/search` over the whole tree returns within a few seconds
- [✅] The interface stays responsive throughout — the spinner keeps moving

## 10 · Security tooling

```bash
npm run security
npm run fuzz
```

- [✅] `govulncheck` reports no vulnerabilities in reachable code
- [✅] `gosec` reports nothing unannotated
- [✅] The secret scan finds nothing
- [✅] All six fuzz targets survive 30 seconds each

If a fuzz target fails, the input is saved to that package's `testdata/fuzz/`
directory. **Commit it** — it becomes a permanent regression test. That is how
the `.ENV` bypass became a test rather than a memory.

---

## Reporting

For anything marked ❌ — and especially anything in **section 4**:

1. The exact command and what happened
2. Your OS and filesystem (`mount | grep " / "` on Linux; macOS is
   case-insensitive by default, which matters for the denylist)
3. The debug log from `npm run start:debug`

A security finding is worth reporting even if you are unsure. The three real
ones in this milestone were all found by tools rather than by reading, and all
three looked like nothing until they were traced.
