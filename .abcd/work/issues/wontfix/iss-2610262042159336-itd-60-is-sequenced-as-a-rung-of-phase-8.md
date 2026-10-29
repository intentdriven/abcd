---
schema_version: 1
id: "iss-2610262042159336"
slug: "itd-60-is-sequenced-as-a-rung-of-phase-8"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1057 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-60-doc-fidelity-anti-drift.md:28"
remedy: "none (filed automatically)"
related_intents: [itd-60]
finding_ref: cfk-74f3affd7ac6d54e
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-60's Phase 8 placement was true when written and the phase document still exists as history; the glossary's 0-7 count is an old point-in-time statement in an entry the record already marks superseded."
---

itd-60 is sequenced as a rung of Phase 8; the phase entry says roadmap phases run 0–7, and it retires the phase.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-60-doc-fidelity-anti-drift.md:28` — "Phase 8 — the brief is the shipped state"
- End B: `.abcd/development/brief/glossary/core/phase.md:43` — "Roadmap phases run 0–7 and the brief's plumbing-phases run 0–6"

itd-60 places itself in 'Phase 8 — the brief is the shipped state' and its Decisions say it is sequenced as Phase 8. The phase glossary entry says roadmap phases run 0–7, and it marks the phase superseded as the sequencing layer on 2026-09-21. Either a ninth phase exists that the 0–7 numbering leaves out, or itd-60's sequencing points at a phase the glossary does not recognise.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1057.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-60's Phase 8 placement was true when written and the phase document still exists as history; the glossary's 0-7 count is an old point-in-time statement in an entry the record already marks superseded.
