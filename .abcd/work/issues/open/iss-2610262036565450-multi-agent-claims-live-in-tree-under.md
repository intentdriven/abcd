---
schema_version: 1
id: "iss-2610262036565450"
slug: "multi-agent-claims-live-in-tree-under"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 686 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/04-universal-patterns.md:139"
remedy: "none (filed automatically)"
finding_ref: cfk-ad09e984cb3b7cba
---

Multi-agent claims live in-tree under `.abcd/coordination/` per the universal patterns, but in the machine-scoped run store per the implement chapter

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/04-universal-patterns.md:139` — "Coordination state (file locks like `shape.lock`, multi-agent claims per itd-33) lives at `.abcd/coordination/`"
- End B: `.abcd/development/brief/04-surfaces/27-implement.md:3` — "It holds a run's shared state in the machine-scoped store, records the sessions that join it, makes a record the unit of exclusion between them with a claim"

04-universal-patterns places multi-agent claims (itd-33) under `.abcd/coordination/`, beside a committed coordination append-log with daily UTC rotation. 27-implement keeps the claim that makes a record the unit of exclusion between concurrent sessions in the machine-scoped store, with its own per-UTC-day run log, and has the sessions write no repository file. Both documents claim the same ground, the exclusion record between concurrent agents, and give it incompatible homes.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 686.
