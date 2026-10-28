---
schema_version: 1
id: "iss-2610262031035884"
slug: "the-register-reserves-claim-vocabulary"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 79 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:202"
remedy: "none (filed automatically)"
finding_ref: cfk-f8147f2670d14b85
---

The register reserves 'claim' vocabulary for itd-33's later-phase coordination unit, while the implement chapter ships a different 'claim' as its unit of exclusion

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:202` — "`claim.status` ∈ `{active, paused, released}` | Three-state claim lifecycle"
- End B: `.abcd/development/brief/04-surfaces/27-implement.md:5` — "makes a record the unit of exclusion between them with a claim"

The naming register reserves claim vocabulary for itd-33, a later phase: a three-state claim.status (active, paused, released) beside take/yield/escalate primitives and a work_item claim unit. The register exists so that a name is not re-minted with a different meaning. The implement chapter ships a 'claim' on a record, with a lease, lapse and release and no paused state or those primitives, as the unit of exclusion between sessions. One name now covers two different coordination concepts.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 79.
