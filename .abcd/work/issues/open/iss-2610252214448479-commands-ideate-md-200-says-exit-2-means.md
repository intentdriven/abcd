---
schema_version: 1
id: "iss-2610252214448479"
slug: "commands-ideate-md-200-says-exit-2-means"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S07"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ideate/record.go"
remedy: "Exit 1 for the rollback-failure fault, or state that exception on both pages."
---

commands/ideate.md:200 says exit 2 means 'nothing was written', and 18-ideate.md:178-180 says 'a run never leaves a verdict nothing points at'; when the pointer write fails and the rollback also fails, internal/core/ideate/record.go:219-221 returns 'THE ROLLBACK FAILED — remove ... by hand', which internal/surface/cli/ideate.go:76-83 exits 2 with the verdict still on disk.
