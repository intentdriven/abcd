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
resolution: "A build that finds the run live now closes each lane whose pull request merged, through the land stage's own cleanup, before it resumes the run, so the work goes on in a new lane on a new branch cut from origin's default branch tip. A lane is read as merged from local refs only, as the status board reads it, with its pushed head on origin's default branch as last fetched; a branch holding commits past the merge resumes as before, uncommitted work is refused by name, and another driver's run is left as it stands. A squash or rebase merge is not closed this way; iss-2610150827113191 holds that half."
impact: fix
resolved_by:
  commit: "7ec71e3059305e981da1f5aa683c68a1a43b5851"
---

When a lane's pull request has merged, abcd build of the same intent resumes the run and reuses the merged lane's worktree, branch name and stale range instead of starting a fresh lane. On 10 Nov at 00:45 build resumed a stopped run and reused the branch of a lane whose pull request (#1475) had merged, and the stop check then named that branch as owed; by 01:25 the reused branch held four new unpushed commits; at 04:05 build resumed the same run a third time, planned a step already landed, and reported the lane's range as f3d98814de19..f3d98814de19 while the worktree sat at a4ffe1dc7. iss-2610150827113191 covers the loop's merged check after a squash or rebase merge.

## Grounds

- pursued: a build after the lane's merge opens a new lane on a new branch cut at the merged tip; shown wrong if a build resumes a merged lane's worktree or branch
