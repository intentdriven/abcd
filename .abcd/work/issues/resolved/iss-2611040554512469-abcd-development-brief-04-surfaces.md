---
schema_version: 1
id: "iss-2611040554512469"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/22-site.md"
remedy: "State the live-site question, the refusal under --yes and the flag in the prose."
resolution: "Chapter 22 now has a paragraph on a site already live: the question naming it, the refusal under the pre-given yes, and the switch that answers in advance, named in words because chapter prose states no flag. TestSurfaceChapterProseStatesNoShape and TestSurfaceAppendicesMatchCommandTree pass."
impact: fix
resolved_by:
  commit: "328fe59f1903fc8531d0e61abf6bae51fe2c7ce9"
---

.abcd/development/brief/04-surfaces/22-site.md:96-99 describes the remote stages' confirmation with no case for a site already live, and --replace-existing appears only in the flag appendix at .abcd/development/brief/04-surfaces/22-site.md:397. `site setup --help` says that flag publishes over a site the host already serves without asking, and that --yes without it refuses when such a site exists. commands/site.md:284 and docs/reference/cli/commands.md:5746 carry this; the chapter's prose does not.
