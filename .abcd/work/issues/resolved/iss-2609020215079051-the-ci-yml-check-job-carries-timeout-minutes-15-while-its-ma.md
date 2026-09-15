---
schema_version: 1
id: "iss-2609020215079051"
slug: "the-ci-yml-check-job-carries-timeout-minutes-15-while-its-ma"
severity: "major"
category: "bug"
source: "agent-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
resolution: "The check job's budget is thirty minutes, twice the quiet-run duration of its macOS leg, so a loaded runner no longer cancels the race lane and drops the merge-queue entry. The merge queue's own check_response_timeout_minutes is thirty, so the job budget now matches the queue's rather than undercutting it."
impact: internal
---

The ci.yml check job carries timeout-minutes: 15 while its macOS leg takes fourteen minutes on a quiet pull-request run (run for PR 589: started 01:28:58Z, completed 01:43:12Z). In the merge-group run for the same PR the macOS leg was cancelled at fifteen minutes inside the race lane, the merge queue removed the entry, and auto-merge was disarmed, with no code at fault. A job budget one minute above the observed quiet-run duration is a wall-clock assertion on a shared runner, the same class as iss-2608292246210181 and the bootstrap self-check site: every queue entry inherits a coin flip. The fix is headroom proportionate to the measured duration, and the durable remedy is the class remedy those records already name.

## Grounds

- pursued: the queue drop was a timing wobble on a shared runner, not a code fault; headroom proportionate to the measured duration is the class remedy the sibling records name, and the workflow is the only place the budget lives
