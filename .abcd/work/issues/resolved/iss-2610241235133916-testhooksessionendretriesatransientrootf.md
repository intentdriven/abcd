---
schema_version: 1
id: "iss-2610241235133916"
slug: "testhooksessionendretriesatransientrootf"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "drain session abcd-0d, 2026-10-24, testing a merge on a machine at load average 25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/hook_subagent_stage_retry_test.go"
remedy: "Widen sessionEndRetryBudget inside TestHookSessionEndRetriesATransientRootFailure, which proves the retry rather than the budget, and add a test that pins the budget's skip path, so neither depends on how fast the machine runs git."
resolution: "The retry test widens sessionEndRetryBudget, and TestHookSessionEndSkipsTheRetryPastItsBudget pins the skip path."
impact: internal
resolved_by:
  commit: "34938571077fae6d9fd821c535b500cd9892734b"
---

TestHookSessionEndRetriesATransientRootFailure fails on main whenever the machine is loaded: the session-end hook begins a retry only when its pause and a detection costed at the first detection's time end inside the 1.2-second sessionEndRetryBudget, and at a load average of 25 the first detection alone used that up, so the hook skipped the retry ('no time is left to ask again'), staged nothing, and the test failed reproducibly, on main and on a merge alike. Every make preflight on a busy machine therefore fails in internal/surface/cli, and the skip path itself had no test.

## Grounds

- pursued: both tests pass at a load average near 25, where the old test failed every run; a test that fails on a busy machine again would show this wrong
