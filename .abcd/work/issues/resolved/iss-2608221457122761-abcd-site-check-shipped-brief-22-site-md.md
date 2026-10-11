---
schema_version: 1
id: "iss-2608221457122761"
slug: "abcd-site-check-shipped-brief-22-site-md"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "all-dimensions bug-hunt round 7"
found_at: ".abcd/development/brief/05-internals/10-site.md"
resolution: "site-check called shipped in the two brief surfaces; 22-site.md pinned in the release-gate manifest."
impact: internal
resolved_by:
  commit: "be8b924"
---

abcd site check shipped (brief 22-site.md marks it shipped) but 04-surfaces/README.md and 05-internals/10-site.md still call the check gates design targets, and the release-gate manifest never pinned 22-site.md so Direction A never reads the site chapter.