---
id: itd-118
slug: merged-work-leaves-no-residue-abcd
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
---

# Merged work leaves no residue: abcd-managed repos delete a PR branch on merge, and abcd tidies the stale local branch, tracking ref, and worktree

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

Merged work leaves no residue: abcd-managed repos delete a PR branch on merge, and abcd tidies the stale local branch, tracking ref, and worktree

This is the post-merge complement of itd-115: that intent gets a ready PR
*into* the base without out-of-sync churn; this one cleans up what the merge
leaves *behind* — the remote PR branch, the stale local branch, its tracking
ref, and any worktree checked out on it.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
