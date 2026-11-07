---
schema_version: 1
id: "iss-2611060809502482"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/37-embargo.md"
remedy: "Say in the chapter, the command page and the help that the location is refused only while something is under embargo."
---

`.abcd/development/brief/04-surfaces/37-embargo.md:88-91` says the check refuses the store's location on this machine, and `abcd embargo check --help` and `commands/embargo.md:144` say the same. `buildDenySet` returns at `internal/core/embargo/check.go:735` when nothing is under embargo, before it adds the store's path, so a text naming the path passes while the stores hold no unpublished advisory.
