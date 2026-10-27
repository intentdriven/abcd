---
schema_version: 1
id: "iss-2610262041158993"
slug: "the-dependency-graph-draft-lints-phase"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 987 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/record-families.md:30"
remedy: "In itd-78 drop the phase-scoping lint ('an intent phase-scoped earlier than its blocker's phase') and the line that phase docs remain the sequencing authority; replace with shelf-based checks such as a planned intent whose blocker is still a draft."
related_intents: [itd-78]
finding_ref: cfk-a19761344d0e34b2
---

The dependency-graph draft lints phase scoping against blockers; the record-families page says nothing sits above the intent for sequence.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/record-families.md:30` — "Sequencing is dependencies (`blocked_by`, `builds_on`) plus the shelves; nothing sits above the intent for sequence."
- End B: `.abcd/development/intents/drafts/itd-78-intent-dependency-graph.md:17` — "a phase that scopes an intent before its blocker's phase"

record-families.md says sequencing is only dependencies plus the shelves, with nothing above the intent. itd-78 proposes a deterministic lint failure for 'a phase that scopes an intent before its blocker's phase', a check that needs phase scoping the record no longer has.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 987.
