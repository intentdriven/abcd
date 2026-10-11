---
schema_version: 1
id: "iss-2610090642394550"
slug: "testmodemadequestionabcdssaysso-failed"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "2026-10-07/08 autonomous drain and v0.13.3 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/guard_question_limits_test.go"
remedy: "Run TestModeMadeQuestionAbcdsSaysSo under -race with -count=200 to reproduce; find whether the hook can exit before flushing its one output object and make it write and flush before exit, or make the test wait for the process to finish before reading."
resolution: "The guard hook runs in-process in the test, so there was no unflushed output. The empty stdout was questionGate admitting the chipless question silently after mode.Root failed, because git's toplevel answer missed the 50ms WaitDelay that f1703a29f (iss-2610100846469473) removed from deadline-free lookups. A regression test now holds git's pipe on every toplevel call and still requires the deny."
impact: internal
resolved_by:
  commit: "103dabf8771e1846c457816d485908f091d1ca79"
---

TestModeMadeQuestionAbcdsSaysSo failed once on the macOS leg of PR #858 with 'stdout is not one hook output object: EOF' after 0.11s, in a change that never touched the guard; a re-run passed. The test arrived with #849. An empty stdout from the question hook reads as a race between the hook's output and the test's read.

## Grounds

- pursued: TestModeMadeQuestionAbcdsSaysSo stays green under -race -count=200 (200/200 at base), and TestModeMadeQuestionSurvivesAGitHoldingItsPipe fails with CI's exact EOF message if the unconditional WaitDelay returns; a future EOF from that test with the WaitDelay fix in place would show a second cause
