---
schema_version: 1
id: "iss-2611220942432452"
slug: "git-stash-keeps-one-stash-list-per"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "autonomous fix run 2026-11-21/22"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/defaults/guard.json"
remedy: "Add a guard registry entry that warns on git stash push, pop, apply and drop in a repository with linked worktrees, naming the shared list and offering a commit on the worktree's own branch, or git diff to a patch file, instead."
resolution: "The guard registry entry git-stash-shared-stack warns on every git stash that changes the shared list where the repository has linked worktrees, and the SHELL rules teach it."
impact: fix
resolved_by:
  commit: "543824621"
---

git stash keeps one stash list per repository, shared by every linked worktree. During an autonomous run on 21 Nov 2026 one builder agent popped another builder's stash in a different worktree, and the work was recovered by hand from a patch.
