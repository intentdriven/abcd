---
schema_version: 1
id: "iss-2608291814566067"
slug: "memory-file-back-writes-pages-unredacted"
severity: "major"
category: "security"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "internal/core/memory/ask.go"
resolution: "WritePages redacts every page body through the store redactor before it renders and locks, so ask --file-back lands scanned like ingest; Ingest's per-body loop collapsed into it and its redactor now serves the --keep-original copy only; a test drives a file-back body carrying a synthetic token and a home path through Ask and reads the page, index and log back"
impact: fix
---

ultra-v0.6.8 altitude 1: the store redactor (internal/core/memory/redact.go) is applied at the Ingest call site only. fileBack in internal/core/memory/ask.go builds PageWrites from the host-delegated distiller's page and hands them to WritePages with no redaction, so a secret or an absolute home path the distiller echoes into a page body lands in the committed .abcd/memory store unscanned — GHSA-j5f5-phgm-9m73 one door over. Fix: redact inside WritePages, the single write primitive, so no PageWrite reaches disk unscanned; Ingest's per-body loop then collapses into it.
