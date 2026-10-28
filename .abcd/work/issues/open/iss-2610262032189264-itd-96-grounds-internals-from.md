---
schema_version: 1
id: "iss-2610262032189264"
slug: "itd-96-grounds-internals-from"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 251 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-96-disembark-reads-a-repo-s-naming-and.md:41"
remedy: "none (filed automatically)"
related_intents: [itd-96]
finding_ref: cfk-c40caa8358a60434
---

itd-96 grounds `internals` from conventions-tier documentation; the generated mapping table caps `internals` at partial for Tier 1 conventions

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-96-disembark-reads-a-repo-s-naming-and.md:41` — "Real documentation → grounded; a thin signal (e.g. package dirs but no architecture prose) → partial"
- End B: `.abcd/development/brief/00-meta.md:76` — "`internals` | `brief/05-internals/` | partial | partial | grounded | package layout, architecture docs, the internals chapters"

itd-96's conventions-tier adapter reports real architecture documentation as grounded. 00-meta's generated mapping table, which states the best status a lifeboat can reach at each tier, gives `internals` partial at Tier 1 conventions and grounded only at Tier 2 abcd-native. The table and the shipped adapter disagree about what the conventions tier can reach.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 251.
