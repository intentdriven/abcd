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
---

The automatic sweep removes a spent store worktree together with its ignored files, which can be a person's own local files such as a .env or a local build config.
