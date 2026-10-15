---
id: dcn-2607080000000012
slug: severity-priority-records-an
date: 2026-07-08
---
2026-07-08 — Severity ≠ priority (records an earlier-session decision):
intents declare `severity` (capture-ledger enum) and edges (`blocked_by`,
`builds_on`); effective priority is DERIVED via priority inheritance (max of
own severity and severity of everything transitively blocked) and never
stored — a minor blocker of a major intent jumps the queue while staying
minor. Phases keep sequencing authority (adr-9); lint makes contradictory
schedules fail. Recorded as itd-78; piloted on itd-76/77.
