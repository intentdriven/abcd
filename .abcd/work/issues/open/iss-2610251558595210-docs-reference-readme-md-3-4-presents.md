---
schema_version: 1
id: "iss-2610251558595210"
slug: "docs-reference-readme-md-3-4-presents"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D3"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/reference/README.md"
remedy: "Decide whether docs/reference gains a configuration reference (keys by file and layer, environment variables, the .abcd JSON file formats), generated from the code's key tables where they exist so it cannot drift; if not, keep the narrowed description."
---

docs/reference/README.md:3-4 presents the reference folder as describing configuration keys, environment variables and file schemas, and no page ever has: the folder holds the generated command reference, the writing style guide and the terminology crosswalk (the wording dates from 157e3ac1f). The proposed fix narrows the line, but the gap stays: keys a person sets by hand, such as load.preflights and reservation.github_username in ~/.abcd.noindex/config.json or rules.force_refresh_every_n in .abcd/config.json (internal/core/rules/inject.go:339-343), are documented only inside command help.
