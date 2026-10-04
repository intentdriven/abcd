---
schema_version: 1
id: "iss-2610040147016103"
slug: "renaming-abcd-s-home-from-abcd-to-abcd-noindex-as-the"
severity: "major"
category: "inconsistency"
source: "user-observation"
found_during: "noindex step 3 planning, abcd-50, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/specs/open/spc-2610031309233367-abcd-keeps-its-home-folder-out-of-desktop-search-by-default.md"
remedy: "Waits on the technical facilitator's ruling on who repairs the links: either the stop's line also tells the person to run git worktree repair in each moved worktree after the rename, or abcd repairs the links itself on its first run after the rename, for the worktrees under its store only (git -C <worktree> worktree repair, which git documents for exactly a moved worktree); either way a test moves a store worktree and proves git worktree list no longer marks it prunable."
---

Renaming abcd's home from ~/.abcd to ~/.abcd.noindex, as the noindex stop tells the person to (mv ~/.abcd ~/.abcd.noindex), moves every git worktree kept in the store under it, and git records a worktree's location in absolute form: after the move each repository lists those worktrees as prunable ('gitdir file points to non-existent location'), and a git worktree prune, by hand or by a tool, deletes their links. Reproduced on a scratch repository 2026-10-04: after mv, git worktree list marks the moved worktree prunable; git -C <moved worktree> worktree repair restores it. spc-2610031309233367 neither names this nor tells the person, and on the product thinker's machine the store holds about thirty live session worktrees.
