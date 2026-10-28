---
schema_version: 1
id: "iss-2610262035439719"
slug: "the-reading-position-glossary-entry-says"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 592 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/reading-position.md:60"
remedy: "none (filed automatically)"
finding_ref: cfk-40af2c5a36de4619
---

The reading-position glossary entry says the comparative position does not assemble and refuses; the reading chapter says it assembles from a prior widening run

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/reading-position.md:60` — "does not assemble and refuses: its object has no channel"
- End B: `.abcd/development/brief/04-surfaces/23-reading.md:61` — "derives its candidate set from the record"

The glossary's worked example states that `comparative` does not assemble and refuses because its object has no channel. The reading surface chapter states that the comparative position derives its candidate set from the record: it selects the one committed widening run at the target and hands the reading that run's items. One position cannot both refuse to assemble and assemble from a derived candidate set. The same glossary entry also still describes a position as reading 'the scope it was commissioned about', the model the chapter says its two-operand invocation superseded.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 592.
