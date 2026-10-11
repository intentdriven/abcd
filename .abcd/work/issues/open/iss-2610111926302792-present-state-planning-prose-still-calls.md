---
schema_version: 1
id: "iss-2610111926302792"
slug: "present-state-planning-prose-still-calls"
severity: "nitpick"
category: "drift"
source: "impl-review"
found_during: "ruthless review, run-2610111822546516 lane-1, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/specs/open/spc-42-every-change-starts-in-its-own-worktree.md"
remedy: "Replace 'the store draft' with 'the store intent' (or name itd-2609091014076309 as planned) in itd-148, spc-42 and iss-213's present-state prose, leaving ADRs, DECISIONS.md and dated research notes as written; extend TestWorktreeStoreRuleNamesTheIntentsActualState's site list to those three files, watched fail first."
---

Present-state planning prose still calls itd-2609091014076309 'the store draft' although the intent is in planned/: itd-148 (lines 64 and 188), spc-42 (lines 27 and 112) and iss-213 (line 11). The drift fix for iss-2610090744073409 corrected the CONCURRENCY rule, the build brief page and lane.go, and left these planning records as out of its remedy's scope; its ruthless review recorded them.
