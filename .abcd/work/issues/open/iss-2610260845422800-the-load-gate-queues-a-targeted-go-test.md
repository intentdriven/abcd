---
schema_version: 1
id: "iss-2610260845422800"
slug: "the-load-gate-queues-a-targeted-go-test"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous-run-2026-10-26"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement"
remedy: "Give the gate a second class for short, targeted runs (a go test naming packages, under a declared budget) that is admitted ahead of whole preflights when the projected load allows, with its own small slot count and an ageing rule so neither class starves; report each waiter's class and wait. Ground the choice in the shortest-job-first / multi-queue scheduling literature and in make -l and Bazel's local resource scheduler before building."
---

The load gate queues a targeted go test (one package, a few minutes) behind whole preflights in one FIFO line, so a short job starves: on 2026-10-26 an implementer's package test for the review-set lane timed out at the gate twice at the 2h limit, behind 19 waiters and 17 make preflight processes (load averages 31.5, 20.7, 40.5 at 09:44). Neither the fixed admission count (iss-2610161756384596) nor the ungated stages (iss-2610241214065079) covers the ordering: even a measured admission count keeps a short job behind every long one already queued.
