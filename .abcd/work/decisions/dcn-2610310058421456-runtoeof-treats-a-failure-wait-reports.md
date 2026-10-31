---
id: dcn-2610310058421456
slug: runtoeof-treats-a-failure-wait-reports
date: 2026-10-31
---
2026-10-31 — RunToEOF treats a failure Wait reports once its context has ended as cut by the context (iss-2610310054494441), rather than wrapping the command's Cancel to record whether exec stopped it. The Cancel wrapper is not exact either: a kill that reaches a command which has exited but is not yet reaped still succeeds, so it shares the same window, and it would mutate the caller's Cancel and add a second path for a command built without one; the ctx.Err() check is one condition. Its cost is that window, up to the scheduling delay before the check under load: a command that failed on its own just as its context ended reads as stopped, and its exit text is still in the message. Every caller that reads an exit code already checks its context first (ctx.Err(), or DeadlineExceeded on a deadline-only context), so none loses an answer.
