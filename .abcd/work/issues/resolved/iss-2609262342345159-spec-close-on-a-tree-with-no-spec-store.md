---
schema_version: 1
id: "iss-2609262342345159"
slug: "spec-close-on-a-tree-with-no-spec-store"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainSpec"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/spec/store.go"
resolution: "Close and Discard check for the spec store before taking its lock: with none, Close refuses the id as not found and Discard has nothing to remove, and neither plants .abcd/development/specs/."
impact: internal
resolved_by:
  commit: "94cecdf7e"
---

spec.Close on a tree with no spec store plants an empty .abcd/development/specs/ before failing 'not found': Close takes the store lock through withStoreLock, whose ensureDir creates the store to lock it, contradicting the rule the spec store states beside WithStoreLock (a verb that writes no spec must not plant an empty store). spec.Discard has the same shape. Unreachable through any in-tree verb today (both Close callers load the spec first), but the writer itself must not plant the store.

## Grounds

- pursued: TestAWriterOnATreeWithNoSpecStorePlantsNone asserts that Close and Discard on a tree with no spec store leave no .abcd/development/specs/ behind, and that Close still refuses the id; a writer that plants the store, or a Close that stops refusing, turns it red.
