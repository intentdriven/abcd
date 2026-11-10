---
schema_version: 1
id: "iss-2610311339274372"
slug: "the-owed-work-reader-names-a-pull"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "landing PR #1291 under away mode"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/owedread.go"
remedy: "Skip the BEHIND item only for a pull request whose base is the default branch, when the forge's live ruleset for that branch (never the in-tree mirror under .abcd/work/rulesets/) declares a merge queue and the pull request has auto-merge armed; keep DIRTY, failing checks and the stacked-base BEHIND case."
resolution: "The stop check no longer owes a merge for a non-draft pull request into the default branch that is behind its base and armed or already queued, where the forge's live rules for that branch declare a merge queue; one nobody armed stays owed with 'arm it' as its next move (dcn-2611081501502395). A rules read that fails, is late or cannot be read whole leaves the item as before, with a note; DIRTY, failing checks and the stacked-base case are unchanged."
impact: fix
resolved_by:
  commit: "e4228929a58a85a74268fa4caaf5a30c1e4d757a"
---

The owed-work reader names a pull request whose merge state is BEHIND as owed a base merge (internal/core/autonomy/owedread.go, the BEHIND case: 'is behind its base: merge it in'), and the away-mode stop hook then refuses the session's stop on it, even in a repository whose default branch is guarded by a merge queue. A queue tests every entry against the latest base itself, so a branch that is merely behind needs no merge; merging the base in moves the head, which voids the head's preflight receipt and review receipt and costs a full preflight and a fresh review pair for nothing. On 2026-10-31 PR #1291 sat BEHIND with auto-merge armed and its last required check still running, and the hook held the session on it across wakes.

## Grounds

- pursued: TestBehindInMergeQueueRepository shows an armed or queued behind pull request in a queue repository is not owed, an unarmed one reads 'arm it', and a draft, a repository with no queue, a failed, hung, unparseable or full-page rules read, an in-tree mirror the forge contradicts, a conflict, failing checks and a stacked base all stay owed as before; it would be shown wrong by a live away-mode stop that still holds a session on an armed or queued pull request that is only BEHIND, or that lets a session stop past an unarmed one.
