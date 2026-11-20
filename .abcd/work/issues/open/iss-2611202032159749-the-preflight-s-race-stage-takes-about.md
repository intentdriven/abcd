---
schema_version: 1
id: "iss-2611202032159749"
slug: "the-preflight-s-race-stage-takes-about"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run of 20 Nov 2026, measuring step 3 of spc-2611012047503720"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/research/notes/2026-11-20-preflight-reuse-over-ten-landings.md"
remedy: "Find why internal/core/implement/loop's cacheable run (the -skip run, internal/testreuse/run.go:504) misses Go's test cache when check-tests' plan reuses it. A likely place to start is an input the test reads that the cache key sees changing from run to run, such as an environment variable, a file's modification time or a path. Fix it, then time the race stage over ten landings again."
---

The preflight's race stage takes about 39 minutes whenever one package's cacheable run misses Go's test cache, and 11m20s in the one landing where it hit. check-tests runs `internal/core/implement/loop` twice in the race stage: a cacheable run in the check directory that skips the package's always-rerun tests (`internal/testreuse/run.go:504`), and an always-rerun run with `-count=1` in the worktree (`internal/testreuse/run.go:537`). The per-package times the preflight logs print show where the stage's time goes:

| PR | Cacheable run | Always-rerun run | Race stage time |
|---|---|---|---|
| #1672 | 1735.326s | 251.534s | 38m41s |
| #1705 | 1962.647s | 255.908s | 43m16s |
| #1706 | 1813.441s | 251.280s | 40m46s |
| #1707 | (cached) | 253.027s | 11m20s |
| #1709 | 1794.410s | 245.846s | 39m35s |
| #1710 | 1715.099s | 252.865s | 39m04s |

The cacheable run missed the cache in five of the six timed landings, taking 1715s to 1962s, and was `(cached)` in #1707. The always-rerun run took about 250s in all six, the fast one included, so the always-rerun tests do not hold the difference. The two earlier landings that printed no stage time, #1657 and #1662, missed in the same way (1806.038s and 1802.820s).

A related finding: check-tests' reason line for `implement/loop` gives the same always-rerun reason whether or not the cacheable part reran ("42 always-rerun tests, as TestACommandPathIsOneAShellTakes undeclared" in both #1707 and #1709), so the line hides a full rerun of the package.
