---
schema_version: 1
id: "iss-2608270930248619"
slug: "pack-go-s-within-is-dead-production-code"
severity: "nitpick"
category: "tech-debt"
source: "agent-finding"
found_during: "issue-sweep-review-2026-08-27"
found_at: "internal/core/lifeboat/pack.go"
resolution: "the dead within wrapper is removed and its case-folding coverage retargeted at pathOverlaps, the production symbol"
impact: internal
resolved_by:
  commit: "4bbe1ab0"
---

pack.go's within() is dead production code after the fsutil consolidation — its only caller is a test while pathOverlaps calls fsutil.PathsOverlap directly; remove the wrapper and point the test at the production path