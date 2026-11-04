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
---

04-naming.md:207 says multi-star activates multiple domains in left-to-right order. parseStarCommands returns a set and RuleSet.Match walks domain names sorted (internal/core/rules/rules.go:618-646,681): abcd rules --prompt "*SCRATCH *COMMITTING" lists COMMITTING before SCRATCH.
