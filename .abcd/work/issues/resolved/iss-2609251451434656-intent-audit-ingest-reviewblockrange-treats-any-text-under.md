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
resolution: "Every renderer closes its block on an abcd-review-end line and the reader ends the block there, so a note below it is not the block's: an identical re-ingest is a noop and a replacement keeps the note (TestReingestKeepsAHumanNoteBelowTheBlock). A block written before the closing line keeps the recorded fallback rule, as this record's own fix states: prose a human writes directly under such a legacy INGESTED or DEAD_LETTER block is still inside its extent, and no tree record has that shape."
impact: fix
resolved_by:
  commit: "55964951091ab03bd9c07167ab6a26f0cb7df7c7"
---

intent audit ingest: reviewBlockRange treats any text under an INGESTED review block, up to the next marker, heading or end of file, as part of the block, so a re-ingest of an IDENTICAL payload on a record with hand-written prose under the block reports replaced:true and deletes that prose, where the documented behaviour is a noop. No record in the tree has the shape today. Fix: bound the block's extent with a closing marker, keeping the current rule as the fallback for blocks written without one.

## Grounds

- pursued: prose written below a block the ingest wrote survives any re-ingest; a re-ingest that deletes or absorbs it would show it wrong
