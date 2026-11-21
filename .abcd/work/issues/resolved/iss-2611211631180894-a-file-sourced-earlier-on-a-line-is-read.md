---
schema_version: 1
id: "iss-2611211631180894"
slug: "a-file-sourced-earlier-on-a-line-is-read"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "review-of-guard-coverage"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Carry a sourced file's exports, and an allexport it turns on, into the rest of the line"
resolution: "A sourced file's exports, and an allexport it turns on, carry into the rest of the line in the guard's export walk."
impact: additive
resolved_by:
  commit: "2f4e6ee86"
---

A file sourced earlier on a line is read for the registry's hazards, but the exports it makes are not carried into the rest of the line, so an opencode command after sourcing an env file is judged without them.
