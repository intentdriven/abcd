---
schema_version: 1
id: "iss-2610150500162972"
slug: "ubuntu-check-leg-cancelled-at-its-30"
severity: "major"
category: "process"
source: "agent-observation"
found_during: "the cancel policy's speed lane for the ubuntu check leg, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
remedy: "Cut the ubuntu check job's wall time by at least five minutes without raising its timeout or changing a required check or the step split: name the slowest packages first in both test steps, share the live-repository fixtures the heaviest race tests rebuild, and make the run lock's retry delay injectable for the take-back test, then confirm from the job's step times on the merge queue."
---

The ubuntu leg of ci.yml's check job (timeout-minutes 30) now runs at its limit for every pull request, and the cancel policy of 2026-09-28 has fired: a check cancelled at the cap twice for the same reason stops that pull request and opens a speed lane. PR #1003 (feat/memory-recall-notes) was cancelled at 30 minutes twice: run 38676280619 attempt 2 spent 03:58:37-04:06:22 in Test (about 8 minutes) and was still in Test (race, internal) from 04:06:22 when it was cancelled at 04:27:48. PR #1008 passed in 25m40 only on a rerun after its first attempt was cancelled at 30m05, and the merge-queue entry for PR #976 (run 38680018438) was cancelled at 30 minutes too, with internal/core/implement/loop still running. On passing runs the race step took 16-18 minutes and the plain step 6-7; the race step's package times sum to about 3,500-3,750 seconds on the four-core runner, so it is bound by total CPU rather than by one package, and its heaviest packages are internal/core/lifeboat (578-594s), internal/core/reading (547-557s), internal/core/launch (357-365s) and internal/surface/cli (333-367s). iss-2610142326511964 covers the macOS leg's slowdown; this is the ubuntu leg. <!-- record-lint: forward-looking -->
