---
id: dcn-2608310000000008
slug: itd-185-s-ingest-containment-is-layered
date: 2026-08-31
---
2026-08-31 — itd-185's ingest containment is layered and the layers are
mutually redundant: writeJSONIn's contained write is currently unreachable
because refuseARerun probes the same path through the root first, and nothing
asserts that ordering. Recorded rather than fixed, because the property is
proved by mutating containment as a whole; a future edit that moves or drops
the rerun probe must re-check it. Rejected: a per-layer test, which would pin
an ordering the design does not promise.
