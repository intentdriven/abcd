---
schema_version: 1
id: "iss-2609170640135192"
slug: "a-local-branch-whose-pull-request-has-merged-is-never-pruned"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "local branch sweep of the [redacted-user] checkout, 2026-09-17"
origin: researcher-authored
production_mode: hand-written
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Plan draft itd-118 (prune merged branches by patch-id or forge state), or does itd-148's merged-half cleanup cover it?"
wontfix_reason: "Duplicate of itd-118 (draft), which scopes abcd tidying the stale local branch, its tracking ref and worktree once a pull request merges, with itd-148 (planned, spc-42) supplying the merged-half detection this record asks for: provably merged means the forge's recorded merge state where a PR exists and patch-equivalence as the local fallback, so squash and rebase merges are covered, and a branch whose merge cannot be proven is never removed. The record's three never-clauses (never a branch with unmerged patches, never one a worktree holds, never a backup-pattern name) are inputs to itd-118's planning."
---

A local branch whose pull request has merged is never pruned, so a checkout accumulates them until somebody sweeps by hand: this checkout held 57 on 2026-09-17, one per landed PR since early September. The sweep cannot be 'git branch --merged', because the repository allows squash and rebase merges and those leave no ancestry, so 20 of the 57 read as unmerged by ancestry and were provably merged only by patch-id or by the forge's PR state; and a branch checked out in a worktree survives the sweep until the worktree is removed. abcd knows the merge state (the resolution gates already read origin/main and the PR), so the install step or a fetch-time pass could list the merged branches and remove them on consent, never one with unmerged patches, never one a worktree holds, and never a name matching a backup pattern.

## Grounds

- declined: the prune is carried by itd-118 with itd-148's merge proof; this would be wrong if itd-118 were planned without a local-branch prune
