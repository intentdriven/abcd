---
schema_version: 1
id: "iss-2609291032194222"
slug: "the-status-line-s-waiting-role-badge-must-show-only-while"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "status after the 2026-09-29 crash of autonomous run A"
origin: researcher-authored
production_mode: hand-written
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (lane drainFresh of autonomous run A, 2026-09-29): The waiting badge reads a mode tied to no session, so it outlives a crash. Which fix: record the session that set the mode and show the badge only there, reset an agent-set mode when a new session starts, or let the mode lapse after a time? Each touches the ruled rule that a mode the human set by hand is never cleared by their next message, because the verb cannot tell a hand-set mode from an agent-set one."
---

The status line's waiting:<role> badge must show only while that role is actually being addressed. After a computer crash ended every session of autonomous run A, we reopened the repo and saw 'waiting:facilitator', which was wrong: no session was waiting on anyone. The mode is a bare word in .abcd/.work.local/mode (set to 'facilitator' at 09:23Z 2026-09-29, before the crash), and it is tied to no session, so it outlives the session that set it and keeps claiming a question is owed. A crash, kill or rotation never runs 'mode managed', so the badge goes stale exactly when a person comes back to check.
