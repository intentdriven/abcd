---
schema_version: 1
id: "iss-2609261636087420"
slug: "the-worktree-step-s-refusal-of-a"
severity: "nitpick"
category: "ux"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-loop2"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/lane.go"
resolution: "laneName's refusals each name, quoted, the id they refuse and carry no lane, so no unvalidated lane id reaches the one-line rendering."
impact: internal
resolved_by:
  commit: "fd503c7fc998b4672bb47058d5f416335492ed87"
---

The worktree step's refusal of a malformed run id names the lane id in its lane field, and its refusal of a malformed lane id names no id there, so neither refusal names the id it refuses. Each refusal should carry the id it refuses.

## Grounds

- pursued: a refusal names what it refuses and nothing it did not; TestALaneNameRefusalNamesTheIDItRefuses would fail if the run-id refusal carried the lane id or a raw control byte
