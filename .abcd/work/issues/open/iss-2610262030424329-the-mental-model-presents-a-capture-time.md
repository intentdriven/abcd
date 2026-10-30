---
schema_version: 1
id: "iss-2610262030424329"
slug: "the-mental-model-presents-a-capture-time"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 34 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:74"
remedy: "Edit 03-mental-model.md (the fourth-capture-verdict paragraph): state that itd-44's classifier and the decision verdict are staged and that nothing yet reads suggested_kind, matching 05-intent."
finding_ref: cfk-019040a05682f92b
---

The mental model presents a capture-time classifier that emits the decision verdict; the intent chapter says nothing of it exists or reads the field

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:74` — "The capture-time classifier (`intent classify-capture-kind`, itd-44) recognises this signature and emits a fourth"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:93` — "Nothing enforces it today, and there is nothing yet to enforce it against"

03-mental-model states in present tense, with no staged marker, that the capture-time classifier (intent classify-capture-kind, itd-44) recognises a standing choice and emits decision, which plan_single/reclassify refuse. 05-intent says this is a later phase: nothing enforces it, the create seeds suggested_kind null, and no shipped code reads the field.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 34.
