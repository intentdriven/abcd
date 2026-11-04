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
---

.abcd/development/brief/04-surfaces/20-banlist.md:222-223 says `abcd ahoy` lays down the guard artefacts. Bare `abcd ahoy` mutates nothing (.abcd/development/brief/04-surfaces/01-ahoy.md:41); `abcd ahoy install` wrote .githooks/pre-commit, .githooks/pre-merge-commit and .gitattributes in a test repository.
