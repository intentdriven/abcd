---
schema_version: 1
id: "iss-2608270500202696"
slug: "validatesourceblock-s-single-source"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "github-ledger-dedup-2026-08-27"
found_at: "internal/core/memory (validateSourceBlock)"
resolution: "validateSourceBlock's single-source branch enforces the class-conditional provenance fields 07-memory.md requires"
impact: fix
resolved_by:
  commit: "753a6a3d"
---

validateSourceBlock's single-source branch requires only class, so an external_* page written without source_hash passes validation and abcd memory lint reports only an MQ003 info whose 'no external source' reason is false, so the MQ001/MQ002 quotation-budget blockers never run. GitHub mirror: #362