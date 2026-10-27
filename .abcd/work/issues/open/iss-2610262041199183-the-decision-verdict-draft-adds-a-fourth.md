---
schema_version: 1
id: "iss-2610262041199183"
slug: "the-decision-verdict-draft-adds-a-fourth"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 992 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md:80"
remedy: "none (filed automatically)"
related_intents: [itd-44]
finding_ref: cfk-a7eb62b7ff9f24c2
---

The decision-verdict draft adds a fourth verdict to the capture-time kind classifier; the verification matrix stages that classifier as unbuilt.

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/02-verification-matrix.md:80` — "the capture-time `suggested_kind` classifier, and the shape-classification role"
- End B: `.abcd/development/intents/drafts/itd-44-fourth-intent-kind-decision.md:106` — "The capture-time kind classifier gains a fourth verdict"

itd-44 extends an existing capture-time kind classifier with a `decision` verdict and keeps the existing suggestions unchanged as regression fixtures. The matrix's Three intent kinds row lists the capture-time `suggested_kind` classifier as staged. The draft extends a classifier that does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 992.
