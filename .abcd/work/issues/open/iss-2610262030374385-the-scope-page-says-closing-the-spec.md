---
schema_version: 1
id: "iss-2610262030374385"
slug: "the-scope-page-says-closing-the-spec"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 24 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/04-scope.md:40"
remedy: "none (filed automatically)"
finding_ref: cfk-d55da1d9657fad74
refines: [iss-2609100508566552]
---

The scope page says closing the spec ships the intent; invariant 17 says closing a spec never ships an intent that still has an open spec

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/04-scope.md:40` — "It reaches `shipped/` when its linked spec is closed: closing the spec is what ships the intent"
- End B: `.abcd/development/brief/02-constraints/03-invariants.md:171` — "closing a spec never ships an intent that still has an open spec"

The scope page says an intent reaches `shipped/` when its linked spec closes, and that closing the spec is what ships it. Invariant 17 says an intent can own several specs, that closing one never ships an intent with another open spec, and that the intent moves only after its last spec closes. The scope page's single-spec rule gives the wrong result for the partial-delivery case.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 24.
