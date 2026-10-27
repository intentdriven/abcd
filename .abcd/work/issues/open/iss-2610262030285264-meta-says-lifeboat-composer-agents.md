---
schema_version: 1
id: "iss-2610262030285264"
slug: "meta-says-lifeboat-composer-agents"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 4 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/00-meta.md:41"
remedy: "none (filed automatically)"
finding_ref: cfk-918b2100ea9b1784
---

Meta says lifeboat-composer agents produce the skeleton's shape; the disembark chapter says pack dispatches no agents

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/00-meta.md:41` — "uses this skeleton's *shape* as a target schema for what lifeboat-composer agents must produce."
- End B: `.abcd/development/brief/04-surfaces/02-disembark.md:75` — "Packing is one deterministic Go run. It dispatches no agents and runs no model passes"

00-meta says disembark uses the brief skeleton as the target schema for what lifeboat-composer agents must produce. The disembark chapter says pack is one deterministic Go run that dispatches no agents and grounds each brief section from the mapping. Neither the skeleton's sections nor any agent named lifeboat-composer is produced by an agent at pack time.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 4.
