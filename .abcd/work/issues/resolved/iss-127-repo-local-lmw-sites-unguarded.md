---
schema_version: 1
id: "iss-127"
slug: "repo-local-lmw-sites-unguarded"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "iss-101/102 class sweep (2026-07-24 run queue, burst 3)"
found_at: "internal/core/intent/lifecycle.go"
resolution: "ahoy's rewrites of .abcd/config.json, the .gitignore block and the CLAUDE.md/AGENTS.md marker block, and the release cut's CHANGELOG writes and undo, now read and write under fsutil.WithFileLock on a retired lock file beside the guarded file; intent transitions were already under the intent store lock at this base"
impact: fix
resolved_by:
  commit: "352b21e19"
---

repo-local load-modify-write sites remain unguarded after the iss-101/102 class fix: intent transitions (review.go, lifecycle.go — strongest candidate), ahoy config.json stepConfigValues, gitignore/marker block rewrites, and the CHANGELOG release path all do load-mutate-write without a lock; they lack the home-global cross-worktree exposure the fixed sites had, so risk is same-worktree concurrency only

## Grounds

- pursued: two abcd runs in one working tree rewriting the same file keep both changes, or the later cut is refused as a release in flight; a concurrent writer's change missing after a race test (rewritelock_test.go, changeloglock_test.go) would show it wrong
