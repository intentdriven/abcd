---
schema_version: 1
id: "iss-2611040554024081"
slug: "04-naming-md-207-calls-domain-a"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md"
remedy: "Say the star command must start the prompt or follow whitespace, dropping the word leading-anchored."
resolution: "The naming table's *<DOMAIN> row now says the star command must start the prompt or follow whitespace, and no longer calls it leading-anchored, which matches parseStarCommands and the row's own regex."
impact: fix
resolved_by:
  commit: "279f1ebf6004683a84a717a9ed5a682f2e39285e"
---

04-naming.md:207 calls *<DOMAIN> a leading-anchored prompt prefix. A star command anywhere after whitespace activates: abcd rules --prompt "hello *SCRATCH there" reports SCRATCH hit by *SCRATCH. parseStarCommands (internal/core/rules/rules.go:681) checks only the bytes around the token, and the row's own regex admits whitespace before the star.

## Grounds

- pursued: the row should describe what parseStarCommands accepts; TestNamingRowStatesWhereAStarCommandMayStand would show it wrong if a mid-prompt star stopped activating, a star glued to a word activated, or the row went back to saying leading-anchored
