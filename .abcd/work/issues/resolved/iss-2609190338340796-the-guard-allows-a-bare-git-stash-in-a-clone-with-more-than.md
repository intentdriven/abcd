---
schema_version: 1
id: "iss-2609190338340796"
slug: "the-guard-allows-a-bare-git-stash-in-a-clone-with-more-than"
severity: "minor"
category: "future-work-seed"
source: "agent-observation"
found_during: "Gropius autonomous sweep, session gropiusllm-66, relayed to abcd-17 on 2026-09-19"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
resolution: "A warn under the reserved id git-stash-shared-stack on a stash or push without a message and a pop or apply without an entry, only when the loaded repository has more than one non-bare worktree; the count is read lazily from git worktree list."
impact: additive
resolved_by:
  commit: "b7538e877cf0ab52285ec4b245ff4e192f23cafb"
---

The guard allows a bare git stash in a clone with more than one worktree, where the stash stack is shared. git stash is one stack per repository, not per worktree, so an agent's stash-then-lint-then-pop on a clean tree in one lane popped another lane's entry in the Gropius sweep of 2026-09-19 (session gropiusllm-66, forty lanes on one clone); the pop succeeded and the entry landed in the wrong tree with no message either lane could act on. abcd guard check "git stash pop" and "git stash" both answer allow at v0.9.0, and the hazard registry carries no stash entry. Wanted: a warn (not a block) on bare git stash and git stash pop when git worktree list reports more than one worktree, naming the shared stack and the safe successors (a stash with a message and a pop by that entry, or a scratch commit on the lane's own branch). A single-worktree clone keeps allow.

## Grounds

- pursued: a bare stash warns where worktrees share the stack and nowhere else; shown wrong by a single-worktree clone warning or a multi-worktree bare pop allowing (TestSharedStashWarnsInAMultiWorktreeClone, TestSharedStashCountsRealWorktrees)
