---
schema_version: 1
id: "iss-2610120640454742"
slug: "ensurestore-stricter-than-group-ruling"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "security fix lane C, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/lane.go"
remedy: "Use the same sole-member check as fsutil's home vetting for the worktree store's levels, so one rule decides both; prove with the tests used for the home"
refines: [iss-2609290703091174]
---

The implement loop's worktree store check (ensureStore in internal/core/implement/loop/lane.go) refuses every group-writable level, which is stricter than the product thinker's 2026-10-11 ruling for abcd's own home (admit a group-writable directory when the owner is the group's sole member, fixed for the home by iss-2609290703091174). Found by lane C on 2026-10-12.
