---
schema_version: 1
id: "iss-2610150733190782"
slug: "the-cold-reading-assembler-s-exclusion"
severity: "nitpick"
category: "security"
source: "review-followup"
found_during: "itd-2609151138388536 build review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/include.go"
remedy: "Restore an explicit exclusion row for the frozen log path (decisions.FrozenLog) beside the folder's, bumping the assembler version as the exclusion rule requires."
---

The cold-reading assembler's exclusion floor now names only the decisions folder; a managed repository's frozen .abcd/work/DECISIONS.md lost its explicit exclusion. No current include row reaches .abcd/work/ beyond issues, so nothing leaks today, but a future glob such as .abcd/work/*.md would admit the frozen log into a cold reading.
