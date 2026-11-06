---
schema_version: 1
id: "iss-2611060809322689"
slug: "abcd-development-brief-05-internals"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "State about 1,600 tokens, or drop the figure so it cannot go stale."
refines: [iss-2611060809312249]
---

`.abcd/development/brief/05-internals/03-configuration.md:1587` says `GRILL` lands at about 1,100 tokens. On the built binary, `*GRILL` through `abcd hook prompt-router --json` injects 6,533 bytes of text, about 1,600 tokens at four bytes a token.
