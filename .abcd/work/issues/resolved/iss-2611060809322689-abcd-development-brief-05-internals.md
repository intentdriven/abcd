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
resolution: "The configuration chapter no longer states a token figure for GRILL (the block is now 7,601 bytes, about 1,900 tokens, past even the issue's 1,600); it points at `abcd rules --json` rules_bytes instead, and a test holds any figure the GRILL paragraph states to the rendered block within a quarter."
impact: internal
resolved_by:
  commit: "404e3565daa9803be463dfeaac554ecadef5231c"
---

`.abcd/development/brief/05-internals/03-configuration.md:1587` says `GRILL` lands at about 1,100 tokens. On the built binary, `*GRILL` through `abcd hook prompt-router --json` injects 6,533 bytes of text, about 1,600 tokens at four bytes a token.

## Grounds

- pursued: the GRILL paragraph states no size that can drift from the bundled block; TestGrillCostInTheChapterMatchesTheBlock failing, or a figure in that paragraph more than a quarter off bytes/4 of the rendered GRILL block, would show it wrong
