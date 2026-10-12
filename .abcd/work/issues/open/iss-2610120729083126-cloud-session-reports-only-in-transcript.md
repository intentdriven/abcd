---
schema_version: 1
id: "iss-2610120729083126"
slug: "cloud-session-reports-only-in-transcript"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "release preparation with cloud sessions, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "cloud session reporting"
remedy: "Give cloud sessions a report channel abcd reads, e.g. a draft pull request or a report file on the work-in-progress namespace, or a comment on a tracking issue, written whether the run succeeded or stopped; and an abcd view that lists those reports"
---

A cloud session's stop report (what it did, what failed, what it needs) is visible only in its transcript at claude.ai/code: on 2026-10-12 the cloud sessions could not message the local sessions and pushed nothing, so the person had to copy one session's report into a local session by hand, and the other two sessions' reports are still unread.
