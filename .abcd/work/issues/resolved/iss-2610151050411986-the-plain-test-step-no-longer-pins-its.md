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
resolution: "TestPlainTestStepRunsTheWholeModule now reads the check job's plain Test step through plainTestStepWords, which refuses every word starting with - but one -count=1 and one -timeout, so a -short, -run, -skip or -p added to the step fails the test as the old exact match did. TestPlainTestStepRefusesAnUnpinnedFlag feeds it the narrowed commands and failed on each before the change."
impact: internal
resolved_by:
  commit: "95d40969a7b6c18f2e9b728554937ee90acd8a71"
---

After the CI speed lane (#1014) let the plain test step name its slowest packages first, the test that guards that step drops every word starting with - before comparing package sets. A plain step changed to go test -short -run=TestA ... ./... now passes both TestMacOSLegRunsItsSuiteOnlyWhereItLands and TestPlainTestStepRunsTheWholeModule, where the old exact match refused it (review note, 2026-10-15).

## Grounds

- pursued: a plain Test step in .github/workflows/ci.yml carrying any flag other than -count=1 and -timeout fails TestPlainTestStepRunsTheWholeModule, while the step as it stands passes; a narrowed step such as go test -short -run=TestA -count=1 -timeout 70m ./... passing that test would show it wrong.
