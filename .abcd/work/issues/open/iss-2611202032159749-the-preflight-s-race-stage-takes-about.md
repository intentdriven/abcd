---
schema_version: 1
id: "iss-2611202032159749"
slug: "the-preflight-s-race-stage-takes-about"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "autonomous run of 20 Nov 2026, measuring step 3 of spc-2611012047503720"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/research/notes/2026-11-20-preflight-reuse-over-ten-landings.md"
remedy: "Time internal/core/implement/loop's cacheable tests the way the preflight runs them: through the load gate, with ABCD_TEST_SHARDS=3, -race -p 1 -count=1 -timeout 1h20m -v and no -skip, so the three shards stay and each shard's --- PASS lines carry every test's time (the 42 tests the classifier reruns, about 250s in all, appear under their own names and are left out of the count, since nothing prints the preflight's skip list); name the few tests that hold most of a shard's time, shorten them or move them out of the race stage, then time the race stage again on a landing that changes what loop's test binary reads."
---

The preflight's race stage takes about 39 minutes on a landing that changes anything loop's test binary reads, its own files or any of the 98 other internal packages its test binary builds in (`go list -deps -test`) (#1706 changed only `internal/core/site`), and about 30 minutes of that is loop's cacheable tests under `-race`. check-tests runs loop twice in the race stage: a cacheable run in the check directory that skips the package's always-rerun tests (`internal/testreuse/run.go:504`), and an always-rerun run with `-count=1` in the worktree (`internal/testreuse/run.go:537`). The per-package times the preflight logs print, and what git shows changed in loop's test binary since the previous log that ran tests:

| PR | Cacheable run | Always-rerun run | Race stage time | Changed since the previous test-running log |
|---|---|---|---|---|
| #1662 | 1802.820s | 252.322s | not printed | eight of loop's non-test files (against #1657's d37c28ce) |
| #1672 | 1735.326s | 251.534s | 38m41s | one of loop's test files, `handback_doc_test.go` |
| #1705 | 1962.647s | 255.908s | 43m16s | one of loop's test files, `itd50_verdicts_doc_test.go` |
| #1706 | 1813.441s | 251.280s | 40m46s | none of loop's files; seven non-test files of `internal/core/site`, which loop imports |
| #1707 | (cached) | 253.027s | 11m20s | nothing loop's test binary is built from |
| #1709 | 1794.410s | 245.846s | 39m35s | four of loop's non-test files: `narrow.go`, `reservation.go`, `schedule.go`, `widen.go` |
| #1710 | 1715.099s | 252.865s | 39m04s | loop's `queue.go` against #1707's head; against #1709's, those four files as well, since the two heads are on sibling branches |

Each comparison is `git diff --stat` between the two heads over `internal/core/implement/loop`. #1657 is the oldest log in the set and has no earlier one to compare with, and the records-only landings #1659 and #1702 ran no tests, so they filled no cache. For #1706, `go list -deps ./internal/core/implement/loop` at d0596e67 lists `internal/core/site` (imported through `capture` and `reading`), so a change to `site`'s non-test files changes loop's test binary. Between #1706's d0596e67 and #1707's 402f6e00 only `_test.go` files of `ahoy`, `interview`, `lint` and `surface/cli` changed, none of which enters loop's test binary, so #1707 replayed the run #1706 had just cached.

So the misses are Go's test cache doing its job, not a defect in the cache key: every miss follows a change to loop's own files or to a package it imports. The cost is structural. Loop's cacheable run takes 1715s to 1962s under `-race`, and in six of the seven landings above loop's test binary changed, five of them through loop's own files. The always-rerun run took about 250s in all seven, the cached one included, so it does not hold the difference.

A related finding: check-tests' reason line for `implement/loop` gives the same always-rerun reason whether or not the cacheable part reran ("42 always-rerun tests, as TestACommandPathIsOneAShellTakes undeclared" in both #1707 and #1709), so the line hides a full rerun of the package.
