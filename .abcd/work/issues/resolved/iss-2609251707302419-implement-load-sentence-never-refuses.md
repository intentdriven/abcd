---
schema_version: 1
id: "iss-2609251707302419"
slug: "implement-load-sentence-never-refuses"
severity: "minor"
category: "documentation"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/surface/sentences.go"
resolution: "The implement load sentence names the unknown --site refusal and keeps that a loaded machine is never refused."
impact: fix
resolved_by:
  commit: "0cd55018"
---

The abcd implement load sentence says it never refuses, but the verb refuses an unknown --site with exit 2 before checking anything (implement_load.go)

## Grounds

- pursued: the sentence matches implement_load.go's exit 2 on an unknown --site; shown wrong by that call exiting 0
