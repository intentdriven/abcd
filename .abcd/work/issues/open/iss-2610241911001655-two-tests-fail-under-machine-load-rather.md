---
schema_version: 1
id: "iss-2610241911001655"
slug: "two-tests-fail-under-machine-load-rather"
severity: "major"
category: "observation"
source: "user-observation"
found_during: "autonomous run 2026-10-24, gated preflight of chore/capture-holdups-3fee"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy"
remedy: "Drive both tests from an injected clock or a deadline the test controls rather than wall-clock time, so a slow machine cannot change their outcome."
---

Two tests fail under machine load rather than on a defect: TestReadOwedHonoursDeadline in internal/core/autonomy found the read marked complete when the forge timed out, and TestHookSessionEndRetriesATransientRootFailure in internal/surface/cli ran out of retry time and staged nothing, both during a gated preflight at load 17 on ten cores

## Diagnosis 2026-10-25

The first reading above was wrong: load was not the cause. All three tests below also fail on their own at a load of 5, on `main` at 2e03c61e4 and on a commit from 2026-10-20, because each git process takes tens of milliseconds to start on this machine and a few dozen of them outlast the budgets the tests set.

- `TestReadOwedHonoursDeadline` (internal/core/autonomy) shows a real defect. Once the 150ms budget has passed, the owed-work reader's git reads fail. The branch listing and the reflog and worktree reads then take the failures to mean "nothing there", so the reading came back complete with no own branches. The verdict should instead fail open and name the cut.
- `TestEachForgeLookupEndsAtTheBudget` (internal/core/claim) measured the lookup's deadline from before the status fetch, whose git work is the machine's to time, not the budget's.
- `TestHookSessionEndRetriesATransientRootFailure` (internal/surface/cli) tests the retry under the real 1.2-second budget. On this machine one detection through the test's git shim costs most of that, so no retry fits.

The same failure is filed from another worktree as iss-2610241829569682 (not yet on the default branch). <!-- record-lint: forward-looking -->
