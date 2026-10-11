---
schema_version: 1
id: "iss-2609291430223385"
slug: "testeightconcurrentshortlivedprocessesdo"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "autonomous run A 2026-09-29, lane statusBlock preflight-1-flake.log"
origin: researcher-authored
production_mode: hand-written
resolution: "The machineload tests read the real machine through one helper that retries a failed read up to three times, so a ps run killed by its timeout under extreme load no longer fails the gate; the assertions are unchanged. The sibling single reads in TestReadSeesThisProcess and the cleanup take the same helper."
impact: internal
resolved_by:
  commit: "6e2e5bd9a"
---

TestEightConcurrentShortLivedProcessesDoNotWarn flaked in a lane's make preflight when the machine load was about 200 on 16 cores: /bin/ps was killed, so the test failed in code the lane did not touch; the rerun passed. A gate that fails on machine load, not on the change, costs a 10-15 minute preflight rerun and teaches lanes to rerun red. Make the test tolerate a killed or timed-out ps (skip with a stated reason, or retry once), or bound what it spawns.

## Grounds

- pursued: a preflight at extreme load no longer fails TestEightConcurrentShortLivedProcessesDoNotWarn on a single killed ps run; shown wrong if the test fails again with a ps kill while a later read in the same run would have succeeded
