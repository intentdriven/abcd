---
schema_version: 1
id: "iss-2609100509533147"
slug: "intent-mints-into-the-checkout-it-runs-in-with-no-repo-target"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-09/10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal (intent) / related iss-89"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Should intent minting target the primary worktree by default?"
---

`abcd intent` creates its draft in the checkout it runs in, with no repository or worktree target, which bites when the record store is being edited across several worktrees at once.

Observed in an autonomous run in a managed repository where several worktrees were live against one record store. The verb has no `--repo` and no worktree target, so where a draft lands is decided by the process's working directory rather than by the operator, and a session working in a feature worktree mints into that worktree's copy of the store. The record then travels only with that branch — or, if the branch is abandoned, not at all.

This is the same shape as iss-89, which reports that `abcd capture` writes only to the cwd repo's ledger and so has no way to route a defect found in one repository into another's store. That record's proposed remedy is exactly the one wanted here: an upstream or `--repo` flag on the writing verbs. The two differ only in which cross-boundary case bites — iss-89 is cross-repository (an abcd defect found while working in a managed repo), this is cross-worktree within one repository — and a `--repo`-shaped fix answers both. Filed rather than folded into iss-89 because the worktree case has a second requirement iss-89 does not: within one repository, the operator usually wants the PRIMARY worktree's store, not an arbitrary path, and the verb could default there rather than requiring the path be named.
