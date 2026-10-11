---
schema_version: 1
id: "iss-2609261206436289"
slug: "the-verification-matrix-s-capture"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md"
resolution: "the verification matrix's capture-resolve row says grounds are recorded when given, not required"
impact: internal
resolved_by:
  commit: "6985cfb8"
---

The verification matrix's capture-resolve row says a note, an impact and grounds are all required, while grounds are optional and recorded when given (the v0.11.0 brief-surface cross-check's x-098, beside x-099 in the same row)

## Grounds

- pursued: the row now matches resolve's optional grounds; a matrix row calling grounds required again would show it wrong
