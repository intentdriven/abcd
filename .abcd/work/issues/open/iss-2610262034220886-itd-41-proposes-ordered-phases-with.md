---
schema_version: 1
id: "iss-2610262034220886"
slug: "itd-41-proposes-ordered-phases-with"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 490 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-41-phase-negotiator.md:30"
remedy: "Supersede the draft phase negotiator itd-41 by adr-2609212115255771 (ruled 2026-10-27): every output it promises names a unit the record retired."
related_intents: [itd-41]
finding_ref: cfk-cd1bb884580b5d17
deferred_after: v0.15.1
deferral_reason: "Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes."
---

itd-41 proposes ordered phases with Milestone and Phase Acceptance sections; the intent chapter says phases are retired.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-41-phase-negotiator.md:30` — "proposes an ordered set of phases, each with a draft `## Expectation`, `## Milestone`, and `## Phase Acceptance`"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:95` — "Phases are retired, and this check is what replaces the same-phase rule a bundle once carried"

itd-41 is a negotiator that proposes ordered phases, each with a draft `## Expectation`, `## Milestone` and `## Phase Acceptance`. 05-intent.md says phases are retired, citing adr-2609212115255771 (phases and milestones are retired), and that a check now replaces the bundle's old same-phase rule. The intent's whole subject is missing from the planning model it would plug into.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 490.

## Deferral 2026-10-27

Deferred past v0.15.1: Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes.
