---
schema_version: 1
id: "iss-2610090642394550"
slug: "testmodemadequestionabcdssaysso-failed-once-on-the-macos-leg"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "2026-10-07/08 autonomous drain and v0.13.3 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/guard_question_limits_test.go"
remedy: "Run TestModeMadeQuestionAbcdsSaysSo under -race with -count=200 to reproduce; find whether the hook can exit before flushing its one output object and make it write and flush before exit, or make the test wait for the process to finish before reading."
---

TestModeMadeQuestionAbcdsSaysSo failed once on the macOS leg of PR #858 with 'stdout is not one hook output object: EOF' after 0.11s, in a change that never touched the guard; a re-run passed. The test arrived with #849. An empty stdout from the question hook reads as a race between the hook's output and the test's read.
