---
schema_version: 1
id: "iss-2610151050411986"
slug: "the-plain-test-step-no-longer-pins-its"
severity: "minor"
category: "tech-debt"
source: "agent-observation"
found_during: "v0.15.0 run 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint"
remedy: "Have TestPlainTestStepRunsTheWholeModule refuse any word starting with - in the plain test step's command, so a -short or -run added to the required plain lane fails the test as the old exact match did."
---

After the CI speed lane (#1014) let the plain test step name its slowest packages first, the test that guards that step drops every word starting with - before comparing package sets. A plain step changed to go test -short -run=TestA ... ./... now passes both TestMacOSLegRunsItsSuiteOnlyWhereItLands and TestPlainTestStepRunsTheWholeModule, where the old exact match refused it (review note, 2026-10-15).
