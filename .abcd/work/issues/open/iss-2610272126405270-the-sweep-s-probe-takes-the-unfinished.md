---
schema_version: 1
id: "iss-2610272126405270"
slug: "the-sweep-s-probe-takes-the-unfinished"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "sweep stack review, lane E salvage"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/sweep/liveness.go"
remedy: "Give the probe the runs' read error (a Runs func returning a list and an error, or a RunsUnread string like ReservationsUnread), so an unread run directory holds every worktree, and wire it that way when the CLI front door lands."
---

The sweep's probe takes the unfinished build runs as ProbeOptions.Runs, a plain list with no error: a front door that reads the runs and swallows a read failure passes an empty list, and the run proof then silently holds nothing, so a worktree a crashed-but-resumable run names could be reclaimed.
