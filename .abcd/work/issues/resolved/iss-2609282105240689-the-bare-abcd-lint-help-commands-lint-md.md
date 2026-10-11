---
schema_version: 1
id: "iss-2609282105240689"
slug: "the-bare-abcd-lint-help-commands-lint-md"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.11.1 release gate crosscheck (autonomous run A)"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/lint.md"
resolution: "The bare lint sentence reads 'every target but outbound' in the manifest, commands/lint.md, docs/reference/cli/commands.md and surface.json; proved by TestBareLintSentenceNamesWhatTheBareRunLeavesOut, watched failing on a scratch archive first."
impact: fix
resolved_by:
  commit: "e3160ebf78b6bd3c9b7a075916a033259c331b57"
---

The bare `abcd lint` help (commands/lint.md frontmatter, abcd --help, docs/reference/cli/commands.md) says it checks the conventions with every target included, but repolint.DefaultRules excludes the outbound target, so a reader believes the bare run checks outbound text when it does not. Found by the v0.11.1 crosscheck (x-045); same text at v0.11.0.

## Grounds

- pursued: a reader of the bare lint help learns the outbound target is not part of the bare run, matching repolint.DefaultRules and the lint chapter; help text still claiming every target, or a DefaultRules that gains an outbound rule without the sentence changing, would show it wrong
