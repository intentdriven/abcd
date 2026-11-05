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

**Approach.** Go's test cache already keys a package's result on the test binary, the cacheable flags, and the files and environment variables the test reads (cmd/go). Three things defeat it today. Each worktree's absolute source paths change the binary, so a fresh worktree never hits; `-trimpath` removes them. Go keys every in-module file a test opens on its size and modification time (cmd/go, hashWriteStat), and a fresh worktree gives every file a new time, and Go hashes a file a test opens by a relative name joined to the package's absolute directory (cmd/go/internal/test/test.go, computeTestInputsID), so such a test misses the cache in a worktree at another path even when nothing it reads changed; a file opened by an absolute name outside the module root is not hashed at all, so a test that reads one can be reused with a stale result; setting file times alone fixes neither. The product thinker chose the fixed check directory on 2026-11-02 (one per repository, ~/.abcd.noindex/tests/<root-sha>/, one check at a time); this spec's own design for it: the preflight brings that directory to the commit's tree, proving each file's bytes against the commit's blob and writing only what differs, and sets each file's time to a fixed past base, 2001-01-01T00:00:00Z, plus bits of the SHA-256 of its content, reduced to a window of whole seconds about twenty years long, ending before 2021, and to the nanoseconds of that second, so every derived time is in the past (Go refuses to cache a test that opened a file whose time is within two seconds of now or later, cmd/go errFileTooNew) and within what APFS and ext4 store; the cache key is that time together with the file's size, and a file's time changes when its content does and is the same in every worktree (never a commit time, which never had the future-time problem and is rejected for the reason that stands: an author time survives an amend, and a committer time is whole seconds, so a same-size edit in the same second would keep it), and runs the test stages there, one check at a time under a lock, so every worktree's run hashes the same paths and times and its results carry over.

**How each criterion is met.**
- Reuse in a fresh worktree (criterion 1): `-trimpath` on both test stages, the tests run from the fixed check directory with content-derived file times, and the person's one Go build cache, so a test whose inputs are unchanged is reused whichever worktree the check started in.
- The always-rerun list (criterion 2): a list of package patterns read with `git show origin/<default>:<path>` at preflight time, never from the working tree, so a change's own diff does not move it (the principle evaluator-outside-the-loop); those packages run in a separate `go test -count=1` line and are excluded from the cached line. A release cut's gates run with `-count=1` throughout, and so does every hosted test step.
- Record-only changes (criterion 3): with the cache working, a test that opened a record reruns when that record changed and is reused when it did not; the report names which packages ran. The records-only path stays as the fast route it is today.
- `-trimpath` passes (criterion 4): every `runtime.Caller` repository lookup in a test is replaced by a walk up from the test's working directory to `go.mod`, and a lint test refuses a new one (fix-the-detector).
- The report (criterion 5): the preflight counts `ok … (cached)` against `ok …` lines per stage and prints "reused N of M" for each; the receipt records both counts.
- The measured saving (criterion 6): after ship, ten consecutive landings' reports are recorded in a dated research note against the 2026-11-01 baseline of no reuse; that note, not an estimate, states the saving.

**Out of scope.** Hosted CI reuse (a fresh checkout's file times defeat it; a follow-up intent), the merge queue's checks, and how many checks run at once (itd-2610262141001890).

## Footprint

- packages: Makefile, scripts, internal/abcdhome, internal/core/lint, internal/core/launch, internal/core/launch/scaffold, internal/surface/cli
- tests: internal/core/lint (the Makefile and preflight gate tests, and the new runtime.Caller detector), the nine test files iss-2611011343052330 names, `go test -trimpath ./...`

## Steps

1. Tests find the repository without their own source path
   - packages: internal/core/lint, internal/core/launch, internal/core/launch/scaffold, internal/surface/cli
   - tests: the nine test files iss-2611011343052330 names, a lint test refusing a `runtime.Caller` repository lookup in a test, `go test -trimpath ./...`
   - needs: none
   - landed: 11418782b08a
2. The preflight runs its tests from the fixed check directory, reuses results, reruns the listed tests and reports the counts
   - packages: Makefile, scripts, internal/core/lint, internal/abcdhome
   - tests: internal/core/lint (the preflight gate and Makefile tests), a preflight in a fresh worktree reporting reused packages, a test that reads a tracked file reused in a second worktree, and a same-size edit, amended or committed within the same second, rerun
   - needs: 1
   - landed: 5795e213f668
3. The saving is measured over ten landings
   - packages: .abcd/development/research/notes
   - tests: the research note's ten recorded reports against the 2026-11-01 baseline
   - needs: 2
