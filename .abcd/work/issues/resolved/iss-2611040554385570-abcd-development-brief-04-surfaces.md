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
resolution: "The lint chapter of the brief now says a finding's `file` is omitted where a `site-gates` finding names no source span, as the command page does; a test holds both pages to it while the field is omitempty."
impact: internal
resolved_by:
  commit: "42e73faa919d1be96f1eb9c5622442a7c0f544bf"
---

.abcd/development/brief/04-surfaces/16-lint.md:130-132 says each finding carries a `file`. Finding.File is omitempty (internal/core/repolint/repolint.go:70) and a site-gates finding with no source span has none (internal/core/repolint/rule_site.go:83). commands/lint.md:39 states this exception; the chapter does not.

## Grounds

- pursued: the chapter and commands/lint.md state the same exception for `file`, and TestLintPagesStateTheFileException fails if either page drops it while Finding.File is omitempty; a site-gates finding with no source span that still serialised a `file` key would show it wrong
