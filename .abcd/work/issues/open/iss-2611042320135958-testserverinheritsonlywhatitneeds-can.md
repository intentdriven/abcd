---
schema_version: 1
id: "iss-2611042320135958"
slug: "testserverinheritsonlywhatitneeds-can"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "a cloud session's preflight, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/dashboard/control_test.go"
remedy: "Have the fake server read its configuration before it reports, or have start prefer a server's own exit report over the failed write, with a test that holds the ordering."
---

internal/surface/dashboard's TestServerInheritsOnlyWhatItNeeds failed once under load with a broken pipe, then passed, in a cloud session's preflight on 2026-11-04: the fake server reports its environment and exits, and when it exits before start has handed it its configuration, start's refusal appears to name the failed write instead of the report the test reads.
