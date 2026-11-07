---
schema_version: 1
id: "iss-2611051727550516"
slug: "test-reuse-has-no-test-for-losing-the"
severity: "minor"
category: "tech-debt"
source: "agent-observation"
found_during: "build run, 5 November"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/testreuse/run_test.go"
remedy: "Add a test in `internal/testreuse/run_test.go` that loses the lock on retake after the race gate and asserts the stage goes back to the gate."
---

Test reuse has no test for a stage that loses the check directory's lock on retake after passing the race gate (`internal/testreuse/run.go`), so that path can change unwatched.
