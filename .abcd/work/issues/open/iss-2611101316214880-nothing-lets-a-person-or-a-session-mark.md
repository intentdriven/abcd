---
schema_version: 1
id: "iss-2611101316214880"
slug: "nothing-lets-a-person-or-a-session-mark"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/owedread.go"
remedy: "Waits on a ruling first: then add one hold state, land after a named tag, that the build loop and land honour by not arming, that the stop check reads as waiting rather than owed, that keeps its reservation renewed, and that the queue and the board report."
---

Nothing lets a person or a session mark a run, lane or pull request as held until a release tag, so a change held back on purpose is treated as live work everywhere. Under auto the build loop arms its own landing, so keeping a schema change out of v0.17.0 on 7 Nov at 20:30 meant stepping the land stage one move at a time and stopping before the arm. The stop check then named the held lane and the held draft pull request #1459 as owed work in every session for days (7 Nov at 22:57, 8 Nov at 03:30 and 15:10); the reservation kept for the waiting draft had nothing to renew it (8 Nov at 06:50); and closing #1459 on 8 Nov at 15:15 moved the owed item instead of clearing it, so its holder released the reservation and relied on a plan note to keep peers off the intent. A person must choose where the hold lives (on the run, the pull request or the record) and which verb sets and lifts it.
