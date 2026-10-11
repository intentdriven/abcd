---
schema_version: 1
id: "iss-2608270908344367"
slug: "synthesis-principles-dedups-principles"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "issue-sweep-2026-08-27"
found_at: "internal/core/lifeboat/synthesis_principles.go"
resolution: "deterministicPrinciples records a duplicate-ADR drop instead of dropping it silently"
impact: internal
resolved_by:
  commit: "8219937a"
---

synthesis_principles dedups principles by ADR id with a silent first-wins map, so a genuine ADR-number collision drops a prn-adr-N with no marker while the neighbouring review and lessons ingests announce their drops