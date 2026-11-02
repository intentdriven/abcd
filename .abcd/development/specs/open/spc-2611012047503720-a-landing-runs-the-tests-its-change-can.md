---
id: spc-2611012047503720
slug: a-landing-runs-the-tests-its-change-can
intent: itd-2611011325421304
origin: researcher-authored
production_mode: hand-written
---
# A one-package change is checked in minutes, not the whole suite

## Summary

Delivers itd-2611011325421304: on the person's machine, the pre-push check reuses every Go test result that cannot have changed, reruns the tests that read what the cache cannot see from a list kept on the default branch, runs everything fresh for a release cut, and reports how many packages ran and how many were reused. The saving is measured over ten landings before it is stated.

**Approach.** Go's test cache already keys a package's result on the test binary, the cacheable flags, and the files and environment variables the test reads (cmd/go). Two things defeat it today. Each worktree's absolute source paths change the binary, so a fresh worktree never hits; `-trimpath` removes them. And nine test files find the repository through `runtime.Caller`, which `-trimpath` turns into a module path, so they fail under it (iss-2611011343052330). So the work is: sweep that pattern and arm a detector against it; add `-trimpath` to the preflight's unit and race `go test` lines; split out the always-rerun tests, read from the default branch's list, into a `-count=1` line of their own; force `-count=1` for a release cut's gates; and count `(cached)` results per stage into the report and the receipt.

**How each criterion is met.**
- Reuse in a fresh worktree (criterion 1): `-trimpath` on both test stages, with the person's one Go build cache shared by every worktree of the checkout, as it already is (GOCACHE is per account).
- The always-rerun list (criterion 2): a list of package patterns read with `git show origin/<default>:<path>` at preflight time, never from the working tree, so a change cannot remove its own tests from it (the principle evaluator-outside-the-loop); those packages run in a separate `go test -count=1` line and are excluded from the cached line. A release cut's gates run with `-count=1` throughout.
- Record-only changes (criterion 3): with the cache working, a test that opened a record reruns when that record changed and is reused when it did not; the report names which packages ran. The records-only path stays as the fast route it is today.
- `-trimpath` passes (criterion 4): every `runtime.Caller` repository lookup in a test is replaced by a walk up from the test's working directory to `go.mod`, and a lint test refuses a new one (fix-the-detector).
- The report (criterion 5): the preflight counts `ok … (cached)` against `ok …` lines per stage and prints "reused N of M" for each; the receipt records both counts.
- The measured saving (criterion 6): after ship, ten consecutive landings' reports are recorded in a dated research note against the 2026-11-01 baseline of no reuse; that note, not an estimate, states the saving.

**Out of scope.** Hosted CI reuse (a fresh checkout's file times defeat it; a follow-up intent), the merge queue's checks, and how many checks run at once (itd-2610262141001890).

## Footprint

- packages: Makefile, scripts, internal/core/lint, internal/core/launch, internal/core/launch/scaffold, internal/surface/cli
- tests: internal/core/lint (the Makefile and preflight gate tests, and the new runtime.Caller detector), the nine test files iss-2611011343052330 names, `go test -trimpath ./...`

## Steps

1. Tests find the repository without their own source path
   - packages: internal/core/lint, internal/core/launch, internal/core/launch/scaffold, internal/surface/cli
   - tests: the nine test files iss-2611011343052330 names, a lint test refusing a `runtime.Caller` repository lookup in a test, `go test -trimpath ./...`
   - needs: none
2. The preflight reuses results, reruns the listed tests and reports the counts
   - packages: Makefile, scripts, internal/core/lint
   - tests: internal/core/lint (the preflight gate and Makefile tests), a preflight in a fresh worktree reporting reused packages
   - needs: 1
3. The saving is measured over ten landings
   - packages: .abcd/development/research/notes
   - tests: the research note's ten recorded reports against the 2026-11-01 baseline
   - needs: 2
