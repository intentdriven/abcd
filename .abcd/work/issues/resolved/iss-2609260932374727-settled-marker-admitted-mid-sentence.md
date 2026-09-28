---
schema_version: 1
id: "iss-2609260932374727"
slug: "settled-marker-admitted-mid-sentence"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review2-loop1 item 3"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/questions.go"
resolution: "A settled label (resolved:, deferred:) counts only opening a line of an Open Questions item, after a closing bold, or after a dash; mid-sentence it is prose and the item stays a question. The bold-span marker keeps its reach. No committed intent changes verdict."
impact: fix
resolved_by:
  commit: "ef9ef3819c6401f4806b6c5353ce8ca0a29d818f"
---

settledMarkRe (internal/core/intent/questions.go:30) admits `resolved:` or `deferred:` anywhere in an open-question item, so an item such as 'Which id wins once the split is resolved: the old or the new?' reads as settled and build starts past a real question. No record trips it today (every intent scanned); the tightening is a label at the item's start or after a closing bold plus dash, not mid-sentence.

## Grounds

- pursued: we expect anchoring the label to its written positions to stop a question that uses the word mid-sentence from reading as settled without refusing any record the tree settles; it is shown wrong if a record written in the settled convention starts reading as open, or a mid-sentence label still admits an item
