---
schema_version: 1
id: "iss-2611040554189675"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/20-banlist.md"
remedy: "Write `abcd ahoy install` in that sentence."
resolution: "The banlist chapter now names the ahoy installer, linked to the ahoy chapter, as what lays down the guard artefacts, and says bare `abcd ahoy` writes none of them. The literal `abcd ahoy install` spelling fails the surface-shape gate (TestSurfaceChapterProseStatesNoShape), so the chapter uses the house spelling instead."
impact: internal
resolved_by:
  commit: "06936ca935f15448d03f85d5d611273e6d95cf58"
---

.abcd/development/brief/04-surfaces/20-banlist.md:222-223 says `abcd ahoy` lays down the guard artefacts. Bare `abcd ahoy` mutates nothing (.abcd/development/brief/04-surfaces/01-ahoy.md:41); `abcd ahoy install` wrote .githooks/pre-commit, .githooks/pre-merge-commit and .gitattributes in a test repository.

## Grounds

- pursued: the chapter no longer credits bare ahoy with the scaffold writes, and TestBriefBanlistChapterNamesAhoyInstallerAsTheScaffolder holds it there; if the test passed on the base text, or if the surface-shape gate flagged the new sentence, that would show it wrong
