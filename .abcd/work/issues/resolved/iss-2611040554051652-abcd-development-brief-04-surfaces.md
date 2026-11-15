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
resolution: "The board chapter's presence-line paragraph now lists the autonomous merging element (key merging) between the branch and the record's counts, shown under facilitator.mode auto, and points at 26-mode.md; a test in internal/core/statusline holds the paragraph to the row's constants."
impact: internal
resolved_by:
  commit: "0f08a8b03d9e92a0cb28cbaa9db35c14dd00f809"
---

.abcd/development/brief/04-surfaces/08-abcd.md:185-188 lists the presence line and JSON statusline as the badge, the repository, the branch and the record's counts. Under facilitator.mode auto the binary adds an element between the branch and the counts: the plain row reads `abcd-managed · <repo> · main · autonomous merging · itd N · iss N`, with key `merging` in statusline.elements. .abcd/development/brief/04-surfaces/26-mode.md:16-26 documents it; this chapter does not.

## Grounds

- pursued: the presence-line paragraph in 08-abcd.md names the merging element, its key and the mode chapter, matching the row the binary draws; TestBoardChapterNamesTheMergingElement failing, or the row under facilitator.mode auto carrying an element the paragraph does not list, would show it wrong
