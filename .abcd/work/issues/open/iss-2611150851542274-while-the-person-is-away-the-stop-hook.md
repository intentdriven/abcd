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
remedy: "Have the owed-work reader skip a lane branch whose run a live drive process holds, read from the drive's state file and its process id, so the branch is not owed until the drive hands the lane back or stops."
---

While the person is away, the stop hook names a branch a live drive is still building (its lane worktree mid-preflight, the drive process running) as owed work with commits not pushed, and asks the session to push it or schedule a look. The session must not push a branch the drive owns, so the item cannot move until the drive pushes it. Once the owed set has not moved over two wakes, a wake no longer releases the stop, so every turn end of a session that correctly leaves the drive's branch alone brings the same stop prompt until the drive's preflight finishes, which on a loaded machine takes an hour or more.
