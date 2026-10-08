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
---

TestStatuslineSlowPreviousCommandIsStopped asserts the status line returns within 300ms; on the CI macOS runner it returned in 326ms for a change that did not touch it (PR #857, run 37707291322), so a fixed wall-clock budget under a race-instrumented shared runner makes it flaky.
