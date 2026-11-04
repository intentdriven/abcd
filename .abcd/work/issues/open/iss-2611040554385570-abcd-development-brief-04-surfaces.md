---
schema_version: 1
id: "iss-2611040554385570"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/16-lint.md"
remedy: "Carry the command page's exception into the chapter."
duplicates: [iss-2610252214323001]
---

.abcd/development/brief/04-surfaces/16-lint.md:130-132 says each finding carries a `file`. Finding.File is omitempty (internal/core/repolint/repolint.go:70) and a site-gates finding with no source span has none (internal/core/repolint/rule_site.go:83). commands/lint.md:39 states this exception; the chapter does not.
