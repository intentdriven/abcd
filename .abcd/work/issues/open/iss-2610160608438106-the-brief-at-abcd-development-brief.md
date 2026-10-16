---
schema_version: 1
id: "iss-2610160608438106"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/08-skills.md"
remedy: "Replace 'audit' with 'lint' among the verbs named at 08-skills.md:84-86."
---

The brief at .abcd/development/brief/05-internals/08-skills.md:84-86 says the ahoy page's install section prepares a repository by running the binary's audit, install and identity verbs. abcd has no audit verb (`abcd audit` exits 2 as unknown); the install section's Audit phase runs `abcd lint --json` (commands/ahoy.md:202), beside ahoy install and identity init and render.
