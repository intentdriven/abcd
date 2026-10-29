---
schema_version: 1
id: "iss-2610262032111048"
slug: "the-mental-model-says-a-standalone"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 233 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:64"
remedy: "Edit 03-mental-model.md standalone bullet: a standalone intent may be delivered through more than one spec and ships when its last spec closes (invariant 17), with itd-2609091034175565 as the example."
related_intents: [itd-2609091034175565]
finding_ref: cfk-04108d6ba2cab3b8
---

The mental model says a standalone intent ships as one spec; itd-2609091034175565, planned alone, is delivered through two specs.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:64` — "one press-release-shaped user moment, ships as one spec"
- End B: `.abcd/development/intents/shipped/itd-2609091034175565-nothing-tells-an-agent-that-a-record-it.md:238` — "Two specs under this one intent, both built today"

The brief defines the standalone kind as one user moment that ships as one spec. The reservation intent is not a bundle, yet it is planned and delivered as two specs, and it ships only when both close. Both statements cannot describe a standalone intent's delivery.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 233.
