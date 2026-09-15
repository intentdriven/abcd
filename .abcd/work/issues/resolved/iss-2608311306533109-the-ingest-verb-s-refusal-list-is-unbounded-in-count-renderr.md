---
schema_version: 1
id: "iss-2608311306533109"
slug: "the-ingest-verb-s-refusal-list-is-unbounded-in-count-renderr"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "The refusal list is capped in count as well as per name, with the total reported separately as refused_count, so nothing is hidden by the truncation."
impact: fix
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

The ingest verb's refusal list is unbounded in count. renderRefusals joins one entry per refused item into the durable refusal reason, and RunRecord.RefusedItems keeps one entry per refused item, with the item count payload-chosen — so a payload of many illegal items produces a refusal record and a terminal message hundreds of kilobytes long. The per-name cap on quoted field names states the principle exactly and does not apply to the refusals themselves.

## Grounds

- pursued: a per-value cap bounds nothing when the payload chooses how many values there are, and a record whose purpose is to be read has to stay readable; a refusal record of hundreds of kilobytes appearing again would show this wrong
