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
resolution: "Added TestIssueKeyedBuildLaneCommitsItsIssue, which starts a build keyed on an issue, lands its lane and reads the pushed records commit: its trailers are the resolution, Admitted-by: build <issue> and the disclosure, in that order, with no Refs: line, and the pull request's body carries the same line. With the records step made to drop the admission for an issue-keyed run, this test fails while the spec-keyed landing test and the trailer composer's unit test still pass."
impact: fix
resolved_by:
  commit: "73209098811b19d65ded8ea9855f4a3dc237146e"
---

No test lands an issue-keyed build run and reads the admission trailer off its records commit: the issue form is shown by the start-state test and a unit test of the trailer composer only. The intent audit of the landings record marked its second criterion met with concerns for this.
