---
schema_version: 1
id: "iss-2610110908443341"
slug: "the-implement-loop-arms-a-pull-request"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "landing lane-1 of run-2610101724174688 (itd-82 step 1), 2026-10-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "at the arm and merged-check moves, read the pull request's mergeability from the forge and refuse naming a conflicting pull request and its paths, with the hand-merge or rebuild as the remedy, instead of repeating 'not merged yet'; the cause it meets most is iss-2609100507439414"
---

The implement loop arms a pull request that GitHub reports as conflicting and then waits on it in silence. #902, lane-1 of run-2610101724174688, had appended to .abcd/work/DECISIONS.md while later merges to main appended too. Local git merges that cleanly through the union driver; the forge ignores the driver, so #902 was armed but never entered the queue. For 13 hours every implement step said only 'pull request #902 is not merged yet', and the loop's rule that nothing is pushed after arming left it no way out.

## Deferral 2026-10-11

Deferred past v0.13.4: the fix is implement-loop code (read the forge's mergeability at the arm and merged-check moves and refuse naming a conflicting pull request), next cycle's work; meanwhile the hand merge the product thinker ruled for on 2026-10-11 is the remedy, and the root cause, appends to DECISIONS.md, is planned away by itd-2609151138388536
