---
schema_version: 1
id: "iss-2608221254560250"
slug: "docs-reference-terminology-md-memory-row-describes-retrieval"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "context-window SOTA investigation"
found_at: "docs/reference/terminology.md"
resolution: "the terminology Memory row describes the shipped retrieval: memory ask's word-overlap ranking over classes, domain and summary, top five with citations"
impact: fix
resolved_by:
  commit: "07af2eb092552739a5360c39e5659457e7b3c734"
---

docs/reference/terminology.md Memory row describes retrieval as 'recall-matched and budget-bracketed', but no retrieval engine exists: the recall: frontmatter field is parsed and round-tripped yet never consumed, and the budget brackets live only in itd-39 (draft). The row overstates shipped behaviour.

## Grounds

- pursued: the row now claims only shipped behaviour; a retrieval claim with no code behind it would show it wrong
