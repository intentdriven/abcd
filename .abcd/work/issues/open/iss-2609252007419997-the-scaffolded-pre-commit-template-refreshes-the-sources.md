---
schema_version: 1
id: "iss-2609252007419997"
slug: "the-scaffolded-pre-commit-template-refreshes-the-sources"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/defaults/pre-commit"
---

The scaffolded pre-commit template refreshes the sources banlist block by default with a binary found on PATH or in ~/.local/bin, which answers by fiat the three questions iss-2609250834251447 holds as a ruling owed to the product thinker (how a scaffolded hook finds a binary, fail open or closed, default or opt-in). Until that ruling, the template should refresh only on repo-local opt-in, git config --local abcd.sourcesBinary naming an absolute path to a regular file, with no PATH search and never an environment variable, and otherwise print one line and proceed.
