---
schema_version: 1
id: "iss-2610262044159830"
slug: "the-dependency-graph-draft-lints-phase"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1194 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-78-intent-dependency-graph.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-78, itd-2609211913453478]
finding_ref: cfk-e6639bb11f7067c8
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262044029050. Same draft edit as iss-2610262044029050, set against a different statement of the same current state."
duplicates: [iss-2610262044029050]
---

The dependency-graph draft lints phase ordering, but phase is a superseded term

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-78-intent-dependency-graph.md:17` — "a phase that scopes an intent before its blocker's phase"
- End B: `.abcd/development/intents/shipped/itd-2609211913453478-one-page-in-the-glossary-maps-abcd-s.md:40` — "`phase`, `milestone` and `roadmap` marked `status: superseded` with the successor named"

itd-78 makes one of its deterministic graph failures a phase that scopes an intent before its blocker's phase. itd-2609211913453478 marks phase, milestone and roadmap as superseded terms, so there is no live phase for that lint rule to read.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1194.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262044029050. Same draft edit as iss-2610262044029050, set against a different statement of the same current state.
