---
schema_version: 1
id: "iss-2611070737139919"
slug: "at-a-resuming-start-s-proof-a"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/drain_drive.go"
remedy: "Use the re-proving wall for that wait, as the loop does."
---

At a resuming start's proof, a usage-limit wait uses the drive's settings wall, so a harness file that goes during the wait ends the run where the loop would re-prove. It needs a failed start, then a limit, then an update, in that order.
