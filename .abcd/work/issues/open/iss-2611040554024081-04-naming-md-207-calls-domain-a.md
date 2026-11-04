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
---

04-naming.md:207 calls *<DOMAIN> a leading-anchored prompt prefix. A star command anywhere after whitespace activates: abcd rules --prompt "hello *SCRATCH there" reports SCRATCH hit by *SCRATCH. parseStarCommands (internal/core/rules/rules.go:681) checks only the bytes around the token, and the row's own regex admits whitespace before the star.
