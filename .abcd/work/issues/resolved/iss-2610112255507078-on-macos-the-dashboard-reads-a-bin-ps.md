---
schema_version: 1
id: "iss-2610112255507078"
slug: "on-macos-the-dashboard-reads-a-bin-ps"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "review of the flaky-test fixes"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/dashboard/proc_darwin.go"
remedy: "Check ctx.Err() before the ExitError mapping; a timed-out ps is an error, never a gone process."
resolution: "readProcess on macOS checks the ps context's deadline before mapping an exit error to errProcessGone, so a ps killed by its own timeout returns an error wrapping context.DeadlineExceeded; TestReadProcessTakesATimedOutPsForAnError drives it through a ps that outlasts a 50ms deadline."
impact: fix
resolved_by:
  commit: "d090e76ef6928f2dd9099735f2a8f50310295c17"
---

On macOS the dashboard reads a /bin/ps that its own 5-second timeout killed as a gone process: any exec.ExitError, including 'signal: killed' from the context deadline, maps to errProcessGone. On a stalled machine Stop then takes the live server for stale, removes its run file and signals nothing, leaving the server running with no record of it.
