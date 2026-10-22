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
resolution: "Both bounds are slack(time.Second), as the package's other wall-clock bounds are: a network wait runs the read out of its budget and the git trace names it, and the held read never ends before cleanup, so any finite bound shows the prompt returned at its deadline."
impact: internal
---

TestThePromptNeverWaitsOnTheNetwork (internal/core/claim/status_test.go) holds two wall-clock bounds to a bare second: that the block never waited on the network, while its read runs under the package's stretched budget, and that a held read returns at the hook's one-second deadline, about 150 ms after the read's cutoff. A loaded runner can fail either although nothing is wrong.
