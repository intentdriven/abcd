---
schema_version: 1
id: "iss-2611040337193723"
slug: "implement-status-keeps-saying-waiting"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "the v0.17.0 cut and the overnight lanes, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "Name the actual blocker in order: a release hold, then a missing receipt; a test mints a receipt under a hold and reads status."
---

implement status keeps saying waiting for its full check, run the repository's preflight, after the receipt is minted and while the release hold is the real wait, so a person or agent is told to do what is already done.
