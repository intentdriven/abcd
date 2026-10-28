---
schema_version: 1
id: "iss-2610262039581980"
slug: "03-configuration-schedules-work-in-a"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 900 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md:428"
remedy: "none (filed automatically)"
related_intents: [itd-2609211913453478]
finding_ref: cfk-ec59b609a78be04c
---

03-configuration schedules work 'in a later phase'; itd-2609211913453478 retired phases as the unit of sequencing.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/03-configuration.md:428` — "Schema versioning and cross-version migration come in a later phase"
- End B: `.abcd/development/intents/shipped/itd-2609211913453478-one-page-in-the-glossary-maps-abcd-s.md:55` — "Phases and milestones are retired; sequencing is dependencies plus the lifecycle shelves"

itd-2609211913453478 rules that phases and milestones are retired and that sequencing is dependencies plus the lifecycle shelves. 03-configuration still says schema versioning and migration come in a later phase, that migrators are added if a later phase changes the shape, that scheduled sync comes in a later phase, and that cross-project memory pages are a later phase. It uses the retired unit as if it still sequences work.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 900.
