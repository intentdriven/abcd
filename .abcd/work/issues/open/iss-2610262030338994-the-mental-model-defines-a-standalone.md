---
schema_version: 1
id: "iss-2610262030338994"
slug: "the-mental-model-defines-a-standalone"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 15 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:64"
remedy: "Edit 03-mental-model.md (standalone bullet and the diagram label '1 or N intents to 1 spec'): a standalone intent usually ships as one spec, may be finished by a remainder spec, and ships when its last spec closes (invariant 17)."
finding_ref: cfk-4c3eff9d1e1226a9
---

The mental model defines a standalone intent as shipping as one spec; invariant 17 makes an intent own one or more specs as the normal state

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:64` — "one press-release-shaped user moment, ships as one spec."
- End B: `.abcd/development/brief/02-constraints/03-invariants.md:171` — "More than one spec naming one intent is the normal state and not an ambiguity"

The mental model defines the standalone kind as one user moment that ships as one spec. Invariant 17 says the intent-to-spec relation is 1:n, that more than one spec naming one intent is normal, and that the intent ships when its last spec closes. A standalone intent planned as one spec and finished by a remainder spec breaks the mental model's definition but follows the invariant.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 15.
