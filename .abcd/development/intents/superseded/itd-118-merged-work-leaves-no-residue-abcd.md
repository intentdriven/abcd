---
id: itd-118
slug: merged-work-leaves-no-residue-abcd
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history:
  - { date: 2026-10-23, from: standalone, to: superseded, reason: "the one-command cleanup covers everything this seed asked for: the merged branch on the server, the local branch, its tracking ref and its worktree (product thinker, 2026-10-23)" }
builds_on: []
severity: minor
kind_at_supersession: standalone
superseded_by: itd-2610222214476887
---

# Merged work leaves no residue: abcd-managed repos delete a PR branch on merge, and abcd tidies the stale local branch, tracking ref, and worktree

> **Superseded by itd-2610222214476887** on 2026-10-23: the one-command cleanup covers everything this seed asked for: the merged branch on the server, the local branch, its tracking ref and its worktree (product thinker, 2026-10-23)

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
