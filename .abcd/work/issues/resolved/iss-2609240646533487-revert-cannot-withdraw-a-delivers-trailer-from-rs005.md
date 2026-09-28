---
schema_version: 1
id: "iss-2609240646533487"
slug: "revert-cannot-withdraw-a-delivers-trailer-from-rs005"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/check-issue-resolution.sh"
resolution: "A commit that a later commit of the same range reverts, in git revert's own words, has its Resolves: and Delivers: declarations withdrawn (RS001 and RS005 alike), but only for the ids whose record the revert's own diff takes back out of resolved/, wontfix/ or shipped/ (d68712713), so a hand-written reverts line over a commit that moves no record withdraws nothing; a revert of that revert reinstates them, and a reverts line naming a commit outside the range, or not an ancestor of the revert, withdraws nothing. Proved by cases in scripts/check-issue-resolution-cases.sh: a reverted delivery and a reverted resolution pass (both refused by the previous gate), a revert of the revert is refused again, a revert naming a commit outside the range withdraws nothing, and a hand-written reverts line over a commit that reverts nothing (RS001 and RS005) or that takes a different record out is refused."
impact: internal
resolved_by:
  commit: "aa9a1c241"
---

RS005 judges every `Delivers: itd-N` trailer in the range, so once a commit carrying the trailer is on a branch, a later `git revert` of that commit cannot withdraw the declaration: the revert takes the intent back out of `shipped/`, the trailer stays in the range, and `lint-issues` refuses the branch. In autonomous run A the load-check lane (itd-2609231434459890) shipped its intent, review found that shipping it over-claimed, and the fix round reverted the ship commit and closed the spec with a remainder instead, which RS005 refused. The branch had not been pushed, so the orchestrator rebuilt it from the commits before the ship plus cherry-picks, with no ship-and-revert pair in the history; a pushed branch under an open pull request has no such exit short of a new branch and a new pull request. Wanted: RS005 treats a delivery whose commit is reverted within the same range as withdrawn (git names the reverted commit in the revert's message), or its refusal names the rebuild as the remedy.

## Grounds

- pursued: a pushed branch can take a delivery or resolution back with git revert instead of being rebuilt; a declaration left in force after a range-internal revert, or one withdrawn by a hand-written line naming a commit outside the range, would show it wrong
