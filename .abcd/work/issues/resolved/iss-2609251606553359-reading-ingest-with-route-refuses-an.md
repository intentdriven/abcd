---
schema_version: 1
id: "iss-2609251606553359"
slug: "reading-ingest-with-route-refuses-an"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
resolution: "reading ingest returns the output read's own refusal (size cap, non-regular file, unreadable) when a --route is given, instead of the no-position refusal; TestReadingIngestRouteOnAnUnreadableOutputGivesTheReadsReason pins both the routed and unrouted cases."
impact: fix
resolved_by:
  commit: "ec535a9ec"
---

reading ingest with --route refuses an oversized output as "names no reading position" instead of giving the size-cap message (internal/surface/cli/reading.go:246-251), so the operator is told the wrong cause (review2-tier2 note b).

## Grounds

- pursued: an operator whose routed ingest fails on the output file is told the file's fault; a routed ingest of an oversized output still naming the position would show it wrong
