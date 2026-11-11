---
schema_version: 1
id: "iss-2611051719381143"
slug: "an-armed-pull-request-blocked-or-dropped"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "build run, 5 November"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/board.go"
remedy: "The status board names an armed pull request whose required check failed, with the check's name."
---

An armed pull request whose required check failed before it reached the queue stays armed and blocked. `abcd land` names the failed check only when it is run again, and the status board never does, so nothing says so between land runs and a queue watch sees an empty queue either way. This is the part before the queue. A drop from the queue is iss-2611022247147314, and retrying a drop that is not the pull request's own fault is iss-2610110401224302.

2026-11-09: seen again at 18:53 in the autonomous run of 7 Nov: the required external-review check failed to start on #1473 and on #1474 (startup_failure, no jobs, no change to main's workflows), and both sat armed and not queued while land reported them armed; it was noticed an hour later, only by comparing main's required checks with those the pull requests reported. #1474's run re-ran clean; a startup failure cannot be re-run, so #1473 was closed and reopened and land re-armed it, restarting every check, about 45 minutes lost.
