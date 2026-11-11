---
schema_version: 1
id: "iss-2611101320485997"
slug: "abcd-ahoy-worktree-prune-reclaims-every"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/worktree/prune.go"
remedy: "Waits on a ruling first: then scope prune to the calling session's worktrees, or to one named worktree, archiving its notes as it does now."
---

abcd ahoy worktree prune reclaims every merged, clean worktree in the repository's worktree store, other sessions' included, so a session told not to prune others' worktrees cannot clean up its own, and the guard refuses a plain removal while the worktree's local tier holds files. Seen on 8 Nov at 04:33. A person must choose whether prune is scoped to the calling session's own worktrees by default, or to one named worktree.
