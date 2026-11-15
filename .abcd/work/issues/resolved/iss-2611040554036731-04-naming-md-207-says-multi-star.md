---
schema_version: 1
id: "iss-2611040554036731"
slug: "04-naming-md-207-says-multi-star"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md"
remedy: "Say several star commands activate every named domain, injected in name-sorted order."
resolution: "The naming table's `*<DOMAIN>` row now says several star commands activate every named domain, injected in name-sorted order, in place of left-to-right order; a test runs Match on a reversed prompt and holds the row to it."
impact: internal
resolved_by:
  commit: "55221446d01996e8624881d0b8726046e8a3f0b5"
---

04-naming.md:207 says multi-star activates multiple domains in left-to-right order. parseStarCommands returns a set and RuleSet.Match walks domain names sorted (internal/core/rules/rules.go:618-646,681): abcd rules --prompt "*SCRATCH *COMMITTING" lists COMMITTING before SCRATCH.

## Grounds

- pursued: the row states the order RuleSet.Match injects star-named domains in; a prompt of "*SCRATCH *COMMITTING" injecting SCRATCH before COMMITTING, or TestNamingRowStatesTheStarOrderMatchUses failing, would show it wrong
