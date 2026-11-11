---
schema_version: 1
id: "iss-2611011943357182"
slug: "the-implement-loop-s-land-stage-refuses"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "landing build run-2610312053537139 lane 2, 2026-11-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "Have the land stage first check whether its landing head, or a descendant of it that a review receipt names, is already on the default branch. If it is, mark the lane landed, ahead of the branch-head and preflight-receipt checks. Otherwise accept a branch that has moved past its records commit when the records commit is an ancestor of the new head and a review receipt names the new head: record the new head as the landing head, compose the body over the whole range, and arm on the receipt, rather than asking for the branch to be restored."
---

The implement loop's land stage refuses a lane whose branch has moved past the loop's own records commit, and offers no path for a reviewed follow-up. On 2026-11-01, lane 2 of run-2610312053537139 passed validation round 2. The land stage then wrote its records commit (0030dea82), and a reviewer's note then had to be fixed before the change merged. That fix went on top as two commits, with its own preflight and a delta review pair (SHIP and APPROVE). implement step then refused: the branch is at 764bc6a8dcc3, not at the head 0030dea82032 the landing lands, so restore the branch to that head. Following that remedy would have thrown the reviewed fix away. The lane was pushed and its pull request (#1333) opened by hand with abcd land and gh pr create, outside the run's record. The run's landing state (body_checked, armed) no longer describes the pull request.

The refusal outlives the merge. #1333 merged at 21:28 with both 0030dea82 and 764bc6a8d on origin/main. implement step still refused at land, saying the branch is at 764bc6a8dcc3, not at the head 0030dea82032 the landing lands. Moving the local branch back to 0030dea82, as that remedy asks, produced a second refusal: waiting for its full check, because no preflight receipt names the lane's head 0030dea82032. So recognising an already-merged lane would have cost a full preflight of a records commit, one to two hours on the loaded machine. The preflight was not run, and run-2610312053537139's record stays at land while its work is on main.

2026-11-08: seen again several times in the autonomous run of 7 Nov. At 00:23 and 00:31 a lane 39 commits behind main could not have main merged in by hand before precheck, because the push step refuses a branch moved off its reviewed head (internal/core/implement/loop/land.go). At 00:45 and 11:54 precheck at land failed an audit after every review round had passed, and with no move from land back to a fix round each fix went outside the loop, through a hand review pair, a preflight and a plain abcd land. At 13:55 three faults found after validation left a lane at land off its judged head. A correction logged at 06:13: the land stage does merge main into the lane itself and send the merged head back to validate, so the gap stands for a branch moved by hand after validation.
