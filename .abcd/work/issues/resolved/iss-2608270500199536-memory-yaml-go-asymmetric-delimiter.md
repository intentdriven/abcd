---
schema_version: 1
id: "iss-2608270500199536"
slug: "memory-yaml-go-asymmetric-delimiter"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "github-ledger-dedup-2026-08-27"
found_at: "internal/core/memory/yaml.go"
resolution: "memory's three close sites route through one trimmed-delimiter predicate matching the opener's tolerance"
impact: fix
resolved_by:
  commit: "cd0c6d9f"
---

memory yaml.go asymmetric delimiter parse: opening --- is TrimSpace-tolerant but closing is byte-exact, so a trailing-space closing delimiter drops source: and bypasses ML001. GitHub mirror: #304