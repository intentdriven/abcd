---
schema_version: 1
id: "iss-2609240646544930"
slug: "a-join-before-the-window-mode-counts-in-the-last-window"
severity: "nitpick"
category: "ux"
source: "agent-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/report.go"
---

`abcd implement report` gives each event to the window open when it happened, so a session that joins a second before the first session sets the window's mode is counted in the previous window. In autonomous run A the second session's session_open landed one second before the first session's `implement mode claim --window 2`, and the report lists that session under `single` with no lanes while its one lane, landed, sits under `claim`. Wanted: a join is attributed to the window whose window_mode line follows it within a short grace, or the report names a session whose open and whose work fall in different windows.
