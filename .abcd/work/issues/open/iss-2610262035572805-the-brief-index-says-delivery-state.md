---
schema_version: 1
id: "iss-2610262035572805"
slug: "the-brief-index-says-delivery-state"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 611 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/README.md:13"
remedy: "none (filed automatically)"
finding_ref: cfk-ce90d36fc9a3e643
---

The brief index says delivery state never lives in brief prose; every surface chapter records each verb's shipped or staged existence in its prose table

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/README.md:13` — "lives in the roadmap dashboard, never in brief prose"
- End B: `.abcd/development/brief/04-surfaces/10-docs.md:17` — "non-assessment verb) and its existence (`shipped` / `staged`)"

The brief README says current delivery state lives in the roadmap dashboard, never in brief prose. The docs chapter, like every surface chapter, carries a machine-checked table whose rows record each verb's existence as `shipped` or `staged`, verified against the command-tree snapshot. That is per-verb delivery state stated in brief prose.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 611.
