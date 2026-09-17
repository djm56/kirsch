# Kirsch — Documentation

> **Mostly written at Milestone 5**, once there is a working tool to document.
> [`usage.md`](usage.md) landed early, and deliberately: the help overlay inside
> the program already points readers at `doc/usage.md`, so leaving the file
> absent meant shipping a pointer to nothing. It documents the interface as it
> stands and says in place where something is not yet wired.

`doc/` is for **people using Kirsch**. When it is complete it will contain:

| Document | Covers | State |
|---|---|---|
| `install.md` | Install (release binary, `go install`, build from source), first-run setup, API key | Not written |
| `configuration.md` | Every key in `config.toml`, its default, and what it does — global vs project precedence | Not written |
| [`usage.md`](usage.md) | Key bindings, slash commands, the approval model, sessions and resume | Written — layout, keys, commands and approvals; sessions and resume arrive with the feature |
| `troubleshooting.md` | `kirsch doctor` output explained, common failures and what they mean | Not written |

**Everything about building Kirsch lives in [`plan/`](../plan/) instead** — the
v0.1 spec, architecture, UI spec, ADRs, and the per-milestone instruction sets.
Build notes do not belong in this folder.
