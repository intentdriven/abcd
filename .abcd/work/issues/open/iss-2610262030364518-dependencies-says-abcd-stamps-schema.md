---
schema_version: 1
id: "iss-2610262030364518"
slug: "dependencies-says-abcd-stamps-schema"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 21 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/02-dependencies.md:95"
remedy: "In 02-dependencies.md Out of scope, replace 'abcd stamps `schema_version: 1` everywhere' with what holds: each record family stamps its own `schema_version` at its own current value (the voyage log and lifeboat provenance at 2, the changelog ingest at 2, the reading manifest at 11), some committed records carry none (03-configuration.md counts six of fourteen), and at least one family already migrates a version-1 file; keep the itd-9 pointer for lifeboat migration."
finding_ref: cfk-5c308a125dd502a9
---

Dependencies says abcd stamps `schema_version: 1` everywhere; the voyage log's documented entry carries `schema_version` 2

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/02-dependencies.md:95` — "abcd stamps `schema_version: 1` everywhere"
- End B: `.abcd/development/brief/04-surfaces/03-embark.md:288` — ""schema_version": 2,"

The dependencies page says abcd stamps `schema_version: 1` everywhere, with migrators added only if a later intent changes the shape. The embark chapter documents the voyage log entry with `schema_version` 2. A shape has already moved past version 1 without the migration stance being updated.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 21.
