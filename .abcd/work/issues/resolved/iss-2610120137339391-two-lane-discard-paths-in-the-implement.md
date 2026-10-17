---
schema_version: 1
id: "iss-2610120137339391"
slug: "two-lane-discard-paths-in-the-implement"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "security review of itd-82 step 3 (run-2610101724174688 lane-3, PR #961)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/hold.go"
remedy: "before each update-ref -d in hold.go and receipt.go, refuse when the branch is checked out in any worktree other than the lane's own, reusing lane-3's branchCheckedOutElsewhere from land.go, with a test per site"
resolution: "implement step --discard and a hand-back receipt's discard now refuse, before removing anything, when another checkout has the lane's branch checked out, naming it; both reuse the landing's branchCheckedOutElsewhere, which now takes the refusing stage and the retry"
impact: fix
---

Two lane-discard paths in the implement loop delete a lane's branch with `git update-ref -d` without first checking whether another worktree has that branch checked out: `abcd implement step --discard <lane>` (internal/core/implement/loop/hold.go, the update-ref after removeLaneWorktree) and a handed-back receipt's discard (internal/core/implement/loop/receipt.go, discardReceiptLane's update-ref). git refuses `git branch -d` for a checked-out branch but not `update-ref -d`, so if the person (or a peer session) has checked the lane's branch out in a worktree other than the lane's own, that worktree is left on an unborn branch with its HEAD dangling. The same gap in landMerged was found by the security review of itd-82 step 3 (run-2610101724174688 lane-3) and closed there by a branchCheckedOutElsewhere check in land.go; these two siblings were outside that step's scope.

## Grounds

- pursued: a lane branch another worktree stands on survives both discards with that worktree's HEAD intact; a discard that deletes it, or removes the lane's worktree before refusing, shows it wrong

## Route

abcd adds this route: the deletion is the loop's own `git update-ref -d`. Without abcd, git's porcelain refuses it (`git branch -d` and `-D` both refuse a branch checked out in any worktree), so the fix is abcd's and changes nothing about how the repository works.
