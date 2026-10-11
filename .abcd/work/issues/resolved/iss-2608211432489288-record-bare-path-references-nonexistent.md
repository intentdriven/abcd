---
schema_version: 1
id: "iss-2608211432489288"
slug: "record-bare-path-references-nonexistent"
severity: "nitpick"
category: "drift"
source: "user-observation"
found_during: "bughunt-b/round-5"
found_at: ".abcd/development/decisions/adrs/README.md"
resolution: "Fix two backticked bare paths that named nonexistent directories"
impact: internal
resolved_by:
  commit: "1c3ae21"
---

two backticked bare relative paths point at directories that do not exist (decisions/adrs/README roadmap/rfcs and intents/README decisions/adrs), invisible to links_resolve