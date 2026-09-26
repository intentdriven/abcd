---
schema_version: 1
id: "iss-2609261215168271"
slug: "the-bundle-spec-close-reconcilebundle-in-internal-core"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-itd34"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/bundle.go"
resolution: "The bundle close judges every member under the intent mint lock and pre-flights each member's shipped/ destination before the first move, as PlanBundle pre-flights planned/."
impact: fix
resolved_by:
  commit: "d14715654a2396e02abf3d07c22fd4ca98309d7c"
---

The bundle spec close (reconcileBundle in internal/core/intent/bundle.go) pre-flights no destination, so a shipped/<name> collision on the second member moves the first member before the rename refuses; the rollback restores it byte-identical, but the doc comment's claim that every other refusal fires before anything moves is false. PlanBundle has the per-member Lstat pre-check this close lacks.

## Grounds

- pursued: we expect a twin at any member's shipped/ path to refuse the close before any member moves, with a refusal that says nothing moved; shown wrong if a refused bundle close ever reports a rollback for a destination it could have seen
