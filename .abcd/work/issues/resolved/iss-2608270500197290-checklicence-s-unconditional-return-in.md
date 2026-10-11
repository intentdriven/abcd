---
schema_version: 1
id: "iss-2608270500197290"
slug: "checklicence-s-unconditional-return-in"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "github-ledger-dedup-2026-08-27"
found_at: "internal/core/memory/lint.go"
resolution: "checkLicence's page-level pass runs unconditionally, skipping only classes a per-source entry accounts for"
impact: fix
resolved_by:
  commit: "dde0c6a4"
---

checkLicence's unconditional return in the sources branch skips the scalar source.class licence check when both shapes are present, so an external_* page with an empty/junk sources: [] passes ML001. GitHub mirror: #330