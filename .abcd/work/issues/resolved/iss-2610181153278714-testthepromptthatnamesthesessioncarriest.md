---
schema_version: 1
id: "iss-2610181153278714"
slug: "testthepromptthatnamesthesessioncarriest"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "review of the fix for iss-2610180735158998"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/status_hook_title_test.go"
remedy: "Give its subtests budgets of their own, as the show-once test now has; the shipped bound is asked by TestThePromptHookReturnsInsideItsSecond."
resolution: "Its subtests run under budgets of their own through ownHookBudgets, and wait for every status read before their directories go."
impact: internal
---

TestThePromptThatNamesTheSessionCarriesTheStatus (internal/surface/cli/status_hook_title_test.go) requires the status line in each of its four subtests under the shipped hook budget (one second) and status read cap (600 ms), with no override, so a loaded runner can spend the read's share and leave the status unavailable, failing it the way iss-2610180735158998 failed.
