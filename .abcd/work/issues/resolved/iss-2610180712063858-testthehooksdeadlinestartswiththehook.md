---
schema_version: 1
id: "iss-2610180712063858"
slug: "testthehooksdeadlinestartswiththehook"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "merge queue run for #1089"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/status_hook_test.go"
remedy: "Give the test a budget that tolerates runner load while still failing when the deadline starts before the hook, rather than the shared one-second hook budget."
resolution: "The test's second half now owns both budgets: the hook gets the payload's delay plus ten seconds, and the status read's 600 ms cap is lifted to ten seconds through claim.SetPromptReadBudgetForTest, so neither the steps before the read nor a slow read can spend its time. The first half keeps the shipped budgets, so a deadline started late still fails it."
impact: internal
---

TestTheHooksDeadlineStartsWithTheHook (internal/surface/cli/status_hook_test.go, the second half's assertion) failed on the merge queue's Linux check for docs-only pull request #1089 (merge-group run 39104650787, job check (ubuntu-latest), 2026-10-18T06:44Z): under the one-second hook budget the next prompt rendered 'status unavailable: no time was left to read it at this prompt' instead of the announcement. The test took 1.83s, and the package 253s beside a 349s implement/loop run on the same runner, so the shared one-second budget was spent by load before the read, and the test cannot tell that from a deadline started too early.
