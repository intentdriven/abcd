---
schema_version: 1
id: "iss-2611060809220159"
slug: "04-surfaces-09-reflect-md-67-says-the"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/09-reflect.md"
remedy: "Name the changelog's target-move notes as the seed's second source in the chapter's list."
---

`04-surfaces/09-reflect.md:67` says the seed lists the planned intents whose `target_release` names the release and which did not ship. `unshippedTargets` (`internal/core/reflect/seed.go:200`) also adds any still-planned intent named in a target-move note of the release's changelog section (`changelog.IsTargetMoveNote`), whatever its `target_release` says, and those intents enter the warning and the `unshipped_targets` refusal. The chapter does not name this second source, which is the answer the ruling on iss-2610262032250865 chose.
