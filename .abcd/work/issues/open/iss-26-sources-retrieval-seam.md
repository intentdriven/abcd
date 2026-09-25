---
schema_version: 1
id: "iss-26"
slug: "sources-retrieval-seam"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "sources-ingest session 2026-07-08"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Plan a pluggable sources-retrieval seam (RAG or CMS backends)?"
---

sources corpus retrieval seam: the MVP is the current native store (per-source folders, text.md extraction, grep-based consult — shipped) and retrieval/processing becomes a pluggable seam with opt-in deeper backends, mirroring the spec seam pattern (native minimal default, adapter for depth): a RAG backend (RAG-Anything, already downloaded locally, has its own content processor that could replace pandoc/pdftotext extraction and serve semantic retrieval) and/or a CMS backend. Adapter contract, not a rewrite: consult and ingest skills keep one surface; the backend is configuration.