---
schema_version: 1
id: "iss-2610262044146391"
slug: "the-phase-negotiator-drafts-phases"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1193 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-41-phase-negotiator.md:30"
remedy: "none (filed automatically)"
related_intents: [itd-41, itd-2609211913453478]
finding_ref: cfk-392892670c0b4283
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262034220886. Same draft edit as iss-2610262034220886, set against a different statement of the same current state."
duplicates: [iss-2610262034220886]
---

The phase negotiator drafts phases, milestones and phase acceptance, but the shipped glossary intent retired phases and milestones

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-41-phase-negotiator.md:30` — "proposes an ordered set of phases, each with a draft `## Expectation`, `## Milestone`, and `## Phase Acceptance`"
- End B: `.abcd/development/intents/shipped/itd-2609211913453478-one-page-in-the-glossary-maps-abcd-s.md:55` — "Phases and milestones are retired; sequencing is dependencies plus the lifecycle shelves"

itd-41's whole deliverable is a proposed ordered set of phases, each with an Expectation, a Milestone and a Phase Acceptance section (authored per adr-9). itd-2609211913453478 rules that phases and milestones are retired, sequencing is dependencies plus the lifecycle shelves, and adr-9 is superseded. A negotiator for a retired concept cannot be built against the current vocabulary, so both records cannot stand.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1193.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262034220886. Same draft edit as iss-2610262034220886, set against a different statement of the same current state.
