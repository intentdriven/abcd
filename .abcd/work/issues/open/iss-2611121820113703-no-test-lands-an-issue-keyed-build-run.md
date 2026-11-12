---
schema_version: 1
id: "iss-2611121820113703"
slug: "no-test-lands-an-issue-keyed-build-run"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "intent audit of the landings record"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land_admitted_test.go"
remedy: "Add an issue-keyed run to the test that lands a build lane and asserts the trailer on the landed records commit."
---

No test lands an issue-keyed build run and reads the admission trailer off its records commit: the issue form is shown by the start-state test and a unit test of the trailer composer only. The intent audit of the landings record marked its second criterion met with concerns for this.
