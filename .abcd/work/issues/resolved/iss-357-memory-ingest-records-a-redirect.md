---
schema_version: 1
id: "iss-357"
slug: "memory-ingest-records-a-redirect"
severity: "nitpick"
category: "security"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: "internal/core/memory/ingest.go"
resolution: "the ingested redirect-controlled FinalURL is percent-encoded before it becomes a stored origin, mirroring the cite fix"
impact: internal
resolved_by:
  commit: "cea65342"
---

memory ingest records a redirect-controlled FinalURL verbatim into stored material origins — the iss-359 class one door over; the cite fix neither touched nor recorded the sibling site