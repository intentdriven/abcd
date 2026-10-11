---
schema_version: 1
id: "iss-2608270908341877"
slug: "externalsourcehashes-derives-external"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "issue-sweep-2026-08-27"
found_at: "internal/core/memory/coverage.go"
resolution: "externalSourceHashes reads the plural classes shape, so a plural-shape page's source_hash reaches coverage"
impact: internal
resolved_by:
  commit: "4eb192a5"
---

externalSourceHashes derives external classes from sources entries and the scalar source.class but never the plural source.classes, so a plural-shape page's source_hash is dropped from quotation-coverage accounting — the resolved plural-classes licence gap's coverage sibling