---
schema_version: 1
id: "iss-2610262041122493"
slug: "the-manual-hold-draft-lifts-a-hold-with"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 984 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md:33"
remedy: "Supersede or re-scope itd-57 together with iss-2610262041112989: abcd intent hold already parks a record, and there is no spec dependency, depends_on field or spec rm-dep verb."
related_intents: [itd-57]
finding_ref: cfk-dd660ddf46e26bcd
---

The manual-hold draft lifts a hold with a native `spec rm-dep`; the verification matrix says the spec core has no graph and `spec` has one sub-verb.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/02-verification-matrix.md:33` — "the spec core carries no graph, and `spec` registers one sub-verb"
- End B: `.abcd/development/intents/drafts/itd-57-manual-hold-sentinel.md:17` — "To unblock, one native `spec rm-dep` lifts the hold."

itd-57 assumes native spec dependencies, holds placed through `depends_on` and lifted with `spec rm-dep`. The matrix stages the dependency graph over specs and says the spec core carries no graph and `spec` registers a single sub-verb. Neither the dependency nor the verb that removes it exists.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 984.
