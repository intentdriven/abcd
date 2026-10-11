---
schema_version: 1
id: "iss-2608270926036966"
slug: "isdelimiter-trims-a-bom-on-every-line-so"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "issue-sweep-review-2026-08-27"
found_at: "internal/core/frontmatter/frontmatter.go"
resolution: "IsDelimiter no longer trims a BOM; the four line-0 read and write sites trim it themselves, matching the file-level BOM convention the repo already states"
impact: fix
resolved_by:
  commit: "c6fe046a"
---

IsDelimiter trims a BOM on every line, so a mid-file zero-width no-break space before --- closes frontmatter.Fields early while intent's writer walks past it — abcd intent plan then inserts kind and spec_id into the record body, reports success, and reload shows an empty spec link; changelog's bodyStart carries the same stale rule under a comment claiming exact agreement. BOM tolerance belongs to the first line only