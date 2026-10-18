---
schema_version: 1
id: "iss-2610181153288506"
slug: "testthepromptneverwaitsonthenetwork"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "review of the fix for iss-2610180735158998"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/status_test.go"
remedy: "Hold the bound to slack(time.Second), as the package's other wall-clock bounds are, now that the origin really hangs and the announcement and the git trace also catch a network wait."
resolution: "The bound is slack(time.Second), as the package's other wall-clock bounds are; the hanging origin, the announcement and the git trace catch a network wait."
impact: internal
---

TestThePromptNeverWaitsOnTheNetwork (internal/core/claim/status_test.go) decides that the block never waited on the network by a bare one-second wall-clock bound, while its read runs under the package's stretched budget, so a loaded runner can fail it although nothing reached the network.
