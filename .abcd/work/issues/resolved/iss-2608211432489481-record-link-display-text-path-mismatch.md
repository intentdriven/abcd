---
schema_version: 1
id: "iss-2608211432489481"
slug: "record-link-display-text-path-mismatch"
severity: "nitpick"
category: "drift"
source: "user-observation"
found_during: "bughunt-b/round-5"
found_at: ".abcd/development/intents/README.md"
resolution: "Align seven record link display texts with their resolving hrefs"
impact: internal
resolved_by:
  commit: "1c3ae21"
---

seven markdown links across the record carry a backticked display path that does not resolve from the containing file while the href does, evading links_resolve (recurrence of iss-391)