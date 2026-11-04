---
schema_version: 1
id: "iss-2611040554051652"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/08-abcd.md"
remedy: "Name the merging element in the chapter's list of the row, pointing at the mode chapter."
---

.abcd/development/brief/04-surfaces/08-abcd.md:185-188 lists the presence line and JSON statusline as the badge, the repository, the branch and the record's counts. Under facilitator.mode auto the binary adds an element between the branch and the counts: the plain row reads `abcd-managed · <repo> · main · autonomous merging · itd N · iss N`, with key `merging` in statusline.elements. .abcd/development/brief/04-surfaces/26-mode.md:16-26 documents it; this chapter does not.
