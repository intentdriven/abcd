---
schema_version: 1
id: "iss-2610302150519818"
slug: "the-status-line-s-plain-render-tests-run"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "merge-queue CI for PR 1275, 2026-10-30"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/statusline_test.go"
remedy: "Widen the budgets with the existing widenBudgets helper in every status-line test that needs real git to answer and is not about the deadline, as the deadline tests already do; leave the production values as they are."
resolution: "The status-line render tests that need real git to answer, and are not about the deadline, now widen the budgets with widenBudgets for their run: every managed-row test in statusline_test.go, and every merging test through mergingCheckout. With git made to answer 0.1s late, both named tests failed on main with the CI message and pass with the change. The production budgets are unchanged and still pinned by TestStatuslineBudgetIsUnderASecond."
impact: internal
---

The status line's plain render tests run at the production git share of 300ms, and since the merging element added its read of the default branch's facilitator.mode, a row asks git several more questions inside that share. On a loaded race-instrumented macOS runner the share ran out: TestStatuslineManagedRendersTheRowBadgeFirst failed in the merge-queue run of PR 1275 with the merging element dropped and a note on stderr, for a change that touched only records.

## Grounds

- pursued: expected that the status-line render tests stop failing on timing alone in the merge queue's race-instrumented macOS run; a later queue failure of one of them naming the git share would show it wrong.
