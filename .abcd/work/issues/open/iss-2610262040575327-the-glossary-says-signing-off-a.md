---
schema_version: 1
id: "iss-2610262040575327"
slug: "the-glossary-says-signing-off-a"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 967 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/technical-facilitator.md:38"
remedy: "Per dcn-2610271112338738: the technical facilitator signs off a new dependency. Change the glossary's and the other entries that name the product thinker, and AGENTS.md's sign-off line, to name the technical facilitator."
related_intents: [itd-2610030810370060]
finding_ref: cfk-684832595c724cee
deferred_after: v0.15.1
deferral_reason: "Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes."
---

The glossary says signing off a dependency is never the technical facilitator's; the Terminal-interview intent records the technical facilitator signing one off.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/technical-facilitator.md:38` — "Not for what is built, a ruling, an adoption, a dependency sign-off or an irreversible act"
- End B: `.abcd/development/intents/planned/itd-2610030810370060-a-person-can-run-abcd-s-interviews-in-a.md:79` — "the technical facilitator, signing off a new dependency"

technical-facilitator.md lists dependency sign-off among the acts that are the product thinker's, never the facilitator's, and product-thinker.md says the same. itd-2610030810370060 decision 5 records golang.org/x/term being adopted as a new direct dependency on the technical facilitator's sign-off. A dependency cannot be signed off by a role the glossary says has no authority to do it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 967.

## Deferral 2026-10-27

Deferred past v0.15.1: Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes.
