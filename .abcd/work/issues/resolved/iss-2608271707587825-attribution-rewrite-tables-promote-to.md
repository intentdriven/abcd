---
schema_version: 1
id: "iss-2608271707587825"
slug: "attribution-rewrite-tables-promote-to"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: ".abcd/work/attribution-rewrite-2026-08-06"
resolution: "attribution-rewrite tables promoted to development/research/data/; reviews charter and DECISIONS.md repointed; sha-map preserved"
impact: internal
resolved_by:
  commit: "d7763bde"
---

promote the attribution-rewrite tables to the durable record: .abcd/work/attribution-rewrite-2026-08-06/ holds the sha and tag translation tables for the 2026-08-06 history rewrite, and its own README declares them historical and complete (one event, nothing appends) — durable-record shape, not working state. The 2026-08-27 structural review also found the sha-map is the only translation key for the five pre-rewrite receipt-directory ids under .abcd/work/reviews/. Move the directory to .abcd/development/research/data/ via the tier charter's promotion path (an artefact that proves durable moves up, in a change that says why), repoint the reviews-charter reference and the DECISIONS.md mention, and log the supersession of the highlight-not-inventory adjudication for this path. Deletion is off the table: the 725-row sha-map is the sole old-to-new history bridge.