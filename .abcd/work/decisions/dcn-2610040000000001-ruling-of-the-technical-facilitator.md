---
id: dcn-2610040000000001
slug: ruling-of-the-technical-facilitator
date: 2026-10-04
---
2026-10-04 — Ruling of the technical facilitator, asked by abcd-50, on who reconnects the git worktrees that renaming abcd's home moves (iss-2610040147016103: git records a worktree's location in absolute form, so after `mv ~/.abcd ~/.abcd.noindex` each repository lists every store worktree as prunable until `git -C <worktree> worktree repair` runs, and a prune deletes its link), verbatim: "Person, one printed command". Not chosen: abcd repairing them itself on its first run after the rename (a write into each repository's `.git/worktrees`, outside abcd's home, against itd-2610030720038073 decision 6); decide later. Applied: both stop lines of spc-2610031309233367 print `for w in ~/.abcd.noindex/worktrees/*/*; do git -C "$w" worktree repair; done` after the rename command, recorded as that spec's open question 6, with `TestPrintedRepairReconnectsMovedWorktrees` proving the loop.
