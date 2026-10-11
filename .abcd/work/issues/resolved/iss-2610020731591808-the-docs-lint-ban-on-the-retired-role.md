---
schema_version: 1
id: "iss-2610020731591808"
slug: "the-docs-lint-ban-on-the-retired-role"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-2609212137129937"
origin: researcher-authored
production_mode: hand-written
remedy: "Add .abcd/development/brief, .abcd/development/principles and .abcd/development/personas.json to the token's extra_roots in .abcd/docs-lint.json (the seam takes a directory or a file, as .abcd/rules.json shows, and refuses a missing or escaping root per TestTokenExtraRootsRefuseAMissingOrEscapingRoot), extend TestRepoRoleWordIsRefusedOnEveryLintRoot's wanted roots and fixture pages to match, and mark any genuine escape in the glossary with the allow comment; the intents directory stays out, since a record's historical text keeps its words. Grounds: the intent's In Scope 'the docs-lint banned-token list gains the word for every lint root including the command pages and the rules' and its press release 'the word cannot come back'."
resolution: "The roles/retired-role-word docs-lint ban now reaches .abcd/development/brief, .abcd/development/principles and .abcd/development/personas.json through its extra_roots; the intents stay out; the repository test pins the new roots, the glossary escape and the intents' exclusion."
impact: internal
resolved_by:
  commit: "a8ebe97866429f76976e4f6121aa955a059557fe"
---

The docs-lint ban on the retired role word (itd-2609212137129937, ac-2) reaches the docs roots, the command pages, .abcd/rules.json and the bundled rules source only: .abcd/docs-lint.json's 'roles/retired-role-word' entry lists extra_roots commands, .abcd/rules.json and internal/core/rules/defaults, and the top-level roots are docs and README.md. The brief (.abcd/development/brief, 22 occurrences before the sweep, the largest share), the principles (.abcd/development/principles, 5) and the personas registry (.abcd/development/personas.json) were swept by ac-1 but are in no root the token reads, and the ac-5 test walks only the rendered help, commands/*.md and agents/*.md, so the word can return to the brief or a principle with nothing refusing it, against the press release's 'the word cannot come back' and the mechanism claim that agents say what they read (the OPINIONS domain points agents at the principles).

## Grounds

- pursued: the retired role word written into a brief page, a principle or the personas registry is refused by abcd lint docs as a blocker, while an intent body is not; a brief or principle page carrying the word unescaped that lints clean would show it wrong.
