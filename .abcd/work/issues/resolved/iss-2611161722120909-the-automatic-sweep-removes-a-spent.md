---
schema_version: 1
id: "iss-2611161722120909"
slug: "the-automatic-sweep-removes-a-spent"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/sweep/auto.go"
remedy: "Unattended, keep a spent store worktree holding any ignored file outside its local tier, and list it for the applying form a person runs."
resolution: "Unattended, the sweep keeps a spent store worktree holding any ignored file outside its local tier, counted by git's status of ignored files as the store's removal deletes them, or whose listing fails, and lists it for the applying sweep (dcn-2611161728290362)."
impact: fix
resolved_by:
  commit: "ae841726002d60d05d17677f9fb75d40fb7d3d13"
---

The automatic sweep removes a spent store worktree together with its ignored files, which can be a person's own local files such as a .env or a local build config.
