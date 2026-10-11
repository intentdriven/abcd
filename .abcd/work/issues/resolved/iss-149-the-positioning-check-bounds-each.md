---
schema_version: 1
id: "iss-149"
slug: "the-positioning-check-bounds-each"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "itd-102-security-review"
found_at: "internal/core/positioning/config.go"
resolution: "Config.Validate caps the surface count before any read, bounding the amplification of the per-surface guarded read"
impact: internal
resolved_by:
  commit: "286784c1"
---

The positioning check bounds each surface read to 1 MiB but nothing bounds the number of surfaces a committed registry may declare, so one abcd audit run over a hostile repo can be made to hold an unbounded multiple of that cap. Detector: a Config.Validate test that refuses a registry declaring more than a fixed surface count.