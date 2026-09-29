---
schema_version: 1
id: "iss-2609282105240689"
slug: "the-bare-abcd-lint-help-commands-lint-md-frontmatter-abcd"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.11.1 release gate crosscheck (autonomous run A)"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/lint.md"
---

The bare `abcd lint` help (commands/lint.md frontmatter, abcd --help, docs/reference/cli/commands.md) says it checks the conventions with every target included, but repolint.DefaultRules excludes the outbound target, so a reader believes the bare run checks outbound text when it does not. Found by the v0.11.1 crosscheck (x-045); same text at v0.11.0.
