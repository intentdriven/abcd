---
schema_version: 1
id: "iss-2610180545134593"
slug: "the-implement-loop-package-s-tests-are"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "Cut the git subprocesses a Drain call makes in the test fixtures, or let the tests that swap no package seam run in parallel, measured against the race step's CPU share"
resolution: "A claim operation names origin's default branch once each side of a fetch, and a view reads its reservation tips in one batch with a per-tip fallback; one drain test fell from 1,107 git processes to 830"
impact: internal
---

The implement loop package's tests are bound by git subprocesses: the slowest, TestPollingWithoutAnAnswerNeverHoldsTheIssuePastOneLease, runs Drain 120 times (a 4-minute poll over four 2-hour leases, kept inside the 5-minute take-over margin on purpose) and took 67s plain on a loaded machine, 31s on a quieter one with 3.84s of its own CPU; the next three take about 20s each, and the package is the slowest in both of the ubuntu leg's passes (320-410s plain, 470-650s race on 2026-10-18), with none of its 500 tests parallel.

## Grounds

- pursued: the loop package's tests and every drain move spend about a quarter fewer git processes, so the ubuntu leg's race step shortens by a minute or more; a merge-group race step no shorter than before would show it wrong

## Outcome

The grounds were shown wrong on the first two merge-group runs after both changes (#1099, #1100, 2026-10-18). The ubuntu leg's race step took 20.8 and 20.6 minutes, at the top of the 14 to 20.6 minutes it had taken before, not a minute shorter. Its package times summed to 4,293 and 4,243 CPU-seconds, against 4,254 at the slowest run before. The loop package took 635 and 621 seconds under race, inside its earlier 470 to 650. The cut in git processes stands where it was measured (one drain test, 1,107 to 830), but that one test overstated it: across the whole of the loop and claim packages a plain run made 126,334 git calls before and 119,327 after, 5.5% fewer, which on CI is below the difference between runners. The longer lever is iss-2610181757540736. The leg's remaining time is tracked in iss-2610181711260494.
