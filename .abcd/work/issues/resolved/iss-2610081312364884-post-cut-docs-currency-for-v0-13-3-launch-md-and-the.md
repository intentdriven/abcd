---
schema_version: 1
id: "iss-2610081312364884"
slug: "post-cut-docs-currency-for-v0-13-3-launch-md-and-the"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/launch.md"
remedy: "Document the open-records refusal in launch.md and the --allow-dirty help, condition the hooksPath step on unarmed, correct the record-lint path and the --discard help order"
resolution: "launch.md and the --allow-dirty help name the findings gate's open/ records refusal (and the help the cut's uncommitted-records refusal); prepare-this-repo conditions the hooksPath step on unarmed and names .abcd/record-lint.json; the --discard help gives the code's order"
impact: fix
resolved_by:
  commit: "afa6e7b169abf9f840a8e5cf7b6bd8af07bbbf6a"
---

Post-cut docs currency for v0.13.3: launch.md and the --allow-dirty help omit the open-records refusal, prepare-this-repo shadows a foreign hooks path and names a bare record-lint.json, and implement step --discard help misorders its steps

## Grounds

- pursued: the five deferred v0.13.3 docs-currency findings now match the code; a docs review that still finds one of them stale would show this wrong
