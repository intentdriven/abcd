---
schema_version: 1
id: "iss-2610230851422913"
slug: "the-brief-s-rules-section-04-surfaces"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.2 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md"
remedy: "Describe `abcd rules --prompt` in the brief's rules section, as commands/ahoy.md does."
resolution: "The brief's verb table now names abcd rules --prompt, and its rules section states --prompt's exit-2 refusals (a DOMAIN beside it; --fill without it or outside 0 to 100) and its JSON form, as commands/ahoy.md does; TestTheBriefsRulesSectionDescribesThePromptForm holds it."
impact: internal
resolved_by:
  commit: "15c3ffbbd6c6da6c1d201f480c39a5f714c7b1c7"
---

The brief's rules section (04-surfaces/README.md, 'The rules verb') describes only the bare and `<DOMAIN>` forms; the registered `abcd rules --prompt` flag appears nowhere in the brief.

## Grounds

- pursued: a reader of the brief alone meets every form and refusal the rules verb registers; shown wrong if cli.go's rules command or its JSON envelope (rulesPromptView) says otherwise than the brief's bullet, or if the detector passes against the base-commit README (it was watched failing there).
