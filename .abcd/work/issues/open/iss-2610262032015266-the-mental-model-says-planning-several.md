---
schema_version: 1
id: "iss-2610262032015266"
slug: "the-mental-model-says-planning-several"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 209 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:56"
remedy: "none (filed automatically)"
related_intents: [itd-34]
finding_ref: cfk-87d8d89f73920b84
---

The mental model says planning several intents onto one spec (itd-34) is planned; itd-34 is shipped and records the bundle command as built.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:56` — "planning several intents onto one spec in a single act is itd-34, planned"
- End B: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:162` — "**Both remainders are built**: the bundle command and the `reclassify` verb"

The brief describes the bundle plan act as itd-34, planned. itd-34 is on the shipped shelf, and its first decision says both remainders, the bundle command and the reclassify verb, are built. The brief presents a shipped capability as not yet delivered.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 209.
