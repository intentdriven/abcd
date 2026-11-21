---
schema_version: 1
id: "iss-2610230851529426"
slug: "03-configuration-md-s-user-scope"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.2 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Add the four folders to the inventory, with the chapter that owns each."
resolution: "03-configuration.md's user-scope inventory now follows away/, after-update/, sessions/ and dashboard/ with a link to the surface chapter that owns each (26-mode, 21-update, 08-abcd, 36-dashboard); TestUserScopeInventoryNamesTheChapterOfEachFolder holds it."
impact: internal
resolved_by:
  commit: "4d729d4313c6f951252404678f3d7c3ebd577882"
---

03-configuration.md's user-scope inventory of ~/.abcd.noindex/ omits four folders the binary writes: away/, sessions/, dashboard/ and after-update/.

## Grounds

- pursued: each of the four folders in the inventory is followed by a link to its owning chapter, and that chapter has the heading the link points to; a renamed heading, a dropped link or a folder taken out of the inventory fails the test.
