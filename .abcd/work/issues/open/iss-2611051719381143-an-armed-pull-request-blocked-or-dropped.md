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
