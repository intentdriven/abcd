---
schema_version: 1
id: "iss-2611040555295193"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/19-identity.md"
remedy: "Say present on disk in the chapter and in the code comment."
---

.abcd/development/brief/04-surfaces/19-identity.md:121-122 says the identity-positioning rule is gated on a committed registry. The gate tests presence on disk (internal/core/repolint/rule_positioning.go:42): with `.abcd/positioning.json` untracked, bare `abcd lint` still ran the rule and reported under it. .abcd/development/brief/04-surfaces/16-lint.md:162 states the presence gate correctly, and the comment at internal/core/repolint/rule_positioning.go:40 repeats the chapter's wording.
