---
schema_version: 1
id: "iss-2610262030591199"
slug: "the-mental-model-calls-per-step-lanes"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 71 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:56"
remedy: "none (filed automatically)"
finding_ref: cfk-54edb2c78e9b16ce
---

The mental model calls per-step lanes planned (itd-2609212103565953), while the build chapter describes one lane per spec step as shipped behaviour

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:56` — "per-step lanes are itd-2609212103565953, planned"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:389` — "each spec step lands as one lane, and the loop takes the lane through its stages"

The mental model marks per-step lanes as planned work under itd-2609212103565953, so not yet delivered. The build chapter says it describes the part of the loop that ships, and states as present behaviour that each spec step lands as one lane, citing the same intent's criterion 4 in the brief it renders. The brief's truth rule allows only shipped or explicitly staged claims, so these two documents disagree about whether per-step lanes exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 71.
