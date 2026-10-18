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
---

The implement loop package's tests are bound by git subprocesses: the slowest, TestPollingWithoutAnAnswerNeverHoldsTheIssuePastOneLease, runs Drain 120 times (a 4-minute poll over four 2-hour leases, kept inside the 5-minute take-over margin on purpose) and took 67s plain on a loaded machine, 31s on a quieter one with 3.84s of its own CPU; the next three take about 20s each, and the package is the slowest in both of the ubuntu leg's passes (320-410s plain, 470-650s race on 2026-10-18), with none of its 500 tests parallel.
