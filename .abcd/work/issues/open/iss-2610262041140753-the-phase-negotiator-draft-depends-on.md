---
schema_version: 1
id: "iss-2610262041140753"
slug: "the-phase-negotiator-draft-depends-on"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 986 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/phase.md:18"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): Is the phase negotiator withdrawn because phases are retired, or re-scoped to negotiate dependency order and Now / Next / Later placement? Recommend withdraw: its output unit no longer exists and a dependency-order negotiator would be a different intent to capture fresh."
related_intents: [itd-41]
finding_ref: cfk-8193a92b0cb1e676
---

The phase-negotiator draft depends on authoring new phase docs; the glossary retires the phase, keeping its documents as history.

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/phase.md:18` — "Superseded on 2026-09-21 (adr-2609212115255771): the sequencing layer is dependencies plus the lifecycle shelves"
- End B: `.abcd/development/intents/drafts/itd-41-phase-negotiator.md:42` — "writing the canonical `roadmap/phases/phase-N-*.md` files is the product thinker's act"

phase.md records that phases were retired on 2026-09-21: sequencing is now dependencies plus the lifecycle shelves, and the phase documents stay only as history. itd-41's output is proposed phases with Expectation, Milestone and Phase Acceptance sections, which the product thinker writes into new `roadmap/phases/phase-N-*.md` files. What it produces and depends on is a unit the record no longer has.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 986.
