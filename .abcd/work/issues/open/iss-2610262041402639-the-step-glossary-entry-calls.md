---
schema_version: 1
id: "iss-2610262041402639"
slug: "the-step-glossary-entry-calls"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1017 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/step.md:18"
remedy: "In step.md change '(itd-2609212103565953, planned)' to shipped."
related_intents: [itd-2609212103565953]
finding_ref: cfk-6eb1fcdaec10a8ee
---

The step glossary entry calls itd-2609212103565953 planned, but that intent has shipped.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/step.md:18` — "is the unit below a spec (itd-2609212103565953, planned)"
- End B: `.abcd/development/intents/shipped/itd-2609212103565953-a-spec-lists-its-steps-and-the-build.md:55` — "The unit below a spec is a section called steps, not a record family"

The step entry introduces the term as 'the unit below a spec (itd-2609212103565953, planned)'. The intent it cites is filed in intents/shipped/, and it records the ruling that the unit below a spec is a Steps section. The glossary therefore gives a stale lifecycle state for the record its own definition rests on.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1017.
