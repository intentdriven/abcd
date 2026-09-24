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
---

RS005 judges every `Delivers: itd-N` trailer in the range, so once a commit carrying the trailer is on a branch, a later `git revert` of that commit cannot withdraw the declaration: the revert takes the intent back out of `shipped/`, the trailer stays in the range, and `lint-issues` refuses the branch. In autonomous run A the load-check lane (itd-2609231434459890) shipped its intent, review found that shipping it over-claimed, and the fix round reverted the ship commit and closed the spec with a remainder instead, which RS005 refused. The branch had not been pushed, so the orchestrator rebuilt it from the commits before the ship plus cherry-picks, with no ship-and-revert pair in the history; a pushed branch under an open pull request has no such exit short of a new branch and a new pull request. Wanted: RS005 treats a delivery whose commit is reverted within the same range as withdrawn (git names the reverted commit in the revert's message), or its refusal names the rebuild as the remedy.
