---
schema_version: 1
id: "iss-2608270500208736"
slug: "a-roots-entry-pointing-at-a-non-existent"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "github-ledger-dedup-2026-08-27"
found_at: "internal/core/record-lint (markdownFiles)"
resolution: "a configured lint root that does not exist fails loudly instead of silently disarming every per-file rule (#360)"
impact: fix
---

a roots entry pointing at a non-existent directory makes docs-lint / record-lint pass with zero findings, silently disarming every per-file blocker rule for that tree; the shipped scaffold's roots default triggers it in any adopter whose docs live elsewhere. GitHub mirror: #360