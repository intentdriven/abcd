---
schema_version: 1
id: "iss-2610262045465811"
slug: "itd-60-is-sequenced-as-phase-8"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1297 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-60-doc-fidelity-anti-drift.md:97"
remedy: "none (filed automatically)"
related_intents: [itd-60, itd-2609211913453478]
finding_ref: cfk-6526f17bba618676
refines: [iss-2609251618079479]
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. A later shipped ruling retires the unit itd-60 was sequenced in and names the replacement, so nothing needs an edit."
---

itd-60 is sequenced as Phase 8; itd-2609211913453478 retires phases as the sequencing device

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-60-doc-fidelity-anti-drift.md:97` — "**Sequenced as Phase 8**, the brief is the shipped state"
- End B: `.abcd/development/intents/shipped/itd-2609211913453478-one-page-in-the-glossary-maps-abcd-s.md:55` — "Phases and milestones are retired; sequencing is dependencies plus the lifecycle shelves"

itd-60 records as a commitment that it is sequenced as Phase 8 and scopes its gate to Phase 8. itd-2609211913453478 retires phases and milestones and makes sequencing dependencies plus the lifecycle shelves, so a shipped intent still states its sequencing in the retired unit.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1297.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. A later shipped ruling retires the unit itd-60 was sequenced in and names the replacement, so nothing needs an edit.
