---
schema_version: 1
id: "iss-2608270500190007"
slug: "maxgraveyardfindingspersignal-documents"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "github-ledger-dedup-2026-08-27"
found_at: "internal/core/lifeboat (graveyard signal cap)"
resolution: "every graveyard signal caps through capSignalFindings, whose last retained finding notes the omission"
impact: fix
resolved_by:
  commit: "854900b5"
---

maxGraveyardFindingsPerSignal documents that the last retained finding notes the cap, but 6 of 10 signals truncate silently with no such note. GitHub mirror: #292