---
schema_version: 1
id: "iss-26"
slug: "sources-retrieval-seam"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "sources-ingest session 2026-07-08"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: pluggable sources retrieval seam (RAG/CMS backends) has no planned intent). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

sources corpus retrieval seam: the MVP is the current native store (per-source folders, text.md extraction, grep-based consult — shipped) and retrieval/processing becomes a pluggable seam with opt-in deeper backends, mirroring the spec seam pattern (native minimal default, adapter for depth): a RAG backend (RAG-Anything, already downloaded locally, has its own content processor that could replace pandoc/pdftotext extraction and serve semantic retrieval) and/or a CMS backend. Adapter contract, not a rewrite: consult and ingest skills keep one surface; the backend is configuration.