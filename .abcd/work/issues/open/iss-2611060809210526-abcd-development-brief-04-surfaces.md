---
schema_version: 1
id: "iss-2611060809210526"
slug: "abcd-development-brief-04-surfaces"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/23-reading.md"
remedy: "Count two unrecorded refusals, adding the rerun of a run that already has an outcome and why it writes no record."
---

`.abcd/development/brief/04-surfaces/23-reading.md:282` says one refusal after the identity point is deliberately unrecorded, the prose citation. `refuseARerun`, called at `internal/core/reading/ingest.go:614` after the identity is proven, also refuses and writes no refusal record, because a run that already has an outcome must not have it overwritten.
