---
schema_version: 1
id: "iss-2610262031246090"
slug: "04-naming-marks-the-voyage-embark"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 126 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:231"
remedy: "none (filed automatically)"
finding_ref: cfk-4eda3c233b34e988
---

04-naming marks the voyage embark subtree as staged names nothing writes; the voyage entry defines the voyage as recording every embark run and lists the embark paths unmarked.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:231` — "names for an embark subtree nothing writes yet"
- End B: `.abcd/development/brief/glossary/core/voyage.md:4` — "every disembark and embark run"

04-naming's voyage/ row says disembark/history.jsonl is the one file written and that embark/provenance.json and embark/from/< timestamp>/ are staged 'names for an embark subtree nothing writes yet'. The glossary's voyage entry defines the voyage as an append-only record of what abcd did to produce a lifeboat, 'every disembark and embark run', and shows the embark paths beside history.jsonl as the layout. An embark run cannot both be recorded in the voyage and have nothing written for it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 126.
