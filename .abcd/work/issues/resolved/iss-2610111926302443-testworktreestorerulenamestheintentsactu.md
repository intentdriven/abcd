---
schema_version: 1
id: "iss-2610111926302443"
slug: "testworktreestorerulenamestheintentsactu"
severity: "nitpick"
category: "tech-debt"
source: "impl-review"
found_during: "ruthless review, run-2610111822546516 lane-1, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/conventionsfile_test.go"
remedy: "Narrow the stale-word check to the line (or list item) that names itd-2609091014076309 instead of the whole paragraph, and add a case with a draft word beside another intent in the same list that must pass, watched fail first."
resolution: "TestWorktreeStoreRuleNamesTheIntentsActualState now judges the list item, prose sentence or front-matter line that names itd-2609091014076309 (or says \"store draft\") instead of the whole blank-line paragraph, so a draft word beside another intent in the same list no longer fails it; TestWorktreeStoreSitesNarrowToTheItemNamingTheIntent pins that case and the historical wrapped stale shapes."
impact: internal
resolved_by:
  commit: "60e1831f77ccad9e9a26d9fc054bb2f11b3cf17a"
---

TestWorktreeStoreRuleNamesTheIntentsActualState (internal/core/lint/conventionsfile_test.go) checks for stale draft wording per blank-line paragraph, and the references list in .abcd/development/brief/04-surfaces/34-build.md is one paragraph naming six intents, so a later '(a draft)' beside a different intent in that list would fail the test naming itd-2609091014076309 falsely.

## Grounds

- pursued: a draft word beside another intent in the same list or prose paragraph passes, while every stale shape the check was written against (wrapped '(a draft)' list item, wrapped Go comment, 'store draft'/'that draft' in itd-148, spc-42, iss-213) is still flagged; shown wrong if a historical stale text (8198d2b8c^, fb3cd6c97^) passes the narrowed check, or a draft word said of another intent in the same list still fails it.
