---
schema_version: 1
id: "iss-2610241157493381"
slug: "testhooksessionendretriesatransientrootf"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "setup redesign build, shell-settings lane test run, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/hook_subagent_stage_retry_test.go"
remedy: "give the retry a budget the test controls (an injectable clock or deadline), so the test proves one retry happens regardless of machine load"
wontfix_reason: "Duplicate of iss-2610241235133916, which records the same session-end retry test failure and is resolved on main."
---

TestHookSessionEndRetriesATransientRootFailure fails on an unchanged branch from origin/main (2026-10-24): the session-end hook's retry after a simulated transient git failure finds 'no time is left to ask again' and stages nothing, so the retry budget is spent before the retry on a loaded machine (several concurrent sessions and test runs); the test asserts timing-dependent behaviour against wall-clock time.

## Grounds

- declined: Duplicate of iss-2610241235133916, which records the same session-end retry test failure and is resolved on main.
