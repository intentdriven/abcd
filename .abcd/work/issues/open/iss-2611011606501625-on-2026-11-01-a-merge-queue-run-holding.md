---
schema_version: 1
id: "iss-2611011606501625"
slug: "on-2026-11-01-a-merge-queue-run-holding"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "landing build lanes through the merge queue, 2026-11-01"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
remedy: "Split the macOS merge-group race pass into shards (as ABCD_TEST_SHARDS already does for the three longest packages) or into separate jobs, so no one job runs within 15 minutes of the cap. Have the watchdog's SIGQUIT print one line naming the cap, so a timeout is not read as a test failure."
---

On 2026-11-01 a merge-queue run holding #1326 and #1322 failed and dropped both pull requests. The cause was not a test failure: the CI watchdog sent SIGQUIT to the macOS job's whole-module race pass about 80 minutes in, against a job cap of 85 (ABCD_CI_JOB_CAP_MINUTES) less a 5-minute margin. Every package that had finished passed. The slowest took 1203s (internal/surface/cli), 1511s (internal/core/implement/loop) and 1133s (internal/adapter/scanner) under the race detector, and the packages still running were killed (internal/core/memory, internal/core/release, internal/core/repolint). The macOS race pass now runs close enough to its cap that an ordinary slow runner drops a merge group, and every pull request queued with it waits through a re-arm and another full run.
