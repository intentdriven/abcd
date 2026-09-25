---
schema_version: 1
id: "iss-2609100508570803"
slug: "the-record-verbs-worked-from-worktrees-throughout-the-run"
severity: "nitpick"
category: "observation"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-09/10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal (capture, intent, spec, ideate) from git worktrees"
resolution: "positive evidence only: it confirms adr-32's one-file-per-record ledger under 27 parallel worktrees; nothing to decide"
impact: internal
---

Positive finding, recorded as evidence rather than as a defect: every record verb the run reached worked correctly from a git worktree, throughout a day of heavy parallel use.

Observed across an autonomous run in a managed repository that landed 27 worker branches, resolved 33 issues, shipped 5 intents, superseded 3 ADRs, held 3 intents with stated reasons and recorded 1 ideate verdict, through roughly 45 review rounds and one pull request. Every worker ran in its own worktree. `abcd capture`, `abcd capture resolve`, `abcd intent ready`, `abcd intent plan`, `abcd spec close` and `abcd ideate record` all behaved correctly from inside a worktree, with two independent workers reporting `capture resolve` success explicitly. The ledger's move operations (open/ to resolved/) never once conflicted across all 27 merges: one-file-per-record is the right shape, and this run is the strongest evidence for it the project has.

Two things this is evidence FOR, worth stating because they were live design questions. The record store's folder-as-status model survives heavy parallel branching provided each record is one file. And the checkout-is-the-unit-of-isolation convention holds in practice: worktrees did not need special handling from the record verbs, which is why the failures this run produced were all at the edges rather than in the store.

The only gap encountered in that whole set is filed separately: `ideate record` prints "the idea does not graduate" for a `reframed` verdict, which reads as killed.

## Grounds

- pursued: the record verbs hold from worktrees under heavy parallel use; shown wrong if a record verb misbehaves from a git worktree
