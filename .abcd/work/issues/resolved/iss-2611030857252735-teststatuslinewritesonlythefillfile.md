---
schema_version: 1
id: "iss-2611030857252735"
slug: "teststatuslinewritesonlythefillfile"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "coordinating the build lanes, 2026-11-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/statusline_readonly_test.go"
remedy: "Widen git's budgets (widenBudgets) in each TestStatuslineWritesOnlyTheFillFile case, as the no-directory case does: the test proves what is written, not the deadline."
resolution: "Fixed: TestStatuslineWritesOnlyTheFillFile and TestStatuslineWritesNothingWithoutASessionID widen git's budgets per case (widenBudgets), as the no-directory case does, so a loaded race runner no longer empties the row."
impact: fix
resolved_by:
  commit: "1a70bcfc3"
---

TestStatuslineWritesOnlyTheFillFile/managed_by_history_registration_only (internal/surface/cli/statusline_readonly_test.go:352) failed in #1369's merge-group macOS race run 41598350207 on 2026-11-03: stdout was empty because git did not name the checkout within the status line's 300ms git budget under the race load, dropping a pull request from the queue whose change was unrelated. #1362 widened the budgets for TestStatuslineCreatesNoDirectory; this sibling still ran at the default.
