---
schema_version: 1
id: "iss-2610270257082620"
slug: "testthestoredlinereadendswithitscontext"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/stored_each_test.go"
remedy: "Hold the test to the property it exists for: a read whose context has already ended starts no git, so count the runs before and after that read and require no new one, with at most one for the first read."
resolution: "The test holds the property it exists for: the read whose context has already ended starts no git (the run count is the same before and after it), with at most one run for the first read, whatever the first read's deadline allowed."
impact: internal
---

TestTheStoredLineReadEndsWithItsContext (internal/gitutil/stored_each_test.go) required the stubbed git to have run exactly once. Its first read has a 300ms deadline, and on a loaded machine that deadline passed before the stub wrote its run marker, so the test counted 0 runs and failed a preflight whose change touched no gitutil code (fix/fork-approval-mirror, 2026-10-27).
