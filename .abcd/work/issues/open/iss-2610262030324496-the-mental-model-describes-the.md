---
schema_version: 1
id: "iss-2610262030324496"
slug: "the-mental-model-describes-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 13 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:74"
remedy: "Edit 03-mental-model.md (the fourth-capture-verdict paragraph): mark the capture-time classifier and the decision verdict as staged (itd-44, a draft, nothing built), in the naming register's words, and keep only the three-valued persisted kind as current."
finding_ref: cfk-a3375e88a7a40b71
---

The mental model describes the capture-kind classifier as operating; the naming register marks it staged with no classifier sub-verb registered

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:74` — "The capture-time classifier (`intent classify-capture-kind`, itd-44) recognises this signature and emits a fourth **verdict**, `decision`."
- End B: `.abcd/development/brief/02-constraints/04-naming.md:184` — "Capture-TIME classifier verdict, produced by the capture-kind classifier the intent surface is designed to gain; no classifier sub-verb is registered today."

The mental model says, without a staged marker, that the capture-time classifier `intent classify-capture-kind` recognises the signature and emits the `decision` verdict. The naming register marks the capture verdict staged and says no classifier sub-verb is registered today. The truth rule needs a staged claim to say so, and here the two pages give the classifier different statuses.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 13.
