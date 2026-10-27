---
schema_version: 1
id: "iss-2610262032464642"
slug: "the-build-sequence-has-the-disembark"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 315 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/02-disembark.md:75"
remedy: "none (filed automatically)"
finding_ref: cfk-387bc770caa83b30
---

the build sequence has the disembark pack run the host-delegated audit; disembark says packing dispatches no agents and runs no model passes

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/02-disembark.md:75` — "It dispatches no agents and runs no model passes"
- End B: `.abcd/development/brief/06-delivery/01-build-sequence.md:155` — "synthesises the lifeboat at the operator-chosen destination, and runs the host-delegated audit"

01-build-sequence's lifeboat milestone says the pack synthesises the lifeboat at the destination and runs the host-delegated audit. 02-disembark says packing is one deterministic Go run that dispatches no agents and runs no model passes, the review and every synthesis artefact being written later by separate sub-verbs over an already-packed lifeboat. The pack cannot both run the audit and dispatch nothing.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 315.
