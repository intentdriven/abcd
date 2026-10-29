---
schema_version: 1
id: "iss-2610262032502060"
slug: "the-voyage-glossary-says-the-voyage"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 325 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/03-embark.md:251"
remedy: "In glossary/core/voyage.md say the log records every disembark run and that the embark/ subtree is staged (nothing writes it), as 03-embark.md and 02-constraints/04-naming.md already say."
finding_ref: cfk-b58173cf71021357
---

the voyage glossary says the voyage records every embark run; embark says voyage provenance on embark is not built

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/03-embark.md:251` — "an `embark/provenance.json` under the voyage store recording the source path, the source manifest hash, the timestamp and the files written"
- End B: `.abcd/development/brief/glossary/core/voyage.md:4` — "an append-only record of what abcd *did* to produce a lifeboat (every disembark and embark run)"

core/voyage defines the voyage as an append-only record of every disembark and embark run, and lists embark/provenance.json in its layout. 03-embark places voyage provenance on embark in its design-target surface, none of which is built, and marks the embark/ subtree of the voyage layout not built yet. An embark run today leaves no voyage record, so the definition overstates what the namespace holds.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 325.
