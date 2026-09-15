---
schema_version: 1
id: "iss-2608291814578333"
slug: "git-dir-exclusion-is-a-one-off-guard"
severity: "minor"
category: "architectural-insight"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "internal/core/positioning/config.go"
---

ultra-v0.6.8 altitude 5: the .git first-segment exclusion in internal/core/positioning/config.go Surface.validate is a one-off guard layered on fsutil.ValidRelPath, which every other repo-relative path consumer (site manifest, lint config, record-lint) relies on without it — so another committed config naming a repo-relative file can still quote .git/config and leak a credential-bearing remote URL. Deeper fix: a shared fsutil predicate (denied root segments, case-folded) used by every validator, or enforcement in ReadGuardedInRoot for repo roots.
