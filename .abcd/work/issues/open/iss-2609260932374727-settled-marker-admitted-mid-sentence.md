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
---

settledMarkRe (internal/core/intent/questions.go:30) admits `resolved:` or `deferred:` anywhere in an open-question item, so an item such as 'Which id wins once the split is resolved: the old or the new?' reads as settled and build starts past a real question. No record trips it today (every intent scanned); the tightening is a label at the item's start or after a closing bold plus dash, not mid-sentence.
