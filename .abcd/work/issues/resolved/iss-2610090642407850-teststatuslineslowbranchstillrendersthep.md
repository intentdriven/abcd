---
schema_version: 1
id: "iss-2610090642407850"
slug: "teststatuslineslowbranchstillrendersthep"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "2026-10-07/08 autonomous drain and v0.13.3 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/statusline_deadline_test.go"
remedy: "Reproduce TestStatuslineSlowBranchStillRendersThePartialRow under load with -count and -race, and give it the same bound the sibling fix used: prove the partial row rendered and the slow git was stopped, with a generous ceiling instead of the production deadline."
resolution: "TestStatuslineSlowBranchStillRendersThePartialRow, already on the sibling fix's widened budgets and stopCeiling, now also proves the slow branch git was killed (its pid gone within stopCeiling); fakeGit primes gitutil's once-per-process git version probe with the real git, so a hanging fake can no longer hold every git in the process when the statusline tests run on their own."
impact: internal
resolved_by:
  commit: "360c3ab11e3f4a16f4cd82b4300af354cb6f3841"
---

TestStatuslineSlowBranchStillRendersThePartialRow failed once at 0.34s in a local landing preflight under a load average above 50; a retry passed. It is the sibling of TestStatuslineSlowPreviousCommandIsStopped, whose wall-clock budget was fixed in iss-2610080243367006, but that fix did not reach this test, so the same family can still fail under a loaded host.

## Grounds

- pursued: the test passes under -race -count on a loaded host and fails if a git at the deadline is abandoned rather than killed; a failure at abcd's 4s overall ceiling, or a branch git pid still alive after the verb returns, would show it wrong.
