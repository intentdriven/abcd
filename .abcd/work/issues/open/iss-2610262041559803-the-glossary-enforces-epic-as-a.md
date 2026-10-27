---
schema_version: 1
id: "iss-2610262041559803"
slug: "the-glossary-enforces-epic-as-a"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1034 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/README.md:139"
remedy: "none (filed automatically)"
related_intents: [itd-4]
finding_ref: cfk-33e4ee346d2d3f12
---

The glossary enforces 'epic' as a forbidden synonym; itd-4's issue schema names a related_epics field holding spec ids.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/README.md:139` — "This repo currently enforces one — `epic` (itd-43)"
- End B: `.abcd/development/intents/shipped/itd-4-issue-capture.md:42` — "`related_epics` (list of `spc-N`)"

The glossary README says this repository enforces one forbidden synonym, 'epic'. itd-4's ledger frontmatter schema names its spec back-references 'related_epics (list of spc-N)', so the issue schema uses the forbidden word for a spec.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1034.
