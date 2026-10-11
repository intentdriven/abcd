---
schema_version: 1
id: "iss-2608221456469558"
slug: "a-site-page-that-fails-to-parse-is"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "all-dimensions bug-hunt round 7"
found_at: "internal/core/site/check.go"
resolution: "An unparsed page now fails every page-walking gate rather than dropping out of their lists."
impact: fix
resolved_by:
  commit: "de19186"
---

A site page that fails to parse is dropped from c.pages, so the page-walking gates (hero, banned-tokens, snippets, mobile, figure-labels) never examine it yet print ok, masking a genuine finding on that page behind the provenance parse fault.