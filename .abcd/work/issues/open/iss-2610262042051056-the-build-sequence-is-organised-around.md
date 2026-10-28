---
schema_version: 1
id: "iss-2610262042051056"
slug: "the-build-sequence-is-organised-around"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1045 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/01-build-sequence.md:15"
remedy: "none (filed automatically)"
related_intents: [itd-2609211913453478]
finding_ref: cfk-38b88cb72e994fa1
---

The build sequence is organised around 'milestones' as a live term; the vocabulary intent marks milestone superseded.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/01-build-sequence.md:15` — "Two commitments shape every milestone."
- End B: `.abcd/development/intents/shipped/itd-2609211913453478-one-page-in-the-glossary-maps-abcd-s.md:23` — "Phase and milestone are on it too, marked superseded, with what replaced them."

The shipped vocabulary intent retires 'milestone' along with 'phase', and the glossary marks both superseded. 01-build-sequence still describes itself as the build-milestone detail and frames its sections and validation cadence by milestone ('Two commitments shape every milestone', 'After every milestone'). It uses the retired word as live vocabulary.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1045.
