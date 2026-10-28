---
schema_version: 1
id: "iss-2610262030567363"
slug: "the-mental-model-says-the-brief-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 63 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:85"
remedy: "none (filed automatically)"
finding_ref: cfk-377b88c76bffa00f
---

The mental model says the brief's surface chapters carry an `## Acceptance` block in Given-When-Then format; the identity chapter has no such block and names an issue as its acceptance corpus instead.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:85` — "The brief's press release and surface chapters carry an `## Acceptance` block in the same format."
- End B: `.abcd/development/brief/04-surfaces/19-identity.md:124` — "Its acceptance corpus is"

03-mental-model says the brief's press release and surface chapters carry an `## Acceptance` block in the same Given-When-Then format, as part of the uniform acceptance discipline the intent-auditor compares reality against. 19-identity has no `## Acceptance` section. Its sections run from Sub-verbs to References, and its only statement of acceptance is a sentence naming iss-143 as its acceptance corpus. None of the twelve surface chapters in this set carries the block either, so the mental model's claim does not hold for them.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 63.
