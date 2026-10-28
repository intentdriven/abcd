---
schema_version: 1
id: "iss-2610262041158645"
slug: "the-brief-change-derivation-draft-reuses"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 988 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md:78"
remedy: "none (filed automatically)"
related_intents: [itd-61]
finding_ref: cfk-8f7f62201922c906
---

The brief-change derivation draft reuses 'the existing grill's interrogation machinery'; the verification matrix says nothing of the grill ships.

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/02-verification-matrix.md:78` — "nothing of itd-27's interview ships: `intent` registers no `grill` sub-verb"
- End B: `.abcd/development/intents/drafts/itd-61-brief-change-derivation.md:40` — "Reuse of the existing grill's interrogation machinery rather than a parallel re-implementation."

itd-61 builds its derivation pass by reusing the grill's interrogation machinery instead of writing its own. The matrix's Grill row is staged in full: no grill sub-verb is registered and nothing of itd-27's interview ships. The draft depends on machinery that does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 988.
