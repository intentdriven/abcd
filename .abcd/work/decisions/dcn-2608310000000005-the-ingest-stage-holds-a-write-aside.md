---
id: dcn-2608310000000005
slug: the-ingest-stage-holds-a-write-aside
date: 2026-08-31
---
2026-08-31 — The ingest stage holds a write-aside MARKER, not the rendered
reading records; `capture.IngestReading` writes them against the real ledger.
Ground: that writer mints every item id under the ledger lock precisely so its
collision probe sees the tree it is about to write into, and staging into a
second issues root would move the probe off the real ledger. The orphan sweep
rolls the run back instead, bounded by the `rdi-N.md` filename grammar.
Rejected: staging rendered records per spc-63's first draft.
