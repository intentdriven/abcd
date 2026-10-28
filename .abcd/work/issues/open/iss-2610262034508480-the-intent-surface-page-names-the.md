---
schema_version: 1
id: "iss-2610262034508480"
slug: "the-intent-surface-page-names-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 527 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:74"
remedy: "none (filed automatically)"
related_intents: [itd-2609211913453478]
finding_ref: cfk-e72e422b14699d9e
---

The intent surface page names the roadmap phase plan as the canonical build order; itd-2609211913453478 retires phases and puts sequencing in dependencies and the lifecycle shelves.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:74` — "ID number is *not* an execution-order guarantee — the canonical build order is the phase plan at"
- End B: `.abcd/development/intents/shipped/itd-2609211913453478-one-page-in-the-glossary-maps-abcd-s.md:55` — "Phases and milestones are retired; sequencing is dependencies plus the lifecycle shelves"

05-intent.md says an intent id is not an execution-order guarantee because the canonical build order is the phase plan under roadmap/phases/, and still describes planned/ as scoped into a roadmap phase. The shipped glossary intent rules that phases and milestones are retired and that sequencing is dependencies plus the lifecycle shelves, with the phase documents kept only as history. Both cannot name the source of build order: a reader following the surface page is sent to a retired plan rather than to the dependency edges and the shelves.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 527.
