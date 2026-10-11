---
schema_version: 1
id: "iss-2610090821527948"
slug: "missing-promisor-object-triggers"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "private security advisory GHSA-4jpc-8vpr-5x2j, filed 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/repo.go"
remedy: "Set `GIT_NO_LAZY_FETCH=1` in `gitEnv` so a missing object returns an error instead of fetching; prove it with a gitutil test (watched fail first) that Plan on a local-path promisor with a missing manifest blob leaves the uploadpack mark empty and returns the git error, `GitExistingTags` with `tag.sort=taggerdate` and a missing v* object leaves the mark empty, a present blob and tag still read, and the sshCommand and ext:: marks stay empty; sweep siblings (ScrubbedEnv and pickGit, and every other git environment abcd builds)."
resolution: "gitEnv sets GIT_NO_LAZY_FETCH=1, so on git 2.44 or later a read of a missing object in a promisor repository (show, cat-file, a tag list under an object-reading tag.sort) is an error and starts no configured transport. git older than 2.44 ignores the variable (Apple's Command Line Tools ship 2.39), so below that floor every isolated git command that reads objects (Run, RunLimited, RunCapped, RunLimitedContext, IsAncestor, ArchiveTree, and GitExistingTags, now routed through Run) refuses, before git starts and naming the 2.44 floor, a repository that declares extensions.partialClone or a true remote.<name>.promisor; a repository with no promisor remote reads as before, root discovery still answers, and the README states the floor"
impact: fix
---

abcd's git environment does not set `GIT_NO_LAZY_FETCH`, so an object read (`show`, `cat-file`, `ls-tree`, or `tag --list` with an object-reading `tag.sort`) of a missing object in a promisor checkout lazy-fetches and runs the repository's transport program (`remote.origin.uploadpack` on a local or file:// URL, `core.sshCommand` on ssh://).

Private security advisory GHSA-4jpc-8vpr-5x2j (draft, severity medium). Full text, evidence and reproduction: the security-drain-2026-10-09 run directory in the main checkout's local tier. This record stays uncommitted until its fix lands; the fix commit adds it directly to resolved/.

Evidence (lines at main 7549ca2d5): `gitEnv` (internal/gitutil/repo.go:74) ends its config keys at core.fsmonitor (internal/gitutil/repo.go:107) and sets no GIT_NO_LAZY_FETCH; `Run`, `RunLimited` and `IsolatedEnv` (internal/gitutil/repo.go:122) share it. Sinks: `lifeboat.Plan` (internal/core/lifeboat/plan.go:187) calls `buildArchaeology` (:305), where `gvRemovedDependencies` shows a historical manifest and ignores the error (internal/core/lifeboat/graveyard_archaeology.go:242), so `abcd disembark plan` and `pack` succeed after the program ran; `committedRegistry` cat-files `HEAD:.abcd/guard.json` and falls back to defaults (internal/core/guard/config.go:99); `GitExistingTags` (internal/core/launch/retention.go:108) is reached by `abcd launch --dry-run` twice (internal/surface/cli/launch_deep.go:77 via `changelog.LatestReleaseTag`, internal/core/changelog/anchor.go:58, and internal/core/launch/dryrun.go:229), by changelog derive and guard, capture's deferral path, `reflect` `previousTag` (internal/core/reflect/seed.go:212) and the receipt gate's `releaseImpact` (internal/core/lint/lint.go:1437). `disembark probe` does not take this path.

Reproduction: on git 2.39.5, delete the loose blob of a historical manifest the plan shows; set `extensions.partialClone=origin`, `remote.origin.promisor=true`, a partial-clone filter, `remote.origin.url` a local path and `remote.origin.uploadpack` a mode-0755 script; run `abcd disembark plan`. The script runs and Plan returns a lifeboat with a nil error. For the tag sink, point `refs/tags/v1.1.0` at a deleted commit, set `tag.sort=taggerdate` and call `GitExistingTags`: the script runs and the call returns exit 128. `--sort=refname` and `-c tag.sort=refname` do not stop it; `GIT_NO_LAZY_FETCH=1` does.
