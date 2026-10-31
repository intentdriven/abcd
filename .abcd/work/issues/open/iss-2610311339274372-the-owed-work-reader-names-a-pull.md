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
remedy: "Skip the BEHIND item when the default branch's ruleset declares a merge queue and the pull request has auto-merge armed; keep DIRTY, which a queue cannot resolve."
---

The owed-work reader names a pull request whose merge state is BEHIND as owed a base merge (internal/core/autonomy/owedread.go, the BEHIND case: 'is behind its base: merge it in'), and the away-mode stop hook then refuses the session's stop on it, even in a repository whose default branch is guarded by a merge queue. A queue tests every entry against the latest base itself, so a branch that is merely behind needs no merge; merging the base in moves the head, which voids the head's preflight receipt and review receipt and costs a full preflight and a fresh review pair for nothing. On 2026-10-31 PR #1291 sat BEHIND with auto-merge armed and its last required check still running, and the hook held the session on it across wakes.
