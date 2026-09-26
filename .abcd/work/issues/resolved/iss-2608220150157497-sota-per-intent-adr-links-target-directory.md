---
schema_version: 1
id: "iss-2608220150157497"
slug: "sota-per-intent-adr-links-target-directory"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "bughunt-b-round-7"
found_at: ".abcd/development/principles/sota-per-intent.md"
resolution: "adr-22/adr-26 links repointed at the ADR files"
impact: internal
resolved_by:
  commit: "554f97f"
---

principles/sota-per-intent.md links `[adr-22](../decisions)` and `[adr-26](../decisions)` to the decisions directory instead of the ADR files; links_resolve passes because the directory exists; the only two ID-labelled links in the record whose target does not match the id