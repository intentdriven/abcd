---
schema_version: 1
id: "iss-2608270500194738"
slug: "backfilllegacy-gates-frontmatter"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "github-ledger-dedup-2026-08-27"
found_at: "internal/core/memory/writer.go"
resolution: "backfillLegacy asks the parser for frontmatter presence; preamble-led pages are skipped, not rebuilt with fabricated provenance"
impact: fix
resolved_by:
  commit: "e7e0437d"
---

backfillLegacy gates frontmatter presence on HasPrefix(text, '---') while the canonical parsers tolerate a leading HTML comment, demoting real source: provenance. GitHub mirror: #288