---
schema_version: 1
id: "iss-2611170410462663"
slug: "while-the-person-is-away-the-stop-check"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "[redacted-user] session waiting on the v0.17.0 tag"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/owedread.go"
remedy: "Have the stop check read the away record's queue and report an owed item whose step is queued as waiting on the person, not owed, while leaving it in the status the person reads on return"
---

While the person is away, the stop check keeps naming a branch push as owed after the session has queued that push for the person with abcd mode queue, so the stop can never release. On 2026-11-17 from 03:30 GMT the [redacted-user] session's stop check named branch records-run-1104b, which another session made and whose push is the person's act, on every turn end for over half an hour; queuing it at 03:44 ('queued for the person's return (16 queued)') changed nothing, and once 'the set has not moved over two wakes' a wake no longer released the stop either, so the session spent dozens of turns answering the same line while its real work waited on a release tag. The hook's own text says a push the session may not send 'stays owed: queue it … and take another item meanwhile', but when no other item is left the queue is the end of what the session can do, and the check does not read it.
