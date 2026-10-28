---
schema_version: 1
id: "iss-2610262032193797"
slug: "the-mental-model-calls-itd-60-planned"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 252 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:79"
remedy: "none (filed automatically)"
related_intents: [itd-60]
finding_ref: cfk-4debaca702ff2863
---

The mental model calls itd-60 planned, but itd-60 sits on the shipped shelf

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:79` — "This work is governed by the planned itd-60/itd-61 doc-fidelity + derivation loop"
- End B: `.abcd/development/intents/shipped/itd-60-doc-fidelity-anti-drift.md:32` — "abcd gains a doc-fidelity gate: a change cannot call itself shipped while"

03-mental-model says the framework-provided discipline is governed by 'the planned itd-60/itd-61 doc-fidelity + derivation loop'. itd-60, the doc-fidelity gate, is in `intents/shipped/`. The brief is meant to record the current state, but the lifecycle state it gives for itd-60 is contradicted by the shelf itd-60 sits on.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 252.
