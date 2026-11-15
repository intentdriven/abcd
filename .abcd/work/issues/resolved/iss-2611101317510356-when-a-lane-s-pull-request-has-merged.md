---
schema_version: 1
id: "iss-2611101317510356"
slug: "when-a-lane-s-pull-request-has-merged"
severity: "major"
category: "bug"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/resume.go"
remedy: "Close a lane in its run once its pull request has merged, releasing its worktree and branch, and have the next build of the intent start a new lane and branch from origin's default branch; a test merges a lane, builds the intent again and finds a new branch name based at the merged tip."
resolution: "A build that finds the run live now closes each lane whose pull request merged by a merge commit, the kind this repository's merge queue makes, once the resume's own checks have passed, so the work goes on in a new lane on a new branch cut from origin's default branch tip. The proof is the land step's own: the default branch is fetched from origin and the lane's pushed head must be on it, and a fetch that fails or runs out of time closes nothing. A branch holding commits past the merge resumes as before, a worktree holding files its removal would lose is refused by name, and a run another driver or a stop has the say over is left as it stands. Limits: a squash or rebase merge leaves the pushed head off the default branch and is not closed this way, so iss-2610150827113191 stays open; a lane that has left its land stage is not closed; and a run whose control record still holds a stop is left alone. The incident's own route could not be re-run, so which stage its lane was at, and how its pull request merged, is not known."
impact: fix
resolved_by:
  commit: "7ec71e3059305e981da1f5aa683c68a1a43b5851"
---

When a lane's pull request has merged, abcd build of the same intent resumes the run and reuses the merged lane's worktree, branch name and stale range instead of starting a fresh lane. On 10 Nov at 00:45 build resumed a stopped run and reused the branch of a lane whose pull request (#1475) had merged, and the stop check then named that branch as owed; by 01:25 the reused branch held four new unpushed commits; at 04:05 build resumed the same run a third time, planned a step already landed, and reported the lane's range as f3d98814de19..f3d98814de19 while the worktree sat at a4ffe1dc7. iss-2610150827113191 covers the loop's merged check after a squash or rebase merge.

## Grounds

- pursued: a build after the lane's merge opens a new lane on a new branch cut at the merged tip; shown wrong if a build resumes a merged lane's worktree or branch
