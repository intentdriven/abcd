---
schema_version: 1
id: "iss-2610262043456762"
slug: "itd-142-says-the-tool-never-recommends"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1159 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-142-the-brief-creation-interview-abcd.md:34"
remedy: "none (filed automatically)"
related_intents: [itd-142, itd-42]
finding_ref: cfk-d4b0bd6cc4d151cb
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. The principle's Bounds section separates conjectural questions from record-settled ones, and itd-142 states its rule at conjectural questions only; itd-42's decision 2 concerns overlaps with named invariants and forbids a marked option."
---

itd-142 says the tool never recommends; itd-42 lets the grill pre-pass recommend an answer in prose

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-142-the-brief-creation-interview-abcd.md:34` — "always available — the tool never recommends"
- End B: `.abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:94` — "the pre-pass may recommend one with its reason in the prose beside the question, never as a marked option"

itd-142 states that at a conjectural question the tool never recommends. itd-42's ruled decision lets the coherence pre-pass recommend one of the standard answers, with its reason, in the prose beside the question. It forbids only a marked option. These cannot both be abcd's interview stance: one forbids any recommendation, and the other allows one in prose.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1159.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. The principle's Bounds section separates conjectural questions from record-settled ones, and itd-142 states its rule at conjectural questions only; itd-42's decision 2 concerns overlaps with named invariants and forbids a marked option.
