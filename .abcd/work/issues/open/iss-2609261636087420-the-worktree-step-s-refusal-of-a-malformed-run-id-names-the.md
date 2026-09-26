---
schema_version: 1
id: "iss-2609261636087420"
slug: "the-worktree-step-s-refusal-of-a-malformed-run-id-names-the"
severity: "nitpick"
category: "ux"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-loop2"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/lane.go"
---

The worktree step's refusal of a malformed run id names the lane id in its lane field, and its refusal of a malformed lane id names no id there, so neither refusal names the id it refuses. Each refusal should carry the id it refuses.
