---
schema_version: 1
id: "iss-2608270500210055"
slug: "readme-and-docs-state-no-requirements"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "github-ledger-dedup-2026-08-27"
found_at: "README.md"
resolution: "README gains a Requirements section above Install, each bullet scoped to its route"
impact: fix
resolved_by:
  commit: "4c6a8c65"
---

README and docs state no requirements, but git is an undocumented hard runtime dependency; there is no Requirements/Prerequisites heading. GitHub mirror: #496