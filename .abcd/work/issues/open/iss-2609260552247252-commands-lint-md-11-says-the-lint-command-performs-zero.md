---
schema_version: 1
id: "iss-2609260552247252"
slug: "commands-lint-md-11-says-the-lint-command-performs-zero"
severity: "minor"
category: "documentation"
source: "drift-detection"
found_during: "v0.11.0 release gate: brief-surface cross-check (autonomous run A, abcd-a2)"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/lint.md"
---

commands/lint.md:11 says the lint command performs zero writes and names only the temp-dir site render, but `abcd lint site` renders the site into `--out`, default ./site under the working directory, when that directory holds no index.html: a bare run in a repository leaves a new directory. Found by the v0.11.0 brief-surface cross-check (x-045).
