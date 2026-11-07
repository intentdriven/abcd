---
schema_version: 1
id: "iss-2611070737148510"
slug: "a-resuming-process-that-began-with-no"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner/wall.go"
remedy: "Decide whether a typed start's acceptance should apply on the retry, and record the ruling; if yes, apply it."
---

A resuming process that began with no harness resolved holds no team, so a harness that comes back under the expected name stops the run on its retry. The spec records this as intended, but the window is real.
