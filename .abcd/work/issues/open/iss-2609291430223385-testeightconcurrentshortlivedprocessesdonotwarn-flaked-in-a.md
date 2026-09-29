---
schema_version: 1
id: "iss-2609291430223385"
slug: "testeightconcurrentshortlivedprocessesdonotwarn-flaked-in-a"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "autonomous run A 2026-09-29, lane statusBlock preflight-1-flake.log"
origin: researcher-authored
production_mode: hand-written
---

TestEightConcurrentShortLivedProcessesDoNotWarn flaked in a lane's make preflight when the machine load was about 200 on 16 cores: /bin/ps was killed, so the test failed in code the lane did not touch; the rerun passed. A gate that fails on machine load, not on the change, costs a 10-15 minute preflight rerun and teaches lanes to rerun red. Make the test tolerate a killed or timed-out ps (skip with a stated reason, or retry once), or bound what it spawns.
