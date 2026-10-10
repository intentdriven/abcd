---
schema_version: 1
id: "iss-2608221457125132"
slug: "commands-site-md-and-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "all-dimensions bug-hunt round 7"
found_at: "commands/site.md"
resolution: "site.md and 22-site.md read/write sets corrected to include record.js, install.sh.tmpl and the credit sources."
impact: fix
resolved_by:
  commit: "be8b924"
---

commands/site.md and brief 04-surfaces/22-site.md claim the site build reads exactly a named set and nowhere else, but the set omits site-src/record.js, site-src/install.sh.tmpl, CONTRIBUTING.md (a build-aborting input) and ACKNOWLEDGEMENTS.md, and the output list omits install.sh and record.js.