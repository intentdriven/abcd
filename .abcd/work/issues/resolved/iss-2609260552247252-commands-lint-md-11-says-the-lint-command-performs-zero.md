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
resolution: "commands/lint.md and the brief's lint chapter name lint site's render into --out as the one write, beside the zero-write bare run and other targets."
impact: fix
resolved_by:
  commit: "1ab64394"
---

commands/lint.md:11 says the lint command performs zero writes and names only the temp-dir site render, but `abcd lint site` renders the site into `--out`, default ./site under the working directory, when that directory holds no index.html: a bare run in a repository leaves a new directory. Found by the v0.11.0 brief-surface cross-check (x-045).

## Grounds

- pursued: the lint page claims zero writes only for the targets that write nothing; a bare lint or a lint docs, identity or outbound run that leaves a file in the repository would show it wrong
