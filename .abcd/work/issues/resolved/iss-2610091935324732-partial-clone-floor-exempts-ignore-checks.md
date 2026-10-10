---
schema_version: 1
id: "iss-2610091935324732"
slug: "partial-clone-floor-exempts-ignore-checks"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "security-drain-2026-10-09 lane W sweep"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/gitignore.go"
remedy: "In readsObjects count check-ignore without --no-index, and ls-files with -o/--others/-i/--ignored/--exclude-standard/--exclude-per-directory/--with-tree, as reading objects, proved by a test with the injected old git version, watched fail first."
resolution: "readsObjects now counts check-ignore without --no-index, config --blob, and ls-files with any flag outside an index-only allowlist (the exclude flags, --with-tree, --eol, --format, -m) or an attr pathspec as reading objects, so on git older than 2.44 a partial clone refuses them before git starts; proved by TestIgnoreReadsRefuseAPartialCloneBelowTheLazyFetchFloor with the injected git 2.39"
impact: fix
---

On git older than 2.44 the partial-clone refusal exempts check-ignore and ls-files as commands that read no objects, but git reads a skip-worktree .gitignore missing from disk out of the object store, so a copied partial clone still lazy-fetches through a promisor remote's transport on those paths (CheckIgnored, IgnoredUnder, lifeboat probe, launch gates, intent consistency, capture reframe). Sibling of iss-2610090821527948, found by the security-drain-2026-10-09 lane W sweep; kept uncommitted until its fix lands.
