---
schema_version: 1
id: "iss-2609301303434847"
slug: "a-handed-back-implement-loop-run-names"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
remedy: "Name the run's directory, .abcd/.work.local/run/<run-id>, as the way out in both the refusal's remedy and the exclusion reason (removing it clears the handed-back run), until itd-50 criterion 3 lands terminal liveness with the drafts/ move; grounds: review-fixLoop item 4, which rules making the run terminal today unsafe because build next would re-pick the falsified intent."
resolution: "The handed-back refusal and build next's exclusion both name the run's directory, .abcd/.work.local/run/<run-id>, as the way out; the exclusion no longer invites a resume that refuses."
impact: fix
resolved_by:
  commit: "528fbe006"
---

A handed-back implement-loop run names no way out: handedBackRefusal's remedy (internal/core/implement/loop/handback.go) says only that the loop starts nothing further, and build next's exclusion of the intent (next.go) says 'resume it with abcd implement step', a step that refuses on a handed-back lane. The run stays live by construction (State.Complete is false at handed-back) and no verb clears it, so the person is told of no way to start over.

## Grounds

- pursued: a person holding a handed-back run is told to remove its directory once the intent is replanned, in the step's refusal and the pick's exclusion; shown wrong if either message omits the directory or build next still says to resume the run
