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
---

The bundle spec close (reconcileBundle in internal/core/intent/bundle.go) pre-flights no destination, so a shipped/<name> collision on the second member moves the first member before the rename refuses; the rollback restores it byte-identical, but the doc comment's claim that every other refusal fires before anything moves is false. PlanBundle has the per-member Lstat pre-check this close lacks.
