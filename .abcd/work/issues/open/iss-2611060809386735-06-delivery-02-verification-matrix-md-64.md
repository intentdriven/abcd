---
schema_version: 1
id: "iss-2611060809386735"
slug: "06-delivery-02-verification-matrix-md-64"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md"
remedy: "Drop the timestamp from both matrix rows, or have the transition write one and update the record schema."
---

`06-delivery/02-verification-matrix.md:64` says `abcd capture resolve` moves the issue with its resolution and timestamp populated, and line 65 says the same of `abcd capture wontfix` with its reason. Neither writes a time. A resolved record carries `resolution:` and `impact:`, a declined one `wontfix_reason:`, and the only time in either is the id's own mint stamp; `internal/core/capture/workflow.go` reads no clock in the transition.
