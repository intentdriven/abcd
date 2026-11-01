---
schema_version: 1
id: "iss-2611011153130169"
slug: "testvulncheckinterruptedleavesnocopybehi"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "merge queue run for #1319"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/preflightscans_test.go"
remedy: "Close the stub's fork window: write the marker and then exec sleep 30 (or re-send SIGINT every few hundred milliseconds until the run ends), keeping the 20-second wait strictly below the stub's sleep and the assertion that no copy is left behind; never widen the wait past the sleep, which would let a scan that ignored the interrupt pass, and leave scripts/preflight-scans.sh's INT trap as it is."
---

TestVulncheckInterruptedLeavesNoCopyBehind (internal/core/lint/preflightscans_test.go) is flaky: it fails with 'the interrupted scan did not end' when its SIGINT races the stub's own fork. The stub `go` writes its started marker with the builtin `:` and only then forks `sleep 30`, so a signal sent in that fork-to-exec window misses the sleep, which is orphaned holding the command-substitution pipes in scripts/preflight-scans.sh open until it ends at 30 seconds, past the test's 20-second wait. The script's interrupt path is not at fault: every slow run still exited 130 through its INT trap with the copy removed. Reproduced locally in 18 of 150 runs with no load and no race detector, each ending at about 30.0 seconds; with the stub changed to `exec sleep 30`, none of 400 runs was slow. On 2026-11-01 it dropped a records-only pull request (#1319) from the merge queue in the macOS race run (Actions run 41298480855), though nothing in that change reaches the lint package. A measurement must use the bash the test runs, which is the first on the test process's PATH, not the stub PATH's.
