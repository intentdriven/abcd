---
schema_version: 1
id: "iss-2611070737124228"
slug: "a-resuming-start-with-no-bound-waits-out"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/drain_drive.go"
remedy: "Wait on a missing verdict on a resuming start too, as on a first start."
---

A resuming start with no bound waits out a usage limit but not a launch that gets no verdict. If a release fails its start and the service then gives no verdict for over three minutes, the run ends with exit 2 where a fresh start would have waited.
