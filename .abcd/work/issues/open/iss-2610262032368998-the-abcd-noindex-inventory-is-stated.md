---
schema_version: 1
id: "iss-2610262032368998"
slug: "the-abcd-noindex-inventory-is-stated"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 292 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:476"
remedy: "none (filed automatically)"
finding_ref: cfk-a5017834b2874c2f
---

the ~/.abcd.noindex inventory is stated twice and says load-limits holds three limits in ahoy but two in configuration

A scope leakage (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:476` — "the load check's per-machine limits (stray-minutes, extreme-load, wait-load)"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:661` — "the load check's two limits in `load-limits`"

Both chapters say their two inventories of ~/.abcd.noindex are one list and must agree. The ahoy tree says load-limits holds three limits (stray-minutes, extreme-load, wait-load), while the configuration prose says it holds two. Because the same ground is covered twice, the two copies now disagree.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 292.
