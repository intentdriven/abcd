---
schema_version: 1
id: "iss-2610262032024089"
slug: "invariant-17-makes-the-intent-to-spec"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 212 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/03-invariants.md:171"
remedy: "none (filed automatically)"
related_intents: [itd-34]
finding_ref: cfk-e065ec8140417a18
refines: [iss-2609100508566552]
---

Invariant 17 makes the intent-to-spec relation 1:n; itd-34 says a standalone intent maps 1:1 to a spec.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/03-invariants.md:171` — "the intent↔spec relation is 1:n"
- End B: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:21` — "A standalone intent maps 1:1 to a flow-next `spec`"

The brief's invariant states that an intent owns one or more specs and ships when its last spec closes. itd-34 asserts that a standalone intent maps one-to-one to a spec. Both make a claim about the same relation and cannot both hold.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 212.
