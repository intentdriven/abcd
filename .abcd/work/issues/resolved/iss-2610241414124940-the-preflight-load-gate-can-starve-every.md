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
resolution: "Whole-run admission puts every heavy preflight stage behind one admission, so queued runs no longer feed the load they wait on, and load.preflights is read again at every look of a wait, so a changed setting reaches waiters already queued. Two parts of the remedy are not done here: releasing a slot whose holder is idle is carried by iss-2610262155232329, and judging capacity by the slot count instead of the load once a run holds a slot is declined in dcn-2610290741094304, on the timeout evidence of this record and iss-2610241911052191."
impact: fix
---

The preflight load gate can starve every waiter on a busy machine and holds its only slot for a hung command. It opens only while the one-minute load is below twice the online cores (20 on 10 cores), but the preflight stages it does not gate (lint, record gates, evals, build, vet) run before it, so seven sessions' preflights on 2026-10-24 kept the load near 25 and the gate never opened: one drain lane's preflight ended 'load gate still shut after 2h0m0s: load 20.9 is not below 20 (2 x 10 online cores); 3 make preflights are running'. Later the same day, at 14:04 UTC, the load was 8 on 10 cores and about nine preflights were still queued, their logs unchanged for 45 minutes, because the one open slot (~/.abcd.noindex/load-wait.json, form command, opened 13:19 UTC, held to 14:19) belonged to a package test run that had used about 11% CPU for 45 minutes; load.preflights was 2, yet only one slot was open, possibly because waiters started before the setting was written read one slot when they began. Nothing reclaims a slot whose holder is idle, and nothing lets the gate open when the load it waits on comes from the ungated stages of its own queued runs. Adjacent records on an unmerged branch cover the ungated stages running unbounded and the unmeasured wait, which this issue does not restate.

## Further evidence 2026-10-24

- A records-only preflight timed out at two hours with the load at 19, below the threshold, so load was not the cause: the gate refused because another make preflight was running and six waiters were queued ahead. Under this load the test stages take 25 to 75 minutes, and only one preflight's test stages pass at a time even with load.preflights at 2, so with six or more waiters no run can meet a two-hour timeout.
- The test stage's `go test -p 4 -timeout 20m ./...` bounds each test binary, not the stage, so one stage ran 73 minutes and more with no overall bound.
- With several preflights' test stages admitted together at a load near 20, six packages hit their 20-minute per-binary timeouts in one run: admitting more than one at a time made all of them fail. The sessions worked around it by running whole preflights one at a time through the gate's command form (`abcd implement load --wait --timeout 8h -- make preflight`).

## Further evidence 2026-10-25

- A whole preflight run alone through the command form, admitted at a load of 8.5 on 10 cores after a 2h12m wait, still timed out three packages at 20 minutes: `internal/core/claim` (43 s in an earlier passing run), `internal/core/implement/loop` and `internal/surface/cli` (16 s earlier). The goroutine dumps show the tests waiting on child processes (git), not on a lock or a deadlock.
- In the same window, the machine's endpoint security scanner used 100 to 210 per cent CPU, sampled every five seconds, while every test binary together used 9 to 38 per cent. The tests make many throwaway repositories under the temporary directory, and an on-access scanner inspects each write and exec. Load average does not show this cost, so the gate opens on a machine whose test throughput is a small fraction of normal.
- So the gate's signal (the one-minute load) is the wrong measure on a machine with on-access scanning. A remedy grounded in that: measure throughput, not load, for example by timing a fixed probe (one throwaway `git init` and commit) before admitting a run, and say so when it is many times slower than the baseline; and ask the machine's administrator to exclude the Go build cache, the temporary directory and `~/.abcd.noindex/` from on-access scanning.
