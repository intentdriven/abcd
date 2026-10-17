---
schema_version: 1
id: "iss-2610161457198410"
slug: "the-rules-package-s-load-idiom-test"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "crash recovery on 2026-10-16 (abcd-22/abcd-a2)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/load_test.go"
remedy: "Before the probe, run pgrep once on the test's own pid. If it cannot list processes, skip naming that reason; otherwise print pgrep's stderr when the wait gives up."
---

The rules package's load-idiom test fails with exit status 3 and an empty stderr when pgrep runs but cannot read the process list. After the 2026-10-16 screen-server restart, every session started before it got 'pgrep: Cannot get process list' (sysmond unreachable). The test only checks that pgrep is on PATH, then waits ten seconds for two processes pgrep can never see, so make preflight failed with no reason given.
