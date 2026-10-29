---
schema_version: 1
id: "iss-2610262037072482"
slug: "the-record-families-page-says-batch-is"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 700 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/record-families.md:32"
remedy: "In record-families.md say a batch is the run's internal order and also the name of one shared-run window mode (single, claim, batch, split-roles); it is still not a record family."
finding_ref: cfk-9a5989670e3a8ecc
---

The record-families page says batch is not a term of the record; the implement chapter makes batch a named window mode.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/record-families.md:32` — "is the autonomous run's internal order, derived from dependencies and the pick, and is not a term of the record"
- End B: `.abcd/development/brief/04-surfaces/27-implement.md:87` — "single, claim, batch or split-roles"

record-families.md says a batch is the autonomous run's internal order, derived from dependencies and the pick, and not a term of the record. 27-implement.md lists batch as one of the window's modes (single, claim, batch or split-roles), set by a sub-verb and logged in the run log. The record uses as a named mode a word the glossary says it does not use, in a sense other than the internal order the glossary describes.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 700.
