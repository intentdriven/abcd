---
id: itd-150
slug: agent-worktrees-commit-without-the-private-name-guard-abcd-w
spec_id: spc-43
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-370
---

# Agent worktrees commit without the private name-guard: .abcd/.work.local/ is per-worktree, so every isolated-worktree agent commit runs with the banlist layer absent — loudly warned, per design, but the isolated-agent pattern now systematically bypasses a protection the main checkout has. Candidate remedies: the worktree-creation path seeds a pointer to the primary checkout's store, or the hook falls back to reading the primary worktree's local tier

## Press Release

> _Seeded by promotion from iss-370. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-370`: Agent worktrees commit without the private name-guard: .abcd/.work.local/ is per-worktree, so every isolated-worktree agent commit runs with the banlist layer absent — loudly warned, per design, but the isolated-agent pattern now systematically bypasses a protection the main checkout has. Candidate remedies: the worktree-creation path seeds a pointer to the primary checkout's store, or the hook falls back to reading the primary worktree's local tier. Read that issue record for the source observation.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a name recorded in the private banlist of the primary checkout's `.work.local` store but absent from any store inside a linked git worktree, **when** an agent stages a commit whose content carries that name from within the linked worktree, **then** the guard hook resolves the banlist from the primary checkout and blocks the commit on that name.
- **Given** a freshly created linked worktree with no private banlist file of its own, **when** the guard hook runs there, **then** it enforces the same banlist as the main checkout without any per-worktree banlist setup having been performed.
- **Given** a name banned in the linked worktree's own local store, **when** a commit runs in that worktree, **then** the guard still blocks it, so the primary store is a fallback and never overrides a worktree-local entry.
- **Given** the guard runs in a standalone checkout that is not a linked worktree, **when** no primary-checkout store can be resolved, **then** it behaves as it does today and emits the existing loud warning rather than failing closed on a resolution error.

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
