---
schema_version: 1
id: "iss-2608261533174500"
slug: "terminology-rag-row-claims-the-sources"
severity: "minor"
category: "documentation"
source: "agent-observation"
found_during: "bughunt-a round 9"
found_at: "docs/reference/terminology.md"
resolution: "The RAG row states the corpus as a user-tier script MVP with core absorption cited, matching the script-first-mvp principle"
impact: fix
resolved_by:
  commit: "6d9a7b24"
---

docs/reference/terminology.md's RAG row claims 'the sources corpus ships the script-first version' — a false shipped-ness claim on a page whose sibling rows use ship as a precise delivery marker (No MCP server ships today; abcd ships zero skills). Nothing corpus-related is in the repo or any released artefact; the consult and ingest commands refuse when the corpus is absent and point at a README no user has; and the committed script-first-mvp principle states a script MVP never ships as product behaviour, naming this corpus tooling as the live instance. Same defect class as the open Memory-row overstatement on the same page. Acceptance: the row states the corpus as a user-tier script MVP with core absorption tracked, not as shipped behaviour.