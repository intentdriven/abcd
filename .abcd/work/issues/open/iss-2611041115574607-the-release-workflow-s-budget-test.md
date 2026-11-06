---
schema_version: 1
id: "iss-2611041115574607"
slug: "the-release-workflow-s-budget-test"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "the v0.16.1 cut, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/release.yml"
remedy: "Sum each profile's go test limits in the test, and add the watchdog ci.yml's race lane uses."
---

The release workflow's budget test checks each go test step alone, not cumulatively, so a scaffolded repository's Test step plus a hung race package can pass it while its 15-minute job is cancelled with no goroutine dump; and the verify job has no watchdog, so a late race package in abcd's own profile can pass the job's limit the same way.
