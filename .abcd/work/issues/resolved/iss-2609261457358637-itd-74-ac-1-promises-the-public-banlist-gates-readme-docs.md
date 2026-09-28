---
schema_version: 1
id: "iss-2609261457358637"
slug: "itd-74-ac-1-promises-the-public-banlist-gates-readme-docs"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-74"
origin: researcher-authored
production_mode: hand-written
resolution: "name_roots in .abcd/docs-lint.json now list commands, agents and hooks, the plugin surfaces the shipped artefact carries, so the public banlist (the names/ family alone) gates them as it gates README and docs/; this repository has no skills/ tree, and TestRepoNameRootsCoverThePublicSurface pins every plugin surface the checkout carries."
impact: fix
resolved_by:
  commit: "868b9c375"
---

itd-74 ac-1 promises the public banlist gates README, docs/ and the shipped artefact; the banned_tokens family walks only the docs-lint roots (docs, README.md) and the payload render reuses those roots, so a banned public token in commands/, agents/ or skills/ ships ungated

## Grounds

- pursued: a names/ token written into commands/, agents/ or hooks/ is now a blocker from abcd lint docs; shown wrong if such a token passes the docs lint
