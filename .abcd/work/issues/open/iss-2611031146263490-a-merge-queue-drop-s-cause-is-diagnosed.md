---
schema_version: 1
id: "iss-2611031146263490"
slug: "a-merge-queue-drop-s-cause-is-diagnosed"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "coordinating the build lanes, 2026-11-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "When the queue watch finds a drop, read the failed job's log, name the failing test and package, and capture it as a flake (found_at the test file) unless an open record already names that test, so the drain takes it; a watchdog stop is reported as such and captures nothing."
---

A merge-queue drop's cause is diagnosed and filed by hand: on 2026-11-03 five pull requests left the queue, three at the ci-watchdog deadline and two on timing-bound tests, and each time the coordinator read the run's failed log, told a watchdog stop from a flaky test, captured the flake, launched its fix and re-armed. itd-82's queue watch re-arms and names the dropped check, but a failing test does not become a ledger record a drain lane could fix.
