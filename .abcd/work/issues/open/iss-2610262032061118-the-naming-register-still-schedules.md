---
schema_version: 1
id: "iss-2610262032061118"
slug: "the-naming-register-still-schedules"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 222 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:76"
remedy: "none (filed automatically)"
related_intents: [itd-2609211913453478]
finding_ref: cfk-72ce7b813a9bdfae
---

The naming register still schedules reserved commands by 'a later phase'; itd-2609211913453478 retired the phase as a unit of sequencing.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:76` — "cross-corpus synthesis (itd-25, a later phase)"
- End B: `.abcd/development/intents/shipped/itd-2609211913453478-one-page-in-the-glossary-maps-abcd-s.md:55` — "Phases and milestones are retired; sequencing is dependencies plus the lifecycle shelves"

itd-2609211913453478 rules that phases are retired and that sequencing is dependencies plus the lifecycle shelves. The register still places `/abcd:dredge`, `/abcd:loot`, `/abcd:audit` and the coordination enums in 'a later phase', which uses the retired term as if it still ordered work.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 222.
