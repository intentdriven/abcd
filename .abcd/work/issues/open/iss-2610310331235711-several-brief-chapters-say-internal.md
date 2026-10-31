---
schema_version: 1
id: "iss-2610310331235711"
slug: "several-brief-chapters-say-internal"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Add internal/surface/session to the layouts in 05-internals/03-configuration.md, 06-delivery/01-build-sequence.md and 06-delivery/02-verification-matrix.md."
---

Several brief chapters say internal/surface holds only cli and dashboard and call the dashboard the one network listener; internal/surface/session is a second front door and listener.
