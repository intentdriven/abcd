---
schema_version: 1
id: "iss-2610262043161268"
slug: "itd-196-makes-the-phase-its-unit-of"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1126 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-196-a-capability-is-rehearsed-end-to-end.md:40"
remedy: "none (filed automatically)"
related_intents: [itd-196, itd-2609211913453478]
finding_ref: cfk-d23b7773a5e48f31
---

itd-196 makes the phase its unit of rehearsal, while itd-2609211913453478 retired the phase as a sequencing unit

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-196-a-capability-is-rehearsed-end-to-end.md:40` — "The cumulative rung is what turns a phase into something more than a list of merged intents"
- End B: `.abcd/development/intents/shipped/itd-2609211913453478-one-page-in-the-glossary-maps-abcd-s.md:55` — "Phases and milestones are retired; sequencing is dependencies plus the lifecycle shelves"

itd-196 defines its cumulative and closing rungs over a phase ('what turns a phase into something more than a list of merged intents'), but the shipped record-families intent rules that phases and milestones are retired and that sequencing is dependencies plus the lifecycle shelves. A reader applying the discipline has no current unit that 'the phase' names, so the rung the discipline calls the one that pays cannot be located.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1126.
