---
schema_version: 1
id: "iss-2610090821527948"
slug: "missing-promisor-object-triggers-configured-fetch"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "private security report, filed 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/repo.go"
remedy: "Set `GIT_NO_LAZY_FETCH=1` in `gitEnv` so a missing object returns an error instead of fetching; prove it with a gitutil test (watched fail first) that Plan on a local-path promisor with a missing manifest blob leaves the uploadpack mark empty and returns the git error, `GitExistingTags` with `tag.sort=taggerdate` and a missing v* object leaves the mark empty, a present blob and tag still read, and the sshCommand and ext:: marks stay empty; sweep siblings (ScrubbedEnv and pickGit, and every other git environment abcd builds)."
resolution: "gitEnv now sets GIT_NO_LAZY_FETCH=1, so a read of a missing object in a promisor repository (show, cat-file, a tag list under an object-reading tag.sort) is an error and starts no configured transport; Run, RunLimited and IsolatedEnv share it, covering GitExistingTags"
impact: fix
---

abcd's git environment does not set `GIT_NO_LAZY_FETCH`, so an object read (`show`, `cat-file`, `ls-tree`, or `tag --list` with an object-reading `tag.sort`) of a missing object in a promisor checkout lazy-fetches and runs the repository's transport program (`remote.origin.uploadpack` on a local or file:// URL, `core.sshCommand` on ssh://).

A private security report, fixed in this release; its advisory, with the full text and reproduction, is published with the release.

Evidence (lines at main 7549ca2d5): `gitEnv` (internal/gitutil/repo.go:74) ends its config keys at core.fsmonitor (internal/gitutil/repo.go:107) and sets no GIT_NO_LAZY_FETCH; `Run`, `RunLimited` and `IsolatedEnv` (internal/gitutil/repo.go:122) share it. Sinks: `lifeboat.Plan` (internal/core/lifeboat/plan.go:187) calls `buildArchaeology` (:305), where `gvRemovedDependencies` shows a historical manifest and ignores the error (internal/core/lifeboat/graveyard_archaeology.go:242), so `abcd disembark plan` and `pack` succeed after the program ran; `committedRegistry` cat-files `HEAD:.abcd/guard.json` and falls back to defaults (internal/core/guard/config.go:99); `GitExistingTags` (internal/core/launch/retention.go:108) is reached by `abcd launch --dry-run` twice (internal/surface/cli/launch_deep.go:77 via `changelog.LatestReleaseTag`, internal/core/changelog/anchor.go:58, and internal/core/launch/dryrun.go:229), by changelog derive and guard, capture's deferral path, `reflect` `previousTag` (internal/core/reflect/seed.go:212) and the receipt gate's `releaseImpact` (internal/core/lint/lint.go:1437). `disembark probe` does not take this path.