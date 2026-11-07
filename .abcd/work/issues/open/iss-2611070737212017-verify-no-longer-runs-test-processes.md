---
schema_version: 1
id: "iss-2611070737212017"
slug: "verify-no-longer-runs-test-processes"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/flush.go"
remedy: "Correct the comment, and add a flush-enabled pass of the flush tests to the cut or the nightly."
---

Verify no longer runs test processes with real flushes, so a defect that only a flush shows is less likely to be caught by a cut's unit passes; the comment in flush.go names only the check job as setting the switch and omits the release verify job.
