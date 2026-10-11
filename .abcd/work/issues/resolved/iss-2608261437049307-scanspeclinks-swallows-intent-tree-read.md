---
schema_version: 1
id: "iss-2608261437049307"
slug: "scanspeclinks-swallows-intent-tree-read"
severity: "nitpick"
category: "bug"
source: "agent-observation"
found_during: "bughunt-b-round-9"
found_at: "internal/core/lint/speclinks.go"
resolution: "ScanSpecLinks propagates intent-tree walk and read errors; missing [redacted-user] stays soft"
impact: fix
resolved_by:
  commit: "5d964a9e"
---

ScanSpecLinks swallows intent-tree read errors the spec half propagates