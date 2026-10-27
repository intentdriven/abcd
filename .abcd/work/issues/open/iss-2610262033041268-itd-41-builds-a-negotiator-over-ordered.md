---
schema_version: 1
id: "iss-2610262033041268"
slug: "itd-41-builds-a-negotiator-over-ordered"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 356 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-41-phase-negotiator.md:30"
remedy: "none (filed automatically)"
related_intents: [itd-41]
finding_ref: cfk-eb21ca85a2d5beca
---

itd-41 builds a negotiator over ordered phases with Expectation, Milestone and Phase Acceptance; the disembark page says sequence is dependencies plus the lifecycle shelves, and phases and milestones are retired.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-41-phase-negotiator.md:30` — "proposes an ordered set of phases, each with a draft `## Expectation`, `## Milestone`, and `## Phase Acceptance`"
- End B: `.abcd/development/brief/04-surfaces/02-disembark.md:18` — "sequence is dependencies plus the lifecycle shelves ([adr-2609212115255771] (../../decisions/adrs/2609212115255771-phases-and-milestones-are-retired.md))"

Everything itd-41 outputs is a phase plan. It proposes ordered phases, each with a draft ## Expectation, ## Milestone and ## Phase Acceptance, grounds costs in phase-acceptance bullets, and leaves the product thinker to write the roadmap/phases/phase-N files under adr-9. The disembark page says sequence is dependencies plus the lifecycle shelves, citing adr-2609212115255771 (phases and milestones are retired), and calls the Phase 6 document retired. A negotiator over phases presupposes a layer the brief says is gone.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 356.
