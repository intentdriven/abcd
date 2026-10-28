---
schema_version: 1
id: "iss-2610262032470373"
slug: "the-voyage-glossary-s-example-has-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 318 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/02-disembark.md:276"
remedy: "none (filed automatically)"
finding_ref: cfk-1d16b1cbc0660ebe
---

the voyage glossary's example has the voyage log record the host oracle a lifeboat was packed with; disembark says the voyage line carries no oracle backend

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/02-disembark.md:276` — "The line carries no oracle backend and no verdict, because nothing produces them yet and an empty field would be a lie."
- End B: `.abcd/development/brief/glossary/core/voyage.md:53` — "The voyage log says this lifeboat was packed from root SHA `3f9a…` with the host oracle."

core/voyage's usage example says the voyage log records that a lifeboat was packed from a root SHA with the host oracle. 02-disembark specifies that the voyage line deliberately carries no oracle backend and no verdict, because nothing produces them at pack time and an empty field would be a lie. The log cannot both name the oracle that packed the lifeboat and carry no oracle field.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 318.
