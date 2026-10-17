---
schema_version: 1
id: "iss-2610142032595919"
slug: "a-gitignore-the-person-already-keeps-in"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "PR #991 review pair"
origin: researcher-authored
production_mode: hand-written
found_at: "the self-ignoring abcd home (internal/core/homeignore, PR #991)"
remedy: "Leave it: abcd never overwrites or second-guesses a .gitignore the person wrote in its home (the self-ignoring home's scope, PR #991); their file states their intent."
wontfix_reason: "abcd never overwrites a .gitignore the person keeps in its home; that is the self-ignoring home's stated scope, and their file is their statement of what to ignore. Rewriting it, or refusing private verdicts until they change it, would override a file the person wrote on purpose."
---

A .gitignore the person already keeps in abcd's home is kept whatever it holds, and the marker is written beside it, so a file that does not ignore ideas/ (say one holding only transcripts/) leaves private ideate verdicts stageable by a dotfiles repository in the home, which the private store admits because abcd's home is taken to ignore itself there. Raised by the Sonnet reviewer of PR #991.

## Grounds

- declined: abcd never overwrites a .gitignore the person keeps in its home; that is the self-ignoring home's stated scope, and their file is their statement of what to ignore. Rewriting it, or refusing private verdicts until they change it, would override a file the person wrote on purpose.
