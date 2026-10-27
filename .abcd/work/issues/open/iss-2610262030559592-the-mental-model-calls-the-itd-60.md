---
schema_version: 1
id: "iss-2610262030559592"
slug: "the-mental-model-calls-the-itd-60"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 62 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:79"
remedy: "none (filed automatically)"
finding_ref: cfk-814d969a0f6bb922
---

The mental model calls the itd-60 doc-fidelity loop planned and not yet exercised; the docs chapter describes the itd-60 doc-fidelity gate as a working gate in the present tense.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:79` — "This work is governed by the planned itd-60/itd-61 doc-fidelity + derivation loop — itd-62 would be that loop's first real exercise."
- End B: `.abcd/development/brief/04-surfaces/10-docs.md:66` — "**The doc-fidelity gate** (itd-60) judges whether the brief describes every surface that ships."

03-mental-model says the framework-provided discipline is governed by 'the planned itd-60/itd-61 doc-fidelity + derivation loop', and that itd-62 'would be that loop's first real exercise'. 10-docs describes the itd-60 doc-fidelity gate in the present tense: it judges whether the brief describes every surface that ships, and it refuses spec closes and release cuts. The doc-fidelity half cannot be both planned and never exercised and also a gate that runs today.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 62.
