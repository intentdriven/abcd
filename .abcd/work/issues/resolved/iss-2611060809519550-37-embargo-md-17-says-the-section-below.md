---
schema_version: 1
id: "iss-2611060809519550"
slug: "37-embargo-md-17-says-the-section-below"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/37-embargo.md"
remedy: "Add putting back redacted record details to the chapter's list of the person's acts."
resolution: "The embargo chapter's 'What stays the person's' section now names putting back any record details redacted to land a fix, once the advisory is published, as commands/embargo.md lists it; a test holds the section to each act on the page's list."
impact: internal
resolved_by:
  commit: "e6570d64a10fe87202599d93ba27739ddc2a8fd7"
---

`37-embargo.md:17` says the section below lists the acts the command page hands the person. The section at lines 132-134 omits one that `commands/embargo.md:309` lists: putting back any record details they redacted to land a fix, once the advisory is published. Restoring each record into the public ledger, which the section does name, is a separate item on the page.

## Grounds

- pursued: the chapter's section names every act the command page's 'What stays the person's act' list hands the person; TestBriefPersonsActsMatchCommandPage failing, or a page act with no counterpart in the chapter, would show it wrong
