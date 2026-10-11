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
resolution: "itd-148, spc-42 and iss-213's deferral_reason now call itd-2609091014076309 the store intent rather than the store draft; TestWorktreeStoreRuleNamesTheIntentsActualState holds the three records as sites (found by id in any lifecycle folder), flags 'store draft' and 'that draft', and skips dated '## Deferral' sections."
impact: internal
resolved_by:
  commit: "fb3cd6c977af2915b822e0244c6db48ab70efcd6"
---

Present-state planning prose still calls itd-2609091014076309 'the store draft' although the intent is in planned/: itd-148 (lines 64 and 188), spc-42 (lines 27 and 112) and iss-213 (line 11). The drift fix for iss-2610090744073409 corrected the CONCURRENCY rule, the build brief page and lane.go, and left these planning records as out of its remedy's scope; its ruthless review recorded them.

## Grounds

- pursued: no present-state paragraph of itd-148, spc-42 or iss-213 calls the planned store intent a draft, and the extended test fails if one does; shown wrong by any 'store draft' or 'that draft' left outside a dated Deferral section of those records, or by the test passing on the pre-fix text (it failed there with 13 findings).
