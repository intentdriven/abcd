---
schema_version: 1
id: "iss-2610080243367006"
slug: "teststatuslineslowpreviouscommandisstopped-asserts-the"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "CI for PR #857, 2026-10-08"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/statusline_deadline_test.go"
remedy: "Replace the fixed 300ms wall-clock assertion in TestStatuslineSlowPreviousCommandIsStopped with a bound that proves the slow command was stopped (it did not run to completion; the return came well before the slow command's own duration) plus a generous ceiling, so a race-instrumented shared runner cannot fail it on scheduling jitter."
resolution: "The statusline deadline tests no longer assert a fixed wall-clock budget. Each slow fixture runs for 30s and the verb must return under a 10s ceiling. The note naming the budget proves which bound stopped it, and a pid poll proves the killed process is gone. The recorded CI failure was really the previous command's shell writing kill reports to stderr ahead of abcd's note, and the note check now allows that. The sibling tests that need real git to answer inside the 300ms git budget (SlowBranch, RootCommit, SlowPrevious, HungGit) widen the budgets for their run. The production values are unchanged and now pinned exactly."
impact: internal
resolved_by:
  commit: "a104ed4992668d3d777eec94671c6942d4bd553c"
---

TestStatuslineSlowPreviousCommandIsStopped asserts the status line returns within 300ms; on the CI macOS runner it returned in 326ms for a change that did not touch it (PR #857, run 37707291322), so a fixed wall-clock budget under a race-instrumented shared runner makes it flaky.

## Grounds

- pursued: expected that TestStatusline* pass repeatedly with and without -race, including when git answers 400ms late, and that the tests still fail when a production bound is removed (group kill, previous-command timeout or git timeout, each mutated on a scratch copy). A CI or loaded-machine failure of any statusline deadline test on timing alone would show it wrong.
