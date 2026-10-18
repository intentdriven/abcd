---
id: dcn-2610180544351869
slug: the-check-job-s-ubuntu-leg-stops
date: 2026-10-18
---
2026-10-18 — The check job's ubuntu leg stops rerunning the race lane's tests in its plain pass, decided by the coordinating session on delegation from the technical facilitator, who is away until 2026-10-23 (iss-2610171803053186). The race step's CPU was saturated, at about 3.4 of the runner's four cores, so sharding its slowest package would not have shortened it. The plain pass reran 106 of its 111 tested packages, the ones under ./internal/... that the race lane runs anyway, at 7 to 9.3 minutes of a leg that reached 31. The precedent is the macOS leg's single pass (iss-2610110938293273). The condition: the tests that assert only without the race detector, cost guards that skip under -race or drop their CPU bound there, must still run in CI. So where the race lane runs, the plain pass covers ./cmd/... and, by name, the 43 such tests, a list derived from the test files and held by a test. On an inert change the plain pass still runs the whole module, since the race lane stands down there. The ceiling returns to 30, and macOS keeps its 40-minute race -timeout, which nothing here changes.
