---
schema_version: 1
id: "iss-2611150851542274"
slug: "while-the-person-is-away-the-stop-hook"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy"
remedy: "Have the owed-work reader skip a lane branch whose run a live drive process holds (read from the drive's state file and its process id), and accept a scheduled session job that fires before the deadline as the scheduled look."
---

While the person is away, the stop hook names a branch a live drive is still building (its lane worktree mid-preflight, the drive process running) as owed work with commits not pushed, and asks the session to push it or schedule a look. It also repeats the same request on every turn end after a heartbeat job is already scheduled to fire before its deadline, so a session that correctly leaves the drive's branch alone is held in a loop of identical stop prompts.
