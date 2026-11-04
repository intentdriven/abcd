---
schema_version: 1
id: "iss-2611040337111563"
slug: "implement-step-and-land-refuse-to-push"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "the v0.17.0 cut and the overnight lanes, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/land.go"
remedy: "Add a push-and-open-unarmed mode, which the release hold permits since nothing can merge from it; a test lands under another copy's hold and asserts a pull request opens unarmed."
---

implement step and land refuse to push during another copy's release, although the repository's rule is that a change ready while a peer is mid-cut is handed over as a pull request: on 2026-11-04 a walled session finished a lane during the v0.17.0 cut and had to push and open its pull request by hand.
