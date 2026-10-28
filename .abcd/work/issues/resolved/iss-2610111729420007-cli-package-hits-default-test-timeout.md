---
schema_version: 1
id: "iss-2610111729420007"
slug: "cli-package-hits-default-test-timeout"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "security triage and fix run, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "Give preflight's go test of internal/surface/cli an explicit -timeout sized to its measured worst case under concurrent load (as the race lane already carries -timeout 20m), or split the slow statusline tests into their own package; prove by recording the package's wall time at the measured load and the new margin"
resolution: "The unit test stage of make preflight now runs with -timeout 20m, the race stage's package budget, for every package, after internal/core/implement/loop hit the 10-minute default on a loaded machine as the cli package had."
impact: internal
resolved_by:
  commit: "2e4bccb7b"
---

make preflight's unit pass runs `go test` with the default ten-minute package timeout, and internal/surface/cli exceeded it on 2026-10-11 (panic: test timed out after 10m0s, in TestStatuslineHungGitDoesNotFreezeTheLine) while the machine's five-minute load average was 133 with several sessions running preflights and sub-agents at once. Every other package passed and the branch changed no Go code, so the failure was load, not the change; a second run was started at load 48. Concurrent sessions are the normal way this repository is worked on, so a load-sensitive package timeout fails preflight for nothing and costs a full re-run.

2026-10-15, a further occurrence: a plain `go test ./...` on the feat/abcd-land branch timed out at ten minutes in four packages, internal/core/implement/loop, internal/core/reading, internal/core/site and internal/surface/cli, under concurrent load. All four passed with `-timeout 40m` (468s, 209s, 218s, 448s). So the default timeout is short for more packages than internal/surface/cli alone.

## A second package, 2026-10-17

The same failure reached a second package. On 2026-10-17 a preflight's unit stage killed `internal/core/implement/loop` at go test's 10-minute default (600.3s), with no test hung: the test running at the alarm was one second old. The same package had taken 346s in a preflight the evening before, and the machine was busier that morning, now that two preflights can run their test stages at once. The class is any slow package in the unit stage under load, not the cli package alone, so the fix gives the whole stage the race stage's 20-minute budget.

2026-10-23, a measurement against the 20-minute budget: a hand run of `go test ./internal/surface/cli` on go's ten-minute default (not preflight) stopped at 597s under concurrent load. The package grew to sit at about ten minutes under load, which is half the 20-minute preflight and race budgets, so the budget still holds with the margin it was sized for. Nothing to change. A hand run of this package under load takes `-timeout 20m` as preflight does.

## Grounds

- pursued: a 20-minute budget covers the slowest package under the load two concurrent preflights put on the machine; shown wrong if a unit-stage package is killed at 20 minutes with no test hung
