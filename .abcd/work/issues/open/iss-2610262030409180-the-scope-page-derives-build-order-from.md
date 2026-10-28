---
schema_version: 1
id: "iss-2610262030409180"
slug: "the-scope-page-derives-build-order-from"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 29 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/04-scope.md:3"
remedy: "none (filed automatically)"
finding_ref: cfk-53eeab88b62911ca
---

The scope page derives build order from dependencies and shelves; the intent chapter names the phase plan as the canonical build order

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/04-scope.md:3` — "order comes from each record's dependencies (`blocked_by`, `builds_on`) and its shelf"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:74` — "ID number is *not* an execution-order guarantee — the canonical build order is the phase plan at"

04-scope says order comes from each record's blocked_by/builds_on edges and its shelf, with the phase documents kept only as history, while 05-intent (twice, in Intent IDs and Lifecycle) says the canonical build order is the phase plan under roadmap/phases/. Both cannot be the source of sequence.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 29.
