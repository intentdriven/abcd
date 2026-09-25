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
---

A local branch whose pull request has merged is never pruned, so a checkout accumulates them until somebody sweeps by hand: this checkout held 57 on 2026-09-17, one per landed PR since early September. The sweep cannot be 'git branch --merged', because the repository allows squash and rebase merges and those leave no ancestry, so 20 of the 57 read as unmerged by ancestry and were provably merged only by patch-id or by the forge's PR state; and a branch checked out in a worktree survives the sweep until the worktree is removed. abcd knows the merge state (the resolution gates already read origin/main and the PR), so the install step or a fetch-time pass could list the merged branches and remove them on consent, never one with unmerged patches, never one a worktree holds, and never a name matching a backup pattern.
