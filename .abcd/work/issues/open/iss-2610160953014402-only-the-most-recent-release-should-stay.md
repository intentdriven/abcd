---
schema_version: 1
id: "iss-2610160953014402"
slug: "only-the-most-recent-release-should-stay"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "v0.15.0 release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "release retention (launch's retention plan and the release workflow)"
remedy: "Change the retention rule from newest-per-line to newest-only (keep the release just published; delete every earlier GitHub release, never its tag), and have the release run apply the plan automatically after publish, so no cut deletes by hand."
---

Only the most recent release should stay published: once a release is published, every earlier GitHub release is deleted automatically, its tag kept. Today launch's retention plan keeps the newest release of each minor line (v0.13.4, v0.14.2, v0.15.0 and the rest), and on 2026-10-15 and 2026-10-16 the previous release was deleted by hand at each cut (v0.14.1, then v0.14.2 on the product thinker's ruling).
