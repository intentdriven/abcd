---
schema_version: 1
id: "iss-2610262033003041"
slug: "the-glossary-calls-a-phase-the-retired"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 346 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/ledger/position.md:25"
remedy: "none (filed automatically)"
finding_ref: cfk-fad23e81814901a1
---

The glossary calls a phase the retired sequencing unit; launch says the repo organises its work by phase

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/ledger/position.md:25` — "Do not call a position a phase (the retired sequencing unit)"
- End B: `.abcd/development/brief/04-surfaces/04-launch.md:489` — "the repo organises work by **phase**"

The ledger glossary's position entry describes 'phase' as the retired sequencing unit, which is delivery vocabulary no longer in use. The launch chapter still states that the repo organises work by phase, citing adr-9, and treats a release as what falls out when a stretch of phase work is published. A reader cannot tell from the two documents whether phases are the live unit of work.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 346.
