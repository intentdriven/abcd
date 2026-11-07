---
schema_version: 1
id: "iss-2611060809397681"
slug: "the-capture-wontfix-row-at-abcd"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md"
remedy: "Drop the timestamp from the row, or have the wontfix transition write one."
refines: [iss-2611060809386735]
---

The capture wontfix row at .abcd/development/brief/06-delivery/02-verification-matrix.md:65 says `abcd capture wontfix` moves the issue with the reason and timestamp populated. The moved record carries `wontfix_reason` and a Grounds section and no date or time field, and neither internal/core/capture nor internal/core/issueschema defines one for a wontfix.
