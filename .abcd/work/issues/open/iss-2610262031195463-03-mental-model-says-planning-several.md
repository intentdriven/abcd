---
schema_version: 1
id: "iss-2610262031195463"
slug: "03-mental-model-says-planning-several"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 114 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:56"
remedy: "none (filed automatically)"
finding_ref: cfk-abfb865855dc944e
---

03-mental-model says planning several intents onto one spec in one act (itd-34) is planned; the verification matrix records the --bundle plan as shipped behaviour.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:56` — "(planning several intents onto one spec in a single act is itd-34, planned)"
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:80` — "plans several drafts as ONE shared spec"

03-mental-model says '(planning several intents onto one spec in a single act is itd-34, planned)', and describes the third auditor role as already suggesting reclassifications. The matrix's Three intent kinds (itd-34) row, a row describing behaviour the binary carries today, says intent plan with --bundle 'plans several drafts as ONE shared spec' whose close ships every member, while it stages the shape-classification role. The two documents disagree on which halves of itd-34 are delivered.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 114.
