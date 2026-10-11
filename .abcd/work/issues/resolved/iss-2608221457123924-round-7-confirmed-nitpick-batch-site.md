---
schema_version: 1
id: "iss-2608221457123924"
slug: "round-7-confirmed-nitpick-batch-site"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "all-dimensions bug-hunt round 7"
found_at: "internal/core/site/recordjson.go"
resolution: "Nitpick batch: principle-id collision refusal, core.quotePath=false, and the doc/string nitpicks."
impact: fix
resolved_by:
  commit: "de19186"
---

Round-7 confirmed nitpick batch: site principle-id collision silently overwrites a typed record in record.json (loud refusal added); the isolatedGit history walk quoted non-ASCII paths so a record lost its dates (core.quotePath=false); docs Windows planned/dry-run --json/no-color colour and the 00-meta open-questions home clause.