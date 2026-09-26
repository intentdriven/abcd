---
schema_version: 1
id: "iss-2609251451434656"
slug: "intent-audit-ingest-reviewblockrange-treats-any-text-under"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/audit.go"
---

intent audit ingest: reviewBlockRange treats any text under an INGESTED review block, up to the next marker, heading or end of file, as part of the block, so a re-ingest of an IDENTICAL payload on a record with hand-written prose under the block reports replaced:true and deletes that prose, where the documented behaviour is a noop. No record in the tree has the shape today. Fix: bound the block's extent with a closing marker, keeping the current rule as the fallback for blocks written without one.
