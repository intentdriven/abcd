---
schema_version: 1
id: "iss-2610170921271252"
slug: "before-its-push-an-issue-lane-re-fetches"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/sync.go"
remedy: "Merge the default branch into a lane at most once per lane before its push, or at most once per N minutes, and say so in the run record; the pull request's own conflict check still catches a later move."
---

Before its push, an issue lane re-fetches origin's default branch every time it arrives at its landing with no landing state, and whenever that branch has moved it merges it and sends the lane through a fresh validation round, with no limit. With the default branch moving every 30 to 60 minutes, a slow lane, whose validators take longer than that, can run round after round of validators and never reach its push.
