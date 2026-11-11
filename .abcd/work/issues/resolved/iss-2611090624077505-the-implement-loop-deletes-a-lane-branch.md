---
schema_version: 1
id: "iss-2611090624077505"
slug: "the-implement-loop-deletes-a-lane-branch"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "Move the held-branch rule the peers check reads (rebase-merge and rebase-apply head-name, BISECT_START, and rebase-merge/update-refs, in the common dir and every worktrees entry, failing closed on a file that cannot be read) into one gitutil primitive, and have the loop's branch delete, the guard's delete proof and the peers check all ask it"
resolution: "gitutil.HeldBranches is the one held-branch rule (rebase and bisect state in every git dir, a rebase's update-refs list, failing closed); the loop's branch delete, the guard's delete proof, the peers check and the session branch recorder ask it"
impact: fix
---

The implement loop deletes a lane branch that a stopped rebase or bisect still holds. branchCheckedOutElsewhere asks only which branch each listed worktree has, and git lists a worktree whose rebase or bisect stopped as detached, so a person who checks a landed lane branch out and rebases it, stopping on a conflict, has the branch deleted by the landing, the held discard or the receipt discard through git update-ref -d, which never asks whether a branch is in use. Their git rebase --continue then fails with cannot lock ref, unable to resolve reference, and the rebase stays stopped; the commits remain reachable from HEAD. The guard's delete proof (sweep branchHazard) and the session branch recorder read the worktree listing the same way. A branch listed in a stopped rebase's update-refs file is also one git refuses to move.
