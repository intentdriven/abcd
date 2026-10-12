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
---

TestWorktreeStoreRuleNamesTheIntentsActualState (internal/core/lint/conventionsfile_test.go) checks for stale draft wording per blank-line paragraph, and the references list in .abcd/development/brief/04-surfaces/34-build.md is one paragraph naming six intents, so a later '(a draft)' beside a different intent in that list would fail the test naming itd-2609091014076309 falsely.
