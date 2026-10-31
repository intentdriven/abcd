---
schema_version: 1
id: "iss-2610310054494441"
slug: "runtoeof-misses-a-context-stop-when"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "lane A, #1284 race-lane failure read by lane E"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/procgroup/run.go"
remedy: "After Wait, treat a failing command whose context has ended as cut, so its error wraps the context's own; pin it with a test whose wait never sees the context end."
resolution: "RunToEOF now treats a failure Wait reports once ctx has ended as cut, so a command its context stopped wraps ctx's error even when the killed pipes drain before the wait sees ctx end. TestRunToEOFStopsACommandThatHangsWhosePipesCloseFirst forces that order with a context whose Done never fires; it failed first with the CI message (signal: killed) and passes now. Every RunToEOF caller was checked: those that read an exit code check ctx.Err() first, the rest pass the error up as a failure."
impact: fix
---

procgroup.RunToEOF reports a command its context stopped as an ordinary failure when the command's pipes close before RunToEOF's wait sees the context end: exec kills the command, the pipes close, the drained branch wins the second select, cut stays false and the bare exit error (signal: killed) is returned without the context's error. internal/procgroup TestRunToEOFStopsACommandThatHangs fails on it intermittently under load and drops merge-queue entries.
