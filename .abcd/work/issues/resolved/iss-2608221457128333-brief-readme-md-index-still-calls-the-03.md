---
schema_version: 1
id: "iss-2608221457128333"
slug: "brief-readme-md-index-still-calls-the-03"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "all-dimensions bug-hunt round 7"
found_at: ".abcd/development/brief/README.md"
resolution: "brief/README.md index calls the evidence chapter live, not placeholders."
impact: internal
resolved_by:
  commit: "be8b924"
---

brief/README.md index still calls the 03-evidence chapter placeholders populated by lifeboat extraction, three commits after that chapter was made live (commit closing iss-2608220750029989).