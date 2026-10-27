---
schema_version: 1
id: "iss-2610262032001815"
slug: "the-mental-model-says-per-step-lanes-are"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 208 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:56"
remedy: "none (filed automatically)"
related_intents: [itd-2609212103565953]
finding_ref: cfk-b3e75bc581941792
---

The mental model says per-step lanes are a planned intent; that intent is on the shipped shelf and says the build lands steps one lane at a time.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:56` — "per-step lanes are itd-2609212103565953, planned"
- End B: `.abcd/development/intents/shipped/itd-2609212103565953-a-spec-lists-its-steps-and-the-build.md:21` — "A spec lists its steps, and `abcd build` lands them one at a time, each its own lane and pull request."

The brief qualifies the step-per-lane model as itd-2609212103565953, planned. That record sits under intents/shipped/ and its press release states that `abcd build` lands the steps one at a time, each as its own lane and pull request. One of the two misstates whether per-step lanes exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 208.
