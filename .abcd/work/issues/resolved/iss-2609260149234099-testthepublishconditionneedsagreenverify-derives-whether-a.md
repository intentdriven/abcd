---
schema_version: 1
id: "iss-2609260149234099"
slug: "testthepublishconditionneedsagreenverify-derives-whether-a"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25 (fix round, review of lane launchkind)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/scaffold/tagorder_workflow_test.go"
resolution: "The publish-condition test takes the gate flag from the rendering profile and evaluates the build job too; deleting either inputs.publish clause now fails it."
impact: internal
resolved_by:
  commit: "98a77f3d"
---

TestThePublishConditionNeedsAGreenVerify derives whether a source is the reusable gate from the condition under test (gate := strings.Contains(cond, "inputs.publish")), so deleting the '&& inputs.publish' clause from the gate profile's build and release jobs in release.yml.tmpl leaves the suite green; a caller that passes publish: false with contents: write would then get a Release from the gate before its own build. The expectation must come from the profile that rendered the source.

## Grounds

- pursued: a gate-profile template missing '&& inputs.publish' on its build or release job fails TestThePublishConditionNeedsAGreenVerify; a mutant that removes either clause and still passes would show it wrong
