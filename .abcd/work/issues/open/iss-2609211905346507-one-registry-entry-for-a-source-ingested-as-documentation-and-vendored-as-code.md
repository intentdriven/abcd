---
schema_version: 1
id: "iss-2609211905346507"
slug: "one-registry-entry-for-a-source-ingested-as-documentation-and-vendored-as-code"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "product thinker interview closing itd-36 as delivered, 2026-09-21"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/memory/provenance.go; itd-26 (loot, draft)"
---

One registry entry for a source ingested as documentation and vendored as code. itd-36 (closed as delivered on 2026-09-21) asked that the same source content ingested by memory ingest and by the code-vendoring path (itd-26, loot, still a draft) share one provenance registry entry with an ingest count of two; the vendoring path does not exist, so the registry has one writer and no count. Wanted, when itd-26 is planned: the registry keyed by source hash across both paths, the count, and one licence declaration read by both.
