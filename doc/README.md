# Kirsch — Documentation

> **This folder is intentionally empty for now.** User documentation is
> written at Milestone 5, once there is a working tool to document.

`doc/` is for **people using Kirsch**. When it is populated it will contain:

| Document | Covers |
|---|---|
| `install.md` | Install (release binary, `go install`, build from source), first-run setup, API key |
| `configuration.md` | Every key in `config.toml`, its default, and what it does — global vs project precedence |
| `usage.md` | Key bindings, slash commands, the approval model, sessions and resume |
| `troubleshooting.md` | `kirsch doctor` output explained, common failures and what they mean |

**Everything about building Kirsch lives in [`plan/`](../plan/) instead** — the
v0.1 spec, architecture, UI spec, ADRs, and the per-milestone instruction sets.
Build notes do not belong in this folder.
