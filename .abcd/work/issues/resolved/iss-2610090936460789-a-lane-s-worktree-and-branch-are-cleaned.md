---
schema_version: 1
id: "iss-2610090936460789"
slug: "a-lane-s-worktree-and-branch-are-cleaned"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "2026-10-09 cleanup of the overnight drain lanes"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "Finish merged lanes without a person: have each drain move (and the build loop's own status) step every lane parked at land whose pushed head is now on the default branch, and let the merged-cleanup step run outside the pacing window, since it does no new work; test that a merged lane parked at land is removed and marked done on the next drain move, watched fail first."
resolution: "each drain move takes the landing's last step for every issue lane parked at land whose pushed head is on the default branch, outside the drain's and the run's window clock, removing its worktree and branch and marking it done"
impact: fix
resolved_by:
  commit: "91765d9c5875d809e6af1a404434d3f860476948"
---

A lane's worktree and branch are cleaned up only by a final 'abcd implement step' run after its pull request merges, and nothing runs it: the land stage waits (exit 3) until the merge, the drain records the lane as 'pull-request' and moves on, and no hook, drain move or schedule comes back. On 2026-10-09 all 18 overnight drain lanes sat merged but still at the land stage, their worktrees in the store and their runs counted as building on the status board (iss-2610090824041378). The step was also refused when it was finally run, because the run's pacing window had elapsed ('nothing before 14:36Z'), so even finishing a merged lane waits hours.

## Grounds

- pursued: merged drain lanes no longer sit at land with their worktrees in the store after the next drain move; shown wrong if a lane whose pull request merged is still at land, its worktree present, after a drain move that followed the merge
