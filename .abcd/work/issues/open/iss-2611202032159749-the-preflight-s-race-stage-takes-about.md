---
schema_version: 1
id: "iss-2611202032159749"
slug: "the-preflight-s-race-stage-takes-about"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run of 20 Nov 2026, measuring step 3 of spc-2611012047503720"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/research/notes/2026-11-20-preflight-reuse-over-ten-landings.md"
remedy: "Time each race package's always-rerun tests from one preflight's -json output, name the few that hold the stage's time, and either narrow the list where a test reads nothing the cache misses or run those tests in a stage of their own; then measure the race stage's time over ten landings again."
---

The preflight's race stage takes about as long with test results reused as without: over eight landings it reused 80% of race packages, yet the six that printed a time ran the stage 11m to 43m, so the reuse cuts the packages run but not the check's time; 44 of 132 packages still run their always-rerun tests and may hold the time.
