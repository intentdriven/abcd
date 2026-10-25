---
schema_version: 1
id: "iss-2610241829569682"
slug: "testreadowedhonoursdeadline-internal"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "abcd build itd-82 session, 2026-10-24, preflight of fix/landing-marks-step-landed"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/owedread.go"
remedy: "reproduce it under a CPU burner capped below the core count; if a read the forge did not answer marks no incomplete, set incomplete whenever any part of the read is cut short or unread and say which in the notes; if only the test races, drive its deadline from an injected clock rather than wall time"
wontfix_reason: "the same failure as iss-2610241911001655, which branch chore/capture-holdups-3fee diagnoses and fixes (a git read failing past the deadline now marks the owed-work reading incomplete); it is followed there"
---

TestReadOwedHonoursDeadline (internal/core/autonomy/owedread_test.go:1198) failed in a preflight on a loaded machine on 2026-10-24 (load near 20, other preflights running): the read it cut short came back with incomplete false while its notes said 'the local branches could not be listed | the pull requests were not read (the forge did not answer within 10s), so the check judged from disk', so either a read cut short under load does not mark itself incomplete, or the test's deadline races the machine. A test that fails under load blocks every preflight that meets it, whatever the branch changed.

## Grounds

- declined: the same failure as iss-2610241911001655, which branch chore/capture-holdups-3fee diagnoses and fixes (a git read failing past the deadline now marks the owed-work reading incomplete); it is followed there
