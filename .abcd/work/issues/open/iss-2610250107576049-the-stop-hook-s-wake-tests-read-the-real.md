---
schema_version: 1
id: "iss-2610250107576049"
slug: "the-stop-hook-s-wake-tests-read-the-real"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "records-branch preflight, 2026-10-25 01:00-02:00 UTC"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/hook_stop_test.go"
remedy: "Give the stop-hook tests a pinned clock and location (the hook already takes loc; thread a now through the test seam), add a case set inside a fall-back's repeated hour and one inside a spring-forward gap, and decide out loud whether a recurring task's wake should match the repeated hour's second occurrence, since the host may run it then."
---

The stop hook's wake tests read the real clock in the machine's local zone, so they fail during a daylight-saving fall-back's repeated hour. Cron.Next (internal/core/autonomy/cron.go) deliberately never matches a repeated wall-clock minute, so between 02:00 and 03:00 CET on 2026-10-25 a */10 or */30 task had no match until 03:00. TestHookStopOutput, TestHookStopWakeCountAcrossRuns and TestHookStopRecordsHandBack (internal/surface/cli/hook_stop_test.go) then failed in make preflight on every branch, at any load. They pass with TZ=UTC and again after the hour, while CI (UTC) stays green. The hook itself shares the behaviour: for that hour a session sees no qualifying wake for any recurring task.
