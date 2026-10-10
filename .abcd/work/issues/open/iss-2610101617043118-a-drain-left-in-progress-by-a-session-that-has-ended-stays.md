---
schema_version: 1
id: "iss-2610101617043118"
slug: "a-drain-left-in-progress-by-a-session-that-has-ended-stays"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "abcd-60 drain run 2026-10-10"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/drain.md"
remedy: "Record the starting session and the time of each move in drain.json; when a drain move comes from another session or after the working window has lapsed, say so in the payload (owner, last move, pace, ceiling) and refuse until --resume adopts it or --restart archives it and begins a new drain with the flags given."
---

A drain left in progress by a session that has ended stays in force with no owner. On 2026-10-10 a new session's bare abcd drain silently resumed a drain begun on 2026-10-07 (last move 2026-10-08 05:20, since when no session drove it), inheriting its --pace 10080/0 and --sub-agents 6. Nothing in the payload named who began it, when it last moved, or that its ceiling was set for another session's share of the machine's agents. Three live peer sessions were asked and none owned it. A second drain cannot start while it stands, and --pace or --sub-agents naming anything else is refused, so the only ways forward were to inherit it or move drain.json aside by hand, as an earlier session already had (drain-2026-10-07-paced.json).
