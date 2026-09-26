---
schema_version: 1
id: "iss-260"
slug: "record-probe-unreadable-bucket-diagnostic"
severity: "nitpick"
category: "ux"
source: "impl-review"
found_during: "spc-25 build, ruthless-reviewer note"
found_at: "internal/core/capture/recordref.go"
resolution: "findRecordFile returns an error for a bucket it cannot read: a symlinked or non-directory bucket is refused as path-unsafe instead of being followed, any other read failure is reported as itself, and only an absent bucket stays soft; resolve and promote relay it with nothing written."
impact: fix
resolved_by:
  commit: "e2abfdf8"
---

findRecordFile treats any os.ReadDir failure as an empty bucket, so an unreadable (not absent) intent/spec bucket — mode 000, or a symlinked bucket dir — makes capture promote --intent and capture resolve --intent/--spec report 'not found in the store' instead of the actual I/O error; the old intent.Load path in promote link mode reported the read error and refused symlinked buckets. Still fail-closed (nothing written), but the diagnostic lies about the cause.

## Grounds

- pursued: an operator told a linked record is missing can trust that it is missing; a symlinked bucket still answered as not found or followed would show it wrong
