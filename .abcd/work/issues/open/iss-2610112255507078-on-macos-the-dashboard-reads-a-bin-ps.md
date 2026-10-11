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
---

On macOS the dashboard reads a /bin/ps that its own 5-second timeout killed as a gone process: any exec.ExitError, including 'signal: killed' from the context deadline, maps to errProcessGone. On a stalled machine Stop then takes the live server for stale, removes its run file and signals nothing, leaving the server running with no record of it.
