# Milestone 6 — start here

You are Bishop working on Kirsch. Read, in order: plan/README.md, plan/PROGRESS.md,
plan/build-plan-v0.2-v0.6.md (amendments and the Milestone 6 section),
plan/adr/0010-model-catalog-and-roles.md, then the code under internal/provider,
internal/config and internal/app.

Mission: m6-d1 — Model catalog in data.

Before writing code: confirm amendments A1–A5 and A7 are in plan/spec/kirsch-plan.md §11
and that O30 is recorded in PROGRESS.md (default: TOML). If either is missing, stop and
ask the operator.

Run the mission filter: the change must keep Kirsch one loop, one session, one pane,
patch-only, approval-first, local-first, tool output as data. If it cannot, stop and
file an amendment instead.

Deliver exactly what the m6-d1 row says under Owns and Done when. Add every new config
key to doc/configuration.md in this mission. Update internal/arch/imports_test.go if
a new package is introduced. Run npm run check and npm run security. Record the
mission in PROGRESS.md (Status: done, mission id, commit) and list carried items under
"Carried from m6-d1". Do not start m6-d2.

---

## Added when filed (2026-10-30, mission-20261030-05)

- Do not start until Milestone 5 is closed and `v0.1.0` is tagged.
- Amendments A1, A2, A3, A4, A5 and A7 are amendments 91, 92, 93, 94, 95 and 97 in `plan/spec/kirsch-plan.md` §11.
- Also confirm before m6-d1: ADR 0010 is Accepted (it was filed Proposed); the ADR 0009 layout paths are settled (open item under *Carried from the build plan filing* in `plan/PROGRESS.md`); and read the "Notes for whoever executes this" in `plan/milestones/milestone-6.md`.
- If O30 is not recorded, ask the operator; if it is still unanswered when the mission starts, the default applies (`plan/milestones/milestone-6.md`, Operator decisions).
