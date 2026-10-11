---
schema_version: 1
id: "iss-2608271711530571"
slug: "docs-readme-advertises-an-empty"
severity: "nitpick"
category: "observation"
source: "agent-finding"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: "docs/tutorials"
resolution: "docs/README tutorials row marked not-yet-populated, naming iss-216 as the real close"
impact: internal
resolved_by:
  commit: "a032377d"
---

docs/README.md advertises tutorials/ as a populated Diataxis home while the directory holds only a 6-line README and zero content pages. Until the iss-216 getting-started page lands (the real close), the table row overstates the tree: mark the tutorials row as not yet populated and name iss-216 so a reader is not routed to an empty home.