---
schema_version: 1
id: "iss-2610262031429165"
slug: "the-mental-model-retires-the-phase-and"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 165 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:56"
remedy: "none (filed automatically)"
related_intents: [itd-41]
finding_ref: cfk-f4593bca60ed6273
---

The mental model retires the phase and the milestone, but itd-41 builds a negotiator that proposes phases and their Phase Acceptance under adr-9.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:56` — "The phase and the milestone are retired; the documents under"
- End B: `.abcd/development/intents/drafts/itd-41-phase-negotiator.md:34` — "The product thinker owns the final `## Expectation` and `## Phase Acceptance` per adr-9."

03-mental-model says the phase and the milestone are retired (adr-2609212115255771, which supersedes adr-9) and that the roadmap/phases documents stay only as history. itd-41's whole surface proposes ordered phases, each with `## Expectation`, `## Milestone` and `## Phase Acceptance`, and leaves the product thinker owning those sections per adr-9. Its subject is a unit the record no longer has, governed by a superseded ADR.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 165.
