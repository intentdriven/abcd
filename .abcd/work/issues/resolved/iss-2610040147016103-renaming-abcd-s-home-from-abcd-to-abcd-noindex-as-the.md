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
remedy: "Both stop lines of spc-2610031309233367 print one ready command after the rename, for w in ~/.abcd.noindex/worktrees/*/*; do git -C \"$w\" worktree repair; done, run by the person (the technical facilitator's ruling of 2026-10-04, recorded as the spec's open question 6); proven by TestPrintedRepairReconnectsMovedWorktrees, which moves a store worktree as the rename does, runs the printed loop and asserts its repository no longer lists it as prunable, and by the stop tests asserting the command in both lines; grounds: git-worktree(1) documents repair for exactly a worktree moved without git worktree move, and a scratch reproduction on 2026-10-04 confirmed it."
resolution: "Both stop lines now print the repair loop for the store's worktrees after the rename; TestPrintedRepairReconnectsMovedWorktrees runs the printed loop against a worktree moved as the rename moves it and asserts its repository no longer lists it as prunable."
impact: fix
resolved_by:
  commit: "a1ed114bd"
---

Renaming abcd's home from ~/.abcd to ~/.abcd.noindex, as the noindex stop tells the person to (mv ~/.abcd ~/.abcd.noindex), moves every git worktree kept in the store under it, and git records a worktree's location in absolute form: after the move each repository lists those worktrees as prunable ('gitdir file points to non-existent location'), and a git worktree prune, by hand or by a tool, deletes their links. Reproduced on a scratch repository 2026-10-04: after mv, git worktree list marks the moved worktree prunable; git -C <moved worktree> worktree repair restores it. spc-2610031309233367 neither names this nor tells the person, and on the product thinker's machine the store holds about thirty live session worktrees.

## Grounds

- pursued: a person who renames ~/.abcd as the stop says and runs the printed loop finds every store worktree listed again by its repository and none prunable; a moved worktree still listed prunable after the loop would show it wrong
