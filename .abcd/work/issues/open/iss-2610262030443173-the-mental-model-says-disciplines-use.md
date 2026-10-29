---
schema_version: 1
id: "iss-2610262030443173"
slug: "the-mental-model-says-disciplines-use"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 38 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:85"
remedy: "Edit 03-mental-model.md (acceptance-discipline paragraph): discipline intents carry the rule under `## Rule` with Given-When-Then acceptance (the seven older) or under `## The rule` with `## The gate`, `## Fit` and `## Staging` (the nine since itd-190), and point at 05-intent."
finding_ref: cfk-3c9665e8c3d68e03
---

The mental model says disciplines use Given-When-Then under ## Rule; the intent chapter says nine of sixteen use a different idiom with no acceptance section

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:85` — "Discipline-kind intents skip the press release but use the same Given-When-Then format under a `## Rule` heading"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:154` — "The nine written since (itd-190 onwards) head the rule `## The rule` and carry `## The gate`, `## Fit` and `## Staging` in place of the scope, acceptance and reference sections"

03-mental-model states discipline-kind intents use the same Given-When-Then format under a ## Rule heading, while 05-intent says the nine disciplines written since itd-190 head the rule ## The rule and carry ## The gate, ## Fit and ## Staging in place of the acceptance section. Most disciplines in the tree do not take the format the mental model asserts.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 38.
