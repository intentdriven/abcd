---
schema_version: 1
id: "iss-2610241414124940"
slug: "the-preflight-load-gate-can-starve-every"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "concurrent preflights, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement"
remedy: "Gate every heavy preflight stage, not only the tests, behind one admission so queued runs stop feeding the load they wait on; judge capacity by the slot count rather than a raw load threshold once a run holds a slot; release a slot whose holder has used little CPU for a set time; and re-read load.preflights on every wait so a changed setting applies to waiters already queued."
---

The preflight load gate can starve every waiter on a busy machine and holds its only slot for a hung command. It opens only while the one-minute load is below twice the online cores (20 on 10 cores), but the preflight stages it does not gate (lint, record gates, evals, build, vet) run before it, so seven sessions' preflights on 2026-10-24 kept the load near 25 and the gate never opened: one drain lane's preflight ended 'load gate still shut after 2h0m0s: load 20.9 is not below 20 (2 x 10 online cores); 3 make preflights are running'. Later the same day, at 14:04 UTC, the load was 8 on 10 cores and about nine preflights were still queued, their logs unchanged for 45 minutes, because the one open slot (~/.abcd.noindex/load-wait.json, form command, opened 13:19 UTC, held to 14:19) belonged to a package test run that had used about 11% CPU for 45 minutes; load.preflights was 2, yet only one slot was open, possibly because waiters started before the setting was written read one slot when they began. Nothing reclaims a slot whose holder is idle, and nothing lets the gate open when the load it waits on comes from the ungated stages of its own queued runs. Adjacent records on an unmerged branch cover the ungated stages running unbounded and the unmeasured wait, which this issue does not restate.
