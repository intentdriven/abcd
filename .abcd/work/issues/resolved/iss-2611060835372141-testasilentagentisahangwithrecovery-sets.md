---
schema_version: 1
id: "iss-2611060835372141"
slug: "testasilentagentisahangwithrecovery-sets"
severity: "minor"
category: "tech-debt"
source: "agent-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/terminal_longrun_test.go"
remedy: "Apply the short limit to the hanging launch alone."
resolution: "TestASilentAgentIsAHangWithRecovery now drives in two runs: the first, under the 200ms hang limit, ends at its bound once the hang has taken its script line; the second resumes under the ordinary 5-minute limit and lands the lane. A test-local transcript store asserts every finished launch ran under the ordinary limit."
impact: internal
resolved_by:
  commit: "a1def8d883b458608a485bf0f145942a778934d5"
---

TestASilentAgentIsAHangWithRecovery sets a 5-second runner limit that also covers the ordinary launches after the hang (`terminal_longrun_test.go:790`), so a loaded machine could kill one of those and fail the check for exactly one suspect file. It has not been seen failing; a failure would cost a queue entry.

## Grounds

- pursued: no launch that ends by itself runs under the short limit any more, so a loaded machine cannot fail the check by killing one; the detector failing again (a finished launch recorded under a limit other than ordinaryLaunchLimit), or the test taking minutes because a hang reached the second run, would show it wrong
