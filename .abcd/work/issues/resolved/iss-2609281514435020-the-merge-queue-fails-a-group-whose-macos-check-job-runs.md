---
schema_version: 1
id: "iss-2609281514435020"
slug: "the-merge-queue-fails-a-group-whose-macos-check-job-runs"
severity: "minor"
category: "process"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: ruling Z"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
resolution: "The macOS leg of ci.yml's check job carries a 45-minute ceiling (the ubuntu leg keeps 30) and the ruleset mirror's merge-queue check_response_timeout_minutes records 45, under ruling Z of 2026-09-28; the live ruleset follows with gh api after the merge. A step at the end of the macOS leg warns in the log and the step summary once the check passes 35 minutes, naming the rerun-once rule and the speed lane, and never fails the job. TestRaceLaneBudgetIsDeclaredAndFitsItsJob holds every leg at or below the mirror's cap and TestCheckJobWarnsBeforeTheQueueCap pins the 35-minute threshold below it."
impact: internal
resolved_by:
  commit: "b248cb647"
---

The merge queue fails a group whose macOS check job runs past 30 minutes, cancelling pull requests whose code is fine: #728 was cancelled at 30.3 minutes, #730 twice at the cap, and #733 once. The check job's timeout-minutes and the main ruleset's merge-queue check_response_timeout_minutes are both 30, while the macOS leg of a source change measures 23 to 30 minutes, so ordinary runner variance cancels a passing run, and nothing warns before the cancellation arrives. Ruling Z (2026-09-28, the technical facilitator) raises both limits to 45 minutes and asks for a warning once the macOS check passes 35 minutes, so a speed lane opens before any cancellation.

## Grounds

- pursued: a passing macOS check that runs between 30 and 45 minutes concludes and merges instead of being cancelled by the queue, and one that passes 35 minutes shows the warning; a merge-group run cancelled at 30 minutes after the live ruleset is raised, a check failed by the warning step, or a run past 35 minutes with no warning would show it wrong.
