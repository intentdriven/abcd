---
schema_version: 1
id: "iss-2608261041029596"
slug: "brief-readme-nine-vs-ten-internals"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-a/round-7"
found_at: ".abcd/development/brief/README.md"
resolution: "brief/README: nine to ten internals chapters and added site to the tree roster."
impact: internal
resolved_by:
  commit: "2811321ba4b6adcd06bf05d14101c4bd76d26cdc"
---

brief/README.md still says the internals README indexes nine chapters and its tree roster omits site, but there are ten. brief/README.md line 52 reads 'indexes the nine chapters' and the tree line 40 lists nine internals topics without site, while 05-internals/README.md indexes ten rows (10-site.md exists on disk). iss-356 designated the tree line and the internals index as the single rosters to keep in sync; a tenth chapter was added and the index updated but the count and tree line were not. No lint rule gates this roster. Fix: nine to ten, add site to the tree line.