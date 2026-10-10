---
schema_version: 1
id: "iss-2610090821510097"
slug: "implement-sync-merge-runs-repo-merge"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "private security advisory GHSA-486x-8h9p-pq5x, filed 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/sync.go"
remedy: "On this computed merge only, blank every configured `merge.<name>.driver` with `-c` arguments passed to `pickGit` before the `merge` subcommand, leaving merges the operator runs untouched; prove it with a loop test (watched fail first) that the overlapping sync leaves the driver mark empty and keeps git's own merge result, the no-overlap control still starts no driver, and a merge run outside `pickGit` still honours the driver; sweep siblings (every merge, rebase or cherry-pick abcd composes that can run a repo-configured program)."
resolution: "the implement loop's sync merge replaces every configured merge driver, and merge.default, with git's built-in text merge (gitutil.MergeDriverOverrides), so no repository-named merge program runs and the result is git's own"
impact: fix
---

The implement loop's lane sync merge (`git merge --no-ff` through `pickGit`) runs a repository-configured merge driver, which executes as the operator and writes the merge result git keeps, although the loop presents this merge as running no hook code.

Private security advisory GHSA-486x-8h9p-pq5x (draft, severity medium). Full text, evidence and reproduction: the security-drain-2026-10-09 run directory in the main checkout's local tier. This record stays uncommitted until its fix lands; the fix commit adds it directly to resolved/.

Evidence (lines at main 7549ca2d5): `syncLane` (internal/core/implement/loop/sync.go:66) merges with `pickGit(lane.Worktree, "merge", "--no-ff", "--no-edit", "-m", msg, merged)` (internal/core/implement/loop/sync.go:95). `pickGit` pins hooks, fsmonitor and quotePath and runs under ScrubbedEnv, so repo and global config stay (internal/core/implement/loop/pickcommit.go:71). `landStage` calls `syncLane` only while `lane.Landing` is nil (internal/core/implement/loop/land.go:131, :139). The other side can come from a planted `refs/remotes/origin/<default>`; no network remote is needed.

Reproduction: on git 2.39.5, set `.git/info/attributes` to `* merge=evil` (or a committed `.gitattributes`) and `merge.evil.driver` to a script; point `refs/remotes/origin/<default>` at a commit editing a file the lane also edits; run `abcd implement step` until the lane reaches `syncLane` with no landing. The script runs, git reports success and the worktree file holds the script's output; the post-merge hook does not run. With no overlapping change the driver does not start.
