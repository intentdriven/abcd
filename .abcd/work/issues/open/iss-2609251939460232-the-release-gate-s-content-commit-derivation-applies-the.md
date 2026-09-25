---
schema_version: 1
id: "iss-2609251939460232"
slug: "the-release-gate-s-content-commit-derivation-applies-the"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/releasegate_derive.go"
---

The release gate's content-commit derivation applies the version binding to the NEAREST receipts directory only. A co-batched pull request that carries its own commit-keyed receipts directory either ties with the release roll's directory or sits nearer and carries the previous version; both refuse although the roll's correct receipts are on the lineage, and the cut cannot be rerun without a fresh reviewed commit. Candidates should be filtered by version first and the nearest taken from those.
