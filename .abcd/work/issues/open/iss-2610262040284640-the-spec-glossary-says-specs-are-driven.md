---
schema_version: 1
id: "iss-2610262040284640"
slug: "the-spec-glossary-says-specs-are-driven"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 933 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/spec.md:29"
remedy: "none (filed automatically)"
finding_ref: cfk-d3c82e7fbee56a73
---

The spec glossary says specs are driven to completion by the pluggable run seam; the build sequence says the run seam does not ship.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/spec.md:29` — "completion by the pluggable run seam"
- End B: `.abcd/development/brief/06-delivery/01-build-sequence.md:141` — "The seam does not ship: the binary registers no `run` verb"

The spec entry names the pluggable run seam (adr-27) as what drives a spec to completion. The build sequence says the seam does not ship: the binary registers no run verb, and the run adapter is on the planned-seams list. Nothing can be driven by a seam that does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 933.
