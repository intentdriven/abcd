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
---

make preflight's unit pass runs `go test` with the default ten-minute package timeout, and internal/surface/cli exceeded it on 2026-10-11 (panic: test timed out after 10m0s, in TestStatuslineHungGitDoesNotFreezeTheLine) while the machine's five-minute load average was 133 with several sessions running preflights and sub-agents at once. Every other package passed and the branch changed no Go code, so the failure was load, not the change; a second run was started at load 48. Concurrent sessions are the normal way this repository is worked on, so a load-sensitive package timeout fails preflight for nothing and costs a full re-run.
