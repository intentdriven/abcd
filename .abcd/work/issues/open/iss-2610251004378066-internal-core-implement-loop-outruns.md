---
schema_version: 1
id: "iss-2610251004378066"
slug: "internal-core-implement-loop-outruns"
severity: "major"
category: "process"
source: "agent-finding"
found_during: "preflight of fix/unit-stage-budget, 2026-10-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "Let the package run as several test processes: its TestMain, when make preflight exports ABCD_TEST_SHARDS=n and go test asked for the whole package, lists its tests, runs n copies of its own binary at once, each with a share of them under -test.run and its own -test.timeout, and passes the combined result back to go test, so each process fits the budget, go test's cache and the pinned recipe lines are untouched, and CI, which sets no such variable, runs exactly as before. Prove with a test that every test runs exactly once across the processes and that a failing one fails the run."
---

internal/core/implement/loop outruns even the unit stage's new 40-minute package budget on this Mac: its 505 tests run one after another in one test binary, and on 2026-10-25 a preflight killed it at 40m00s with no test hung (the test running at the alarm was three seconds old), after a gated run of the package alone took 3590s at a load of about 13. The time is spread across the suite, not in a few tests: the ten slowest are 17% of it and the median test takes 4.4s, so no single fix shortens it. On CI's dedicated macOS runner the same package takes 1830s under -race, 76% of that budget. Almost every test sets HOME and git's environment through t.Setenv in the shared git fixture (gittest.NewRepo), and the production code reads them from the process, so the tests cannot run in parallel without threading the environment through about 30 to 40 files.
