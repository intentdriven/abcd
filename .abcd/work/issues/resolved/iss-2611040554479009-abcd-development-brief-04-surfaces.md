---
schema_version: 1
id: "iss-2611040554479009"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/22-site.md"
remedy: "Add those pages to the page set and written list, and mark which are opt-in or have no switch."
refines: [iss-2610151422085155]
resolution: "The site brief's page set now names every page the explorer writes, with its route: the seven switched pages, plus foundations, development, the decision log (opt-in through record.decisions), contributors and references, which have no switch. The written list names each route. A test holds both sections to the explorer's route constants."
impact: internal
resolved_by:
  commit: "8368a58da7b359ebe031e412b0f6c93c1f2f154c"
---

.abcd/development/brief/04-surfaces/22-site.md:160-166 says every repository gets the same seven pages and the pages block cannot add one, and .abcd/development/brief/04-surfaces/22-site.md:249-254 ends its written list with nothing else, nowhere else. A build also writes contributors/, references/, record/decision-log/, record/development/ and record/foundations/. PageNames (internal/core/site/pages.go:37) has seven names, with no switch for contributors or references, and the decision pages exist only under record.decisions. Lines 246-248 mention contributors and references only as inputs.

## Grounds

- pursued: the brief's page set and written list name every route explorer.Pages writes; a route constant missing from either section, or a condition in the brief that disagrees with hasFoundations/hasDevelopment/hasDecisionLog/hasContributors/hasReferences, would show it wrong
