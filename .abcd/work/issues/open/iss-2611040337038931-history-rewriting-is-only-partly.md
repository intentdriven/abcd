---
schema_version: 1
id: "iss-2611040337038931"
slug: "history-rewriting-is-only-partly"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "the v0.17.0 cut and the overnight lanes, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Add those commands to the guard's registry with a test each, and have ahoy's remote audit propose a ruleset refusing non-fast-forward on all branches, excluding dependabot's and the merge queue's own, which it has checked against abcd's own flows."
---

History rewriting is only partly prevented, against the product thinker's ruling of 2026-11-03 that abcd never rewrites history: the guard's coverage is partial, and server-side rules cover main alone.
