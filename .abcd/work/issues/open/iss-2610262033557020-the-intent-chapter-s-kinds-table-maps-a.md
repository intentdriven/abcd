---
schema_version: 1
id: "iss-2610262033557020"
slug: "the-intent-chapter-s-kinds-table-maps-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 450 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:83"
remedy: "Change the standalone row's mapping cell at 05-intent.md line 83 from 'One spec (1:1)' to one spec at a time with a remainder spec possible (1:n, invariant 17, adr-2609151513118583)."
finding_ref: cfk-8ffb357cb9856868
---

The intent chapter's kinds table maps a standalone intent to one spec (1:1); the record-families map says an intent has one or more specs.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:83` — "One spec (1:1) | itd-3, itd-4, itd-7, most of the corpus"
- End B: `.abcd/development/brief/glossary/core/record-families.md:23` — "its intent (one or more specs per intent)"

05-intent's kinds table gives standalone the mapping 'One spec (1:1)'. glossary/core/record-families.md says a spec is grouped by 'its intent (one or more specs per intent)', the 1:n relation a remainder spec depends on. A standalone intent with an open remainder spec breaks the 1:1 cell, so the two cannot both describe the relation.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 450.
