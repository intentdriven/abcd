---
schema_version: 1
id: "iss-2609251451432601"
slug: "intent-audit-ingest-the-first-ingest-and-any-replacement"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/audit.go"
---

intent audit ingest: the first ingest, and any replacement, deletes trailing link-reference definitions under Audit Notes when the owed stub sits above them (the itd-114 shape appendToAuditNotes parks a stub above, per iss-2608210737265820). Same root cause as the unbounded review-block extent; same fix (a bounded extent). No record in the tree has a stub followed by refs today.
