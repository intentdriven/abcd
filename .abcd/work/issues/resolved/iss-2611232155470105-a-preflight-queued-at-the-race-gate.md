---
schema_version: 1
id: "iss-2611232155470105"
slug: "a-preflight-queued-at-the-race-gate"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "release run 2026-11-23"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "keep a fairly queued race waiter in place past the timeout while it is still progressing in the queue, or resume from the passed stages, and fail only on a stalled queue"
resolution: "the race gate's timeout now counts only time its race queue is stalled: a look at which a progressing race queue holds the gate shut pauses the clock, up to six times the timeout in all"
impact: fix
---

A preflight queued at the race gate fails when the gate's two-hour timeout passes, throwing away the stages it already passed, although nothing failed and it was waiting its turn
