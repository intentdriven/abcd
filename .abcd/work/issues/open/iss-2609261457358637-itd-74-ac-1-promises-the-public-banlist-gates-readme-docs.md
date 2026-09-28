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
---

itd-74 ac-1 promises the public banlist gates README, docs/ and the shipped artefact; the banned_tokens family walks only the docs-lint roots (docs, README.md) and the payload render reuses those roots, so a banned public token in commands/, agents/ or skills/ ships ungated
