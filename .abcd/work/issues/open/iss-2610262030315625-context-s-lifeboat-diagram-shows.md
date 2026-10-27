---
schema_version: 1
id: "iss-2610262030315625"
slug: "context-s-lifeboat-diagram-shows"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 10 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/02-context.md:31"
remedy: "none (filed automatically)"
finding_ref: cfk-da77b2b302b86740
---

Context's lifeboat diagram shows research/ and audit/ as pass B/C outputs; the disembark chapter says pack runs no model passes and its output shape has no such directories

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/02-context.md:31` — "passes B/C outputs + audits"
- End B: `.abcd/development/brief/04-surfaces/02-disembark.md:75` — "It dispatches no agents and runs no model passes"

The context page's pack/unpack diagram, which points to the disembark § 5 output shape as the full version, shows `research/` and `audit/` holding pass B and C outputs and audits. The disembark chapter says pack dispatches no agents and runs no model passes, and its § 5 tree has neither directory. The diagram describes outputs no pass produces.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 10.
