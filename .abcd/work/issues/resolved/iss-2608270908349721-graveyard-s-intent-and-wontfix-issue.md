---
schema_version: 1
id: "iss-2608270908349721"
slug: "graveyard-s-intent-and-wontfix-issue"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "issue-sweep-2026-08-27"
found_at: "internal/core/lifeboat/graveyard_abandoned.go"
resolution: "intent and issue graveyard ids canonicalise and record shadows like the ADR path, so itd-7 and itd-007 dedupe to one announced finding"
impact: internal
resolved_by:
  commit: "8219937a"
---

graveyard's intent and wontfix-issue signals keep the silent first-wins drop the ADR path just lost, and the package has no canonicaliser for itd or iss ids, so itd-7 and itd-007 read as distinct findings — the two remaining families of the resolved ADR collision defect