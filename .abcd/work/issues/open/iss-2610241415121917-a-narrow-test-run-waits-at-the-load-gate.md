---
schema_version: 1
id: "iss-2610241415121917"
slug: "a-narrow-test-run-waits-at-the-load-gate"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement"
remedy: "Give the load gate a size class, as Bazel's test sizes do: a run that declares itself small (one package, a -run filter, or an explicit --small) is admitted on a short lane alongside at most one large run, instead of queueing FIFO behind every preflight, and the waiter reports its position and what it is waiting behind."
---

A narrow test run waits at the load gate behind whole preflights from other sessions: on 2026-10-24, during the setup redesign build, a single-package test run queued through abcd implement load --wait sat behind several full preflights at a load average above 100, never started, and was cancelled by hand, so a few seconds of checking cost the lane most of an hour.
