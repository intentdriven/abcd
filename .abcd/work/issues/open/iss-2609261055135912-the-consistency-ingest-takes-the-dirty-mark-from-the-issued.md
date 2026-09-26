---
schema_version: 1
id: "iss-2609261055135912"
slug: "the-consistency-ingest-takes-the-dirty-mark-from-the-issued"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review2-itd48 item 2"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/consistency.go"
---

The consistency ingest takes the dirty mark from the issued request alone: validateConsistency (internal/core/intent/consistency.go) reads the dirty and dirty_path lines of the request and never recomputes them, though it re-reads the tree for the receipt check anyway. A request hand-edited to dirty: false over an uncommitted corpus therefore ingests rc=0, and the append-only report pins review_of_commit with no dirty mark while its findings quote text that commit does not hold.
