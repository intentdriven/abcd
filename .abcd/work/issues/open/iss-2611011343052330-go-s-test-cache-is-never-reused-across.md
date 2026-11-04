---
schema_version: 1
id: "iss-2611011343052330"
slug: "go-s-test-cache-is-never-reused-across"
severity: "major"
category: "tech-debt"
source: "agent-finding"
found_during: "CI and local compute investigation, 2026-11-01"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "Make every test helper that finds the repository through runtime.Caller find it by walking up from the test's working directory to go.mod (sweep the pattern, not the nine files listed), then add -trimpath to the preflight's unit and race go test lines and to CI's go test steps, so Go's test cache reuses unchanged packages whose tests' logs name no path inside the module (no open, stat, lstat or chdir of one; criterion 4's walk up to go.mod is such a lookup) across worktrees (whether packages whose tests open files may be reused across worktrees is settled by itd-2611011325421304's fixed check directory, ~/.abcd.noindex/tests/<root-sha>/, and a test that reads a file outside the module, which Go does not hash, belongs on the always-rerun list); prove it with a preflight in a fresh worktree that reports reused packages, and with the whole suite passing under -trimpath."
related_intents: [itd-2611011325421304]
deferred_after: "v0.16.0"
deferral_reason: "Deferred by the product thinker on 2026-11-03 for this cut, v0.16.1 as the records derive it: the fix sweeps every test helper that locates the repository; the planned test-reuse build (itd-2611011325421304) takes it up."
---

Go's test cache is never reused across worktrees, so every preflight runs all packages twice from scratch. A full preflight on 2026-11-01 reported 0 cached results of 133 packages. The cache keys on the test binary, and each worktree's absolute source paths change it: the same unchanged package in a second worktree re-ran, while with -trimpath it came straight from the cache. -trimpath cannot simply be switched on: nine test files locate the repository through runtime.Caller (internal/core/lint/releasereceipts_test.go, internal/core/launch/payload_completeness_test.go, internal/core/launch/scaffold/scaffold_test.go, and six in internal/surface/cli: hooks_sessionstart, hook_scripts, barerender, hooks_selfprovision, bootstrap, githook_prepush), and under -trimpath every one fails, since the path becomes the module path (open github.com/intentdriven/abcd/.github/workflows/release.yml: no such file or directory). Across the last 60 merges, about two thirds of all test time re-ran results that could not have changed.
