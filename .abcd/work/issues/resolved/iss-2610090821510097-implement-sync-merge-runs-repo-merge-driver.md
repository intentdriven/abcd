---
schema_version: 1
id: "iss-2610090821510097"
slug: "implement-sync-merge-runs-repo-merge-driver"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "private security report, filed 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/sync.go"
remedy: "On this computed merge only, blank every configured `merge.<name>.driver` with `-c` arguments passed to `pickGit` before the `merge` subcommand, leaving merges the operator runs untouched; prove it with a loop test (watched fail first) that the overlapping sync leaves the driver mark empty and keeps git's own merge result, the no-overlap control still starts no driver, and a merge run outside `pickGit` still honours the driver; sweep siblings (every merge, rebase or cherry-pick abcd composes that can run a repo-configured program)."
resolution: "the implement loop's sync merge replaces every configured merge driver, and merge.default, with git's built-in text merge (gitutil.MergeDriverOverrides), so no repository-named merge program runs and the result is git's own"
impact: fix
---

The implement loop's lane sync merge (`git merge --no-ff` through `pickGit`) runs a repository-configured merge driver, which executes as the operator and writes the merge result git keeps, although the loop presents this merge as running no hook code.

A private security report, fixed in this release; its advisory, with the full text and reproduction, is published with the release.

Evidence (lines at main 7549ca2d5): `syncLane` (internal/core/implement/loop/sync.go:66) merges with `pickGit(lane.Worktree, "merge", "--no-ff", "--no-edit", "-m", msg, merged)` (internal/core/implement/loop/sync.go:95). `pickGit` pins hooks, fsmonitor and quotePath and runs under ScrubbedEnv, so repo and global config stay (internal/core/implement/loop/pickcommit.go:71). `landStage` calls `syncLane` only while `lane.Landing` is nil (internal/core/implement/loop/land.go:131, :139). The other side can come from a planted `refs/remotes/origin/<default>`; no network remote is needed.