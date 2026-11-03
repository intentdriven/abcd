---
schema_version: 1
id: "iss-2611022331542403"
slug: "testtheforgeisnotaskedpastitsbudget"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "coordinating the merge queue, 2026-11-02"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/status_pushers_test.go"
remedy: "Drive the budget from a clock or deadline the test controls, so the first ask always happens and the second always finds the budget spent, instead of racing a 100ms wall-clock budget."
resolution: "the budget's context is a seam the test spends at the first answer, so the first ask always happens and the second always finds the budget gone"
impact: internal
resolved_by:
  commit: "7dddd62ae"
---

TestTheForgeIsNotAskedPastItsBudget fails under load: on 2026-11-02 in #1347's merge-group macOS race run it reported 'asked []; want only the newest before the budget ran out', because a 100ms pusher budget can be spent before the first branch is asked when the runner is loaded, so the test asserts timing rather than the budget's behaviour.
