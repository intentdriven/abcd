---
schema_version: 1
id: "iss-2608270500195431"
slug: "graveyard-layer-2-keys-adr-findings-by"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "github-ledger-dedup-2026-08-27"
found_at: "internal/core/lifeboat (graveyard layer-2)"
resolution: "graveyard ADR findings key on the canonical adr handle and announce shadowed claimants"
impact: fix
resolved_by:
  commit: "78487283"
---

graveyard layer-2 keys ADR findings by the filename number, so two ADRs sharing it collapse to one finding and adr-012 vs adr-12 defeats cross-home dedup. GitHub mirror: #291