---
schema_version: 1
id: "iss-2611241052289841"
slug: "finished-worktrees-and-branches-pile-up"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "worktree cleanup after the v0.17.1 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy"
remedy: "Remove a worktree and its branch automatically once its work is verified on the default branch: merged, or its tree carried by the default branch. Move its logs and scratch to the main checkout's local tier and drop its stale reservation markers and built binaries first. Run this at a landing's merge and at session start, and keep a bundle of the branch so the removal can be undone."
---

Finished worktrees and branches pile up until someone removes them by hand. ahoy worktree prune reclaims only merged, clean lanes when it is run, and nothing runs it. A lane whose work reached main squashed or superseded, or whose local tier holds logs, scratch notes, reservation markers or a built binary, is refused by the guard's delete check. On 24 November six superseded worktrees needed their local tiers emptied by hand, and then a person's own command, before they could go. Each one left behind stays in the stop check, the status sweeps and the lane list.
