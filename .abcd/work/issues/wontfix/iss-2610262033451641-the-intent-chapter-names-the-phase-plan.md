---
schema_version: 1
id: "iss-2610262033451641"
slug: "the-intent-chapter-names-the-phase-plan"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 435 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:74"
remedy: "none (filed automatically)"
finding_ref: cfk-e5058b642b326bc7
refines: [iss-2609251618079479]
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262033248041. Same End A (05-intent.md line 74, phase plan as canonical build order); End B is the superseded phase glossary entry."
duplicates: [iss-2610262033248041]
---

The intent chapter names the phase plan under roadmap/phases/ as the canonical build order; the glossary marks phase superseded, with sequencing now dependencies plus the lifecycle shelves.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:74` — "the canonical build order is the phase plan at"
- End B: `.abcd/development/brief/glossary/core/phase.md:18` — "the sequencing layer is dependencies plus the lifecycle shelves, rendered as the Now / Next / Later status block; the phase documents stay as history."

05-intent says 'the canonical build order is the phase plan at roadmap/phases/', and elsewhere that a planned intent is 'scoped into a roadmap phase', so it uses phase as the live sequencing unit. The glossary's phase entry has status superseded and says the sequencing layer is dependencies plus the lifecycle shelves, rendered as Now / Next / Later, with the phase documents kept as history. The intent chapter states a retired term in its retired sense as current fact.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 435.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262033248041. Same End A (05-intent.md line 74, phase plan as canonical build order); End B is the superseded phase glossary entry.
