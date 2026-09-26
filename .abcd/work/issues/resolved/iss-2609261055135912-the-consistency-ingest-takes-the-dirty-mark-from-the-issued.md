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
resolution: "The consistency ingest reads the corpus's dirtiness again against the pinned commit and marks the report with the union of that reading and the request's paths, so a request edited to dirty: false, or an edit committed since the emit, still yields a report marked dirty that names the real path."
impact: fix
resolved_by:
  commit: "10c430c2"
---

The consistency ingest takes the dirty mark from the issued request alone: validateConsistency (internal/core/intent/consistency.go) reads the dirty and dirty_path lines of the request and never recomputes them, though it re-reads the tree for the receipt check anyway. A request hand-edited to dirty: false over an uncommitted corpus therefore ingests rc=0, and the append-only report pins review_of_commit with no dirty mark while its findings quote text that commit does not hold.

## Grounds

- pursued: a forged dirty: false request over an uncommitted corpus ingests to a report carrying dirty: true and the uncommitted path (TestConsistencyIngestRecomputesTheDirtyMark, and a CLI probe on a scratch copy); a report left unmarked, or naming a path other than the edited one, would show it wrong
