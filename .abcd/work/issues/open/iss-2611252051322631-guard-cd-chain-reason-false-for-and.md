---
schema_version: 1
id: "iss-2611252051322631"
slug: "guard-cd-chain-reason-false-for-and"
severity: "nitpick"
category: "drift"
source: "agent-finding"
found_during: "manual build of 25 November, logged as a test of what abcd relies on"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/defaults/guard.json"
remedy: "Word the entry's reason so it holds for every join it refuses: the delete's directory is wherever the chain left the shell, and where the join lets a failed cd through it is the wrong one."
---

The guard refuses a recursive forced delete after a cd joined by && and gives the reason that the delete still runs if the cd fails. With && it does not run. Refusing the form is deliberate; the reason is true only of the other joins, so a person told it can show it false. Tested with the guard on 2026-11-25.
