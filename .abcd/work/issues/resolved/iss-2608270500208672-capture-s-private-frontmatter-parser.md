---
schema_version: 1
id: "iss-2608270500208672"
slug: "capture-s-private-frontmatter-parser"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "github-ledger-dedup-2026-08-27"
found_at: "internal/core/capture (frontmatter parser)"
resolution: "capture reads ledger delimiters via the canonical frontmatter.IsDelimiter; four divergent compares removed"
impact: fix
resolved_by:
  commit: "2bc7569a"
---

capture's private frontmatter parser matches ledger delimiters byte-exact while record-lint and the graveyard use the canonical whitespace-tolerant scanner, so a '--- ' (trailing-space) delimiter yields a lint-green issue file every capture verb refuses and abcd iss-N reports 'not found'. GitHub mirror: #338